// Package operation is the pgx-backed ledger for async jobs (ADR-006).
//
// Responsibilities:
//   * Submit — handler writes PENDING, returns operation id.
//   * Claim  — worker atomically picks up a PENDING row, transitions to RUNNING.
//   * Progress / Result / Error — worker updates as it advances.
//   * Lookup — handler polls GET /operations/:id.
//
// Claims are serialized by `SELECT ... FOR UPDATE SKIP LOCKED`, so many worker
// replicas can poll simultaneously without stepping on each other.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

// ErrNotFound is returned when the requested id does not exist or is out of scope.
var ErrNotFound = errors.New("operation not found")

// Repository is a pgx-backed store for operations.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Submit inserts a PENDING operation and returns it.
func (r *Repository) Submit(ctx context.Context, op domain.Operation) (domain.Operation, error) {
	if op.OrganizationID == uuid.Nil {
		return domain.Operation{}, errors.New("organization_id is required")
	}
	if op.Kind == "" {
		return domain.Operation{}, errors.New("kind is required")
	}
	if len(op.Input) == 0 {
		op.Input = []byte(`{}`)
	}
	if op.ActorType == "" {
		op.ActorType = "system"
	}
	const q = `
		INSERT INTO operations (
		  organization_id, actor_user_id, actor_type,
		  kind, status, input, retry_remaining,
		  scheduled_at, expires_at
		)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, scheduled_at
	`
	err := r.pool.QueryRow(ctx, q,
		op.OrganizationID, op.ActorUserID, op.ActorType,
		op.Kind, op.Input, op.RetryRemaining,
		defaultIfZero(op.ScheduledAt), op.ExpiresAt,
	).Scan(&op.ID, &op.CreatedAt, &op.UpdatedAt, &op.ScheduledAt)
	if err != nil {
		return domain.Operation{}, fmt.Errorf("submit operation: %w", err)
	}
	op.Status = domain.OperationPending
	return op, nil
}

// Lookup fetches a single operation scoped to the given org. Cross-tenant
// reads are impossible here by construction.
func (r *Repository) Lookup(ctx context.Context, orgID, id uuid.UUID) (domain.Operation, error) {
	const q = `
		SELECT id, organization_id, actor_user_id, actor_type,
		       kind, status, worker_id, input, result, error_payload,
		       progress, retry_remaining,
		       scheduled_at, started_at, ended_at, expires_at,
		       created_at, updated_at
		  FROM operations
		 WHERE id = $1
		   AND organization_id = $2
		 LIMIT 1
	`
	var op domain.Operation
	var status string
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&op.ID, &op.OrganizationID, &op.ActorUserID, &op.ActorType,
		&op.Kind, &status, &op.WorkerID, &op.Input, &op.Result, &op.ErrorPayload,
		&op.Progress, &op.RetryRemaining,
		&op.ScheduledAt, &op.StartedAt, &op.EndedAt, &op.ExpiresAt,
		&op.CreatedAt, &op.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Operation{}, ErrNotFound
	}
	if err != nil {
		return domain.Operation{}, fmt.Errorf("lookup operation: %w", err)
	}
	op.Status = domain.OperationStatus(status)
	return op, nil
}

// ClaimOne picks the oldest pending operation matching any of `kinds`, atomically
// transitioning it to RUNNING and stamping the worker id. Returns ErrNotFound when
// no claimable row is available — the worker should back off briefly.
//
// Uses `FOR UPDATE SKIP LOCKED` so multiple workers can poll concurrently.
func (r *Repository) ClaimOne(ctx context.Context, kinds []string, workerID string) (domain.Operation, error) {
	if len(kinds) == 0 {
		return domain.Operation{}, errors.New("kinds is empty")
	}
	if workerID == "" {
		return domain.Operation{}, errors.New("worker_id is required")
	}
	const q = `
		WITH pick AS (
		  SELECT id
		    FROM operations
		   WHERE status = 'pending'
		     AND kind = ANY($1::text[])
		     AND scheduled_at <= now()
		   ORDER BY scheduled_at
		   LIMIT 1
		   FOR UPDATE SKIP LOCKED
		)
		UPDATE operations o
		   SET status = 'running',
		       worker_id = $2,
		       started_at = now()
		  FROM pick
		 WHERE o.id = pick.id
		 RETURNING o.id, o.organization_id, o.actor_user_id, o.actor_type,
		           o.kind, o.input, o.retry_remaining, o.scheduled_at,
		           o.started_at, o.created_at, o.updated_at
	`
	var op domain.Operation
	err := r.pool.QueryRow(ctx, q, kinds, workerID).Scan(
		&op.ID, &op.OrganizationID, &op.ActorUserID, &op.ActorType,
		&op.Kind, &op.Input, &op.RetryRemaining, &op.ScheduledAt,
		&op.StartedAt, &op.CreatedAt, &op.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Operation{}, ErrNotFound
	}
	if err != nil {
		return domain.Operation{}, fmt.Errorf("claim: %w", err)
	}
	op.Status = domain.OperationRunning
	op.WorkerID = &workerID
	return op, nil
}

// UpdateProgress writes a progress value (float in [0,1]). Idempotent —
// the worker may call this multiple times per step.
func (r *Repository) UpdateProgress(ctx context.Context, id uuid.UUID, progress float64) error {
	if progress < 0 || progress > 1 {
		return fmt.Errorf("progress out of range: %f", progress)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE operations SET progress = $2 WHERE id = $1 AND status = 'running'`,
		id, progress,
	)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

// Succeed transitions the operation to SUCCEEDED with an optional result envelope.
// No-op if the operation is already terminal.
func (r *Repository) Succeed(ctx context.Context, id uuid.UUID, result any) error {
	payload := []byte(`null`)
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		payload = b
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE operations
		    SET status = 'succeeded',
		        result = $2,
		        progress = 1,
		        ended_at = now()
		  WHERE id = $1 AND status = 'running'`,
		id, payload,
	)
	if err != nil {
		return fmt.Errorf("succeed: %w", err)
	}
	return nil
}

// Fail transitions the operation to FAILED when retry budget is exhausted, or
// back to PENDING with retry_remaining-1 when the error is retryable and budget
// remains. The caller decides retryability.
func (r *Repository) Fail(ctx context.Context, id uuid.UUID, errPayload any, retryable bool) error {
	payload, err := json.Marshal(errPayload)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	// Atomic: decrement if retryable; flip to failed otherwise.
	const q = `
		UPDATE operations
		   SET error_payload = $2,
		       retry_remaining = CASE WHEN $3 AND retry_remaining > 0
		                              THEN retry_remaining - 1
		                              ELSE retry_remaining END,
		       status = CASE WHEN $3 AND retry_remaining > 0
		                     THEN 'pending'::operation_status
		                     ELSE 'failed'::operation_status END,
		       started_at = CASE WHEN $3 AND retry_remaining > 0 THEN NULL ELSE started_at END,
		       ended_at = CASE WHEN $3 AND retry_remaining > 0 THEN NULL ELSE now() END
		 WHERE id = $1 AND status = 'running'
	`
	if _, err := r.pool.Exec(ctx, q, id, payload, retryable); err != nil {
		return fmt.Errorf("fail: %w", err)
	}
	return nil
}

// Cancel is a tenant-initiated terminal transition. Only legal while the
// operation is pending or running.
func (r *Repository) Cancel(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE operations
		    SET status = 'cancelled', ended_at = now()
		  WHERE id = $1 AND organization_id = $2
		    AND status IN ('pending','running')`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("cancel: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func defaultIfZero(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

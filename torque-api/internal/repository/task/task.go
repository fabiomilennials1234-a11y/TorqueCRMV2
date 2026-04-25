// Package task persists F05 tasks (ADR-007 unified Task/Follow-up).
//
// A single invariant rules this package: per (organization_id, assigned_to)
// there is AT MOST ONE task in status='in_progress' at a time. That is
// enforced by the partial unique index `uq_tasks_one_in_progress_per_assignee`
// (migration 0003). Transitions here respect the invariant by using
// explicit WHERE status=... clauses so Postgres rejects the dup before
// any application code runs.
package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("task not found")
	ErrInvalidState   = errors.New("illegal task state transition")
	ErrAssigneeBusy   = errors.New("assignee already has an in_progress task")
)

// Task is the in-memory view.
type Task struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	LeadID          *uuid.UUID
	AssignedTo      uuid.UUID
	CreatedBy       *uuid.UUID
	Kind            string
	Title           string
	Description     *string
	Priority        string
	Status          string
	InQueue         bool
	QueuePosition   *int
	DueAt           *time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	CompletedBy     *uuid.UUID
	CancelledAt     *time.Time
	CancelledReason *string
	MissedReason    *string
	Origin          string
	Context         []byte // jsonb
	ResultNote      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// CreateInput is the payload for POST /tasks.
type CreateInput struct {
	OrganizationID uuid.UUID
	LeadID         *uuid.UUID
	AssignedTo     uuid.UUID
	CreatedBy      *uuid.UUID
	Kind           string
	Title          string
	Description    *string
	Priority       string
	DueAt          *time.Time
	Origin         string
	Context        json.RawMessage
}

// assigneeInTenant verifies the assigned_to member belongs to the caller's
// tenant. Prevents a cross-tenant member-id probe from creating rows whose
// organization_id points at org A while assigned_to points into org B's team.
func (r *Repository) assigneeInTenant(ctx context.Context, orgID, memberID uuid.UUID) error {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM team_members
		                 WHERE id = $1 AND organization_id = $2 AND is_active)`,
		memberID, orgID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verify assignee tenant: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

// Create inserts a pending task.
func (r *Repository) Create(ctx context.Context, in CreateInput) (Task, error) {
	if in.Title == "" {
		return Task{}, errors.New("title is required")
	}
	if in.AssignedTo == uuid.Nil {
		return Task{}, errors.New("assigned_to is required")
	}
	if err := r.assigneeInTenant(ctx, in.OrganizationID, in.AssignedTo); err != nil {
		return Task{}, err
	}
	if in.Kind == "" {
		in.Kind = "generic"
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	if in.Origin == "" {
		in.Origin = "manual"
	}
	var ctxJSON any
	if len(in.Context) > 0 {
		ctxJSON = []byte(in.Context)
	}
	const q = `
		INSERT INTO tasks (
		  organization_id, lead_id, assigned_to, created_by,
		  kind, title, description, priority, origin, due_at, context
		) VALUES ($1,$2,$3,$4,$5::task_kind,$6,$7,$8::task_priority,$9::task_origin,$10,$11)
		RETURNING id, created_at, updated_at
	`
	var t Task
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.LeadID, in.AssignedTo, in.CreatedBy,
		in.Kind, in.Title, in.Description, in.Priority, in.Origin, in.DueAt, ctxJSON,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	t.OrganizationID = in.OrganizationID
	t.LeadID = in.LeadID
	t.AssignedTo = in.AssignedTo
	t.CreatedBy = in.CreatedBy
	t.Kind = in.Kind
	t.Title = in.Title
	t.Description = in.Description
	t.Priority = in.Priority
	t.Status = "pending"
	t.Origin = in.Origin
	t.DueAt = in.DueAt
	t.Context = in.Context
	return t, nil
}

// Get returns a task within the tenant.
func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Task, error) {
	const q = `
		SELECT id, organization_id, lead_id, assigned_to, created_by,
		       kind::text, title, description, priority::text, status::text,
		       in_queue, queue_position, due_at, started_at, completed_at, completed_by,
		       cancelled_at, cancelled_reason, missed_reason,
		       origin::text, context, result_note, created_at, updated_at
		  FROM tasks
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var t Task
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&t.ID, &t.OrganizationID, &t.LeadID, &t.AssignedTo, &t.CreatedBy,
		&t.Kind, &t.Title, &t.Description, &t.Priority, &t.Status,
		&t.InQueue, &t.QueuePosition, &t.DueAt, &t.StartedAt, &t.CompletedAt, &t.CompletedBy,
		&t.CancelledAt, &t.CancelledReason, &t.MissedReason,
		&t.Origin, &t.Context, &t.ResultNote, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

// ListOptions filters the tenant task list.
type ListOptions struct {
	AssigneeID *uuid.UUID
	Status     string
	Kind       string
	LeadID     *uuid.UUID
	Limit      int
}

// List returns tasks ordered by (status-first, due_at ASC NULLS LAST, created_at DESC).
func (r *Repository) List(ctx context.Context, orgID uuid.UUID, opts ListOptions) ([]Task, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{orgID, limit}
	where := []string{"organization_id = $1"}
	if opts.AssigneeID != nil {
		args = append(args, *opts.AssigneeID)
		where = append(where, fmt.Sprintf("assigned_to = $%d", len(args)))
	}
	if opts.Status != "" {
		args = append(args, opts.Status)
		where = append(where, fmt.Sprintf("status = $%d::task_status", len(args)))
	}
	if opts.Kind != "" {
		args = append(args, opts.Kind)
		where = append(where, fmt.Sprintf("kind = $%d::task_kind", len(args)))
	}
	if opts.LeadID != nil {
		args = append(args, *opts.LeadID)
		where = append(where, fmt.Sprintf("lead_id = $%d", len(args)))
	}
	q := `
		SELECT id, organization_id, lead_id, assigned_to, created_by,
		       kind::text, title, description, priority::text, status::text,
		       in_queue, queue_position, due_at, started_at, completed_at, completed_by,
		       cancelled_at, cancelled_reason, missed_reason,
		       origin::text, context, result_note, created_at, updated_at
		  FROM tasks
		 WHERE ` + strings.Join(where, " AND ") + `
		 ORDER BY
		   CASE status
		     WHEN 'in_progress' THEN 0
		     WHEN 'pending'     THEN 1
		     WHEN 'done'        THEN 2
		     WHEN 'missed'      THEN 3
		     WHEN 'cancelled'   THEN 4
		   END,
		   due_at ASC NULLS LAST,
		   created_at DESC
		 LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	out := make([]Task, 0, limit)
	for rows.Next() {
		var t Task
		if err := rows.Scan(
			&t.ID, &t.OrganizationID, &t.LeadID, &t.AssignedTo, &t.CreatedBy,
			&t.Kind, &t.Title, &t.Description, &t.Priority, &t.Status,
			&t.InQueue, &t.QueuePosition, &t.DueAt, &t.StartedAt, &t.CompletedAt, &t.CompletedBy,
			&t.CancelledAt, &t.CancelledReason, &t.MissedReason,
			&t.Origin, &t.Context, &t.ResultNote, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Start transitions pending → in_progress. Atomic — Postgres's partial unique
// index guarantees AT MOST ONE in_progress per (org, assigned_to).
//
// Distinguishes three failure modes:
//   ErrAssigneeBusy   → assignee already has an in_progress task (23505).
//   ErrNotFound       → task does not exist in this tenant.
//   ErrInvalidState   → task exists but is past pending.
func (r *Repository) Start(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE tasks
		    SET status = 'in_progress', started_at = now()
		  WHERE id = $1 AND organization_id = $2 AND status = 'pending'`,
		id, orgID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAssigneeBusy
		}
		return fmt.Errorf("start task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, gerr := r.Get(ctx, orgID, id); gerr != nil {
			return gerr
		}
		return ErrInvalidState
	}
	return nil
}

// Complete transitions in_progress → done with optional result note.
func (r *Repository) Complete(ctx context.Context, orgID, id uuid.UUID, completedBy uuid.UUID, resultNote *string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE tasks
		    SET status = 'done', completed_at = now(),
		        completed_by = $3, result_note = $4
		  WHERE id = $1 AND organization_id = $2 AND status = 'in_progress'`,
		id, orgID, completedBy, resultNote,
	)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, err := r.Get(ctx, orgID, id); err != nil {
			return err
		}
		return ErrInvalidState
	}
	return nil
}

// Cancel sets status=cancelled with reason. Legal from pending or in_progress.
func (r *Repository) Cancel(ctx context.Context, orgID, id uuid.UUID, reason *string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE tasks
		    SET status = 'cancelled', cancelled_at = now(), cancelled_reason = $3
		  WHERE id = $1 AND organization_id = $2 AND status IN ('pending','in_progress')`,
		id, orgID, reason,
	)
	if err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, err := r.Get(ctx, orgID, id); err != nil {
			return err
		}
		return ErrInvalidState
	}
	return nil
}

// Miss flags a task missed (due_at passed without completion).
func (r *Repository) Miss(ctx context.Context, orgID, id uuid.UUID, reason *string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE tasks
		    SET status = 'missed', missed_reason = $3
		  WHERE id = $1 AND organization_id = $2 AND status IN ('pending','in_progress')`,
		id, orgID, reason,
	)
	if err != nil {
		return fmt.Errorf("miss task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrInvalidState
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

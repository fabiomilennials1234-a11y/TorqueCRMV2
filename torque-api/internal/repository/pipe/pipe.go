// Package pipe is the pgx-backed repository for pipes, stages, and entries.
//
// A lead is in one stage per pipe at a time — that invariant is enforced by
// the `Move` method inside a Serializable transaction: it closes the previous
// entry (left_at = now()) and inserts the new one atomically.
package pipe

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

// ErrNotFound is returned when a pipe/stage/entry is missing or out of scope.
var ErrNotFound = errors.New("not found")

// ErrStageMismatch — destination stage does not belong to the target pipe.
var ErrStageMismatch = errors.New("stage does not belong to pipe")

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ListPipes returns every non-archived pipe in the tenant, ordered by position.
func (r *Repository) ListPipes(ctx context.Context, orgID uuid.UUID) ([]domain.Pipe, error) {
	const q = `
		SELECT id, organization_id, kind::text, name, is_default, is_archived, position,
		       created_at, updated_at
		  FROM pipes
		 WHERE organization_id = $1 AND is_archived = false
		 ORDER BY position, name
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list pipes: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Pipe, 0, 4)
	for rows.Next() {
		var p domain.Pipe
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Kind, &p.Name, &p.IsDefault, &p.IsArchived, &p.Position, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Stages returns the columns of a pipe, ordered by position.
func (r *Repository) Stages(ctx context.Context, orgID, pipeID uuid.UUID) ([]domain.PipeStage, error) {
	const q = `
		SELECT id, organization_id, pipe_id, name, color_token, position,
		       is_final_positive, is_final_negative, created_at, updated_at
		  FROM pipe_stages
		 WHERE organization_id = $1 AND pipe_id = $2
		 ORDER BY position
	`
	rows, err := r.pool.Query(ctx, q, orgID, pipeID)
	if err != nil {
		return nil, fmt.Errorf("stages: %w", err)
	}
	defer rows.Close()
	out := make([]domain.PipeStage, 0, 8)
	for rows.Next() {
		var s domain.PipeStage
		if err := rows.Scan(&s.ID, &s.OrganizationID, &s.PipeID, &s.Name, &s.ColorToken, &s.Position,
			&s.IsFinalPositive, &s.IsFinalNegative, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Entries returns every ACTIVE pipe entry (left_at IS NULL) for the pipe.
// Grouping by stage is a consumer concern.
func (r *Repository) Entries(ctx context.Context, orgID, pipeID uuid.UUID) ([]domain.PipeEntry, error) {
	const q = `
		SELECT id, organization_id, pipe_id, stage_id, lead_id,
		       entered_stage_at, left_at, created_at, updated_at
		  FROM pipe_entries
		 WHERE organization_id = $1 AND pipe_id = $2 AND left_at IS NULL
		 ORDER BY entered_stage_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, pipeID)
	if err != nil {
		return nil, fmt.Errorf("entries: %w", err)
	}
	defer rows.Close()
	out := make([]domain.PipeEntry, 0, 32)
	for rows.Next() {
		var e domain.PipeEntry
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.PipeID, &e.StageID, &e.LeadID,
			&e.EnteredStageAt, &e.LeftAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Move transitions a lead from its current stage in `pipeID` to `newStageID`.
// Atomic: closes the open entry (left_at = now) and inserts a new one. If the
// lead has no current entry in the pipe, creates the first one. The transaction
// is Serializable so concurrent moves on the same lead cannot produce two
// open rows.
func (r *Repository) Move(ctx context.Context, orgID, pipeID, leadID, newStageID uuid.UUID) (domain.PipeEntry, error) {
	// Validate the stage belongs to the pipe up-front; saves a wasted write.
	var count int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM pipe_stages WHERE id = $1 AND pipe_id = $2 AND organization_id = $3`,
		newStageID, pipeID, orgID,
	).Scan(&count); err != nil {
		return domain.PipeEntry{}, fmt.Errorf("validate stage: %w", err)
	}
	if count == 0 {
		return domain.PipeEntry{}, ErrStageMismatch
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.PipeEntry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Close the existing open entry, if any.
	if _, err := tx.Exec(ctx,
		`UPDATE pipe_entries
		    SET left_at = now()
		  WHERE organization_id = $1 AND pipe_id = $2 AND lead_id = $3 AND left_at IS NULL`,
		orgID, pipeID, leadID,
	); err != nil {
		return domain.PipeEntry{}, fmt.Errorf("close previous entry: %w", err)
	}

	// Insert the new entry.
	var e domain.PipeEntry
	err = tx.QueryRow(ctx,
		`INSERT INTO pipe_entries (organization_id, pipe_id, stage_id, lead_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, organization_id, pipe_id, stage_id, lead_id,
		           entered_stage_at, left_at, created_at, updated_at`,
		orgID, pipeID, newStageID, leadID,
	).Scan(&e.ID, &e.OrganizationID, &e.PipeID, &e.StageID, &e.LeadID,
		&e.EnteredStageAt, &e.LeftAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return domain.PipeEntry{}, fmt.Errorf("insert entry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PipeEntry{}, fmt.Errorf("commit: %w", err)
	}
	return e, nil
}

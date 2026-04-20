// Package confirmation is the pgx-backed store for F02 pipe_confirmations.
//
// Every row is 1:1 with a pipe_entry in a confirmation-kind pipe. The
// repository does NOT create pipe_entries itself — callers move the lead
// into a confirmation stage via pipe.Move and then Upsert the confirmation
// metadata onto the resulting entry.
package confirmation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound — no confirmation row for the given pipe_entry (tenant-scoped).
var ErrNotFound = errors.New("confirmation not found")

// Confirmation is the in-memory representation.
type Confirmation struct {
	PipeEntryID     uuid.UUID
	OrganizationID  uuid.UUID
	LeadID          uuid.UUID
	MeetingAt       time.Time
	MeetingChannel  *string
	MeetingNotes    *string
	ConfirmedAt     *time.Time
	NoShow          bool
	NoShowReason    *string
	ReminderSentAt  *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Repository wraps the pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// UpsertInput is the write shape.
type UpsertInput struct {
	OrganizationID uuid.UUID
	PipeEntryID    uuid.UUID
	LeadID         uuid.UUID
	MeetingAt      time.Time
	MeetingChannel *string
	MeetingNotes   *string
}

// Upsert inserts or updates the confirmation row for a pipe_entry. The entry
// must already exist in a confirmation-kind pipe — the constraint is that the
// pipe_entry_id must exist, enforced by the FK.
func (r *Repository) Upsert(ctx context.Context, in UpsertInput) (Confirmation, error) {
	if in.MeetingAt.IsZero() {
		return Confirmation{}, errors.New("meeting_at is required")
	}
	const q = `
		INSERT INTO pipe_confirmations (
		  pipe_entry_id, organization_id, lead_id,
		  meeting_at, meeting_channel, meeting_notes
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pipe_entry_id) DO UPDATE SET
		  meeting_at      = EXCLUDED.meeting_at,
		  meeting_channel = EXCLUDED.meeting_channel,
		  meeting_notes   = EXCLUDED.meeting_notes
		RETURNING pipe_entry_id, organization_id, lead_id,
		          meeting_at, meeting_channel, meeting_notes,
		          confirmed_at, no_show, no_show_reason, reminder_sent_at,
		          created_at, updated_at
	`
	var c Confirmation
	err := r.pool.QueryRow(ctx, q,
		in.PipeEntryID, in.OrganizationID, in.LeadID,
		in.MeetingAt, in.MeetingChannel, in.MeetingNotes,
	).Scan(
		&c.PipeEntryID, &c.OrganizationID, &c.LeadID,
		&c.MeetingAt, &c.MeetingChannel, &c.MeetingNotes,
		&c.ConfirmedAt, &c.NoShow, &c.NoShowReason, &c.ReminderSentAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return Confirmation{}, fmt.Errorf("upsert confirmation: %w", err)
	}
	return c, nil
}

// Get returns the confirmation for a pipe_entry in the caller's tenant.
func (r *Repository) Get(ctx context.Context, orgID, pipeEntryID uuid.UUID) (Confirmation, error) {
	const q = `
		SELECT pipe_entry_id, organization_id, lead_id,
		       meeting_at, meeting_channel, meeting_notes,
		       confirmed_at, no_show, no_show_reason, reminder_sent_at,
		       created_at, updated_at
		  FROM pipe_confirmations
		 WHERE pipe_entry_id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var c Confirmation
	err := r.pool.QueryRow(ctx, q, pipeEntryID, orgID).Scan(
		&c.PipeEntryID, &c.OrganizationID, &c.LeadID,
		&c.MeetingAt, &c.MeetingChannel, &c.MeetingNotes,
		&c.ConfirmedAt, &c.NoShow, &c.NoShowReason, &c.ReminderSentAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Confirmation{}, ErrNotFound
	}
	if err != nil {
		return Confirmation{}, fmt.Errorf("get confirmation: %w", err)
	}
	return c, nil
}

// MarkConfirmed sets confirmed_at to now(). Idempotent — subsequent calls
// keep the original timestamp (ON CONFLICT DO NOTHING style via the WHERE).
func (r *Repository) MarkConfirmed(ctx context.Context, orgID, pipeEntryID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_confirmations
		    SET confirmed_at = now()
		  WHERE pipe_entry_id = $1 AND organization_id = $2 AND confirmed_at IS NULL`,
		pipeEntryID, orgID,
	)
	if err != nil {
		return fmt.Errorf("mark confirmed: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// Either already confirmed or row missing. Treat the former as success
		// (idempotent); use Get to disambiguate when needed.
		if _, err := r.Get(ctx, orgID, pipeEntryID); err != nil {
			return err
		}
	}
	return nil
}

// MarkNoShow flags the meeting as a no-show and stores an optional reason.
func (r *Repository) MarkNoShow(ctx context.Context, orgID, pipeEntryID uuid.UUID, reason *string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_confirmations
		    SET no_show = true, no_show_reason = $3
		  WHERE pipe_entry_id = $1 AND organization_id = $2`,
		pipeEntryID, orgID, reason,
	)
	if err != nil {
		return fmt.Errorf("mark no-show: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Overdue returns rows whose meeting_at is in the past and are still
// unconfirmed and not flagged as no-show. Used by a scheduled worker in
// future sprints; exposed here so the endpoint that surfaces them is ready.
func (r *Repository) Overdue(ctx context.Context, orgID uuid.UUID, now time.Time, limit int) ([]Confirmation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
		SELECT pipe_entry_id, organization_id, lead_id,
		       meeting_at, meeting_channel, meeting_notes,
		       confirmed_at, no_show, no_show_reason, reminder_sent_at,
		       created_at, updated_at
		  FROM pipe_confirmations
		 WHERE organization_id = $1
		   AND confirmed_at IS NULL
		   AND no_show = false
		   AND meeting_at < $2
		 ORDER BY meeting_at
		 LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, orgID, now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("overdue: %w", err)
	}
	defer rows.Close()
	out := make([]Confirmation, 0, limit)
	for rows.Next() {
		var c Confirmation
		if err := rows.Scan(
			&c.PipeEntryID, &c.OrganizationID, &c.LeadID,
			&c.MeetingAt, &c.MeetingChannel, &c.MeetingNotes,
			&c.ConfirmedAt, &c.NoShow, &c.NoShowReason, &c.ReminderSentAt,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

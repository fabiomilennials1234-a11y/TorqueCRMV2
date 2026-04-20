// Package proposal persists F03 pipe_proposals.
//
// Lifecycle transitions are atomic single-column updates with a WHERE clause
// that guards against illegal jumps (e.g. cannot Accept from draft; must be
// sent or viewed first). The repository is the single keeper of that truth
// so handlers never need to re-check.
package proposal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("proposal not found")
	ErrInvalidState    = errors.New("illegal proposal state transition")
)

// Status mirrors proposal_status ENUM.
type Status string

const (
	StatusDraft    Status = "draft"
	StatusSent     Status = "sent"
	StatusViewed   Status = "viewed"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"
	StatusExpired  Status = "expired"
)

// Proposal is the in-memory view.
type Proposal struct {
	PipeEntryID     uuid.UUID
	OrganizationID  uuid.UUID
	LeadID          uuid.UUID
	Title           string
	AmountCents     int64
	Currency        string
	AttachmentKey   *string
	AttachmentSize  *int64
	Status          Status
	SentAt          *time.Time
	FirstViewedAt   *time.Time
	AcceptedAt      *time.Time
	RejectedAt      *time.Time
	RejectionReason *string
	ExpiresAt       *time.Time
	Notes           *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// UpsertInput is the write-side shape for Upsert (draft-create / draft-update).
type UpsertInput struct {
	OrganizationID uuid.UUID
	PipeEntryID    uuid.UUID
	LeadID         uuid.UUID
	Title          string
	AmountCents    int64
	Currency       string
	AttachmentKey  *string
	AttachmentSize *int64
	Notes          *string
	ExpiresAt      *time.Time
}

// MaxAmountCents is the upper cap accepted for amount_cents. 1e14 cents =
// one trillion currency units — high enough for enterprise deals, low enough
// to prevent silent int64 overflow when SUM()ing thousands of proposals.
const MaxAmountCents int64 = 1e14

// Upsert creates or updates a draft proposal. It refuses to touch a proposal
// already past draft — those transitions go through MarkSent/MarkViewed/etc.
func (r *Repository) Upsert(ctx context.Context, in UpsertInput) (Proposal, error) {
	if in.Title == "" {
		return Proposal{}, errors.New("title is required")
	}
	if in.AmountCents < 0 {
		return Proposal{}, errors.New("amount_cents must be >= 0")
	}
	if in.AmountCents > MaxAmountCents {
		return Proposal{}, fmt.Errorf("amount_cents exceeds max (%d)", MaxAmountCents)
	}
	if in.Currency == "" {
		in.Currency = "BRL"
	}

	const q = `
		INSERT INTO pipe_proposals (
		  pipe_entry_id, organization_id, lead_id,
		  title, amount_cents, currency, attachment_key, attachment_size,
		  notes, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (pipe_entry_id) DO UPDATE SET
		  title           = EXCLUDED.title,
		  amount_cents    = EXCLUDED.amount_cents,
		  currency        = EXCLUDED.currency,
		  attachment_key  = EXCLUDED.attachment_key,
		  attachment_size = EXCLUDED.attachment_size,
		  notes           = EXCLUDED.notes,
		  expires_at      = EXCLUDED.expires_at
		  WHERE pipe_proposals.status = 'draft'
		RETURNING pipe_entry_id, organization_id, lead_id, title, amount_cents,
		          currency, attachment_key, attachment_size, status,
		          sent_at, first_viewed_at, accepted_at, rejected_at, rejection_reason,
		          expires_at, notes, created_at, updated_at
	`
	var p Proposal
	var status string
	err := r.pool.QueryRow(ctx, q,
		in.PipeEntryID, in.OrganizationID, in.LeadID,
		in.Title, in.AmountCents, in.Currency, in.AttachmentKey, in.AttachmentSize,
		in.Notes, in.ExpiresAt,
	).Scan(
		&p.PipeEntryID, &p.OrganizationID, &p.LeadID, &p.Title, &p.AmountCents,
		&p.Currency, &p.AttachmentKey, &p.AttachmentSize, &status,
		&p.SentAt, &p.FirstViewedAt, &p.AcceptedAt, &p.RejectedAt, &p.RejectionReason,
		&p.ExpiresAt, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT WHERE status='draft' filtered out — already beyond draft.
		return Proposal{}, ErrInvalidState
	}
	if err != nil {
		return Proposal{}, fmt.Errorf("upsert proposal: %w", err)
	}
	p.Status = Status(status)
	return p, nil
}

// Get returns the proposal for the entry within the tenant.
func (r *Repository) Get(ctx context.Context, orgID, entryID uuid.UUID) (Proposal, error) {
	const q = `
		SELECT pipe_entry_id, organization_id, lead_id, title, amount_cents,
		       currency, attachment_key, attachment_size, status,
		       sent_at, first_viewed_at, accepted_at, rejected_at, rejection_reason,
		       expires_at, notes, created_at, updated_at
		  FROM pipe_proposals
		 WHERE pipe_entry_id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var p Proposal
	var status string
	err := r.pool.QueryRow(ctx, q, entryID, orgID).Scan(
		&p.PipeEntryID, &p.OrganizationID, &p.LeadID, &p.Title, &p.AmountCents,
		&p.Currency, &p.AttachmentKey, &p.AttachmentSize, &status,
		&p.SentAt, &p.FirstViewedAt, &p.AcceptedAt, &p.RejectedAt, &p.RejectionReason,
		&p.ExpiresAt, &p.Notes, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Proposal{}, ErrNotFound
	}
	if err != nil {
		return Proposal{}, fmt.Errorf("get proposal: %w", err)
	}
	p.Status = Status(status)
	return p, nil
}

// MarkSent moves draft → sent, stamps sent_at, and records the actor.
// Legal only from draft.
func (r *Repository) MarkSent(ctx context.Context, orgID, entryID, actorMember uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_proposals
		    SET status = 'sent',
		        sent_at = COALESCE(sent_at, now()),
		        sent_by_member_id = $3
		  WHERE pipe_entry_id = $1 AND organization_id = $2 AND status = 'draft'`,
		entryID, orgID, actorMember,
	)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, err := r.Get(ctx, orgID, entryID); err != nil {
			return err
		}
		return ErrInvalidState
	}
	return nil
}

// MarkViewed moves sent → viewed. Legal from sent only. No-op from viewed+.
// first_viewed_at is set on the FIRST viewed transition and never overwritten.
func (r *Repository) MarkViewed(ctx context.Context, orgID, entryID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pipe_proposals
		    SET status = 'viewed',
		        first_viewed_at = COALESCE(first_viewed_at, now())
		  WHERE pipe_entry_id = $1 AND organization_id = $2 AND status = 'sent'`,
		entryID, orgID,
	)
	if err != nil {
		return fmt.Errorf("mark viewed: %w", err)
	}
	// Silent no-op when already viewed/accepted/rejected — idempotent pixel.
	return nil
}

// MarkAccepted moves sent|viewed → accepted and records the actor.
func (r *Repository) MarkAccepted(ctx context.Context, orgID, entryID, actorMember uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_proposals
		    SET status = 'accepted', accepted_at = now(), accepted_by_member_id = $3
		  WHERE pipe_entry_id = $1 AND organization_id = $2 AND status IN ('sent','viewed')`,
		entryID, orgID, actorMember,
	)
	if err != nil {
		return fmt.Errorf("mark accepted: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, err := r.Get(ctx, orgID, entryID); err != nil {
			return err
		}
		return ErrInvalidState
	}
	return nil
}

// MarkRejected moves sent|viewed → rejected with optional reason + actor.
func (r *Repository) MarkRejected(ctx context.Context, orgID, entryID, actorMember uuid.UUID, reason *string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_proposals
		    SET status = 'rejected', rejected_at = now(),
		        rejection_reason = $3, rejected_by_member_id = $4
		  WHERE pipe_entry_id = $1 AND organization_id = $2 AND status IN ('sent','viewed')`,
		entryID, orgID, reason, actorMember,
	)
	if err != nil {
		return fmt.Errorf("mark rejected: %w", err)
	}
	if ct.RowsAffected() == 0 {
		if _, err := r.Get(ctx, orgID, entryID); err != nil {
			return err
		}
		return ErrInvalidState
	}
	return nil
}

// SweepExpired flips sent/viewed rows past expires_at to 'expired'. Returns
// the number of rows expired for metrics. Scheduled workers call this.
func (r *Repository) SweepExpired(ctx context.Context, orgID uuid.UUID, now time.Time) (int64, error) {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipe_proposals
		    SET status = 'expired'
		  WHERE organization_id = $1
		    AND status IN ('sent','viewed')
		    AND expires_at IS NOT NULL
		    AND expires_at < $2`,
		orgID, now.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("sweep expired: %w", err)
	}
	return ct.RowsAffected(), nil
}

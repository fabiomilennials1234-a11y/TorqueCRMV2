// Package subscription is the pgx-backed repository for F14 billing.
//
// Invariant: at most one subscription per tenant in any of the "live" states
// (pending/active/past_due). Enforced by the partial unique index
// uq_subscriptions_one_active_per_org.
package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("subscription not found")
	ErrAlreadyActive  = errors.New("tenant already has an active or pending subscription")
	ErrDuplicateEvent = errors.New("billing event already processed")
)

// Subscription is the hydrated row.
type Subscription struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	PlanID               string
	Status               string
	Provider             string
	ProviderCustomerID   *string
	ProviderChargeID     *string
	AmountCents          int64
	Currency             string
	PixQRCode            *string
	PixQRCodeImage       *string
	PixExpiresAt         *time.Time
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	CancelledAt          *time.Time
	CreatedBy            *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// CreateInput carries the shape used by Create.
type CreateInput struct {
	OrganizationID   uuid.UUID
	PlanID           string
	Provider         string
	AmountCents      int64
	Currency         string
	ProviderChargeID string
	PixQRCode        string
	PixQRCodeImage   string
	PixExpiresAt     time.Time
	CreatedBy        uuid.UUID
}

// Create inserts the initial 'pending' row after the provider issues the
// charge. Unique partial index catches the race where two admins launch a
// checkout at the same time — ErrAlreadyActive gives the handler the right
// status (409).
func (r *Repository) Create(ctx context.Context, in CreateInput) (Subscription, error) {
	const q = `
		INSERT INTO subscriptions (
		  organization_id, plan_id, provider, amount_cents, currency,
		  provider_charge_id, pix_qr_code, pix_qr_code_image, pix_expires_at,
		  created_by
		) VALUES ($1, $2, $3::billing_provider, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, status::text, created_at, updated_at
	`
	var s Subscription
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.PlanID, in.Provider, in.AmountCents, in.Currency,
		in.ProviderChargeID, in.PixQRCode, in.PixQRCodeImage, in.PixExpiresAt,
		in.CreatedBy,
	).Scan(&s.ID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Subscription{}, ErrAlreadyActive
		}
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}
	s.OrganizationID = in.OrganizationID
	s.PlanID = in.PlanID
	s.Provider = in.Provider
	s.AmountCents = in.AmountCents
	s.Currency = in.Currency
	pc := in.ProviderChargeID
	s.ProviderChargeID = &pc
	qr := in.PixQRCode
	s.PixQRCode = &qr
	img := in.PixQRCodeImage
	s.PixQRCodeImage = &img
	exp := in.PixExpiresAt
	s.PixExpiresAt = &exp
	creator := in.CreatedBy
	s.CreatedBy = &creator
	return s, nil
}

// GetActive returns the tenant's current live subscription (pending/active/
// past_due). Returns ErrNotFound if there is none.
func (r *Repository) GetActive(ctx context.Context, orgID uuid.UUID) (Subscription, error) {
	const q = `
		SELECT id, organization_id, plan_id, status::text, provider::text,
		       provider_customer_id, provider_charge_id, amount_cents, currency,
		       pix_qr_code, pix_qr_code_image, pix_expires_at,
		       current_period_start, current_period_end, cancelled_at,
		       created_by, created_at, updated_at
		  FROM subscriptions
		 WHERE organization_id = $1
		   AND status IN ('pending','active','past_due')
		 ORDER BY created_at DESC
		 LIMIT 1
	`
	return r.scanOne(ctx, q, orgID)
}

// Get returns a subscription by id (tenant-scoped).
func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Subscription, error) {
	const q = `
		SELECT id, organization_id, plan_id, status::text, provider::text,
		       provider_customer_id, provider_charge_id, amount_cents, currency,
		       pix_qr_code, pix_qr_code_image, pix_expires_at,
		       current_period_start, current_period_end, cancelled_at,
		       created_by, created_at, updated_at
		  FROM subscriptions
		 WHERE id = $1 AND organization_id = $2
	`
	return r.scanOne(ctx, q, id, orgID)
}

func (r *Repository) scanOne(ctx context.Context, q string, args ...any) (Subscription, error) {
	var s Subscription
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&s.ID, &s.OrganizationID, &s.PlanID, &s.Status, &s.Provider,
		&s.ProviderCustomerID, &s.ProviderChargeID, &s.AmountCents, &s.Currency,
		&s.PixQRCode, &s.PixQRCodeImage, &s.PixExpiresAt,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelledAt,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("scan subscription: %w", err)
	}
	return s, nil
}

// MarkPaid flips pending → active and stamps the billing period.
func (r *Repository) MarkPaid(ctx context.Context, id uuid.UUID, periodEnd time.Time) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE subscriptions
		    SET status = 'active',
		        current_period_start = COALESCE(current_period_start, now()),
		        current_period_end = $2
		  WHERE id = $1 AND status IN ('pending','past_due')`,
		id, periodEnd,
	)
	if err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkPastDue flips active → past_due (provider reports delinquency).
func (r *Repository) MarkPastDue(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE subscriptions SET status = 'past_due'
		  WHERE id = $1 AND status = 'active'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark past_due: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Cancel moves any live state → cancelled. Idempotent returns ErrNotFound.
func (r *Repository) Cancel(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE subscriptions
		    SET status = 'cancelled', cancelled_at = now()
		  WHERE id = $1 AND organization_id = $2
		    AND status IN ('pending','active','past_due')`,
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

// -------- billing events ---------------------------------------------

// RecordEvent appends a webhook event; dedup on (provider, event_id).
// Returns ErrDuplicateEvent when the unique constraint trips — the caller
// must NOT retry state transitions on duplicates.
func (r *Repository) RecordEvent(ctx context.Context, orgID *uuid.UUID, subID *uuid.UUID, provider, eventID, eventType string, rawPayload json.RawMessage) error {
	if len(rawPayload) == 0 {
		rawPayload = json.RawMessage(`{}`)
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO billing_events
		   (organization_id, subscription_id, provider, provider_event_id, event_type, raw_payload)
		 VALUES ($1, $2, $3::billing_provider, $4, $5, $6)`,
		orgID, subID, provider, eventID, eventType, rawPayload,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEvent
		}
		return fmt.Errorf("record event: %w", err)
	}
	return nil
}

// FindByProviderCharge resolves a subscription by the provider's charge id.
// Used by the webhook handler to map raw provider events to subscriptions.
func (r *Repository) FindByProviderCharge(ctx context.Context, provider, chargeID string) (Subscription, error) {
	const q = `
		SELECT id, organization_id, plan_id, status::text, provider::text,
		       provider_customer_id, provider_charge_id, amount_cents, currency,
		       pix_qr_code, pix_qr_code_image, pix_expires_at,
		       current_period_start, current_period_end, cancelled_at,
		       created_by, created_at, updated_at
		  FROM subscriptions
		 WHERE provider = $1::billing_provider
		   AND provider_charge_id = $2
		 LIMIT 1
	`
	return r.scanOne(ctx, q, provider, chargeID)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Package leadwebhook persists the audit log + dedup key for inbound
// lead webhook events (S50 / F.2). Schema in 0025_s50_*.up.sql.
//
// ErrDuplicate is returned when the (organization_id, external_id)
// unique constraint trips — the caller treats it as idempotent
// success (HTTP 200) and does NOT create a second lead.
package leadwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicate is the sentinel for (organization_id, external_id)
// unique-constraint violations. The webhook handler returns 200 on
// this condition so the provider does not retry.
var ErrDuplicate = errors.New("lead webhook event: duplicate external_id")

// Repository is the pgx-backed persistence layer.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// RecordInput carries the write-side shape. The repository serializes
// Payload into the raw_payload jsonb column.
type RecordInput struct {
	OrganizationID uuid.UUID
	ExternalID     string
	Source         string // 'webhook' | 'meta' | 'szchat' | 'manual'
	LeadID         *uuid.UUID
	Payload        json.RawMessage
	SignatureOK    bool
}

// Record inserts the row. Returns ErrDuplicate when the unique
// constraint (organization_id, external_id) trips.
func (r *Repository) Record(ctx context.Context, in RecordInput) error {
	if in.OrganizationID == uuid.Nil {
		return errors.New("leadwebhook: organization_id required")
	}
	if in.ExternalID == "" {
		return errors.New("leadwebhook: external_id required")
	}
	source := in.Source
	if source == "" {
		source = "webhook"
	}
	payload := in.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO lead_webhook_events
		  (organization_id, external_id, source, lead_id, raw_payload, signature_ok)
		VALUES ($1,$2,$3,$4,$5,$6)
	`
	_, err := r.pool.Exec(ctx, q,
		in.OrganizationID, in.ExternalID, source, in.LeadID, payload, in.SignatureOK,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("record lead webhook event: %w", err)
	}
	return nil
}

// AttachLead updates the already-recorded event with the lead_id that
// was created in the same handler. Safe to call at most once per
// event — we do not enforce that at the DB level.
func (r *Repository) AttachLead(ctx context.Context, orgID uuid.UUID, externalID string, leadID uuid.UUID) error {
	const q = `
		UPDATE lead_webhook_events
		   SET lead_id = $3
		 WHERE organization_id = $1
		   AND external_id = $2
		   AND lead_id IS NULL
	`
	_, err := r.pool.Exec(ctx, q, orgID, externalID, leadID)
	if err != nil {
		return fmt.Errorf("attach lead to webhook event: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

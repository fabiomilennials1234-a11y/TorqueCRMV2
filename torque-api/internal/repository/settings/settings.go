// Package settings backs F15 — organization profile patch, webhook
// endpoints CRUD, and per-member notification preferences.
package settings

import (
	"context"
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
	ErrNotFound  = errors.New("settings row not found")
	ErrURLTaken  = errors.New("webhook url already exists in this organization")
	ErrURLScheme = errors.New("webhook url must use https://")
)

// Organization is the read-side view the settings UI consumes.
type Organization struct {
	ID         uuid.UUID
	Slug       string
	Name       string
	LegalName  *string
	CNPJ       *string
	Timezone   *string
	LogoURL    *string
	UpdatedAt  time.Time
}

// WebhookEndpoint is one outbound destination.
type WebhookEndpoint struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	URL            string
	Description    *string
	EventTypes     []string
	Secret         string
	IsActive       bool
	LastSuccessAt  *time.Time
	LastFailureAt  *time.Time
	LastError      *string
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NotificationPref is one row per (member, channel, topic).
type NotificationPref struct {
	TeamMemberID uuid.UUID
	Channel      string
	Topic        string
	Enabled      bool
	UpdatedAt    time.Time
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- organization ----------------------------------------------

// GetOrganization returns the hydrated org row for the settings UI.
func (r *Repository) GetOrganization(ctx context.Context, orgID uuid.UUID) (Organization, error) {
	const q = `
		SELECT id, slug, name, legal_name, cnpj, timezone, logo_url, updated_at
		  FROM organizations
		 WHERE id = $1 AND deleted_at IS NULL
		 LIMIT 1
	`
	var o Organization
	err := r.pool.QueryRow(ctx, q, orgID).Scan(
		&o.ID, &o.Slug, &o.Name, &o.LegalName, &o.CNPJ, &o.Timezone, &o.LogoURL, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, fmt.Errorf("get organization: %w", err)
	}
	return o, nil
}

// UpdateOrganizationInput is the patch shape.
type UpdateOrganizationInput struct {
	Name      *string
	LegalName *string
	CNPJ      *string
	Timezone  *string
	LogoURL   *string
}

// UpdateOrganization applies a partial patch. Slug is intentionally NOT
// patchable here — re-slugging a tenant would break every bookmark; a
// separate master-only path owns that operation.
func (r *Repository) UpdateOrganization(ctx context.Context, orgID uuid.UUID, in UpdateOrganizationInput) (Organization, error) {
	sets := make([]string, 0, 5)
	args := []any{orgID}
	idx := 2
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if len(n) < 2 || len(n) > 120 {
			return Organization{}, errors.New("name must be between 2 and 120 chars")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, n)
		idx++
	}
	if in.LegalName != nil {
		sets = append(sets, fmt.Sprintf("legal_name = $%d", idx))
		args = append(args, *in.LegalName)
		idx++
	}
	if in.CNPJ != nil {
		sets = append(sets, fmt.Sprintf("cnpj = $%d", idx))
		args = append(args, *in.CNPJ)
		idx++
	}
	if in.Timezone != nil {
		sets = append(sets, fmt.Sprintf("timezone = $%d", idx))
		args = append(args, *in.Timezone)
		idx++
	}
	if in.LogoURL != nil {
		sets = append(sets, fmt.Sprintf("logo_url = $%d", idx))
		args = append(args, *in.LogoURL)
		// idx not bumped — LogoURL is the last optional UpdateOrganization field.
	}
	if len(sets) == 0 {
		return r.GetOrganization(ctx, orgID)
	}
	q := fmt.Sprintf(`UPDATE organizations SET %s WHERE id = $1 AND deleted_at IS NULL`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return Organization{}, fmt.Errorf("update org: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return Organization{}, ErrNotFound
	}
	return r.GetOrganization(ctx, orgID)
}

// -------- webhook endpoints -----------------------------------------

// ListWebhooks returns all endpoints for a tenant ordered by creation asc.
func (r *Repository) ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]WebhookEndpoint, error) {
	const q = `
		SELECT id, organization_id, url, description, event_types, secret,
		       is_active, last_success_at, last_failure_at, last_error,
		       created_by, created_at, updated_at
		  FROM webhook_endpoints
		 WHERE organization_id = $1
		 ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()
	out := make([]WebhookEndpoint, 0, 4)
	for rows.Next() {
		var wep WebhookEndpoint
		if err := rows.Scan(
			&wep.ID, &wep.OrganizationID, &wep.URL, &wep.Description, &wep.EventTypes, &wep.Secret,
			&wep.IsActive, &wep.LastSuccessAt, &wep.LastFailureAt, &wep.LastError,
			&wep.CreatedBy, &wep.CreatedAt, &wep.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, wep)
	}
	return out, rows.Err()
}

// CreateWebhookInput is the write shape.
type CreateWebhookInput struct {
	URL         string
	Description *string
	EventTypes  []string
	Secret      string
	CreatedBy   uuid.UUID
}

// CreateWebhook inserts a new endpoint. The secret is caller-generated; the
// repo does not invent one (API caller owns rotation).
func (r *Repository) CreateWebhook(ctx context.Context, orgID uuid.UUID, in CreateWebhookInput) (WebhookEndpoint, error) {
	if !strings.HasPrefix(strings.ToLower(in.URL), "https://") {
		return WebhookEndpoint{}, ErrURLScheme
	}
	if len(in.Secret) < 16 || len(in.Secret) > 200 {
		return WebhookEndpoint{}, errors.New("secret must be between 16 and 200 chars")
	}
	if in.EventTypes == nil {
		in.EventTypes = []string{}
	}
	const q = `
		INSERT INTO webhook_endpoints
		  (organization_id, url, description, event_types, secret, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, is_active, created_at, updated_at
	`
	var w WebhookEndpoint
	err := r.pool.QueryRow(ctx, q,
		orgID, in.URL, in.Description, in.EventTypes, in.Secret, in.CreatedBy,
	).Scan(&w.ID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return WebhookEndpoint{}, ErrURLTaken
		}
		return WebhookEndpoint{}, fmt.Errorf("create webhook: %w", err)
	}
	w.OrganizationID = orgID
	w.URL = in.URL
	w.Description = in.Description
	w.EventTypes = in.EventTypes
	w.Secret = in.Secret
	cb := in.CreatedBy
	w.CreatedBy = &cb
	return w, nil
}

// UpdateWebhookInput is the patch shape.
type UpdateWebhookInput struct {
	Description *string
	EventTypes  *[]string
	IsActive    *bool
}

// UpdateWebhook applies a partial patch. URL and secret are intentionally
// immutable — rotating the secret means deleting + re-creating so the
// destination's verification code goes through a reset.
func (r *Repository) UpdateWebhook(ctx context.Context, orgID, id uuid.UUID, in UpdateWebhookInput) (WebhookEndpoint, error) {
	sets := make([]string, 0, 3)
	args := []any{id, orgID}
	idx := 3
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *in.Description)
		idx++
	}
	if in.EventTypes != nil {
		sets = append(sets, fmt.Sprintf("event_types = $%d", idx))
		args = append(args, *in.EventTypes)
		idx++
	}
	if in.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		// idx not bumped — IsActive is the last optional UpdateWebhook field.
	}
	if len(sets) == 0 {
		return r.GetWebhook(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE webhook_endpoints SET %s WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return WebhookEndpoint{}, fmt.Errorf("update webhook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return WebhookEndpoint{}, ErrNotFound
	}
	return r.GetWebhook(ctx, orgID, id)
}

// GetWebhook loads a single endpoint.
func (r *Repository) GetWebhook(ctx context.Context, orgID, id uuid.UUID) (WebhookEndpoint, error) {
	const q = `
		SELECT id, organization_id, url, description, event_types, secret,
		       is_active, last_success_at, last_failure_at, last_error,
		       created_by, created_at, updated_at
		  FROM webhook_endpoints
		 WHERE id = $1 AND organization_id = $2
	`
	var w WebhookEndpoint
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&w.ID, &w.OrganizationID, &w.URL, &w.Description, &w.EventTypes, &w.Secret,
		&w.IsActive, &w.LastSuccessAt, &w.LastFailureAt, &w.LastError,
		&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookEndpoint{}, ErrNotFound
	}
	if err != nil {
		return WebhookEndpoint{}, fmt.Errorf("get webhook: %w", err)
	}
	return w, nil
}

// DeleteWebhook removes the endpoint. Idempotent → ErrNotFound on absent.
func (r *Repository) DeleteWebhook(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM webhook_endpoints WHERE id = $1 AND organization_id = $2`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- notification preferences ----------------------------------

// ListPreferences returns all opt-outs (and any explicit opt-ins) for a
// member. Absent rows mean the default (true) applies.
func (r *Repository) ListPreferences(ctx context.Context, teamMemberID uuid.UUID) ([]NotificationPref, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT team_member_id, channel, topic, enabled, updated_at
		   FROM notification_preferences
		  WHERE team_member_id = $1
		  ORDER BY channel, topic`,
		teamMemberID,
	)
	if err != nil {
		return nil, fmt.Errorf("list prefs: %w", err)
	}
	defer rows.Close()
	out := make([]NotificationPref, 0, 8)
	for rows.Next() {
		var p NotificationPref
		if err := rows.Scan(&p.TeamMemberID, &p.Channel, &p.Topic, &p.Enabled, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPreference upserts (channel, topic, enabled). The composite primary
// key makes this a natural ON CONFLICT.
func (r *Repository) SetPreference(ctx context.Context, teamMemberID uuid.UUID, channel, topic string, enabled bool) error {
	if channel != "email" && channel != "in_app" && channel != "push" {
		return fmt.Errorf("channel must be email|in_app|push")
	}
	if topic == "" || len(topic) > 80 {
		return fmt.Errorf("topic must be 1..80 chars")
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_preferences (team_member_id, channel, topic, enabled)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (team_member_id, channel, topic)
		 DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now()`,
		teamMemberID, channel, topic, enabled,
	)
	if err != nil {
		return fmt.Errorf("set pref: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

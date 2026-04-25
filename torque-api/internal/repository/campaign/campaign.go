// Package campaign is the pgx-backed repository for F08 bulk outbound
// messaging. The actual message-send worker (kind `campaign.dispatch`) reads
// campaign_recipients rows where status='queued' and advances them; this
// package handles the synchronous CRUD + launch/pause/cancel lifecycle.
package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("campaign not found")
	ErrInvalidState = errors.New("illegal campaign state transition")
)

type Campaign struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	ChannelID      *uuid.UUID
	TemplateBody   string
	AudienceQuery  []byte
	Status         string
	ScheduledAt    *time.Time
	StartedAt      *time.Time
	EndedAt        *time.Time
	StatsQueued    int
	StatsSent      int
	StatsFailed    int
	StatsSkipped   int
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Recipient struct {
	ID             uuid.UUID
	CampaignID     uuid.UUID
	LeadID         uuid.UUID
	Status         string
	MessageID      *uuid.UUID
	SentAt         *time.Time
	FailedAt       *time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- campaigns --------------------------------------------------

type CreateInput struct {
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	ChannelID      *uuid.UUID
	TemplateBody   string
	AudienceQuery  json.RawMessage
	ScheduledAt    *time.Time
	CreatedBy      *uuid.UUID
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Campaign, error) {
	if in.Name == "" || in.TemplateBody == "" {
		return Campaign{}, errors.New("name and template_body are required")
	}
	audience := in.AudienceQuery
	if len(audience) == 0 {
		audience = []byte(`{}`)
	}
	// Channel ownership check (if provided).
	if in.ChannelID != nil {
		var ownedBy uuid.UUID
		err := r.pool.QueryRow(ctx,
			`SELECT organization_id FROM channels WHERE id = $1`,
			*in.ChannelID,
		).Scan(&ownedBy)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && ownedBy != in.OrganizationID) {
			return Campaign{}, ErrNotFound
		}
		if err != nil {
			return Campaign{}, fmt.Errorf("channel ownership: %w", err)
		}
	}

	const q = `
		INSERT INTO campaigns (
		  organization_id, name, description, channel_id, template_body,
		  audience_query, scheduled_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status::text, created_at, updated_at
	`
	var c Campaign
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.Name, in.Description, in.ChannelID, in.TemplateBody,
		audience, in.ScheduledAt, in.CreatedBy,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Campaign{}, fmt.Errorf("campaign name must be unique within tenant")
		}
		return Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	c.OrganizationID = in.OrganizationID
	c.Name = in.Name
	c.Description = in.Description
	c.ChannelID = in.ChannelID
	c.TemplateBody = in.TemplateBody
	c.AudienceQuery = audience
	c.ScheduledAt = in.ScheduledAt
	c.CreatedBy = in.CreatedBy
	return c, nil
}

func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Campaign, error) {
	const q = `
		SELECT id, organization_id, name, description, channel_id, template_body,
		       audience_query, status::text, scheduled_at, started_at, ended_at,
		       stats_queued, stats_sent, stats_failed, stats_skipped,
		       created_by, created_at, updated_at
		  FROM campaigns
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var c Campaign
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.ChannelID, &c.TemplateBody,
		&c.AudienceQuery, &c.Status, &c.ScheduledAt, &c.StartedAt, &c.EndedAt,
		&c.StatsQueued, &c.StatsSent, &c.StatsFailed, &c.StatsSkipped,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	if err != nil {
		return Campaign{}, fmt.Errorf("get campaign: %w", err)
	}
	return c, nil
}

func (r *Repository) List(ctx context.Context, orgID uuid.UUID, statusFilter string) ([]Campaign, error) {
	args := []any{orgID}
	q := `
		SELECT id, organization_id, name, description, channel_id, template_body,
		       audience_query, status::text, scheduled_at, started_at, ended_at,
		       stats_queued, stats_sent, stats_failed, stats_skipped,
		       created_by, created_at, updated_at
		  FROM campaigns
		 WHERE organization_id = $1
	`
	if statusFilter != "" {
		args = append(args, statusFilter)
		q += fmt.Sprintf(" AND status = $%d::campaign_status", len(args))
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()
	out := make([]Campaign, 0, 8)
	for rows.Next() {
		var c Campaign
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.ChannelID, &c.TemplateBody,
			&c.AudienceQuery, &c.Status, &c.ScheduledAt, &c.StartedAt, &c.EndedAt,
			&c.StatsQueued, &c.StatsSent, &c.StatsFailed, &c.StatsSkipped,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Launch materializes recipients and transitions draft|scheduled → running.
// Runs in a Serializable tx so the launch cannot race with another admin's
// attempt to launch or the scheduler.
//
// leadIDs comes from the handler — typically the result of executing the
// `audience_query` against the leads table. The repository does not evaluate
// the query itself; that is the audience-resolver's job.
func (r *Repository) Launch(
	ctx context.Context, orgID, campaignID uuid.UUID, leadIDs []uuid.UUID,
) (int64, error) {
	if len(leadIDs) == 0 {
		return 0, errors.New("at least one recipient is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, fmt.Errorf("begin launch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verify campaign + tenant + state.
	var status string
	err = tx.QueryRow(ctx,
		`SELECT status::text FROM campaigns
		  WHERE id = $1 AND organization_id = $2 LIMIT 1`,
		campaignID, orgID,
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("load campaign for launch: %w", err)
	}
	if status != "draft" && status != "scheduled" {
		return 0, ErrInvalidState
	}

	// Bulk-insert recipients; ignore collisions (anti-duplicate invariant).
	// We use a single INSERT ... SELECT FROM unnest(...) to avoid the
	// round-trip-per-row cost.
	ct, err := tx.Exec(ctx,
		`INSERT INTO campaign_recipients (organization_id, campaign_id, lead_id)
		 SELECT $1, $2, lead_id
		   FROM unnest($3::uuid[]) AS lead_id
		 ON CONFLICT (campaign_id, lead_id) DO NOTHING`,
		orgID, campaignID, leadIDs,
	)
	if err != nil {
		return 0, fmt.Errorf("insert recipients: %w", err)
	}
	queued := ct.RowsAffected()

	if _, err := tx.Exec(ctx,
		`UPDATE campaigns
		    SET status = 'running', started_at = now(), stats_queued = $3
		  WHERE id = $1 AND organization_id = $2`,
		campaignID, orgID, queued,
	); err != nil {
		return 0, fmt.Errorf("mark running: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit launch: %w", err)
	}
	return queued, nil
}

func (r *Repository) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE campaigns SET status = $3::campaign_status
		  WHERE id = $1 AND organization_id = $2`,
		id, orgID, status,
	)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordRecipientSent / Failed are called by the campaign worker. They
// update the recipient row AND the campaign stats counters atomically.
func (r *Repository) RecordRecipientSent(
	ctx context.Context, orgID, recipientID, messageID uuid.UUID,
) error {
	return r.recordRecipient(ctx, orgID, recipientID, "sent", &messageID, nil)
}

func (r *Repository) RecordRecipientFailed(
	ctx context.Context, orgID, recipientID uuid.UUID, errorPayload json.RawMessage,
) error {
	return r.recordRecipient(ctx, orgID, recipientID, "failed", nil, errorPayload)
}

func (r *Repository) recordRecipient(
	ctx context.Context, orgID, recipientID uuid.UUID,
	newStatus string, messageID *uuid.UUID, errorPayload json.RawMessage,
) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin record: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var campaignID uuid.UUID
	var oldStatus string
	err = tx.QueryRow(ctx,
		`UPDATE campaign_recipients
		    SET status = $3::campaign_recipient_status,
		        message_id = COALESCE($4, message_id),
		        error_payload = $5,
		        sent_at = CASE WHEN $3 = 'sent' THEN now() ELSE sent_at END,
		        failed_at = CASE WHEN $3 = 'failed' THEN now() ELSE failed_at END
		  WHERE id = $1 AND organization_id = $2 AND status = 'queued'
		 RETURNING campaign_id, 'queued'::text`,
		recipientID, orgID, newStatus, messageID, errorPayload,
	).Scan(&campaignID, &oldStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update recipient: %w", err)
	}

	inc := map[string]string{"sent": "stats_sent", "failed": "stats_failed"}
	col, ok := inc[newStatus]
	if !ok {
		return fmt.Errorf("unsupported transition: %s", newStatus)
	}
	if _, err := tx.Exec(ctx,
		fmt.Sprintf(
			`UPDATE campaigns
			    SET %s = %s + 1,
			        stats_queued = GREATEST(stats_queued - 1, 0)
			  WHERE id = $1 AND organization_id = $2`,
			col, col,
		),
		campaignID, orgID,
	); err != nil {
		return fmt.Errorf("bump campaign stats: %w", err)
	}
	return tx.Commit(ctx)
}

// ListRecipients returns per-lead delivery state for the campaign.
func (r *Repository) ListRecipients(
	ctx context.Context, orgID, campaignID uuid.UUID, limit int,
) ([]Recipient, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
		SELECT id, campaign_id, lead_id, status::text, message_id, sent_at, failed_at
		  FROM campaign_recipients
		 WHERE organization_id = $1 AND campaign_id = $2
		 ORDER BY created_at
		 LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, orgID, campaignID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}
	defer rows.Close()
	out := make([]Recipient, 0, limit)
	for rows.Next() {
		var rp Recipient
		if err := rows.Scan(
			&rp.ID, &rp.CampaignID, &rp.LeadID, &rp.Status, &rp.MessageID, &rp.SentAt, &rp.FailedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

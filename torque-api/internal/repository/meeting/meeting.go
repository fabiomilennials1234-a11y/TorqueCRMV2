// Package meeting persists F13 agenda rows (S48).
//
// Scope S48: CRUD + range list. Google Calendar sync (external_provider/
// external_id columns already carved) chega em S49 com OAuth2 real +
// token crypto.
package meeting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("meeting: not found")

type Meeting struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Title           string
	Description     *string
	StartsAt        time.Time
	EndsAt          time.Time
	Location        *string
	LeadID          *uuid.UUID
	OwnerMemberID   *uuid.UUID
	Status          string
	ExternalProvider *string
	ExternalID      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type CreateInput struct {
	OrganizationID uuid.UUID
	Title          string
	Description    *string
	StartsAt       time.Time
	EndsAt         time.Time
	Location       *string
	LeadID         *uuid.UUID
	OwnerMemberID  *uuid.UUID
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Meeting, error) {
	title := strings.TrimSpace(in.Title)
	if len(title) < 2 || len(title) > 200 {
		return Meeting{}, errors.New("title must be 2-200 chars")
	}
	if !in.EndsAt.After(in.StartsAt) {
		return Meeting{}, errors.New("ends_at must be after starts_at")
	}
	const q = `
		INSERT INTO meetings (
		  organization_id, title, description, starts_at, ends_at,
		  location, lead_id, owner_member_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, status::text, created_at, updated_at
	`
	var m Meeting
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, title, in.Description, in.StartsAt, in.EndsAt,
		in.Location, in.LeadID, in.OwnerMemberID,
	).Scan(&m.ID, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return Meeting{}, fmt.Errorf("create meeting: %w", err)
	}
	m.OrganizationID = in.OrganizationID
	m.Title = title
	m.Description = in.Description
	m.StartsAt = in.StartsAt
	m.EndsAt = in.EndsAt
	m.Location = in.Location
	m.LeadID = in.LeadID
	m.OwnerMemberID = in.OwnerMemberID
	return m, nil
}

// ListRange returns meetings that intersect the closed-open window.
// Two meetings intersect when m.starts_at < windowEnd AND m.ends_at > windowStart.
func (r *Repository) ListRange(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]Meeting, error) {
	if !to.After(from) {
		return nil, errors.New("to must be after from")
	}
	const q = `
		SELECT id, organization_id, title, description, starts_at, ends_at,
		       location, lead_id, owner_member_id, status::text,
		       external_provider, external_id, created_at, updated_at
		  FROM meetings
		 WHERE organization_id = $1
		   AND starts_at < $3
		   AND ends_at > $2
		 ORDER BY starts_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list meetings: %w", err)
	}
	defer rows.Close()
	out := make([]Meeting, 0, 16)
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.Title, &m.Description, &m.StartsAt, &m.EndsAt,
			&m.Location, &m.LeadID, &m.OwnerMemberID, &m.Status,
			&m.ExternalProvider, &m.ExternalID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Meeting, error) {
	const q = `
		SELECT id, organization_id, title, description, starts_at, ends_at,
		       location, lead_id, owner_member_id, status::text,
		       external_provider, external_id, created_at, updated_at
		  FROM meetings
		 WHERE organization_id = $1 AND id = $2
		 LIMIT 1
	`
	var m Meeting
	err := r.pool.QueryRow(ctx, q, orgID, id).Scan(
		&m.ID, &m.OrganizationID, &m.Title, &m.Description, &m.StartsAt, &m.EndsAt,
		&m.Location, &m.LeadID, &m.OwnerMemberID, &m.Status,
		&m.ExternalProvider, &m.ExternalID, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Meeting{}, ErrNotFound
	}
	if err != nil {
		return Meeting{}, fmt.Errorf("get meeting: %w", err)
	}
	return m, nil
}

func (r *Repository) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	if status != "scheduled" && status != "completed" && status != "cancelled" && status != "no_show" {
		return fmt.Errorf("invalid status: %s", status)
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE meetings SET status = $3::meeting_status
		  WHERE organization_id = $1 AND id = $2`,
		orgID, id, status,
	)
	if err != nil {
		return fmt.Errorf("set meeting status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetExternal writes the (external_provider, external_id) tuple onto a
// meeting row. Idempotent: re-setting the same pair is a no-op; a
// different pair replaces. Used by the S49 GCal sync goroutine.
func (r *Repository) SetExternal(ctx context.Context, orgID, id uuid.UUID, provider, externalID string) error {
	if provider != "gcal" && provider != "outlook" {
		return fmt.Errorf("invalid external_provider: %s", provider)
	}
	if externalID == "" {
		return errors.New("external_id required")
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE meetings
		    SET external_provider = $3, external_id = $4
		  WHERE organization_id = $1 AND id = $2`,
		orgID, id, provider, externalID,
	)
	if err != nil {
		return fmt.Errorf("set meeting external: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM meetings WHERE organization_id = $1 AND id = $2`,
		orgID, id,
	)
	if err != nil {
		return fmt.Errorf("delete meeting: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

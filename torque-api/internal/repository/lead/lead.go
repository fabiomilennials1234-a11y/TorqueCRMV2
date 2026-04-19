// Package lead is the pgx-backed repository for leads (F01).
//
// Every query is tenant-scoped by organization_id — never accept one from the
// caller's body (StripOrganizationID middleware enforces this at the HTTP
// boundary). Cursor pagination per ADR-004: cursors are opaque base64url
// tokens encoding (updated_at RFC3339, id). Offset pagination is not supported.
package lead

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

// ErrNotFound is returned when the lead is absent or out of tenant scope.
var ErrNotFound = errors.New("lead not found")

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ListOptions carries the filter set for List.
type ListOptions struct {
	Cursor        string // opaque; "" = first page
	Limit         int    // 1..100; zero → default 25
	ResponsibleID *uuid.UUID
	Search        string // name/email/phone substring
}

// ListResult carries the page + next cursor (ADR-004).
type ListResult struct {
	Items      []domain.Lead
	NextCursor string // "" when no more pages
}

// List returns a cursor-paginated slice of active leads. The page is ordered
// by (updated_at DESC, id DESC) so the cursor tuple is monotonic.
func (r *Repository) List(ctx context.Context, orgID uuid.UUID, opts ListOptions) (ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	var cursorTime *time.Time
	var cursorID *uuid.UUID
	if opts.Cursor != "" {
		t, id, err := decodeCursor(opts.Cursor)
		if err != nil {
			return ListResult{}, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorTime = &t
		cursorID = &id
	}

	// Build filters with $N placeholders that survive prepared statements.
	args := []any{orgID, limit + 1}
	var where strings.Builder
	where.WriteString("organization_id = $1 AND deleted_at IS NULL")
	if cursorTime != nil && cursorID != nil {
		args = append(args, *cursorTime, *cursorID)
		where.WriteString(fmt.Sprintf(
			" AND (updated_at, id) < ($%d, $%d)",
			len(args)-1, len(args),
		))
	}
	if opts.ResponsibleID != nil {
		args = append(args, *opts.ResponsibleID)
		where.WriteString(fmt.Sprintf(" AND responsible_id = $%d", len(args)))
	}
	if opts.Search != "" {
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		where.WriteString(fmt.Sprintf(
			" AND (LOWER(name) LIKE $%d OR LOWER(COALESCE(email::text,'')) LIKE $%[1]d OR COALESCE(phone,'') LIKE $%[1]d)",
			len(args),
		))
	}

	q := `
		SELECT id, organization_id, external_id, name, company, phone, email::text,
		       position, cnpj_cpf, responsible_id, sdr_id, closer_id,
		       rating, qualification_score, segment, origin,
		       utm_source, utm_medium, utm_campaign, utm_term, utm_content,
		       custom_fields, first_response_at, last_interaction_at,
		       created_at, updated_at
		  FROM leads
		 WHERE ` + where.String() + `
		 ORDER BY updated_at DESC, id DESC
		 LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Lead, 0, limit)
	for rows.Next() {
		var l domain.Lead
		if err := rows.Scan(
			&l.ID, &l.OrganizationID, &l.ExternalID, &l.Name, &l.Company, &l.Phone, &l.Email,
			&l.Position, &l.CNPJCPF, &l.ResponsibleID, &l.SDRID, &l.CloserID,
			&l.Rating, &l.QualificationScore, &l.Segment, &l.Origin,
			&l.UTMSource, &l.UTMMedium, &l.UTMCampaign, &l.UTMTerm, &l.UTMContent,
			&l.CustomFields, &l.FirstResponseAt, &l.LastInteractionAt,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return ListResult{}, fmt.Errorf("scan lead: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	// If we fetched limit+1, the last one is the cursor for the next page.
	var next string
	if len(out) > limit {
		last := out[limit-1]
		next = encodeCursor(last.UpdatedAt, last.ID)
		out = out[:limit]
	}
	return ListResult{Items: out, NextCursor: next}, nil
}

// Get returns one lead; ErrNotFound if missing or deleted.
func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Lead, error) {
	const q = `
		SELECT id, organization_id, external_id, name, company, phone, email::text,
		       position, cnpj_cpf, responsible_id, sdr_id, closer_id,
		       rating, qualification_score, segment, origin,
		       utm_source, utm_medium, utm_campaign, utm_term, utm_content,
		       custom_fields, first_response_at, last_interaction_at,
		       created_at, updated_at
		  FROM leads
		 WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
		 LIMIT 1
	`
	var l domain.Lead
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&l.ID, &l.OrganizationID, &l.ExternalID, &l.Name, &l.Company, &l.Phone, &l.Email,
		&l.Position, &l.CNPJCPF, &l.ResponsibleID, &l.SDRID, &l.CloserID,
		&l.Rating, &l.QualificationScore, &l.Segment, &l.Origin,
		&l.UTMSource, &l.UTMMedium, &l.UTMCampaign, &l.UTMTerm, &l.UTMContent,
		&l.CustomFields, &l.FirstResponseAt, &l.LastInteractionAt,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Lead{}, ErrNotFound
	}
	if err != nil {
		return domain.Lead{}, fmt.Errorf("get lead: %w", err)
	}
	return l, nil
}

// CreateInput is the write-side shape used by Create.
type CreateInput struct {
	Name          string
	Company       *string
	Phone         *string
	Email         *string
	Position      *string
	ResponsibleID *uuid.UUID
	Origin        *string
	CustomFields  []byte
}

// Create inserts a new lead. Name is required; at least one of phone/email must be set.
func (r *Repository) Create(ctx context.Context, orgID uuid.UUID, in CreateInput) (domain.Lead, error) {
	if in.Name == "" {
		return domain.Lead{}, errors.New("name is required")
	}
	if in.Phone == nil && in.Email == nil {
		return domain.Lead{}, errors.New("phone or email is required")
	}
	if len(in.CustomFields) == 0 {
		in.CustomFields = []byte(`{}`)
	}
	const q = `
		INSERT INTO leads (
		  organization_id, name, company, phone, email, position,
		  responsible_id, origin, custom_fields
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at
	`
	var l domain.Lead
	err := r.pool.QueryRow(ctx, q,
		orgID, in.Name, in.Company, in.Phone, in.Email, in.Position,
		in.ResponsibleID, in.Origin, in.CustomFields,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return domain.Lead{}, fmt.Errorf("insert lead: %w", err)
	}
	l.OrganizationID = orgID
	l.Name = in.Name
	l.Company = in.Company
	l.Phone = in.Phone
	l.Email = in.Email
	l.Position = in.Position
	l.ResponsibleID = in.ResponsibleID
	l.Origin = in.Origin
	l.CustomFields = in.CustomFields
	return l, nil
}

// UpdateInput is the patch shape; only non-nil fields are applied.
type UpdateInput struct {
	Name          *string
	Company       *string
	Phone         *string
	Email         *string
	Position      *string
	ResponsibleID *uuid.UUID
	Rating        *int16
	Segment       *string
}

// Update applies a partial patch. Returns ErrNotFound if the row is missing.
func (r *Repository) Update(ctx context.Context, orgID, id uuid.UUID, p UpdateInput) (domain.Lead, error) {
	sets := make([]string, 0, 8)
	args := []any{id, orgID}
	idx := 3
	add := func(col string, v any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, v)
		idx++
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Company != nil {
		add("company", *p.Company)
	}
	if p.Phone != nil {
		add("phone", *p.Phone)
	}
	if p.Email != nil {
		add("email", *p.Email)
	}
	if p.Position != nil {
		add("position", *p.Position)
	}
	if p.ResponsibleID != nil {
		add("responsible_id", *p.ResponsibleID)
	}
	if p.Rating != nil {
		add("rating", *p.Rating)
	}
	if p.Segment != nil {
		add("segment", *p.Segment)
	}
	if len(sets) == 0 {
		return r.Get(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE leads SET %s WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return domain.Lead{}, fmt.Errorf("update lead: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.Lead{}, ErrNotFound
	}
	return r.Get(ctx, orgID, id)
}

// SoftDelete sets deleted_at to now. Idempotent.
func (r *Repository) SoftDelete(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE leads SET deleted_at = now() WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete lead: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- cursor encoding --------------------------------------------

func encodeCursor(t time.Time, id uuid.UUID) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, errors.New("malformed cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return t, id, nil
}

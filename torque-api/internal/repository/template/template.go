// Package template é o repo pgx dos message_templates (S35).
//
// Member-accessible read + admin-only mutations; ownership checks via
// `organization_id` em toda query. Variáveis são text[] no DB — o
// backend NÃO faz merge de template; isso vive no frontend para
// garantir WYSIWYG.
package template

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("message template not found")
	ErrNameTaken = errors.New("template name already exists in this organization")
	ErrInvalid   = errors.New("invalid template payload")
)

type Template struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Body           string
	Variables      []string
	IsActive       bool
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// List retorna todos os templates do tenant, ordenados alfabeticamente.
// Sem paginação — 100-200 templates por tenant caem na UI confortavelmente.
func (r *Repository) List(ctx context.Context, orgID uuid.UUID, activeOnly bool) ([]Template, error) {
	q := `SELECT id, organization_id, name, body, variables, is_active, created_by,
	             created_at, updated_at
	        FROM message_templates
	       WHERE organization_id = $1`
	if activeOnly {
		q += ` AND is_active = true`
	}
	q += ` ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	out := make([]Template, 0, 16)
	for rows.Next() {
		var t Template
		if err := rows.Scan(
			&t.ID, &t.OrganizationID, &t.Name, &t.Body, &t.Variables,
			&t.IsActive, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Template, error) {
	const q = `SELECT id, organization_id, name, body, variables, is_active, created_by,
	                  created_at, updated_at
	             FROM message_templates
	            WHERE id = $1 AND organization_id = $2
	            LIMIT 1`
	var t Template
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&t.ID, &t.OrganizationID, &t.Name, &t.Body, &t.Variables,
		&t.IsActive, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get template: %w", err)
	}
	return t, nil
}

type CreateInput struct {
	Name      string
	Body      string
	Variables []string
	CreatedBy *uuid.UUID
}

func (r *Repository) Create(ctx context.Context, orgID uuid.UUID, in CreateInput) (Template, error) {
	name := strings.TrimSpace(in.Name)
	body := strings.TrimSpace(in.Body)
	if len(name) < 2 || len(name) > 120 {
		return Template{}, ErrInvalid
	}
	if len(body) < 1 || len(body) > 16000 {
		return Template{}, ErrInvalid
	}
	vars := in.Variables
	if vars == nil {
		vars = []string{}
	}
	const q = `INSERT INTO message_templates (organization_id, name, body, variables, created_by)
	           VALUES ($1, $2, $3, $4, $5)
	           RETURNING id, is_active, created_at, updated_at`
	var t Template
	err := r.pool.QueryRow(ctx, q, orgID, name, body, vars, in.CreatedBy).
		Scan(&t.ID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Template{}, ErrNameTaken
		}
		return Template{}, fmt.Errorf("insert template: %w", err)
	}
	t.OrganizationID = orgID
	t.Name = name
	t.Body = body
	t.Variables = vars
	t.CreatedBy = in.CreatedBy
	return t, nil
}

type UpdateInput struct {
	Name      *string
	Body      *string
	Variables *[]string
	IsActive  *bool
}

func (r *Repository) Update(ctx context.Context, orgID, id uuid.UUID, in UpdateInput) (Template, error) {
	sets := make([]string, 0, 4)
	args := []any{id, orgID}
	idx := 3
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if len(name) < 2 || len(name) > 120 {
			return Template{}, ErrInvalid
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, name)
		idx++
	}
	if in.Body != nil {
		body := strings.TrimSpace(*in.Body)
		if len(body) < 1 || len(body) > 16000 {
			return Template{}, ErrInvalid
		}
		sets = append(sets, fmt.Sprintf("body = $%d", idx))
		args = append(args, body)
		idx++
	}
	if in.Variables != nil {
		sets = append(sets, fmt.Sprintf("variables = $%d", idx))
		args = append(args, *in.Variables)
		idx++
	}
	if in.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		idx++
	}
	if len(sets) == 0 {
		return r.Get(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE message_templates SET %s
	                  WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return Template{}, ErrNameTaken
		}
		return Template{}, fmt.Errorf("update template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return Template{}, ErrNotFound
	}
	return r.Get(ctx, orgID, id)
}

// Delete is hard-delete. ComposerBar client filters is_active=true first,
// so if a tenant ever wants to preserve history, flip is_active via
// Update rather than calling Delete.
func (r *Repository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM message_templates WHERE id = $1 AND organization_id = $2`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

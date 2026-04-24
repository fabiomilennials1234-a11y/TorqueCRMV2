// Package product is the pgx-backed repository for F11 Produtos.
//
// Catalog of products/services that Propostas (F03/F12) reference as line
// items. Prices are integer cents (never float — ever). Cursor pagination
// per ADR-004: cursor is opaque base64url(updated_at RFC3339|id).
package product

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

// MaxPriceCents caps any product line at 1e14 cents (R$ 1 trillion). Above
// this we refuse the insert — defense against accidental int64 overflow in
// proposal totals downstream.
const MaxPriceCents int64 = 1e14

var (
	ErrNotFound       = errors.New("product not found")
	ErrInvalidPrice   = errors.New("price_cents must be between 0 and 1e14")
	ErrInvalidName    = errors.New("name must be between 2 and 160 chars")
	ErrSKUConflict    = errors.New("sku already exists in this organization")
	ErrInvalidCurrency = errors.New("currency must be a 3-letter code")
)

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ListOptions for cursor pagination.
type ListOptions struct {
	Cursor         string
	Limit          int
	ActiveOnly     bool
	Search         string
}

// ListResult is the page + next cursor.
type ListResult struct {
	Items      []domain.Product
	NextCursor string
}

// List returns a cursor-paginated page ordered by (updated_at DESC, id DESC).
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

	args := []any{orgID, limit + 1}
	var where strings.Builder
	where.WriteString("organization_id = $1")
	if opts.ActiveOnly {
		where.WriteString(" AND is_active = true")
	}
	if cursorTime != nil && cursorID != nil {
		args = append(args, *cursorTime, *cursorID)
		where.WriteString(fmt.Sprintf(" AND (updated_at, id) < ($%d, $%d)",
			len(args)-1, len(args)))
	}
	if opts.Search != "" {
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		where.WriteString(fmt.Sprintf(
			" AND (LOWER(name) LIKE $%d OR LOWER(COALESCE(sku,'')) LIKE $%[1]d)",
			len(args),
		))
	}

	q := `
		SELECT id, organization_id, name, description, sku,
		       price_cents, currency, is_active, metadata, created_by,
		       created_at, updated_at
		  FROM products
		 WHERE ` + where.String() + `
		 ORDER BY updated_at DESC, id DESC
		 LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Product, 0, limit)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.Name, &p.Description, &p.SKU,
			&p.PriceCents, &p.Currency, &p.IsActive, &p.Metadata, &p.CreatedBy,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return ListResult{}, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	var next string
	if len(out) > limit {
		last := out[limit-1]
		next = encodeCursor(last.UpdatedAt, last.ID)
		out = out[:limit]
	}
	return ListResult{Items: out, NextCursor: next}, nil
}

// Get returns one product; ErrNotFound if missing or cross-tenant.
func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Product, error) {
	const q = `
		SELECT id, organization_id, name, description, sku,
		       price_cents, currency, is_active, metadata, created_by,
		       created_at, updated_at
		  FROM products
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var p domain.Product
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Description, &p.SKU,
		&p.PriceCents, &p.Currency, &p.IsActive, &p.Metadata, &p.CreatedBy,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, ErrNotFound
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

// CreateInput carries the write-side shape.
type CreateInput struct {
	Name        string
	Description *string
	SKU         *string
	PriceCents  int64
	Currency    string
	Metadata    []byte
	CreatedBy   *uuid.UUID
}

// Create inserts a new product.
func (r *Repository) Create(ctx context.Context, orgID uuid.UUID, in CreateInput) (domain.Product, error) {
	trimmed := strings.TrimSpace(in.Name)
	if len(trimmed) < 2 || len(trimmed) > 160 {
		return domain.Product{}, ErrInvalidName
	}
	if in.PriceCents < 0 || in.PriceCents > MaxPriceCents {
		return domain.Product{}, ErrInvalidPrice
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "BRL"
	}
	if len(currency) != 3 {
		return domain.Product{}, ErrInvalidCurrency
	}
	metadata := in.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}

	const q = `
		INSERT INTO products
		  (organization_id, name, description, sku, price_cents, currency, metadata, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, is_active, created_at, updated_at
	`
	var p domain.Product
	err := r.pool.QueryRow(ctx, q,
		orgID, trimmed, in.Description, in.SKU, in.PriceCents, currency, metadata, in.CreatedBy,
	).Scan(&p.ID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Product{}, ErrSKUConflict
		}
		return domain.Product{}, fmt.Errorf("insert product: %w", err)
	}
	p.OrganizationID = orgID
	p.Name = trimmed
	p.Description = in.Description
	p.SKU = in.SKU
	p.PriceCents = in.PriceCents
	p.Currency = currency
	p.Metadata = metadata
	p.CreatedBy = in.CreatedBy
	return p, nil
}

// UpdateInput is the patch shape.
type UpdateInput struct {
	Name        *string
	Description *string
	SKU         *string
	PriceCents  *int64
	Currency    *string
	IsActive    *bool
	Metadata    []byte
}

// Update applies a partial patch.
func (r *Repository) Update(ctx context.Context, orgID, id uuid.UUID, in UpdateInput) (domain.Product, error) {
	sets := make([]string, 0, 6)
	args := []any{id, orgID}
	idx := 3
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if len(trimmed) < 2 || len(trimmed) > 160 {
			return domain.Product{}, ErrInvalidName
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, trimmed)
		idx++
	}
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *in.Description)
		idx++
	}
	if in.SKU != nil {
		sets = append(sets, fmt.Sprintf("sku = $%d", idx))
		args = append(args, *in.SKU)
		idx++
	}
	if in.PriceCents != nil {
		if *in.PriceCents < 0 || *in.PriceCents > MaxPriceCents {
			return domain.Product{}, ErrInvalidPrice
		}
		sets = append(sets, fmt.Sprintf("price_cents = $%d", idx))
		args = append(args, *in.PriceCents)
		idx++
	}
	if in.Currency != nil {
		c := strings.ToUpper(strings.TrimSpace(*in.Currency))
		if len(c) != 3 {
			return domain.Product{}, ErrInvalidCurrency
		}
		sets = append(sets, fmt.Sprintf("currency = $%d", idx))
		args = append(args, c)
		idx++
	}
	if in.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		idx++
	}
	if len(in.Metadata) > 0 {
		sets = append(sets, fmt.Sprintf("metadata = $%d", idx))
		args = append(args, in.Metadata)
		idx++
	}
	if len(sets) == 0 {
		return r.Get(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE products SET %s WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Product{}, ErrSKUConflict
		}
		return domain.Product{}, fmt.Errorf("update product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.Product{}, ErrNotFound
	}
	return r.Get(ctx, orgID, id)
}

// UpsertBySKUInput is the write-side shape for the S50 TinyERP sync
// path. SKU is required — a product with no SKU cannot be safely
// reconciled across sync calls because we would duplicate it on every
// run.
type UpsertBySKUInput struct {
	Name        string
	Description *string
	SKU         string
	PriceCents  int64
	Currency    string
	Metadata    []byte
}

// UpsertBySKU inserts or updates a product keyed by (organization_id,
// sku). Returns the resulting row + a bool indicating whether it was
// newly inserted (true) or updated in place (false). Used by the
// TinyERP SyncProducts flow.
func (r *Repository) UpsertBySKU(ctx context.Context, orgID uuid.UUID, in UpsertBySKUInput) (domain.Product, bool, error) {
	trimmed := strings.TrimSpace(in.Name)
	if len(trimmed) < 2 || len(trimmed) > 160 {
		return domain.Product{}, false, ErrInvalidName
	}
	sku := strings.TrimSpace(in.SKU)
	if sku == "" {
		return domain.Product{}, false, errors.New("sku is required for upsert-by-sku")
	}
	if in.PriceCents < 0 || in.PriceCents > MaxPriceCents {
		return domain.Product{}, false, ErrInvalidPrice
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "BRL"
	}
	if len(currency) != 3 {
		return domain.Product{}, false, ErrInvalidCurrency
	}
	metadata := in.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}

	const q = `
		INSERT INTO products
		  (organization_id, name, description, sku, price_cents, currency, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (organization_id, sku)
		  WHERE sku IS NOT NULL
		DO UPDATE SET
		  name         = EXCLUDED.name,
		  description  = EXCLUDED.description,
		  price_cents  = EXCLUDED.price_cents,
		  currency     = EXCLUDED.currency,
		  metadata     = EXCLUDED.metadata
		RETURNING id, is_active, created_at, updated_at,
		         (xmax = 0) AS inserted
	`
	var p domain.Product
	var inserted bool
	err := r.pool.QueryRow(ctx, q,
		orgID, trimmed, in.Description, sku, in.PriceCents, currency, metadata,
	).Scan(&p.ID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &inserted)
	if err != nil {
		return domain.Product{}, false, fmt.Errorf("upsert product by sku: %w", err)
	}
	p.OrganizationID = orgID
	p.Name = trimmed
	p.Description = in.Description
	p.SKU = &sku
	p.PriceCents = in.PriceCents
	p.Currency = currency
	p.Metadata = metadata
	return p, inserted, nil
}

// Archive is a soft-delete: flips is_active=false. Actual row is kept so
// existing proposals that reference the product keep their line items
// intact.
func (r *Repository) Archive(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE products SET is_active = false
		  WHERE id = $1 AND organization_id = $2 AND is_active = true`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("archive product: %w", err)
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

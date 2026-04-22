// Package metainsights backs the 15-minute cache for Meta Graph API
// Ads Insights responses (S50 / F.2). Schema in 0025_s50_*.up.sql.
//
// TTL is enforced at query time via `fetched_at > now() - interval`,
// not at write time — a fresh INSERT always ON-CONFLICT-replaces the
// prior row for (org, account_id, date_range), and stale rows are
// simply ignored on read.
package metainsights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no row (fresh or stale) exists.
var ErrNotFound = errors.New("meta insights cache: not found")

// ErrStale is returned when a row exists but is older than the TTL.
// Callers use it as a signal to refetch; the stale row remains until
// the next successful Upsert overwrites it.
var ErrStale = errors.New("meta insights cache: stale")

// Repository is the pgx-backed cache facade.
type Repository struct {
	pool *pgxpool.Pool
	ttl  time.Duration
}

// New binds the repository with the default 15-minute TTL.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, ttl: 15 * time.Minute}
}

// NewWithTTL lets tests pin the TTL.
func NewWithTTL(pool *pgxpool.Pool, ttl time.Duration) *Repository {
	return &Repository{pool: pool, ttl: ttl}
}

// Entry is the cached payload + metadata.
type Entry struct {
	OrganizationID uuid.UUID
	AccountID      string
	DateRange      string
	Payload        json.RawMessage
	FetchedAt      time.Time
}

// Get returns the fresh entry for (org, account, range) or ErrStale /
// ErrNotFound. Caller re-fetches and calls Upsert on ErrStale.
func (r *Repository) Get(ctx context.Context, orgID uuid.UUID, accountID, dateRange string) (Entry, error) {
	if orgID == uuid.Nil || accountID == "" || dateRange == "" {
		return Entry{}, errors.New("metainsights: org/account/range required")
	}
	const q = `
		SELECT organization_id, account_id, date_range, payload, fetched_at
		  FROM meta_insights_cache
		 WHERE organization_id = $1 AND account_id = $2 AND date_range = $3
		 LIMIT 1
	`
	var e Entry
	err := r.pool.QueryRow(ctx, q, orgID, accountID, dateRange).Scan(
		&e.OrganizationID, &e.AccountID, &e.DateRange, &e.Payload, &e.FetchedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("metainsights: get: %w", err)
	}
	if time.Since(e.FetchedAt) > r.ttl {
		return e, ErrStale
	}
	return e, nil
}

// Upsert replaces the cached payload for (org, account, range).
func (r *Repository) Upsert(ctx context.Context, orgID uuid.UUID, accountID, dateRange string, payload json.RawMessage) error {
	if orgID == uuid.Nil || accountID == "" || dateRange == "" {
		return errors.New("metainsights: org/account/range required")
	}
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO meta_insights_cache
		  (organization_id, account_id, date_range, payload, fetched_at)
		VALUES ($1,$2,$3,$4, now())
		ON CONFLICT (organization_id, account_id, date_range) DO UPDATE SET
		  payload    = EXCLUDED.payload,
		  fetched_at = now()
	`
	if _, err := r.pool.Exec(ctx, q, orgID, accountID, dateRange, payload); err != nil {
		return fmt.Errorf("metainsights: upsert: %w", err)
	}
	return nil
}

// Package quota owns the tenant-resource cap bookkeeping that S51 added
// at runtime. Schema: `org_quotas` (per-tenant delta model, 0002) +
// `plan_quotas` (per-plan baseline catalog, 0026).
//
// effective_limit = plan_base + purchased_addons + admin_adjustment
//   (min 0 — admin_adjustment may be negative for manual sanctions)
//
// The middleware in `httpx/middleware/quota.go` hits Get() at request
// admission time. Handlers call IncrementUsage() in the same request
// when the create succeeds — miss the increment and the usage count
// drifts low (safe); double-count and the tenant gets 402 one step
// early (also safe). Neither breaks data integrity.
package quota

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Well-known resource keys. Mirrors the CHECK regex in migration 0002
// (`^[a-z][a-z0-9_]{1,40}$`). Handlers import these rather than typing
// strings so a grep surfaces every call site.
const (
	ResourceLeads        = "leads"
	ResourceTeamMembers  = "team_members"
	ResourceWorkflows    = "workflows"
	ResourceAgents       = "agents"
)

// ErrNotFound is the sentinel for a missing (org, resource) row.
// Callers treat this as "no quota configured" — the middleware returns
// 402 on missing rows to force explicit provisioning.
var ErrNotFound = errors.New("quota: not found for (org, resource)")

// Quota is the full denormalized view the middleware + dashboard consume.
type Quota struct {
	OrganizationID   uuid.UUID
	ResourceKey      string
	PlanBase         int
	PurchasedAddons  int
	AdminAdjustment  int // may be negative
	CurrentUsage     int
	// EffectiveLimit is a derived value: max(0, plan_base + addons + adjustment).
	EffectiveLimit int
	Remaining      int // max(0, effective_limit - current_usage)
}

// Repository is the pgx-backed facade.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Get returns the quota for (org, resource). Returns ErrNotFound when
// no row exists — callers decide whether that means "deny" (middleware)
// or "hide" (dashboard).
func (r *Repository) Get(ctx context.Context, orgID uuid.UUID, resource string) (Quota, error) {
	if orgID == uuid.Nil || resource == "" {
		return Quota{}, errors.New("quota: org and resource required")
	}
	const q = `
		SELECT plan_base, purchased_addons, admin_adjustment, current_usage
		  FROM org_quotas
		 WHERE organization_id = $1 AND resource_key = $2
		 LIMIT 1
	`
	var base, addons, adj, usage int
	err := r.pool.QueryRow(ctx, q, orgID, resource).Scan(&base, &addons, &adj, &usage)
	if errors.Is(err, pgx.ErrNoRows) {
		return Quota{}, ErrNotFound
	}
	if err != nil {
		return Quota{}, fmt.Errorf("quota get: %w", err)
	}
	return toQuota(orgID, resource, base, addons, adj, usage), nil
}

// List returns every quota row for an org. Used by GET /quotas.
func (r *Repository) List(ctx context.Context, orgID uuid.UUID) ([]Quota, error) {
	const q = `
		SELECT resource_key, plan_base, purchased_addons, admin_adjustment, current_usage
		  FROM org_quotas
		 WHERE organization_id = $1
		 ORDER BY resource_key
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("quota list: %w", err)
	}
	defer rows.Close()
	out := make([]Quota, 0, 4)
	for rows.Next() {
		var key string
		var base, addons, adj, usage int
		if err := rows.Scan(&key, &base, &addons, &adj, &usage); err != nil {
			return nil, err
		}
		out = append(out, toQuota(orgID, key, base, addons, adj, usage))
	}
	return out, rows.Err()
}

// IncrementUsage bumps the current_usage for (org, resource) by delta
// (usually 1; -1 for reclaim). Atomic via single UPDATE. Decrements
// that would push below 0 are clamped via GREATEST so the CHECK
// constraint never trips — this is correct because usage is a
// monotonic "how many slots are consumed" scalar, not an accounting
// ledger.
func (r *Repository) IncrementUsage(ctx context.Context, orgID uuid.UUID, resource string, delta int) (int, error) {
	if orgID == uuid.Nil || resource == "" {
		return 0, errors.New("quota: org and resource required")
	}
	const q = `
		UPDATE org_quotas
		   SET current_usage = GREATEST(current_usage + $3, 0),
		       updated_at    = now()
		 WHERE organization_id = $1 AND resource_key = $2
		 RETURNING current_usage
	`
	var newUsage int
	err := r.pool.QueryRow(ctx, q, orgID, resource, delta).Scan(&newUsage)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("quota increment: %w", err)
	}
	return newUsage, nil
}

// SeedPlanDefaults populates org_quotas for (orgID) with plan_base
// values taken from plan_quotas for the supplied plan_id. Safe to call
// repeatedly — ON CONFLICT (org, resource) DO UPDATE only refreshes
// plan_base, preserving purchased_addons / admin_adjustment /
// current_usage.
//
// Called from the subscription activation path (billing webhook) so
// that a tenant graduating from free → growth gets the new ceiling
// applied immediately without a dba intervention.
func (r *Repository) SeedPlanDefaults(ctx context.Context, orgID uuid.UUID, planID string) error {
	if orgID == uuid.Nil || planID == "" {
		return errors.New("quota: org and plan required")
	}
	const q = `
		INSERT INTO org_quotas (organization_id, resource_key, plan_base, current_usage)
		SELECT $1, pq.resource_key, pq.plan_base, 0
		  FROM plan_quotas pq
		 WHERE pq.plan_id = $2
		ON CONFLICT (organization_id, resource_key) DO UPDATE SET
		  plan_base  = EXCLUDED.plan_base,
		  updated_at = now()
	`
	if _, err := r.pool.Exec(ctx, q, orgID, planID); err != nil {
		return fmt.Errorf("quota seed plan defaults: %w", err)
	}
	return nil
}

// SetAdminAdjustment lets a master user override a tenant's effective
// limit without touching plan_base. Positive values grant extra
// headroom; negatives sanction. The ON CONFLICT path creates the row
// if missing so the call never 404s on a resource the tenant has not
// touched yet.
func (r *Repository) SetAdminAdjustment(ctx context.Context, orgID uuid.UUID, resource string, adjustment int) error {
	if orgID == uuid.Nil || resource == "" {
		return errors.New("quota: org and resource required")
	}
	const q = `
		INSERT INTO org_quotas (organization_id, resource_key, admin_adjustment)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, resource_key) DO UPDATE SET
		  admin_adjustment = EXCLUDED.admin_adjustment,
		  updated_at       = now()
	`
	if _, err := r.pool.Exec(ctx, q, orgID, resource, adjustment); err != nil {
		return fmt.Errorf("quota set admin_adjustment: %w", err)
	}
	return nil
}

// SetPurchasedAddons records the purchased-addon count (e.g. after an
// upsell workflow that sells extra seats). Non-negative.
func (r *Repository) SetPurchasedAddons(ctx context.Context, orgID uuid.UUID, resource string, count int) error {
	if orgID == uuid.Nil || resource == "" {
		return errors.New("quota: org and resource required")
	}
	if count < 0 {
		return errors.New("quota: purchased_addons must be non-negative")
	}
	const q = `
		INSERT INTO org_quotas (organization_id, resource_key, purchased_addons)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, resource_key) DO UPDATE SET
		  purchased_addons = EXCLUDED.purchased_addons,
		  updated_at       = now()
	`
	if _, err := r.pool.Exec(ctx, q, orgID, resource, count); err != nil {
		return fmt.Errorf("quota set purchased_addons: %w", err)
	}
	return nil
}

// toQuota computes the derived fields once so the consumer doesn't
// have to.
func toQuota(orgID uuid.UUID, resource string, base, addons, adj, usage int) Quota {
	limit := base + addons + adj
	if limit < 0 {
		limit = 0
	}
	remaining := limit - usage
	if remaining < 0 {
		remaining = 0
	}
	return Quota{
		OrganizationID:  orgID,
		ResourceKey:     resource,
		PlanBase:        base,
		PurchasedAddons: addons,
		AdminAdjustment: adj,
		CurrentUsage:    usage,
		EffectiveLimit:  limit,
		Remaining:       remaining,
	}
}

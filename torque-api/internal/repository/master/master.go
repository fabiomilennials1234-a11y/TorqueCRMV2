// Package master backs F16 — cross-org views available only to users_master
// members. Every query here deliberately does NOT filter by
// organization_id because the caller is by definition global.
//
// Write operations here are limited to Impersonate, which produces a
// tenant-scoped session for a specific organization and logs the action
// to audit_log with actor_type='master'.
package master

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("record not found")

// OrgSummary is the cross-tenant view for the master org list.
type OrgSummary struct {
	ID             uuid.UUID
	Slug           string
	Name           string
	PlanID         *string
	PaymentStatus  string
	MemberCount    int
	LeadCount      int
	CreatedAt      time.Time
}

// SystemHealth snapshots counts across the whole installation.
type SystemHealth struct {
	OrgCount              int
	ActiveOrgCount        int
	UserCount             int
	LeadCount             int
	ActiveSubscriptions   int
	PendingSubscriptions  int
	OperationsRunning     int
	OperationsFailed24h   int
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ListOrganizations returns the master-level org list ordered by created_at
// DESC. Includes soft-deleted rows — master needs visibility.
func (r *Repository) ListOrganizations(ctx context.Context, limit int) ([]OrgSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
		SELECT o.id, o.slug, o.name, o.plan_id, o.payment_status,
		       COALESCE(tm.cnt, 0)   AS member_count,
		       COALESCE(lc.cnt, 0)   AS lead_count,
		       o.created_at
		  FROM organizations o
		  LEFT JOIN (
		    SELECT organization_id, COUNT(*) AS cnt
		      FROM team_members WHERE is_active = true
		     GROUP BY organization_id
		  ) tm ON tm.organization_id = o.id
		  LEFT JOIN (
		    SELECT organization_id, COUNT(*) AS cnt
		      FROM leads WHERE deleted_at IS NULL
		     GROUP BY organization_id
		  ) lc ON lc.organization_id = o.id
		 ORDER BY o.created_at DESC
		 LIMIT $1
	`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()
	out := make([]OrgSummary, 0, 16)
	for rows.Next() {
		var o OrgSummary
		if err := rows.Scan(
			&o.ID, &o.Slug, &o.Name, &o.PlanID, &o.PaymentStatus,
			&o.MemberCount, &o.LeadCount, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetOrganization loads a single org regardless of soft-delete state.
func (r *Repository) GetOrganization(ctx context.Context, id uuid.UUID) (OrgSummary, error) {
	const q = `
		SELECT o.id, o.slug, o.name, o.plan_id, o.payment_status,
		       COALESCE(tm.cnt, 0), COALESCE(lc.cnt, 0), o.created_at
		  FROM organizations o
		  LEFT JOIN (
		    SELECT organization_id, COUNT(*) AS cnt FROM team_members WHERE is_active = true
		     GROUP BY organization_id
		  ) tm ON tm.organization_id = o.id
		  LEFT JOIN (
		    SELECT organization_id, COUNT(*) AS cnt FROM leads WHERE deleted_at IS NULL
		     GROUP BY organization_id
		  ) lc ON lc.organization_id = o.id
		 WHERE o.id = $1
		 LIMIT 1
	`
	var o OrgSummary
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.Slug, &o.Name, &o.PlanID, &o.PaymentStatus,
		&o.MemberCount, &o.LeadCount, &o.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrgSummary{}, ErrNotFound
	}
	if err != nil {
		return OrgSummary{}, fmt.Errorf("get org: %w", err)
	}
	return o, nil
}

// ImpersonationTarget resolves the tenant + admin-level team_member the
// master should assume. If the org has no admin, returns ErrNotFound — a
// master cannot enter an empty tenant (would have no role to assume).
func (r *Repository) ImpersonationTarget(ctx context.Context, orgID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	// Returns (team_member_id, user_id) of the tenant's oldest active admin.
	const q = `
		SELECT tm.id, tm.user_id
		  FROM team_members tm
		  JOIN organizations o ON o.id = tm.organization_id
		 WHERE tm.organization_id = $1
		   AND tm.role = 'admin' AND tm.is_active = true
		   AND o.deleted_at IS NULL
		 ORDER BY tm.joined_at ASC
		 LIMIT 1
	`
	var memberID, userID uuid.UUID
	err := r.pool.QueryRow(ctx, q, orgID).Scan(&memberID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("impersonation target: %w", err)
	}
	return memberID, userID, nil
}

// SystemHealthSnapshot returns counters for the Operations Center view.
func (r *Repository) SystemHealthSnapshot(ctx context.Context) (SystemHealth, error) {
	// Single round-trip via parallel COUNT FILTER. All cheap reads.
	const q = `
		SELECT
		  (SELECT COUNT(*) FROM organizations)                                            AS org_count,
		  (SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL)                   AS active_org_count,
		  (SELECT COUNT(*) FROM users WHERE is_active = true)                             AS user_count,
		  (SELECT COUNT(*) FROM leads WHERE deleted_at IS NULL)                           AS lead_count,
		  (SELECT COUNT(*) FROM subscriptions WHERE status = 'active')                    AS active_subs,
		  (SELECT COUNT(*) FROM subscriptions WHERE status = 'pending')                   AS pending_subs,
		  (SELECT COUNT(*) FROM operations WHERE status = 'running')                      AS ops_running,
		  (SELECT COUNT(*) FROM operations WHERE status = 'failed'
		     AND ended_at >= now() - interval '24 hours')                                 AS ops_failed_24h
	`
	var h SystemHealth
	err := r.pool.QueryRow(ctx, q).Scan(
		&h.OrgCount, &h.ActiveOrgCount, &h.UserCount, &h.LeadCount,
		&h.ActiveSubscriptions, &h.PendingSubscriptions,
		&h.OperationsRunning, &h.OperationsFailed24h,
	)
	if err != nil {
		return SystemHealth{}, fmt.Errorf("system health: %w", err)
	}
	return h, nil
}

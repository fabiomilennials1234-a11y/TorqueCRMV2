// Package user is the persistence port for the global user identity and its
// memberships. All reads happen through pgxpool; every method takes a context
// so timeouts propagate from the request.
package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

// ErrNotFound is returned when the requested row does not exist.
// Callers MUST NOT distinguish this from "bad password" in auth responses.
var ErrNotFound = errors.New("user not found")

// Repository is a pgx-backed store for users + memberships + permissions.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// FindByEmail returns the user record with the given (case-insensitive) email.
// Returns ErrNotFound if no active user matches.
func (r *Repository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `
		SELECT id, email, display_name, password_hash, is_active,
		       ui_mode_preference, email_verified_at, last_login_at,
		       created_at, updated_at
		  FROM users
		 WHERE email = $1
		   AND is_active = true
		 LIMIT 1
	`
	var u domain.User
	var uiMode string
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive,
		&uiMode, &u.EmailVerifiedAt, &u.LastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}
	u.UIMode = domain.UIMode(uiMode)
	return u, nil
}

// FindByID returns the user with the given id. ErrNotFound on absence.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const q = `
		SELECT id, email, display_name, password_hash, is_active,
		       ui_mode_preference, email_verified_at, last_login_at,
		       created_at, updated_at
		  FROM users
		 WHERE id = $1
		   AND is_active = true
		 LIMIT 1
	`
	var u domain.User
	var uiMode string
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.IsActive,
		&uiMode, &u.EmailVerifiedAt, &u.LastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}
	u.UIMode = domain.UIMode(uiMode)
	return u, nil
}

// IsMaster reports whether the user is listed in users_master.
func (r *Repository) IsMaster(ctx context.Context, userID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM users_master WHERE user_id = $1)`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("users_master lookup: %w", err)
	}
	return exists, nil
}

// Memberships returns all active memberships of the user, most recent first.
// Empty slice is valid (a master without any tenant membership).
func (r *Repository) Memberships(ctx context.Context, userID uuid.UUID) ([]domain.Membership, error) {
	const q = `
		SELECT tm.id, tm.organization_id, o.slug, o.name, tm.role, tm.is_active, tm.joined_at
		  FROM team_members tm
		  JOIN organizations o ON o.id = tm.organization_id
		 WHERE tm.user_id = $1
		   AND tm.is_active = true
		   AND o.deleted_at IS NULL
		 ORDER BY tm.joined_at DESC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("memberships: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Membership, 0, 4)
	for rows.Next() {
		var m domain.Membership
		var role string
		if err := rows.Scan(&m.TeamMemberID, &m.OrganizationID, &m.OrgSlug, &m.OrgName, &role, &m.IsActive, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		m.Role = domain.Role(role)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// FindMembership returns the team_members row for (user, org). ErrNotFound if absent.
func (r *Repository) FindMembership(ctx context.Context, userID, orgID uuid.UUID) (domain.Membership, error) {
	const q = `
		SELECT tm.id, tm.organization_id, o.slug, o.name, tm.role, tm.is_active, tm.joined_at
		  FROM team_members tm
		  JOIN organizations o ON o.id = tm.organization_id
		 WHERE tm.user_id = $1
		   AND tm.organization_id = $2
		   AND tm.is_active = true
		   AND o.deleted_at IS NULL
		 LIMIT 1
	`
	var m domain.Membership
	var role string
	err := r.pool.QueryRow(ctx, q, userID, orgID).Scan(
		&m.TeamMemberID, &m.OrganizationID, &m.OrgSlug, &m.OrgName, &role, &m.IsActive, &m.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Membership{}, ErrNotFound
	}
	if err != nil {
		return domain.Membership{}, fmt.Errorf("find membership: %w", err)
	}
	m.Role = domain.Role(role)
	return m, nil
}

// Organization returns the tenant record. ErrNotFound if absent or soft-deleted.
func (r *Repository) Organization(ctx context.Context, orgID uuid.UUID) (domain.Organization, error) {
	const q = `
		SELECT id, slug, name, plan_id, payment_status, logo_url
		  FROM organizations
		 WHERE id = $1
		   AND deleted_at IS NULL
		 LIMIT 1
	`
	var o domain.Organization
	err := r.pool.QueryRow(ctx, q, orgID).Scan(&o.ID, &o.Slug, &o.Name, &o.PlanID, &o.PaymentStatus, &o.LogoURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Organization{}, ErrNotFound
	}
	if err != nil {
		return domain.Organization{}, fmt.Errorf("organization: %w", err)
	}
	return o, nil
}

// TouchLastLogin updates users.last_login_at to now().
// Failures here MUST NOT block login; callers should log and continue.
func (r *Repository) TouchLastLogin(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE users SET last_login_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("touch last_login: %w", err)
	}
	return nil
}

// UpdateUIMode persists a new ui_mode_preference on users.
func (r *Repository) UpdateUIMode(ctx context.Context, userID uuid.UUID, mode domain.UIMode) error {
	if !mode.IsValid() {
		return fmt.Errorf("invalid ui_mode: %s", mode)
	}
	const q = `UPDATE users SET ui_mode_preference = $2 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, userID, string(mode))
	if err != nil {
		return fmt.Errorf("update ui_mode: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EffectivePermissions resolves the full permission bundle for a membership,
// applying the RBAC cascade described in domain.FeaturePermission.
//
// Master bypass is computed by the caller (middleware has the master flag on
// the session). This method returns the RAW resolution without master — the
// caller merges as needed.
func (r *Repository) EffectivePermissions(ctx context.Context, teamMemberID uuid.UUID, role domain.Role) ([]domain.FeaturePermission, error) {
	// Left-join the catalog with the member override. One row per feature_key.
	const q = `
		SELECT fp.feature_key,
		       fp.is_admin_only,
		       fp.master_only,
		       fp.default_value,
		       mfp.value AS member_override
		  FROM feature_permissions fp
		  LEFT JOIN member_feature_permissions mfp
		    ON mfp.feature_key = fp.feature_key
		   AND mfp.team_member_id = $1
	`
	rows, err := r.pool.Query(ctx, q, teamMemberID)
	if err != nil {
		return nil, fmt.Errorf("effective permissions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.FeaturePermission, 0, 32)
	for rows.Next() {
		var key string
		var adminOnly, masterOnly, defaultValue bool
		var override *bool
		if err := rows.Scan(&key, &adminOnly, &masterOnly, &defaultValue, &override); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		out = append(out, resolve(key, role, adminOnly, masterOnly, defaultValue, override))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// resolve applies the non-master portion of the RBAC cascade. Kept as a pure
// function so it is trivially unit-testable.
func resolve(key string, role domain.Role, adminOnly, masterOnly, defaultValue bool, override *bool) domain.FeaturePermission {
	switch {
	case masterOnly:
		return domain.FeaturePermission{Key: key, Allowed: false, Source: "master_only"}
	case adminOnly && role != domain.RoleAdmin:
		return domain.FeaturePermission{Key: key, Allowed: false, Source: "admin_only"}
	case override != nil:
		return domain.FeaturePermission{Key: key, Allowed: *override, Source: "member_override"}
	default:
		return domain.FeaturePermission{Key: key, Allowed: defaultValue, Source: "default"}
	}
}

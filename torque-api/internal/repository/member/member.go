// Package member is the pgx-backed repository for F10 Equipe.
//
// team_members is a tenant-scoped join between a global user and an
// organization. S21 exposes CRUD on membership + per-member permission
// overrides. Full user creation (invite flow with email token + signup) is
// deferred; S21 accepts a pre-existing user_id or an email that already
// matches an active user — the admin flow assumes onboarding happens
// elsewhere.
package member

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
)

var (
	ErrNotFound       = errors.New("team member not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrAlreadyMember  = errors.New("user is already a member of this organization")
	ErrInvalidRole    = errors.New("invalid role")
	ErrCannotSelfRole = errors.New("cannot demote own admin role")
)

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// List returns all memberships of the org. Ordered by display_name ASC so the
// Settings UI can render without post-sorting. No cursor — teams rarely
// exceed a few hundred; if a tenant grows past that we add pagination then.
func (r *Repository) List(ctx context.Context, orgID uuid.UUID, includeInactive bool) ([]domain.TeamMember, error) {
	q := `
		SELECT tm.id, tm.organization_id, tm.user_id, tm.role,
		       tm.display_name, u.email::text, tm.avatar_url, tm.is_active,
		       tm.invited_by, tm.invited_at, tm.joined_at, tm.deactivated_at,
		       tm.created_at, tm.updated_at
		  FROM team_members tm
		  JOIN users u ON u.id = tm.user_id
		 WHERE tm.organization_id = $1
	`
	if !includeInactive {
		q += " AND tm.is_active = true"
	}
	q += " ORDER BY tm.display_name ASC"

	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	out := make([]domain.TeamMember, 0, 16)
	for rows.Next() {
		var tm domain.TeamMember
		var role string
		if err := rows.Scan(
			&tm.ID, &tm.OrganizationID, &tm.UserID, &role,
			&tm.DisplayName, &tm.Email, &tm.AvatarURL, &tm.IsActive,
			&tm.InvitedBy, &tm.InvitedAt, &tm.JoinedAt, &tm.DeactivatedAt,
			&tm.CreatedAt, &tm.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		tm.Role = domain.Role(role)
		out = append(out, tm)
	}
	return out, rows.Err()
}

// Get returns a single membership scoped to the tenant.
func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (domain.TeamMember, error) {
	const q = `
		SELECT tm.id, tm.organization_id, tm.user_id, tm.role,
		       tm.display_name, u.email::text, tm.avatar_url, tm.is_active,
		       tm.invited_by, tm.invited_at, tm.joined_at, tm.deactivated_at,
		       tm.created_at, tm.updated_at
		  FROM team_members tm
		  JOIN users u ON u.id = tm.user_id
		 WHERE tm.id = $1 AND tm.organization_id = $2
		 LIMIT 1
	`
	var tm domain.TeamMember
	var role string
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&tm.ID, &tm.OrganizationID, &tm.UserID, &role,
		&tm.DisplayName, &tm.Email, &tm.AvatarURL, &tm.IsActive,
		&tm.InvitedBy, &tm.InvitedAt, &tm.JoinedAt, &tm.DeactivatedAt,
		&tm.CreatedAt, &tm.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TeamMember{}, ErrNotFound
	}
	if err != nil {
		return domain.TeamMember{}, fmt.Errorf("get member: %w", err)
	}
	tm.Role = domain.Role(role)
	return tm, nil
}

// AddByEmailInput is the request shape for binding an existing user to the
// tenant as a member. If no user with the email exists the caller gets
// ErrUserNotFound and is expected to drive the signup flow elsewhere.
type AddByEmailInput struct {
	Email       string
	DisplayName string
	Role        domain.Role
	InvitedBy   uuid.UUID
}

// AddByEmail binds an existing user (lookup by email) to the organization
// with the given role. This is not a full invite — it assumes the user
// account already exists (admin-created elsewhere or pre-provisioned).
func (r *Repository) AddByEmail(ctx context.Context, orgID uuid.UUID, in AddByEmailInput) (domain.TeamMember, error) {
	if !in.Role.IsValid() {
		return domain.TeamMember{}, ErrInvalidRole
	}
	if strings.TrimSpace(in.DisplayName) == "" {
		return domain.TeamMember{}, errors.New("display_name is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.TeamMember{}, fmt.Errorf("begin add: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1 AND is_active = true LIMIT 1`,
		strings.ToLower(strings.TrimSpace(in.Email)),
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TeamMember{}, ErrUserNotFound
	}
	if err != nil {
		return domain.TeamMember{}, fmt.Errorf("lookup user: %w", err)
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO team_members
		    (organization_id, user_id, role, display_name, invited_by, invited_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 RETURNING id`,
		orgID, userID, string(in.Role), in.DisplayName, in.InvitedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.TeamMember{}, ErrAlreadyMember
		}
		return domain.TeamMember{}, fmt.Errorf("insert member: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.TeamMember{}, fmt.Errorf("commit add: %w", err)
	}
	return r.Get(ctx, orgID, id)
}

// UpdateInput patches a membership. Only non-nil fields are applied.
type UpdateInput struct {
	DisplayName *string
	Role        *domain.Role
	AvatarURL   *string
	IsActive    *bool
}

// Update patches a membership. Cannot be used to change tenant or user.
func (r *Repository) Update(ctx context.Context, orgID, id uuid.UUID, in UpdateInput) (domain.TeamMember, error) {
	sets := make([]string, 0, 4)
	args := []any{id, orgID}
	idx := 3
	if in.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", idx))
		args = append(args, *in.DisplayName)
		idx++
	}
	if in.Role != nil {
		if !in.Role.IsValid() {
			return domain.TeamMember{}, ErrInvalidRole
		}
		sets = append(sets, fmt.Sprintf("role = $%d", idx))
		args = append(args, string(*in.Role))
		idx++
	}
	if in.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", idx))
		args = append(args, *in.AvatarURL)
		idx++
	}
	if in.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		idx++
		if !*in.IsActive {
			sets = append(sets, "deactivated_at = now()")
		} else {
			sets = append(sets, "deactivated_at = NULL")
		}
	}
	if len(sets) == 0 {
		return r.Get(ctx, orgID, id)
	}
	q := fmt.Sprintf(
		`UPDATE team_members SET %s WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "),
	)
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return domain.TeamMember{}, fmt.Errorf("update member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.TeamMember{}, ErrNotFound
	}
	return r.Get(ctx, orgID, id)
}

// Deactivate flips is_active=false; joined_at is preserved for audit.
func (r *Repository) Deactivate(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE team_members
		    SET is_active = false, deactivated_at = now()
		  WHERE id = $1 AND organization_id = $2 AND is_active = true`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("deactivate member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- per-member permission overrides ----------------------------

// Override represents one row in member_feature_permissions.
type Override struct {
	FeatureKey string `json:"feature_key"`
	Value      bool   `json:"value"`
}

// ListOverrides returns the override rows only (not the cascade resolution).
// The auth bundle returns the resolved cascade via EffectivePermissions; this
// endpoint is for the Settings UI that wants to show which keys are explicitly
// overridden vs inheriting the default.
func (r *Repository) ListOverrides(ctx context.Context, orgID, teamMemberID uuid.UUID) ([]Override, error) {
	// Tenant-guard: fail fast if the member is out of tenant scope.
	var ok bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE id=$1 AND organization_id=$2)`,
		teamMemberID, orgID,
	).Scan(&ok); err != nil {
		return nil, fmt.Errorf("check member: %w", err)
	}
	if !ok {
		return nil, ErrNotFound
	}
	rows, err := r.pool.Query(ctx,
		`SELECT feature_key, value
		   FROM member_feature_permissions
		  WHERE team_member_id = $1
		  ORDER BY feature_key ASC`,
		teamMemberID,
	)
	if err != nil {
		return nil, fmt.Errorf("list overrides: %w", err)
	}
	defer rows.Close()
	out := make([]Override, 0, 8)
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.FeatureKey, &o.Value); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// SetOverride upserts a single override. Validates that featureKey exists in
// the catalog (FK would catch it, but we return a typed error instead of a
// pgconn leak).
func (r *Repository) SetOverride(ctx context.Context, orgID, teamMemberID uuid.UUID, featureKey string, value bool, updatedBy uuid.UUID) error {
	// Tenant-guard before write.
	var ok bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE id=$1 AND organization_id=$2)`,
		teamMemberID, orgID,
	).Scan(&ok); err != nil {
		return fmt.Errorf("check member: %w", err)
	}
	if !ok {
		return ErrNotFound
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO member_feature_permissions (team_member_id, feature_key, value, updated_by)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (team_member_id, feature_key)
		 DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		teamMemberID, featureKey, value, updatedBy,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("unknown feature_key: %s", featureKey)
		}
		return fmt.Errorf("upsert override: %w", err)
	}
	return nil
}

// ClearOverride removes an explicit override so the member reverts to the
// feature default (or admin_only cascade).
func (r *Repository) ClearOverride(ctx context.Context, orgID, teamMemberID uuid.UUID, featureKey string) error {
	var ok bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE id=$1 AND organization_id=$2)`,
		teamMemberID, orgID,
	).Scan(&ok); err != nil {
		return fmt.Errorf("check member: %w", err)
	}
	if !ok {
		return ErrNotFound
	}
	_, err := r.pool.Exec(ctx,
		`DELETE FROM member_feature_permissions
		  WHERE team_member_id = $1 AND feature_key = $2`,
		teamMemberID, featureKey,
	)
	if err != nil {
		return fmt.Errorf("clear override: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

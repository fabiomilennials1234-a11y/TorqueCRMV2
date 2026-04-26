// Package permission resolves the effective feature permissions for a session.
//
// It sits between the RBAC middleware and the user repository so the middleware
// package stays free of database imports. The resolver caches nothing by
// default — tenant state changes must be reflected within one request. If
// profiling later reveals hot keys, we can layer an LRU here; do not add one
// without measurement.
package permission

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
)

// EffectivePermissionsRepo is the narrow slice of the user repository
// the resolver needs. Keeping it local lets the unit tests drop in a
// fake without spinning a pgxpool — S69 (D080).
type EffectivePermissionsRepo interface {
	EffectivePermissions(ctx context.Context, teamMemberID uuid.UUID, role domain.Role) ([]domain.FeaturePermission, error)
}

// Resolver is the PermissionResolver wiring the RBAC middleware expects.
type Resolver struct {
	users EffectivePermissionsRepo
}

// New returns a resolver backed by the user repository.
func New(users *userrepo.Repository) *Resolver {
	return &Resolver{users: users}
}

// NewWithRepo returns a resolver backed by any EffectivePermissionsRepo.
// Used in tests to inject a fake without pgxpool.
func NewWithRepo(users EffectivePermissionsRepo) *Resolver {
	return &Resolver{users: users}
}

// Resolve returns the effective permission for (session, featureKey).
// Master is handled by the middleware BEFORE Resolve is called, so this method
// does not need to re-check master.
func (r *Resolver) Resolve(ctx context.Context, sess domain.Session, featureKey string) (domain.FeaturePermission, error) {
	perms, err := r.users.EffectivePermissions(ctx, sess.TeamMemberID, sess.Role)
	if err != nil {
		return domain.FeaturePermission{}, fmt.Errorf("resolve %s: %w", featureKey, err)
	}
	for _, p := range perms {
		if p.Key == featureKey {
			return p, nil
		}
	}
	// Unknown key — fail closed. The catalog is the source of truth, so a missing
	// key means either a typo in the RequireFeature call site or a migration gap.
	return domain.FeaturePermission{Key: featureKey, Allowed: false, Source: "unknown"}, nil
}

// Bundle returns the full permission set for the session (master bypass
// applied). Used by /auth/me to ship a single bootstrap bundle to the client.
func (r *Resolver) Bundle(ctx context.Context, sess domain.Session) ([]domain.FeaturePermission, error) {
	perms, err := r.users.EffectivePermissions(ctx, sess.TeamMemberID, sess.Role)
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}
	if !sess.IsMaster {
		return perms, nil
	}
	// Master: replace any denials from admin_only / master_only with allow, so
	// the frontend shell does not render negative UI for an omnipotent user.
	out := make([]domain.FeaturePermission, len(perms))
	for i, p := range perms {
		p.Allowed = true
		p.Source = "master_bypass"
		out[i] = p
	}
	return out, nil
}

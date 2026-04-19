package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/milennials/torque-api/internal/domain"
)

// PermissionResolver loads the effective permission for a single feature_key.
//
// Implementations typically cache the full bundle per team_member_id for the
// duration of the request; the middleware calls into this interface rather
// than holding a repository directly to keep packaging acyclic.
type PermissionResolver interface {
	Resolve(ctx context.Context, sess domain.Session, featureKey string) (domain.FeaturePermission, error)
}

// RequireFeature gates a route on a single feature permission.
//
// Cascade order (highest priority first):
//   1. Master bypass  — users_master entries access everything.
//   2. master_only    — even admins cannot bypass.
//   3. is_admin_only  — non-admins denied.
//   4. member_override — explicit per-member boolean wins.
//   5. default_value  — catalog default.
//
// Any failure to resolve is 500 (we prefer fail-closed over a permissive
// degraded mode when the permission store is unavailable).
func RequireFeature(resolver PermissionResolver, featureKey string) func(next http.Handler) http.Handler {
	if featureKey == "" {
		panic("RequireFeature: empty feature_key")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := SessionFrom(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}

			// 1. Master bypass — short-circuit without a store hit.
			if sess.IsMaster {
				next.ServeHTTP(w, r)
				return
			}

			perm, err := resolver.Resolve(r.Context(), sess, featureKey)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "PERMISSION_RESOLVE_FAILED",
					fmt.Sprintf("could not resolve %s", featureKey))
				return
			}
			if !perm.Allowed {
				writeError(w, http.StatusForbidden, "PERMISSION_DENIED",
					fmt.Sprintf("permission %s denied", featureKey))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole gates a route on a minimum role. Master always passes.
//
// Use only where the permission catalog has no matching key — prefer
// RequireFeature when possible, since explicit catalog keys are auditable.
func RequireRole(role domain.Role) func(next http.Handler) http.Handler {
	if !role.IsValid() {
		panic(fmt.Sprintf("RequireRole: invalid role %q", role))
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := SessionFrom(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			if sess.IsMaster {
				next.ServeHTTP(w, r)
				return
			}
			if role == domain.RoleAdmin && sess.Role != domain.RoleAdmin {
				writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireMaster gates a route to master-only operations.
// These routes ALWAYS produce an audit log entry; that is handled one layer up.
func RequireMaster(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		if !sess.IsMaster {
			writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "master role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

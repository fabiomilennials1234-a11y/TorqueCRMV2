package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// tenantCtxKey is the context key for the resolved organization_id.
// Even though the session already carries org_id, repositories frequently
// want it alone — exposing it as a standalone value keeps their signatures
// from depending on domain.Session.
type tenantCtxKey struct{}

// OrgIDFrom returns the active organization id. Zero uuid and false when
// the request did not pass through RequireAuth (or a public route).
func OrgIDFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(tenantCtxKey{}).(uuid.UUID)
	return v, ok
}

// TenantScope promotes the session's organization_id into a standalone
// context value that repositories read. Composes AFTER RequireAuth.
//
// This middleware does NOT touch request bodies — that responsibility is in
// StripOrganizationID, which runs earlier in the stack as a second line of
// defense independent of auth.
func TenantScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFrom(r.Context())
		if !ok {
			// Defensive: do not 500 here. RequireAuth should have rejected.
			// If the stack is misconfigured, surface it as 401 rather than expose.
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		if sess.OrganizationID == uuid.Nil {
			// A session without org_id is valid ONLY for /auth/me when the user
			// has zero memberships — not for tenant-scoped routes.
			writeError(w, http.StatusForbidden, "NO_TENANT", "session has no active organization")
			return
		}
		ctx := context.WithValue(r.Context(), tenantCtxKey{}, sess.OrganizationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

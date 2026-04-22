// Quota middleware (S51 / Fase G.1) — 402 QUOTA_EXCEEDED gate.
//
// Drops the request with 402 when the tenant's current_usage has
// caught up to effective_limit for the named resource. Runs AFTER
// TenantScope (needs orgID in context) and BEFORE the handler.
//
// Increment happens in the handler AFTER a successful create — the
// middleware doesn't try to auto-increment because it can't tell
// whether the handler will actually go through. This is a deliberate
// two-step pattern:
//
//   1. middleware: "do we have room?"           → 402 if not
//   2. handler:    "try the create; if it works, increment"
//
// The tiny window between admission and increment lets one extra
// create slip past a race but never a destructive overrun — the
// worst case is one-over-cap, which is indistinguishable from the
// tenant's own concurrent clicks.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/repository/quota"
)

// QuotaReader is the narrow slice the middleware needs from the quota
// repository. Exposed as an interface so tests can drop in a fake
// without spinning a pgxpool.
type QuotaReader interface {
	Get(ctx context.Context, orgID uuid.UUID, resource string) (quota.Quota, error)
}

// RequireQuota builds middleware that 402s when the tenant is at cap
// for `resource`. Apply it at the route-group level right after
// TenantScope:
//
//	t.With(mw.RequireQuota(quotaRepo, quota.ResourceLeads)).Post("/leads", h.create)
//
// Master users bypass the check — they routinely drive tenants from
// impersonation sessions where the usage would otherwise balloon.
func RequireQuota(reader QuotaReader, resource string) func(next http.Handler) http.Handler {
	if resource == "" {
		panic("RequireQuota: empty resource")
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
			orgID, ok := OrgIDFrom(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "tenant scope required")
				return
			}

			q, err := reader.Get(r.Context(), orgID, resource)
			if errors.Is(err, quota.ErrNotFound) {
				// No row means the tenant has NOT been provisioned for
				// this resource. Fail-closed (402) is safer than
				// fail-open — the subscription.activated path seeds
				// quotas, so a missing row indicates a misconfiguration
				// the operator needs to see as a hard error, not a
				// silent free pass.
				writeQuotaExceeded(w, resource, 0, 0, 0)
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "QUOTA_LOOKUP_FAILED",
					"could not resolve quota")
				return
			}
			if q.CurrentUsage >= q.EffectiveLimit {
				writeQuotaExceeded(w, resource, q.EffectiveLimit, q.CurrentUsage, q.Remaining)
				return
			}

			// Helpful observability headers; not contractual. Frontend
			// upsell banners can read them without a second round-trip.
			w.Header().Set("X-Quota-Resource", resource)
			w.Header().Set("X-Quota-Limit", strconv.Itoa(q.EffectiveLimit))
			w.Header().Set("X-Quota-Usage", strconv.Itoa(q.CurrentUsage))
			w.Header().Set("X-Quota-Remaining", strconv.Itoa(q.Remaining))

			next.ServeHTTP(w, r)
		})
	}
}

// writeQuotaExceeded emits the canonical 402 response with details
// the frontend upsell flow consumes (see useQuotas + QuotaBanner).
func writeQuotaExceeded(w http.ResponseWriter, resource string, limit, usage, remaining int) {
	w.Header().Set("X-Quota-Resource", resource)
	w.Header().Set("X-Quota-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Quota-Usage", strconv.Itoa(usage))
	w.Header().Set("X-Quota-Remaining", strconv.Itoa(remaining))
	// 402 Payment Required is the cleanest semantic match — the
	// tenant CAN unblock by upgrading the plan or buying addons.
	// HTTP/1.1 reserves the status code for "future use"; we reclaim
	// it per the common SaaS convention.
	writeQuotaJSON(w, http.StatusPaymentRequired, map[string]any{
		"error": map[string]any{
			"code":    "QUOTA_EXCEEDED",
			"message": "limite do recurso atingido — atualize o plano ou compre add-ons",
			"details": map[string]any{
				"resource":       resource,
				"limit":          limit,
				"current_usage":  usage,
				"remaining":      remaining,
			},
		},
	})
}

func writeQuotaJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

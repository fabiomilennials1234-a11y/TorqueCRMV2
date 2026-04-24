// AI-specific quota middleware (S52).
//
// Why a separate file from quota.go:
//
//   * Regular quotas (leads, team_members, workflows, agents) track
//     unit-cost resources. The handler knows the cost is "1" at
//     admission time and can pre-compute admission via
//     `current_usage < effective_limit`. The middleware doesn't even
//     need the actual cost.
//
//   * AI tokens (and, to a lesser degree, TTS seconds) are variable-
//     cost. The handler only knows the final cost AFTER the upstream
//     provider closes the stream. The middleware therefore does ONLY
//     the admission check — no pre-increment, no speculative charge.
//     The handler increments post-stream once `InputTokens + OutputTokens`
//     comes back on the terminal frame.
//
//   * A second distinct fail mode: a tenant who is already AT cap
//     must 402 BEFORE the request touches the provider at all (a
//     "you've burned through the month's budget" response should cost
//     zero tokens). RequireAITokenBudget is the last gate before the
//     provider dial.
package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/milennials/torque-api/internal/repository/quota"
)

// RequireAITokenBudget builds middleware that 402s when the tenant has
// reached its `ai_tokens` cap. Apply at the Playground + any other
// endpoint that dials a paid LLM provider:
//
//	admin.With(mw.RequireAITokenBudget(quotaRepo)).
//	      Post("/agents/{id}/playground/message", h.playground)
//
// Behavioural contract:
//
//   * Master users bypass (impersonation sessions must not double-charge
//     the target tenant).
//   * Missing org_quotas row → fail-CLOSED with 402. The subscription
//     webhook seeds quotas on plan activation, so a missing row means
//     the operator misconfigured something — silent free pass would
//     hide that.
//   * No pre-increment. The handler itself calls
//     `quotaRepo.IncrementUsage(ctx, orgID, ResourceAITokens, total)`
//     ONCE per successful stream. A mid-stream provider error skips
//     the increment so a retry doesn't double-charge.
//
// The 402 body is JSON and mirrors the shape returned by the existing
// RequireQuota middleware (`error.code=AI_QUOTA_EXCEEDED`, details
// with remaining + limit in PT-BR).
func RequireAITokenBudget(reader QuotaReader) func(next http.Handler) http.Handler {
	return requireAIBudget(reader, quota.ResourceAITokens, "AI_QUOTA_EXCEEDED",
		"cota de tokens de IA atingida — atualize o plano ou aguarde o próximo ciclo")
}

// RequireTTSBudget is the twin for `tts_seconds`. Same semantics.
func RequireTTSBudget(reader QuotaReader) func(next http.Handler) http.Handler {
	return requireAIBudget(reader, quota.ResourceTTSSeconds, "TTS_QUOTA_EXCEEDED",
		"cota de segundos de TTS atingida — atualize o plano ou aguarde o próximo ciclo")
}

func requireAIBudget(reader QuotaReader, resource, code, message string) func(next http.Handler) http.Handler {
	if resource == "" {
		panic("requireAIBudget: empty resource")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := SessionFrom(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			if sess.IsMaster {
				// Master sessions are an operator tool; they must never
				// charge the impersonated tenant's AI meter.
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
				// Fail-closed. See doc comment.
				writeAIBudgetExceeded(w, resource, code, message, 0, 0, 0)
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "QUOTA_LOOKUP_FAILED",
					"could not resolve ai quota")
				return
			}
			if q.CurrentUsage >= q.EffectiveLimit {
				writeAIBudgetExceeded(w, resource, code, message,
					q.EffectiveLimit, q.CurrentUsage, q.Remaining)
				return
			}
			// Observability — frontend upsell banners read these.
			w.Header().Set("X-Quota-Resource", resource)
			w.Header().Set("X-Quota-Limit", strconv.Itoa(q.EffectiveLimit))
			w.Header().Set("X-Quota-Usage", strconv.Itoa(q.CurrentUsage))
			w.Header().Set("X-Quota-Remaining", strconv.Itoa(q.Remaining))
			next.ServeHTTP(w, r)
		})
	}
}

// writeAIBudgetExceeded renders the 402 body. Distinct code constants
// let the frontend disambiguate an AI budget miss from a regular quota
// miss (different upsell copy).
func writeAIBudgetExceeded(w http.ResponseWriter, resource, code, msgPT string, limit, usage, remaining int) {
	w.Header().Set("X-Quota-Resource", resource)
	w.Header().Set("X-Quota-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Quota-Usage", strconv.Itoa(usage))
	w.Header().Set("X-Quota-Remaining", strconv.Itoa(remaining))
	writeQuotaJSON(w, http.StatusPaymentRequired, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msgPT,
			"details": map[string]any{
				"resource":      resource,
				"limit":         limit,
				"current_usage": usage,
				"remaining":     remaining,
			},
		},
	})
}

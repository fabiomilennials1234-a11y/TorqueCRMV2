package middleware

import (
	"net/http"

	"github.com/milennials/torque-api/internal/service/token"
)

// CSRFCookieName is the readable (non-httpOnly) cookie that holds the
// double-submit token. Frontend copies its value into X-CSRF-Token on mutations.
const CSRFCookieName = "__torque_csrf"

// CSRFHeaderName is the HTTP header the frontend echoes the cookie value in.
const CSRFHeaderName = "X-CSRF-Token"

// CSRF enforces the double-submit-cookie pattern for state-changing methods.
//
// Strict SameSite already blocks cross-site cookie sends; CSRF here is a
// defense-in-depth layer against same-site variants (compromised subdomain,
// reflected XSS that can set headers but not read cookies, etc).
//
// Behavior:
//   * Safe methods (GET, HEAD, OPTIONS) pass through untouched.
//   * On mutation, both the cookie and the header MUST be present AND equal
//     in constant time. Any miss returns 403.
//   * Auth routes that intentionally lack a prior cookie (login) are expected
//     to be mounted OUTSIDE this chain.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusForbidden, "CSRF_MISSING_COOKIE",
				"csrf token cookie required for mutations")
			return
		}
		header := r.Header.Get(CSRFHeaderName)
		if header == "" {
			writeError(w, http.StatusForbidden, "CSRF_MISSING_HEADER",
				"csrf token header required for mutations")
			return
		}
		if err := token.ConstantTimeEqual(cookie.Value, header); err != nil {
			writeError(w, http.StatusForbidden, "CSRF_MISMATCH",
				"csrf token mismatch")
			return
		}

		next.ServeHTTP(w, r)
	})
}

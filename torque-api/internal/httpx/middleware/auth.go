// Package middleware's auth.go implements cookie-based session authentication.
//
// The access token lives in `__torque_session` (httpOnly, SameSite=Strict,
// Secure in non-dev). This middleware:
//
//  1. Reads the cookie.
//  2. Verifies the signature and expiry (service/jwt).
//  3. Rehydrates a domain.Session.
//  4. Injects it into the request context.
//  5. On any failure, 401s WITHOUT leaking which step failed.
//
// The middleware is offered in two flavors:
//   * Authenticator — parses the cookie if present, does NOT require it.
//     Use on public routes that want to render differently for logged-in users
//     (e.g. /api/bootstrap).
//   * RequireAuth   — fails closed with 401.
//     Wrap /api/v1 routes with this.
package middleware

import (
	"errors"
	"net/http"

	"github.com/milennials/torque-api/internal/service/jwt"
)

// SessionCookieName is the httpOnly cookie carrying the access JWT.
const SessionCookieName = "__torque_session"

// Authenticator parses the session cookie if present and attaches it to the
// context. On failure, the request proceeds as anonymous. Expired or tampered
// cookies produce an anonymous request, never an error response.
func Authenticator(jwtsvc *jwt.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := jwtsvc.Parse(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r) // silently anonymous
				return
			}
			sess, err := claims.Session()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithSession(r.Context(), sess)))
		})
	}
}

// RequireAuth is the gate for protected routes. Must be composed AFTER
// Authenticator; it only reads from context.
//
// Returns 401 with a generic body — clients must not distinguish "no cookie"
// from "expired cookie" from "invalid cookie".
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := SessionFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ErrNoSession is a convenience error for callers that need to distinguish
// "no session" in their own logic after calling SessionFrom.
var ErrNoSession = errors.New("no session in context")

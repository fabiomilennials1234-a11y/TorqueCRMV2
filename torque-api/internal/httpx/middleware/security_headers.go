package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// SecurityHeadersConfig parameterizes the defensive header set.
// Zero-value is production-safe (HSTS enabled, CSP restrictive).
type SecurityHeadersConfig struct {
	// EnableHSTS toggles Strict-Transport-Security. Must be true in staging/prod,
	// and false only over plain-http localhost.
	EnableHSTS bool
	// HSTSMaxAge is the HSTS lifetime in seconds. 63072000 = 2 years.
	HSTSMaxAge time.Duration
	// HSTSPreload requests browser preload-list inclusion. Requires
	// includeSubDomains and a permanent HTTPS commitment.
	HSTSPreload bool
}

// DefaultSecurityHeadersConfig returns the 2026 hardening baseline.
// Use this in prod/staging. Dev may override EnableHSTS to false.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableHSTS:  true,
		HSTSMaxAge:  2 * 365 * 24 * time.Hour, // 2 years
		HSTSPreload: true,
	}
}

// SecurityHeaders sets defensive headers on every response.
//
// Headers included:
//   * Content-Security-Policy — restrictive default-src 'none'; API never renders
//     HTML, so no script-src nonce is required. If we ever serve an HTML error
//     page, tighten here with a per-request nonce via a separate middleware.
//   * Strict-Transport-Security — 2-year preload (toggled by config).
//   * X-Frame-Options DENY — legacy clickjacking guard below frame-ancestors.
//   * X-Content-Type-Options nosniff — stop MIME sniffing downgrades.
//   * Referrer-Policy strict-origin-when-cross-origin — do not leak paths.
//   * Permissions-Policy — deny every power we do not use.
//   * Cross-Origin-Opener-Policy same-origin — isolate browsing context.
//   * Cross-Origin-Resource-Policy same-site — block cross-origin embeds.
//
// The signature returns a `func(http.Handler) http.Handler` so it composes in
// `chi.Router.Use`. Existing callers using the legacy `SecurityHeaders` handler
// still work — it now delegates to this configured version with the default.
func SecurityHeadersWith(cfg SecurityHeadersConfig) func(next http.Handler) http.Handler {
	hstsValue := ""
	if cfg.EnableHSTS {
		seconds := int(cfg.HSTSMaxAge.Seconds())
		if seconds <= 0 {
			seconds = int((2 * 365 * 24 * time.Hour).Seconds())
		}
		hstsValue = fmt.Sprintf("max-age=%d; includeSubDomains", seconds)
		if cfg.HSTSPreload {
			hstsValue += "; preload"
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			// CSP tight enough for a pure-JSON API. No inline scripts, no images,
			// no frames. If a response ever serves HTML intentionally, override
			// this per-route with a nonced policy.
			h.Set("Content-Security-Policy",
				"default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
			if hstsValue != "" {
				h.Set("Strict-Transport-Security", hstsValue)
			}
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy",
				"accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-site")
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders is the legacy entrypoint preserved for backward compatibility.
// It uses DefaultSecurityHeadersConfig. New callers should prefer
// SecurityHeadersWith to make HSTS explicit per environment.
func SecurityHeaders(next http.Handler) http.Handler {
	return SecurityHeadersWith(DefaultSecurityHeadersConfig())(next)
}

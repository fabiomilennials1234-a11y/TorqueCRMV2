package middleware

import "net/http"

// SecurityHeaders sets a conservative baseline of defensive headers.
//
// CSP is intentionally NOT set here — it requires per-response nonces and is
// wired in Sprint S03 (SecurityHeaders + Observabilidade + Bootstrap). This
// middleware is the floor, not the ceiling.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy",
			"accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
		next.ServeHTTP(w, r)
	})
}

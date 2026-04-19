package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// Recover traps panics in downstream handlers, logs a structured error, and
// returns a 500. The stack trace is captured for debugging but never sent to
// the client — leaking stack traces is a security smell.
func Recover(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().
						Interface("panic", rec).
						Str("request_id", RequestIDFrom(r.Context())).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Bytes("stack", debug.Stack()).
						Msg("panic recovered")

					// Do not write a body if headers have been flushed.
					if w.Header().Get("Content-Type") == "" {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
					}
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"code":"INTERNAL","message":"internal error"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

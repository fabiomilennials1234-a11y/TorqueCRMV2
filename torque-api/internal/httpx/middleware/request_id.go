package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ctxKey is unexported to prevent collisions with other packages.
type ctxKey int

const requestIDKey ctxKey = iota

// RequestIDHeader is the canonical header for trace propagation.
const RequestIDHeader = "X-Request-ID"

// RequestID attaches a request id to the context and echoes it on the
// response. If the caller already sent one, we trust it — this makes
// client-side tracing work end-to-end.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom returns the request id attached by RequestID, or "" if the
// middleware did not run. Panics are not used on purpose — callers handle the
// empty case gracefully.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

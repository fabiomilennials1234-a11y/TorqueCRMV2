package middleware

import (
	"context"

	"github.com/milennials/torque-api/internal/domain"
)

// Context-key type is unexported so other packages cannot collide or forge
// values. They read via the With/From helpers below.
type sessionCtxKey struct{}

// WithSession returns a derived context carrying the authenticated session.
// Only the auth middleware should call this.
func WithSession(ctx context.Context, sess domain.Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, sess)
}

// SessionFrom returns the session attached by the auth middleware.
// The second return is false when the middleware did not run or produced no
// session (public route).
func SessionFrom(ctx context.Context) (domain.Session, bool) {
	s, ok := ctx.Value(sessionCtxKey{}).(domain.Session)
	return s, ok
}

// MustSession is the panic-on-miss variant for internal callers who know the
// middleware stack guarantees a session (behind RequireAuth). Use sparingly.
func MustSession(ctx context.Context) domain.Session {
	s, ok := SessionFrom(ctx)
	if !ok {
		panic("middleware.MustSession: no session in context — route not behind RequireAuth")
	}
	return s
}

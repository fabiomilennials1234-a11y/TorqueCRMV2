// Package domain — context helpers for propagating tenant scope across
// goroutine boundaries.
//
// The HTTP middleware (internal/httpx/middleware) owns the canonical
// `TenantScope` context key and exposes its own OrgIDFrom. That key is
// unexported, so background workers (e.g. the GCal goroutine spawned by
// the meetings handler) cannot reuse it when they build a fresh
// context.Background() to outlive the request.
//
// WithOrgID / OrgIDFrom in this package fill that gap with a second,
// independently-keyed carrier that sits at the domain layer — any
// package is free to import it, and it never leaks out of the process.
// Adapters that need the active tenant pull it via this helper.
package domain

import (
	"context"

	"github.com/google/uuid"
)

type orgIDCtxKey struct{}

// WithOrgID returns a derived context carrying the active organization id.
// Intended for background goroutines that outlive the originating request
// and cannot reuse the middleware-owned context key.
func WithOrgID(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, orgIDCtxKey{}, orgID)
}

// OrgIDFrom returns the active organization id attached via WithOrgID.
// Zero uuid + false when absent.
func OrgIDFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(orgIDCtxKey{}).(uuid.UUID)
	return v, ok
}

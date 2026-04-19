// Package ws owns the real-time surface of the API (ADR-002).
//
// The Hub holds one goroutine-safe registry of active connections keyed by
// tenant. Domain services publish Events into the event bus; the Hub drains
// the bus and broadcasts each Event to the matching tenant's connections.
//
// Nothing in this package imports handlers/services — the dependency flows
// inward.
package ws

import (
	"time"

	"github.com/google/uuid"
)

// Event is the wire shape of every message the API pushes to a client.
//
// Version is a monotonically increasing integer per EntityID, scoped by
// EntityType. Clients deduplicate by (EntityType, EntityID, Version) — a WS
// reconnect may deliver a patch the client already applied.
type Event struct {
	// Type namespaces the event, e.g. "lead.updated", "operation.succeeded".
	Type string `json:"type"`
	// TenantID is ALWAYS set by the publisher — the Hub uses it to route.
	TenantID uuid.UUID `json:"tenant_id"`
	// EntityType is the domain noun ("lead", "pipe_entry", "operation").
	EntityType string `json:"entity_type,omitempty"`
	// EntityID is the affected row when the event is row-scoped.
	EntityID *uuid.UUID `json:"entity_id,omitempty"`
	// Version supports client-side dedup across reconnects (monotonic per
	// entity). Zero means "not applicable".
	Version int64 `json:"version,omitempty"`
	// Patch is the minimum diff the client needs to update its view. The
	// schema is event-specific; the client MUST NOT treat it as a full row.
	Patch any `json:"patch,omitempty"`
	// OccurredAt is authoritative server time (RFC3339 / UTC Z).
	OccurredAt time.Time `json:"occurred_at"`
}

// OperationPatch is the Patch shape emitted on every operation transition.
// Reused for `operation.updated`, `.succeeded`, `.failed`, `.cancelled`.
type OperationPatch struct {
	ID       uuid.UUID `json:"id"`
	Status   string    `json:"status"`
	Progress *float64  `json:"progress,omitempty"`
	Result   any       `json:"result,omitempty"`
	Error    any       `json:"error,omitempty"`
}

// EventType constants collect the well-known names in one place. Handlers and
// publishers reference these rather than literals so a grep finds every call
// site.
const (
	TypeOperationUpdated   = "operation.updated"
	TypeOperationSucceeded = "operation.succeeded"
	TypeOperationFailed    = "operation.failed"
)

// Package audit persists entries in the audit_log table (migration 0002).
//
// Append-only by contract — no update, no delete. Master impersonation writes
// rows with actor_type='master' and target_org_id set to the impersonated
// organization; tenants never see those rows.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry is the write-side shape. Every field is optional except action.
type Entry struct {
	OrganizationID *uuid.UUID // caller's org (nil only for system/master ops)
	TargetOrgID    *uuid.UUID // impersonation target; nil for normal calls
	ActorType      string     // 'admin' | 'membro' | 'master' | 'system'
	ActorUserID    *uuid.UUID
	Action         string
	EntityType     *string
	EntityID       *uuid.UUID
	Payload        any // serialized as jsonb; nil → '{}'
	RequestID      string
}

// Repository is the pgx-backed store.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ActorTypes is the allowlist accepted by the schema CHECK constraint.
// Exported so callers can validate without duplicating the literal.
var ActorTypes = map[string]struct{}{
	"admin":  {},
	"membro": {},
	"master": {},
	"system": {},
}

// Append writes a single audit row. Never returns an opaque error on encoding
// issues — callers benefit from seeing which field was invalid.
func (r *Repository) Append(ctx context.Context, e Entry) error {
	if _, ok := ActorTypes[e.ActorType]; !ok {
		return fmt.Errorf("audit: invalid actor_type %q", e.ActorType)
	}
	if e.Action == "" {
		return errors.New("audit: action is required")
	}

	var payload []byte
	if e.Payload == nil {
		payload = []byte(`{}`)
	} else {
		// Scrub sensitive keys out of the payload BEFORE it hits the DB.
		// Callers that pass raw request bodies get password/token/email
		// redacted automatically instead of leaking into audit_log.
		scrubbed := scrubPayload(e.Payload)
		var err error
		payload, err = json.Marshal(scrubbed)
		if err != nil {
			return fmt.Errorf("audit: marshal payload: %w", err)
		}
	}

	const q = `
		INSERT INTO audit_log (
		  organization_id, target_org_id, actor_type, actor_user_id,
		  action, entity_type, entity_id, payload, request_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, q,
		e.OrganizationID, e.TargetOrgID, e.ActorType, e.ActorUserID,
		e.Action, e.EntityType, e.EntityID, payload, nullableString(e.RequestID),
	)
	if err != nil {
		return fmt.Errorf("audit: insert: %w", err)
	}
	return nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// sensitiveAuditKeys is the case-insensitive allowlist of payload fields
// to redact before the row hits audit_log. Mirrors the Sentry scrub list
// in `observability/sentry`, keeping a single mental model for
// "what never leaves the server".
var sensitiveAuditKeys = map[string]struct{}{
	"password":       {},
	"token":          {},
	"access_token":   {},
	"refresh_token":  {},
	"authorization":  {},
	"cookie":         {},
	"csrf":           {},
	"x-csrf-token":   {},
	"api_key":        {},
	"apikey":         {},
	"secret":         {},
	"email":          {},
	"password_hash":  {},
}

const redacted = "[Scrubbed]"

// scrubPayload walks the payload recursively and replaces the value of
// any key that matches sensitiveAuditKeys (case-insensitive). Non-map
// values pass through unchanged. Slices are walked element-by-element.
func scrubPayload(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			if _, hit := sensitiveAuditKeys[strings.ToLower(k)]; hit {
				out[k] = redacted
				continue
			}
			out[k] = scrubPayload(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = scrubPayload(e)
		}
		return out
	default:
		return v
	}
}

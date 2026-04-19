// Package audit is the write-path for audit_log.
//
// It sits above the repository so handlers don't deal with uuid parsing or
// actor_type inference — they pass a domain.Session and a business Action,
// and the service computes the full row.
package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	auditrepo "github.com/milennials/torque-api/internal/repository/audit"
)

// Action labels are free-form text in the DB (char_length <= 80). Canonical
// verbs live here so a reviewer can find every call site with one grep.
const (
	ActionLogin               = "auth.login"
	ActionLogout              = "auth.logout"
	ActionRefresh             = "auth.refresh"
	ActionRefreshReuse        = "auth.refresh_reuse_detected"
	ActionPermissionDenied    = "permission.denied"
	ActionImpersonationStart  = "master.impersonation_start"
	ActionImpersonationEnd    = "master.impersonation_end"
	ActionPasswordChanged     = "auth.password_changed"
	ActionPreferenceUpdated   = "user.preference_updated"
)

// Service hides repository shape from handlers.
type Service struct {
	repo *auditrepo.Repository
}

// New binds the service to a repository.
func New(repo *auditrepo.Repository) *Service { return &Service{repo: repo} }

// Record writes an entry inferring actor_type from the session.
// `entityType` and `entityID` are optional (empty string / nil uuid skip them).
func (s *Service) Record(
	ctx context.Context,
	sess domain.Session,
	action string,
	entityType string,
	entityID uuid.UUID,
	payload any,
	requestID string,
) error {
	actor := "system"
	switch {
	case sess.IsMaster:
		actor = "master"
	case sess.Role == domain.RoleAdmin:
		actor = "admin"
	case sess.Role == domain.RoleMember:
		actor = "membro"
	}

	entry := auditrepo.Entry{
		ActorType: actor,
		Action:    action,
		Payload:   payload,
		RequestID: requestID,
	}
	if sess.UserID != uuid.Nil {
		uid := sess.UserID
		entry.ActorUserID = &uid
	}
	if sess.OrganizationID != uuid.Nil {
		oid := sess.OrganizationID
		entry.OrganizationID = &oid
	}
	if entityType != "" {
		entry.EntityType = &entityType
	}
	if entityID != uuid.Nil {
		eid := entityID
		entry.EntityID = &eid
	}

	if err := s.repo.Append(ctx, entry); err != nil {
		return fmt.Errorf("audit.Record: %w", err)
	}
	return nil
}

// RecordSystem writes an entry with actor_type='system' (no session required).
// Use for background jobs, seed-time events, and boot diagnostics.
func (s *Service) RecordSystem(ctx context.Context, action string, payload any, requestID string) error {
	return s.repo.Append(ctx, auditrepo.Entry{
		ActorType: "system",
		Action:    action,
		Payload:   payload,
		RequestID: requestID,
	})
}

// RecordImpersonation writes a two-field entry used when a master acts on
// behalf of a tenant. Both the acting master's org (usually nil/global) AND
// the targetOrgID are persisted so audit queries from BOTH directions find it.
func (s *Service) RecordImpersonation(
	ctx context.Context,
	masterUserID uuid.UUID,
	targetOrgID uuid.UUID,
	action string,
	payload any,
	requestID string,
) error {
	muid := masterUserID
	toid := targetOrgID
	return s.repo.Append(ctx, auditrepo.Entry{
		TargetOrgID: &toid,
		ActorType:   "master",
		ActorUserID: &muid,
		Action:      action,
		Payload:     payload,
		RequestID:   requestID,
	})
}

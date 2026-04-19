// Package domain holds the tenant-agnostic types that cross package boundaries.
//
// Nothing in this package imports anything below internal/. It is the
// vocabulary of the system — handlers, services, and repositories all speak it.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role is the membership role inside a single organization.
//
// Master is intentionally absent: master privilege lives in users_master and is
// orthogonal to organization membership. The RBAC evaluator treats master as a
// bypass at the top of the cascade.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "membro"
)

// IsValid reports whether r is one of the persisted enum values.
func (r Role) IsValid() bool { return r == RoleAdmin || r == RoleMember }

// UIMode is the persisted preference for rendering the shell.
//
// Defined in ADR-007. Never a role; a user can freely flip between modes
// provided the corresponding feature permission is granted.
type UIMode string

const (
	UIModeManager     UIMode = "manager"
	UIModeSalesperson UIMode = "salesperson"
)

// IsValid reports whether m is one of the persisted enum values.
func (m UIMode) IsValid() bool { return m == UIModeManager || m == UIModeSalesperson }

// Session is the authenticated identity attached to a request by the auth
// middleware. Every handler that requires auth reads it out of context.
//
// Fields here mirror the JWT claims one-to-one so we can round-trip without
// a database hit on every request.
type Session struct {
	UserID         uuid.UUID // users.id
	OrganizationID uuid.UUID // team_members.organization_id (active tenant)
	TeamMemberID  uuid.UUID // team_members.id (active membership)
	Role           Role      // team_members.role in the active org
	IsMaster       bool      // true if user_id is present in users_master
	UIMode         UIMode    // user preference; safe default "manager"
	IssuedAt       time.Time
	ExpiresAt      time.Time
}

// User is the global identity. Not tenant-scoped.
type User struct {
	ID              uuid.UUID
	Email           string
	DisplayName     string
	PasswordHash    string
	IsActive        bool
	UIMode          UIMode
	EmailVerifiedAt *time.Time
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Membership is a User's seat inside a specific Organization.
type Membership struct {
	TeamMemberID   uuid.UUID
	OrganizationID uuid.UUID
	OrgSlug        string
	OrgName        string
	Role           Role
	IsActive       bool
	JoinedAt       time.Time
}

// Organization is the tenant record exposed to authenticated callers.
type Organization struct {
	ID             uuid.UUID
	Slug           string
	Name           string
	PlanID         *string
	PaymentStatus  string
	LogoURL        *string
}

// FeaturePermission is one entry in the effective permission set for a session.
//
// Resolution order (highest-priority wins):
//   1. Master bypass                     → allow
//   2. feature.master_only               → deny for non-master
//   3. feature.is_admin_only + non-admin → deny
//   4. member_feature_permissions row    → that boolean
//   5. feature.default_value             → fallback
type FeaturePermission struct {
	Key     string
	Allowed bool
	// Source describes which rule produced the decision, purely for audit.
	Source string
}

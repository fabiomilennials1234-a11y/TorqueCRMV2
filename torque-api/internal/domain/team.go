package domain

import (
	"time"

	"github.com/google/uuid"
)

// TeamMember is the hydrated membership record returned by the members
// handler. Unlike [Membership] (which is auth-facing and trimmed to what the
// session needs) this carries the full profile needed by the Settings UI.
type TeamMember struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           Role
	DisplayName    string
	Email          string
	AvatarURL      *string
	IsActive       bool
	InvitedBy      *uuid.UUID
	InvitedAt      *time.Time
	JoinedAt       time.Time
	DeactivatedAt  *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Product is a catalog entry for Propostas. Prices are integer cents; the
// repository refuses negative values.
type Product struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	SKU            *string
	PriceCents     int64
	Currency       string
	IsActive       bool
	Metadata       []byte
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

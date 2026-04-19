package domain

import (
	"time"

	"github.com/google/uuid"
)

// Lead is the central tenant-scoped entity.
type Lead struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	ExternalID         *string
	Name               string
	Company            *string
	Phone              *string // E.164
	Email              *string
	Position           *string
	CNPJCPF            *string
	ResponsibleID      *uuid.UUID
	SDRID              *uuid.UUID
	CloserID           *uuid.UUID
	Rating             *int16
	QualificationScore *int16
	Segment            *string
	Origin             *string
	UTMSource          *string
	UTMMedium          *string
	UTMCampaign        *string
	UTMTerm            *string
	UTMContent         *string
	CustomFields       []byte // jsonb
	FirstResponseAt    *time.Time
	LastInteractionAt  *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Pipe is a named funnel definition (whatsapp | confirmation | proposal | custom).
type Pipe struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Kind           string
	Name           string
	IsDefault      bool
	IsArchived     bool
	Position       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PipeStage is a column inside a Pipe.
type PipeStage struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	PipeID          uuid.UUID
	Name            string
	ColorToken      *string
	Position        int
	IsFinalPositive bool
	IsFinalNegative bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PipeEntry is a single lead's current placement inside a stage.
type PipeEntry struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	PipeID          uuid.UUID
	StageID         uuid.UUID
	LeadID          uuid.UUID
	EnteredStageAt  time.Time
	LeftAt          *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

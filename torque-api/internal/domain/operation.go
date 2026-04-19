package domain

import (
	"time"

	"github.com/google/uuid"
)

// OperationStatus mirrors the operations.status ENUM.
type OperationStatus string

const (
	OperationPending   OperationStatus = "pending"
	OperationRunning   OperationStatus = "running"
	OperationSucceeded OperationStatus = "succeeded"
	OperationFailed    OperationStatus = "failed"
	OperationCancelled OperationStatus = "cancelled"
)

// IsTerminal reports whether the status is a final resting state.
// Workers must not move out of a terminal state.
func (s OperationStatus) IsTerminal() bool {
	return s == OperationSucceeded || s == OperationFailed || s == OperationCancelled
}

// Operation is the domain model for an async job ledger entry.
type Operation struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ActorUserID    *uuid.UUID
	ActorType      string

	Kind     string
	Status   OperationStatus
	WorkerID *string

	Input        []byte // raw JSON; decoded by the worker
	Result       []byte
	ErrorPayload []byte

	Progress        *float64
	RetryRemaining  int

	ScheduledAt time.Time
	StartedAt   *time.Time
	EndedAt     *time.Time
	ExpiresAt   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

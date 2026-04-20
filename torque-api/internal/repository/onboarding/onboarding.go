// Package onboarding is the pgx-backed repository for F13 — the first-run
// wizard. State is per-team-member and the canonical step order is declared
// here so repository and handler share it.
package onboarding

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the onboarding row does not exist.
var ErrNotFound = errors.New("onboarding status not found")

// CanonicalSteps is the ordered list of steps. Adding a step here surfaces
// it as pending to every existing row (because steps_completed is explicit).
var CanonicalSteps = []string{
	"welcome",
	"organization_profile",
	"invite_team",
	"connect_whatsapp",
	"create_first_lead",
	"finish",
}

// Status is the wizard state for one member.
type Status struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	TeamMemberID    uuid.UUID
	CurrentStep     string
	StepsCompleted  []string
	Dismissed       bool
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// EnsureForMember creates the row on first access; subsequent calls read the
// existing row. Idempotent.
func (r *Repository) EnsureForMember(ctx context.Context, orgID, teamMemberID uuid.UUID) (Status, error) {
	const upsert = `
		INSERT INTO onboarding_status (organization_id, team_member_id, current_step)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_member_id) DO UPDATE SET updated_at = onboarding_status.updated_at
		RETURNING id, organization_id, team_member_id, current_step, steps_completed,
		          dismissed, completed_at, created_at, updated_at
	`
	var s Status
	err := r.pool.QueryRow(ctx, upsert, orgID, teamMemberID, CanonicalSteps[0]).Scan(
		&s.ID, &s.OrganizationID, &s.TeamMemberID, &s.CurrentStep, &s.StepsCompleted,
		&s.Dismissed, &s.CompletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return Status{}, fmt.Errorf("ensure onboarding: %w", err)
	}
	return s, nil
}

// Get returns the existing row or ErrNotFound.
func (r *Repository) Get(ctx context.Context, orgID, teamMemberID uuid.UUID) (Status, error) {
	const q = `
		SELECT id, organization_id, team_member_id, current_step, steps_completed,
		       dismissed, completed_at, created_at, updated_at
		  FROM onboarding_status
		 WHERE team_member_id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var s Status
	err := r.pool.QueryRow(ctx, q, teamMemberID, orgID).Scan(
		&s.ID, &s.OrganizationID, &s.TeamMemberID, &s.CurrentStep, &s.StepsCompleted,
		&s.Dismissed, &s.CompletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Status{}, ErrNotFound
	}
	if err != nil {
		return Status{}, fmt.Errorf("get onboarding: %w", err)
	}
	return s, nil
}

// CompleteStep marks the given step as done and advances current_step to the
// next canonical entry. If the step is already complete it is a no-op. If
// the step is 'finish' or all steps are done, completed_at is stamped.
func (r *Repository) CompleteStep(ctx context.Context, orgID, teamMemberID uuid.UUID, step string) (Status, error) {
	if !isCanonical(step) {
		return Status{}, fmt.Errorf("unknown step: %s", step)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Status{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	s, err := r.Get(ctx, orgID, teamMemberID)
	if err != nil {
		return Status{}, err
	}
	if contains(s.StepsCompleted, step) {
		return s, nil
	}
	s.StepsCompleted = append(s.StepsCompleted, step)
	next, done := advance(s.StepsCompleted)
	s.CurrentStep = next

	var completedAt any = nil
	if done {
		now := time.Now().UTC()
		s.CompletedAt = &now
		completedAt = now
	}

	if _, err := tx.Exec(ctx,
		`UPDATE onboarding_status
		    SET current_step = $3,
		        steps_completed = $4,
		        completed_at = COALESCE($5, completed_at)
		  WHERE team_member_id = $1 AND organization_id = $2`,
		teamMemberID, orgID, s.CurrentStep, s.StepsCompleted, completedAt,
	); err != nil {
		return Status{}, fmt.Errorf("persist step: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Status{}, fmt.Errorf("commit: %w", err)
	}
	return s, nil
}

// Dismiss marks the wizard as dismissed by the user. Completion state is
// preserved; dismissal is user intent to hide the prompt even if incomplete.
func (r *Repository) Dismiss(ctx context.Context, orgID, teamMemberID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE onboarding_status SET dismissed = true
		  WHERE team_member_id = $1 AND organization_id = $2 AND dismissed = false`,
		teamMemberID, orgID,
	)
	if err != nil {
		return fmt.Errorf("dismiss onboarding: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Reset wipes progress so the wizard is shown again. Admin-only affordance.
func (r *Repository) Reset(ctx context.Context, orgID, teamMemberID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE onboarding_status
		    SET current_step = $3, steps_completed = ARRAY[]::text[],
		        dismissed = false, completed_at = NULL
		  WHERE team_member_id = $1 AND organization_id = $2`,
		teamMemberID, orgID, CanonicalSteps[0],
	)
	if err != nil {
		return fmt.Errorf("reset onboarding: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- helpers ---------------------------------------------------

func isCanonical(step string) bool {
	for _, s := range CanonicalSteps {
		if s == step {
			return true
		}
	}
	return false
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// advance returns the next uncompleted step and a boolean indicating whether
// the whole wizard is now complete.
func advance(completed []string) (string, bool) {
	for _, step := range CanonicalSteps {
		if !contains(completed, step) {
			return step, false
		}
	}
	return "finish", true
}

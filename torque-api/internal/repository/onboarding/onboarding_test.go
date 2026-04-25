// Integration test for F13 onboarding_status (DATABASE_URL gated).
package onboarding_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	obrepo "github.com/milennials/torque-api/internal/repository/onboarding"
)

func TestOnboarding_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	runID := uuid.New().String()[:8]
	orgID := uuid.New()
	userID := uuid.New()
	memberID := uuid.New()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "ob-"+runID, "OB "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','O')`,
		userID, "ob-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','O')`, memberID, orgID, userID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM onboarding_status WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := obrepo.New(pool)

	// --- Ensure is idempotent + starts on first canonical step --------
	s, err := repo.EnsureForMember(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("EnsureForMember: %v", err)
	}
	if s.CurrentStep != obrepo.CanonicalSteps[0] {
		t.Fatalf("want first step %s, got %s", obrepo.CanonicalSteps[0], s.CurrentStep)
	}
	if len(s.StepsCompleted) != 0 {
		t.Fatalf("want empty steps, got %v", s.StepsCompleted)
	}
	// Second call returns same row.
	s2, _ := repo.EnsureForMember(ctx, orgID, memberID)
	if s2.ID != s.ID {
		t.Fatalf("Ensure must be idempotent; got different id")
	}

	// --- CompleteStep advances current_step ---------------------------
	s, err = repo.CompleteStep(ctx, orgID, memberID, "welcome")
	if err != nil {
		t.Fatalf("CompleteStep welcome: %v", err)
	}
	if s.CurrentStep != "organization_profile" {
		t.Fatalf("want organization_profile, got %s", s.CurrentStep)
	}
	// Second call for same step is idempotent.
	s, _ = repo.CompleteStep(ctx, orgID, memberID, "welcome")
	if len(s.StepsCompleted) != 1 {
		t.Fatalf("duplicate complete leaked: %v", s.StepsCompleted)
	}

	// --- Unknown step rejected ---------------------------------------
	_, err = repo.CompleteStep(ctx, orgID, memberID, "bogus")
	if err == nil {
		t.Fatalf("want error on unknown step")
	}

	// --- Complete all → completed_at stamped ------------------------
	for _, step := range obrepo.CanonicalSteps[1:] {
		s, err = repo.CompleteStep(ctx, orgID, memberID, step)
		if err != nil {
			t.Fatalf("CompleteStep %s: %v", step, err)
		}
	}
	if s.CompletedAt == nil {
		t.Fatalf("completed_at must be set after all steps")
	}

	// --- Reset wipes progress ---------------------------------------
	if err := repo.Reset(ctx, orgID, memberID); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	s, _ = repo.Get(ctx, orgID, memberID)
	if len(s.StepsCompleted) != 0 || s.CurrentStep != obrepo.CanonicalSteps[0] || s.CompletedAt != nil {
		t.Fatalf("Reset did not wipe: %+v", s)
	}

	// --- Dismiss idempotent -----------------------------------------
	if err := repo.Dismiss(ctx, orgID, memberID); err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	if err := repo.Dismiss(ctx, orgID, memberID); !errors.Is(err, obrepo.ErrNotFound) {
		t.Fatalf("second Dismiss must return ErrNotFound; got %v", err)
	}

	// --- Cross-tenant guard -----------------------------------------
	otherOrg := uuid.New()
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "other-"+runID, "Other")
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg) }()

	_, err = repo.Get(ctx, otherOrg, memberID)
	if !errors.Is(err, obrepo.ErrNotFound) {
		t.Fatalf("cross-tenant Get must refuse; got %v", err)
	}
}

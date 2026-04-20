// Integration test for F02 pipe_confirmations (DATABASE_URL gated).
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/repository/confirmation/...
package confirmation_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	confirmationrepo "github.com/milennials/torque-api/internal/repository/confirmation"
)

func TestConfirmation_UpsertConfirmNoShow(t *testing.T) {
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
	leadID := uuid.New()
	pipeID := uuid.New()
	stageID := uuid.New()
	entryID := uuid.New()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec %s: %v", sql, err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1, $2, $3, 'free')`,
		orgID, "conf-"+runID, "Conf "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','T')`,
		userID, "conf-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','T')`, memberID, orgID, userID)
	mustExec(`INSERT INTO leads (id, organization_id, name, phone) VALUES ($1,$2,'Lead','+5511999999999')`,
		leadID, orgID)
	mustExec(`INSERT INTO pipes (id, organization_id, kind, name, position)
	          VALUES ($1,$2,'confirmation','Conf',0)`, pipeID, orgID)
	mustExec(`INSERT INTO pipe_stages (id, organization_id, pipe_id, name, position)
	          VALUES ($1,$2,$3,'Agendado',0)`, stageID, orgID, pipeID)
	mustExec(`INSERT INTO pipe_entries (id, organization_id, pipe_id, stage_id, lead_id)
	          VALUES ($1,$2,$3,$4,$5)`, entryID, orgID, pipeID, stageID, leadID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM pipe_confirmations WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipe_entries WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipe_stages WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipes WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM leads WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := confirmationrepo.New(pool)
	meetingAt := time.Now().Add(2 * time.Hour).UTC()

	// Upsert → then Get → then MarkConfirmed → idempotent re-confirm → MarkNoShow.
	c1, err := repo.Upsert(ctx, confirmationrepo.UpsertInput{
		OrganizationID: orgID, PipeEntryID: entryID, LeadID: leadID,
		MeetingAt: meetingAt,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !c1.MeetingAt.Equal(meetingAt) {
		t.Errorf("meeting_at drift: %v vs %v", c1.MeetingAt, meetingAt)
	}

	// Update meeting time → same row (ON CONFLICT DO UPDATE).
	newMeeting := meetingAt.Add(30 * time.Minute)
	c2, err := repo.Upsert(ctx, confirmationrepo.UpsertInput{
		OrganizationID: orgID, PipeEntryID: entryID, LeadID: leadID,
		MeetingAt: newMeeting,
	})
	if err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	if !c2.MeetingAt.Equal(newMeeting) {
		t.Errorf("update did not apply: %v", c2.MeetingAt)
	}

	if err := repo.MarkConfirmed(ctx, orgID, entryID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// Idempotent — second call must not error.
	if err := repo.MarkConfirmed(ctx, orgID, entryID); err != nil {
		t.Fatalf("confirm 2: %v", err)
	}

	reason := "cliente nao atendeu"
	if err := repo.MarkNoShow(ctx, orgID, entryID, &reason); err != nil {
		t.Fatalf("no-show: %v", err)
	}

	got, err := repo.Get(ctx, orgID, entryID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ConfirmedAt == nil {
		t.Error("confirmed_at must be set")
	}
	if !got.NoShow {
		t.Error("no_show must be true")
	}
	if got.NoShowReason == nil || *got.NoShowReason != reason {
		t.Error("reason not persisted")
	}

	// Cross-tenant isolation.
	otherOrg := uuid.New()
	if _, err := repo.Get(ctx, otherOrg, entryID); !errors.Is(err, confirmationrepo.ErrNotFound) {
		t.Errorf("cross-tenant read must be NotFound, got %v", err)
	}
}

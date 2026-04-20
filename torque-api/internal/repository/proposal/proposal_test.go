// Integration test for F03 pipe_proposals (DATABASE_URL gated).
package proposal_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	proposalrepo "github.com/milennials/torque-api/internal/repository/proposal"
)

func TestProposal_Lifecycle(t *testing.T) {
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

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec %s: %v", sql, err)
		}
	}
	exec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "prop-"+runID, "Prop "+runID)
	exec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','T')`,
		userID, "prop-"+runID+"@example.test")
	exec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	      VALUES ($1,$2,$3,'admin','T')`, memberID, orgID, userID)
	exec(`INSERT INTO leads (id, organization_id, name, phone) VALUES ($1,$2,'Lead','+5511999999999')`,
		leadID, orgID)
	exec(`INSERT INTO pipes (id, organization_id, kind, name, position)
	      VALUES ($1,$2,'proposal','Prop',0)`, pipeID, orgID)
	exec(`INSERT INTO pipe_stages (id, organization_id, pipe_id, name, position)
	      VALUES ($1,$2,$3,'Draft',0)`, stageID, orgID, pipeID)
	exec(`INSERT INTO pipe_entries (id, organization_id, pipe_id, stage_id, lead_id)
	      VALUES ($1,$2,$3,$4,$5)`, entryID, orgID, pipeID, stageID, leadID)
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM pipe_proposals WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipe_entries WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipe_stages WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipes WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM leads WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := proposalrepo.New(pool)

	// Upsert draft.
	p, err := repo.Upsert(ctx, proposalrepo.UpsertInput{
		OrganizationID: orgID, PipeEntryID: entryID, LeadID: leadID,
		Title: "Proposta 2026-Q2", AmountCents: 2500000, Currency: "BRL",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if p.Status != proposalrepo.StatusDraft {
		t.Fatalf("status=%s, want draft", p.Status)
	}

	// draft → sent.
	if err := repo.MarkSent(ctx, orgID, entryID, memberID); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Upsert now must refuse (proposal is locked past draft).
	if _, err := repo.Upsert(ctx, proposalrepo.UpsertInput{
		OrganizationID: orgID, PipeEntryID: entryID, LeadID: leadID,
		Title: "Tentativa de edicao", AmountCents: 999,
	}); !errors.Is(err, proposalrepo.ErrInvalidState) {
		t.Fatalf("edit past draft must return ErrInvalidState, got %v", err)
	}

	// sent → viewed is idempotent (second call is no-op).
	if err := repo.MarkViewed(ctx, orgID, entryID); err != nil {
		t.Fatalf("viewed: %v", err)
	}
	if err := repo.MarkViewed(ctx, orgID, entryID); err != nil {
		t.Fatalf("viewed 2: %v", err)
	}

	// viewed → accepted.
	if err := repo.MarkAccepted(ctx, orgID, entryID, memberID); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// Accepted is terminal — cannot reject.
	if err := repo.MarkRejected(ctx, orgID, entryID, memberID, nil); !errors.Is(err, proposalrepo.ErrInvalidState) {
		t.Fatalf("reject after accept must be ErrInvalidState, got %v", err)
	}

	got, _ := repo.Get(ctx, orgID, entryID)
	if got.Status != proposalrepo.StatusAccepted {
		t.Fatalf("final status = %s, want accepted", got.Status)
	}
	if got.FirstViewedAt == nil || got.AcceptedAt == nil {
		t.Error("first_viewed_at and accepted_at must be set")
	}
}

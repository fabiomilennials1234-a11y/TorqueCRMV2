// Integration test for the F06 agent repo, gated by DATABASE_URL.
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/repository/agent/...
package agent_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
)

func TestAgent_CreateUpdateKillSwitchIsolation(t *testing.T) {
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
	orgA := uuid.New()
	orgB := uuid.New()
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgA, "ag-a-"+runID, "AgA")
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "ag-b-"+runID, "AgB")

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM agents WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	repo := agentrepo.New(pool)

	// Create (draft)
	a, err := repo.CreateAgent(ctx, agentrepo.CreateAgentInput{
		OrganizationID: orgA,
		Name:           "SDR primário",
		SystemPrompt:   "Você é um SDR útil e direto.",
		Model:          "anthropic/claude-sonnet-4",
		Temperature:    0.3,
		MaxOutputTokens: 1024,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Status != "draft" {
		t.Fatalf("want draft, got %s", a.Status)
	}

	// Update: muda system_prompt + max_output_tokens
	newPrompt := "Você é um SDR consultivo, tom calmo, foco em perguntas."
	newTokens := 2048
	updated, err := repo.UpdateAgent(ctx, orgA, a.ID, agentrepo.UpdateAgentInput{
		SystemPrompt:    &newPrompt,
		MaxOutputTokens: &newTokens,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.SystemPrompt != newPrompt || updated.MaxOutputTokens != 2048 {
		t.Fatalf("patch not applied: %+v", updated)
	}

	// Invalid patch (temperature fora do intervalo)
	bogus := 10.0
	_, err = repo.UpdateAgent(ctx, orgA, a.ID, agentrepo.UpdateAgentInput{Temperature: &bogus})
	if err == nil {
		t.Errorf("want validation error for temperature=10")
	}

	// Activate + kill-switch
	if err := repo.SetAgentStatus(ctx, orgA, a.ID, "active"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	if err := repo.SetKillSwitch(ctx, orgA, a.ID, true); err != nil {
		t.Fatalf("kill switch: %v", err)
	}
	post, _ := repo.GetAgent(ctx, orgA, a.ID)
	if !post.KillSwitch || post.Status != "active" {
		t.Fatalf("state not persisted: %+v", post)
	}

	// Cross-tenant Get must refuse
	if _, err := repo.GetAgent(ctx, orgB, a.ID); !errors.Is(err, agentrepo.ErrNotFound) {
		t.Fatalf("cross-tenant Get must refuse; got %v", err)
	}
	// Cross-tenant Update must refuse
	name := "stolen"
	if _, err := repo.UpdateAgent(ctx, orgB, a.ID, agentrepo.UpdateAgentInput{Name: &name}); !errors.Is(err, agentrepo.ErrNotFound) {
		t.Fatalf("cross-tenant Update must refuse; got %v", err)
	}
	// Confirma que a não foi renomeado
	after, _ := repo.GetAgent(ctx, orgA, a.ID)
	if after.Name != "SDR primário" {
		t.Fatalf("cross-tenant update leaked: name=%q", after.Name)
	}
}

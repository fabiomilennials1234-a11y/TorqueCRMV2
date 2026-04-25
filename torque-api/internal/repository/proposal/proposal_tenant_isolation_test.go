// Cross-tenant isolation test for proposals, gated by DATABASE_URL.
// Money flow — an org_id leak here would mis-attribute deals.
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

func TestProposal_CrossTenantRefuses(t *testing.T) {
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
	leadA := uuid.New()
	pipeA := uuid.New()
	stageA := uuid.New()
	entryA := uuid.New()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgA, "prop-iso-a-"+runID, "PropIsoA")
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "prop-iso-b-"+runID, "PropIsoB")
	mustExec(`INSERT INTO leads (id, organization_id, name, phone) VALUES ($1,$2,'Lead Teste','+5511999999999')`,
		leadA, orgA)
	mustExec(`INSERT INTO pipes (id, organization_id, kind, name, position)
	          VALUES ($1,$2,'proposal','Prop',0)`, pipeA, orgA)
	mustExec(`INSERT INTO pipe_stages (id, organization_id, pipe_id, name, position)
	          VALUES ($1,$2,$3,'Draft',0)`, stageA, orgA, pipeA)
	mustExec(`INSERT INTO pipe_entries (id, organization_id, pipe_id, stage_id, lead_id)
	          VALUES ($1,$2,$3,$4,$5)`, entryA, orgA, pipeA, stageA, leadA)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM pipe_proposals WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM pipe_entries WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM pipe_stages WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM pipes WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM leads WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	repo := proposalrepo.New(pool)

	_, err = repo.Upsert(ctx, proposalrepo.UpsertInput{
		OrganizationID: orgA,
		PipeEntryID:    entryA,
		LeadID:         leadA,
		Title:          "Proposta R$ 10k",
		AmountCents:    1_000_000,
		Currency:       "BRL",
	})
	if err != nil {
		t.Fatalf("Upsert orgA: %v", err)
	}

	_, err = repo.Get(ctx, orgB, entryA)
	if !errors.Is(err, proposalrepo.ErrNotFound) {
		t.Fatalf("orgB.Get(orgA.proposal) must refuse with ErrNotFound; got %v", err)
	}
}

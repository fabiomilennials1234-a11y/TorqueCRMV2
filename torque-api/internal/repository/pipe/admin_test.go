// Integration test for F12 pipe admin CRUD (DATABASE_URL gated).
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/repository/pipe/...
package pipe_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
)

func TestPipeAdmin_CRUD(t *testing.T) {
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
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "pipe-"+runID, "Pipe "+runID)
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM pipe_entries WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipe_stages WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM pipes WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := piperepo.New(pool)

	// --- CreatePipe happy path ----------------------------------------
	p, err := repo.CreatePipe(ctx, orgID, piperepo.CreatePipeInput{
		Kind: "custom", Name: "Upsell", Position: 10,
	})
	if err != nil {
		t.Fatalf("CreatePipe: %v", err)
	}
	if p.Kind != "custom" || p.Name != "Upsell" {
		t.Fatalf("unexpected pipe: %+v", p)
	}

	// --- CreatePipe invalid kind --------------------------------------
	_, err = repo.CreatePipe(ctx, orgID, piperepo.CreatePipeInput{Kind: "bogus", Name: "X"})
	if !errors.Is(err, piperepo.ErrInvalidKind) {
		t.Fatalf("want ErrInvalidKind, got %v", err)
	}

	// --- CreatePipe duplicate name ------------------------------------
	_, err = repo.CreatePipe(ctx, orgID, piperepo.CreatePipeInput{Kind: "custom", Name: "Upsell"})
	if !errors.Is(err, piperepo.ErrNameTaken) {
		t.Fatalf("want ErrNameTaken, got %v", err)
	}

	// --- CreateStage + XOR guard --------------------------------------
	s1, err := repo.CreateStage(ctx, orgID, p.ID, piperepo.CreateStageInput{
		Name: "Open", Position: 0,
	})
	if err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	_, err = repo.CreateStage(ctx, orgID, p.ID, piperepo.CreateStageInput{
		Name: "Both", Position: 99, IsFinalPositive: true, IsFinalNegative: true,
	})
	if err == nil {
		t.Fatalf("want XOR rejection")
	}

	// --- UpdateStage patch --------------------------------------------
	newName := "Aberto"
	s1u, err := repo.UpdateStage(ctx, orgID, s1.ID, piperepo.UpdateStageInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateStage: %v", err)
	}
	if s1u.Name != "Aberto" {
		t.Fatalf("patch not applied: %+v", s1u)
	}

	// --- DeleteStage with active entry blocks -------------------------
	leadID := uuid.New()
	entryID := uuid.New()
	mustExec(`INSERT INTO leads (id, organization_id, name, phone) VALUES ($1,$2,'Lead Teste','+5511999999999')`,
		leadID, orgID)
	mustExec(`INSERT INTO pipe_entries (id, organization_id, pipe_id, stage_id, lead_id)
	          VALUES ($1,$2,$3,$4,$5)`, entryID, orgID, p.ID, s1.ID, leadID)

	if err := repo.DeleteStage(ctx, orgID, s1.ID); !errors.Is(err, piperepo.ErrStageHasEntries) {
		t.Fatalf("want ErrStageHasEntries with live entry; got %v", err)
	}
	// Clear the entry. left_at marca historico mas o FK pipe_entries.stage_id
	// nao tem ON DELETE SET NULL — deletar a stage requer remover entries
	// fisicamente. Audit trail fica em pipe_entry_history (separado).
	mustExec(`DELETE FROM pipe_entries WHERE id = $1`, entryID)
	if err := repo.DeleteStage(ctx, orgID, s1.ID); err != nil {
		t.Fatalf("DeleteStage after clearing: %v", err)
	}

	// --- ArchivePipe + idempotent -------------------------------------
	if err := repo.ArchivePipe(ctx, orgID, p.ID); err != nil {
		t.Fatalf("ArchivePipe: %v", err)
	}
	if err := repo.ArchivePipe(ctx, orgID, p.ID); !errors.Is(err, piperepo.ErrNotFound) {
		t.Fatalf("second Archive must return ErrNotFound; got %v", err)
	}

	// --- Cross-tenant guard -------------------------------------------
	otherOrg := uuid.New()
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "other-"+runID, "Other")
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg) }()

	_, err = repo.GetPipe(ctx, otherOrg, p.ID)
	if !errors.Is(err, piperepo.ErrNotFound) {
		t.Fatalf("cross-tenant GetPipe must refuse; got %v", err)
	}
}

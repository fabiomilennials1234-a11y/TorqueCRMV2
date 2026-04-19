// Integration tests for the operations repository (DATABASE_URL gated).
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/repository/operation/...
package operation_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
)

func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

func seedOrg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	runID := uuid.New().String()[:8]
	_, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1, $2, $3, 'free')`,
		orgID, "op-"+runID, "Op IT "+runID)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}
	t.Cleanup(func() {
		cctx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_, _ = pool.Exec(cctx, `DELETE FROM operations WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(cctx, `DELETE FROM organizations WHERE id = $1`, orgID)
	})
	return orgID
}

func TestOperationRepo_SubmitLookupSucceed(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	ctx := context.Background()
	orgID := seedOrg(t, ctx, pool)

	repo := operationrepo.New(pool)
	op, err := repo.Submit(ctx, domain.Operation{
		OrganizationID: orgID,
		Kind:           "leads.import",
		Input:          []byte(`{"rows":42}`),
		RetryRemaining: 1,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if op.ID == uuid.Nil {
		t.Fatal("id not assigned")
	}
	if op.Status != domain.OperationPending {
		t.Fatalf("status=%s, want pending", op.Status)
	}

	got, err := repo.Lookup(ctx, orgID, op.ID)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got.Kind != "leads.import" {
		t.Errorf("kind = %q", got.Kind)
	}

	// Claim → succeed.
	claimed, err := repo.ClaimOne(ctx, []string{"leads.import"}, "worker-test")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed.ID != op.ID {
		t.Fatalf("claim picked a different op: %s != %s", claimed.ID, op.ID)
	}
	if err := repo.UpdateProgress(ctx, op.ID, 0.42); err != nil {
		t.Fatalf("progress: %v", err)
	}
	if err := repo.Succeed(ctx, op.ID, map[string]int{"imported": 42}); err != nil {
		t.Fatalf("succeed: %v", err)
	}

	final, err := repo.Lookup(ctx, orgID, op.ID)
	if err != nil {
		t.Fatalf("lookup 2: %v", err)
	}
	if final.Status != domain.OperationSucceeded {
		t.Fatalf("status = %s, want succeeded", final.Status)
	}
	if final.Progress == nil || *final.Progress != 1 {
		t.Fatalf("progress must be 1 on success, got %v", final.Progress)
	}
	if len(final.Result) == 0 {
		t.Fatal("result not persisted")
	}
}

func TestOperationRepo_ClaimSkipLockedIsFair(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	ctx := context.Background()
	orgID := seedOrg(t, ctx, pool)
	repo := operationrepo.New(pool)

	for i := 0; i < 3; i++ {
		_, err := repo.Submit(ctx, domain.Operation{
			OrganizationID: orgID,
			Kind:           "test.kind",
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	claimed := map[uuid.UUID]bool{}
	for i := 0; i < 3; i++ {
		op, err := repo.ClaimOne(ctx, []string{"test.kind"}, "w")
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		if claimed[op.ID] {
			t.Fatalf("claimed duplicate id %s", op.ID)
		}
		claimed[op.ID] = true
	}

	// Next claim is empty.
	if _, err := repo.ClaimOne(ctx, []string{"test.kind"}, "w"); !errors.Is(err, operationrepo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after exhaustion, got %v", err)
	}
}

func TestOperationRepo_FailRetryable(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	ctx := context.Background()
	orgID := seedOrg(t, ctx, pool)
	repo := operationrepo.New(pool)

	op, err := repo.Submit(ctx, domain.Operation{
		OrganizationID: orgID,
		Kind:           "flaky",
		RetryRemaining: 2,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	// claim → fail retryable → back to pending with decremented budget
	if _, err := repo.ClaimOne(ctx, []string{"flaky"}, "w"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := repo.Fail(ctx, op.ID, map[string]string{"msg": "timeout"}, true); err != nil {
		t.Fatalf("fail: %v", err)
	}
	after, _ := repo.Lookup(ctx, orgID, op.ID)
	if after.Status != domain.OperationPending {
		t.Fatalf("status = %s, want pending (retry)", after.Status)
	}
	if after.RetryRemaining != 1 {
		t.Fatalf("retry_remaining = %d, want 1", after.RetryRemaining)
	}

	// Exhaust → failed
	if _, err := repo.ClaimOne(ctx, []string{"flaky"}, "w"); err != nil {
		t.Fatalf("claim 2: %v", err)
	}
	if err := repo.Fail(ctx, op.ID, map[string]string{"msg": "timeout"}, true); err != nil {
		t.Fatalf("fail 2: %v", err)
	}
	if _, err := repo.ClaimOne(ctx, []string{"flaky"}, "w"); err != nil {
		t.Fatalf("claim 3: %v", err)
	}
	if err := repo.Fail(ctx, op.ID, map[string]string{"msg": "timeout"}, true); err != nil {
		t.Fatalf("fail 3: %v", err)
	}
	after, _ = repo.Lookup(ctx, orgID, op.ID)
	if after.Status != domain.OperationFailed {
		t.Fatalf("after exhaustion status = %s, want failed", after.Status)
	}
}

func TestOperationRepo_Cancel(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	ctx := context.Background()
	orgID := seedOrg(t, ctx, pool)
	repo := operationrepo.New(pool)

	op, err := repo.Submit(ctx, domain.Operation{OrganizationID: orgID, Kind: "slow"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := repo.Cancel(ctx, orgID, op.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := repo.Cancel(ctx, orgID, op.ID); !errors.Is(err, operationrepo.ErrNotFound) {
		t.Fatalf("second cancel expected NotFound, got %v", err)
	}
}

func TestOperationRepo_LookupTenantIsolation(t *testing.T) {
	pool := requirePool(t)
	defer pool.Close()
	ctx := context.Background()
	orgA := seedOrg(t, ctx, pool)
	orgB := seedOrg(t, ctx, pool)
	repo := operationrepo.New(pool)

	op, err := repo.Submit(ctx, domain.Operation{OrganizationID: orgA, Kind: "x"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	// Org B must not see it.
	if _, err := repo.Lookup(ctx, orgB, op.ID); !errors.Is(err, operationrepo.ErrNotFound) {
		t.Fatalf("cross-tenant read must return NotFound, got %v", err)
	}
}

// Integration test for F16 master queries (DATABASE_URL gated).
package master_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	masterrepo "github.com/milennials/torque-api/internal/repository/master"
)

func TestMaster_CrossOrgViews(t *testing.T) {
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
	userA := uuid.New()
	memberA := uuid.New()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgA, "ma-"+runID+"-a", "MA "+runID+" A")
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "ma-"+runID+"-b", "MA "+runID+" B")
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','U')`,
		userA, "ma-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','A')`, memberA, orgA, userA)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userA)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	repo := masterrepo.New(pool)

	// --- Health snapshot contains both orgs ---------------------------
	h, err := repo.SystemHealthSnapshot(ctx)
	if err != nil {
		t.Fatalf("SystemHealthSnapshot: %v", err)
	}
	if h.OrgCount < 2 {
		t.Fatalf("expected at least 2 orgs in health snapshot; got %d", h.OrgCount)
	}

	// --- ListOrganizations sees both orgs -----------------------------
	orgs, err := repo.ListOrganizations(ctx, 500)
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	var sawA, sawB bool
	for _, o := range orgs {
		if o.ID == orgA {
			sawA = true
			if o.MemberCount < 1 {
				t.Fatalf("org A should have at least 1 member; got %d", o.MemberCount)
			}
		}
		if o.ID == orgB {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Fatalf("master list must include both orgs; sawA=%v sawB=%v", sawA, sawB)
	}

	// --- ImpersonationTarget resolves orgA's admin --------------------
	memberID, userID, err := repo.ImpersonationTarget(ctx, orgA)
	if err != nil {
		t.Fatalf("ImpersonationTarget: %v", err)
	}
	if memberID != memberA || userID != userA {
		t.Fatalf("unexpected target: member=%v user=%v", memberID, userID)
	}

	// --- Empty org (no admin) returns ErrNotFound ---------------------
	_, _, err = repo.ImpersonationTarget(ctx, orgB)
	if !errors.Is(err, masterrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound for admin-less org; got %v", err)
	}
}

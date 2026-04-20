// Cross-tenant isolation integration test for leads, gated by DATABASE_URL.
//
// Separate file from lead_test.go so pure-unit tests (cursor round-trip)
// stay runnable with -short. This file exercises the SQL-level filter
// (`WHERE organization_id = $1`) that every repository method must apply.
//
// The pattern here is the canonical one for every tenant-scoped repo —
// new domains added post-S32 should follow this shape.
package lead_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
)

// TestLead_CrossTenantRefuses asserts that every read+write method on the
// lead repo honors the org filter. Violations here would leak data across
// tenants — a catastrophic security regression.
func TestLead_CrossTenantRefuses(t *testing.T) {
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
		orgA, "lead-iso-a-"+runID, "LeadIsoA")
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "lead-iso-b-"+runID, "LeadIsoB")

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM leads WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	repo := leadrepo.New(pool)

	// orgA creates a lead. orgB must not see it.
	phone := "+5511999999999"
	created, err := repo.Create(ctx, orgA, leadrepo.CreateInput{
		Name:  "Alice from A",
		Phone: &phone,
	})
	if err != nil {
		t.Fatalf("Create orgA: %v", err)
	}

	// --- Get cross-tenant ---------------------------------------------
	_, err = repo.Get(ctx, orgB, created.ID)
	if !errors.Is(err, leadrepo.ErrNotFound) {
		t.Fatalf("orgB.Get(orgA.lead) must refuse with ErrNotFound; got %v", err)
	}

	// --- Update cross-tenant ------------------------------------------
	newName := "Malicious rename from orgB"
	_, err = repo.Update(ctx, orgB, created.ID, leadrepo.UpdateInput{Name: &newName})
	if !errors.Is(err, leadrepo.ErrNotFound) {
		t.Fatalf("orgB.Update must refuse with ErrNotFound; got %v", err)
	}
	// Verify the name did NOT change after the attempted attack.
	after, err := repo.Get(ctx, orgA, created.ID)
	if err != nil {
		t.Fatalf("Get after failed cross-tenant Update: %v", err)
	}
	if after.Name != "Alice from A" {
		t.Fatalf("cross-tenant Update leaked: name=%q", after.Name)
	}

	// --- SoftDelete cross-tenant --------------------------------------
	err = repo.SoftDelete(ctx, orgB, created.ID)
	if !errors.Is(err, leadrepo.ErrNotFound) {
		t.Fatalf("orgB.SoftDelete must refuse with ErrNotFound; got %v", err)
	}
	// orgA still sees the lead.
	if _, err := repo.Get(ctx, orgA, created.ID); err != nil {
		t.Fatalf("orgA.Get after failed cross-tenant SoftDelete: %v", err)
	}

	// --- List scopes to the caller's org only -------------------------
	listA, err := repo.List(ctx, orgA, leadrepo.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List orgA: %v", err)
	}
	seen := false
	for _, l := range listA.Items {
		if l.ID == created.ID {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("orgA.List must surface its own lead")
	}
	listB, err := repo.List(ctx, orgB, leadrepo.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List orgB: %v", err)
	}
	for _, l := range listB.Items {
		if l.ID == created.ID {
			t.Fatalf("orgB.List leaked orgA lead: %v", l.ID)
		}
	}
}

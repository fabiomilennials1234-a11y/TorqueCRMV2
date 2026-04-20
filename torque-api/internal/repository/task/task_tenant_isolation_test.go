// Cross-tenant isolation test for tasks, gated by DATABASE_URL.
// Companion to lead_tenant_isolation_test.go — the canonical pattern.
package task_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	taskrepo "github.com/milennials/torque-api/internal/repository/task"
)

func TestTask_CrossTenantRefuses(t *testing.T) {
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
		orgA, "task-iso-a-"+runID, "TaskIsoA")
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "task-iso-b-"+runID, "TaskIsoB")
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','U')`,
		userA, "task-iso-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','Admin A')`, memberA, orgA, userA)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM tasks WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userA)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	repo := taskrepo.New(pool)

	created, err := repo.Create(ctx, taskrepo.CreateInput{
		OrganizationID: orgA,
		AssignedTo:     memberA,
		CreatedBy:      memberA,
		Title:          "Ligar para Alice",
	})
	if err != nil {
		t.Fatalf("Create orgA task: %v", err)
	}

	// Cross-tenant Get refuses.
	_, err = repo.Get(ctx, orgB, created.ID)
	if !errors.Is(err, taskrepo.ErrNotFound) {
		t.Fatalf("orgB.Get(orgA.task) must refuse with ErrNotFound; got %v", err)
	}

	// Cross-tenant List does not surface orgA's task.
	listB, err := repo.List(ctx, orgB, taskrepo.ListOptions{})
	if err != nil {
		t.Fatalf("List orgB: %v", err)
	}
	for _, tk := range listB {
		if tk.ID == created.ID {
			t.Fatalf("orgB.List leaked orgA task: %v", tk.ID)
		}
	}
}

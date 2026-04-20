// Integration test for S35 message_templates (DATABASE_URL gated).
package template_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	templaterepo "github.com/milennials/torque-api/internal/repository/template"
)

func TestTemplate_CRUD(t *testing.T) {
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
		orgID, "tpl-"+runID, "TplOrg "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','T')`,
		userID, "tpl-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','T')`, memberID, orgID, userID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM message_templates WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := templaterepo.New(pool)

	// Create happy path
	t1, err := repo.Create(ctx, orgID, templaterepo.CreateInput{
		Name:      "Saudação",
		Body:      "Olá {{nome}}! Bem-vindo à {{empresa}}.",
		Variables: []string{"nome", "empresa"},
		CreatedBy: &memberID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !t1.IsActive || len(t1.Variables) != 2 {
		t.Fatalf("unexpected template: %+v", t1)
	}

	// Duplicate name → ErrNameTaken
	_, err = repo.Create(ctx, orgID, templaterepo.CreateInput{
		Name: "Saudação", Body: "outro body",
	})
	if !errors.Is(err, templaterepo.ErrNameTaken) {
		t.Fatalf("want ErrNameTaken, got %v", err)
	}

	// Invalid name (too short)
	_, err = repo.Create(ctx, orgID, templaterepo.CreateInput{Name: "x", Body: "ok"})
	if !errors.Is(err, templaterepo.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}

	// Update body + flip active
	newBody := "Olá {{nome}}!"
	active := false
	t2, err := repo.Update(ctx, orgID, t1.ID, templaterepo.UpdateInput{
		Body: &newBody, IsActive: &active,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if t2.Body != newBody || t2.IsActive {
		t.Fatalf("patch not applied: %+v", t2)
	}

	// active_only filter excludes
	list, err := repo.List(ctx, orgID, true)
	if err != nil {
		t.Fatalf("List active_only: %v", err)
	}
	for _, x := range list {
		if x.ID == t1.ID {
			t.Fatalf("deactivated template leaked into active_only list")
		}
	}

	// include inactive
	list, _ = repo.List(ctx, orgID, false)
	seen := false
	for _, x := range list {
		if x.ID == t1.ID {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("full List should include the inactive template")
	}

	// Delete is idempotent via ErrNotFound on second call.
	if err := repo.Delete(ctx, orgID, t1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Delete(ctx, orgID, t1.ID); !errors.Is(err, templaterepo.ErrNotFound) {
		t.Fatalf("second Delete must return ErrNotFound; got %v", err)
	}

	// Cross-tenant guard
	otherOrg := uuid.New()
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "tpl-iso-"+runID, "OtherOrg")
	defer pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg)

	t3, err := repo.Create(ctx, orgID, templaterepo.CreateInput{Name: "Ouro A", Body: "hi"})
	if err != nil {
		t.Fatalf("setup cross-tenant: %v", err)
	}
	if _, err := repo.Get(ctx, otherOrg, t3.ID); !errors.Is(err, templaterepo.ErrNotFound) {
		t.Fatalf("cross-tenant Get must refuse; got %v", err)
	}
}

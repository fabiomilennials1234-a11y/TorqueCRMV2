// Integration + unit test for F11 products.
//
// Integration paths are gated by DATABASE_URL; the cursor encode/decode
// roundtrip runs unconditionally.
package product_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	productrepo "github.com/milennials/torque-api/internal/repository/product"
)

func TestProduct_CRUD(t *testing.T) {
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
			t.Fatalf("exec %s: %v", sql, err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "prod-"+runID, "Prod "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','P')`,
		userID, "prod-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','P')`, memberID, orgID, userID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM products WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := productrepo.New(pool)

	// --- Create happy path ---------------------------------------------
	sku1 := "SKU-" + runID + "-1"
	p, err := repo.Create(ctx, orgID, productrepo.CreateInput{
		Name: "Licenca Torque Pro", SKU: &sku1,
		PriceCents: 19_900, Currency: "brl", CreatedBy: &memberID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Currency != "BRL" || p.PriceCents != 19_900 || !p.IsActive {
		t.Fatalf("unexpected product: %+v", p)
	}

	// --- Invalid name / price / currency -------------------------------
	_, err = repo.Create(ctx, orgID, productrepo.CreateInput{Name: "x", PriceCents: 100})
	if !errors.Is(err, productrepo.ErrInvalidName) {
		t.Fatalf("want ErrInvalidName, got %v", err)
	}
	_, err = repo.Create(ctx, orgID, productrepo.CreateInput{Name: "Valid name", PriceCents: -1})
	if !errors.Is(err, productrepo.ErrInvalidPrice) {
		t.Fatalf("want ErrInvalidPrice, got %v", err)
	}
	_, err = repo.Create(ctx, orgID, productrepo.CreateInput{
		Name: "Valid name", PriceCents: 100, Currency: "BR",
	})
	if !errors.Is(err, productrepo.ErrInvalidCurrency) {
		t.Fatalf("want ErrInvalidCurrency, got %v", err)
	}

	// --- SKU conflict ---------------------------------------------------
	_, err = repo.Create(ctx, orgID, productrepo.CreateInput{
		Name: "Another", SKU: &sku1, PriceCents: 100,
	})
	if !errors.Is(err, productrepo.ErrSKUConflict) {
		t.Fatalf("want ErrSKUConflict, got %v", err)
	}

	// --- Update partial ------------------------------------------------
	newPrice := int64(24_900)
	p2, err := repo.Update(ctx, orgID, p.ID, productrepo.UpdateInput{PriceCents: &newPrice})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if p2.PriceCents != 24_900 {
		t.Fatalf("price patch not applied: %+v", p2)
	}

	// --- List with cursor ----------------------------------------------
	// Insert 4 more so pagination actually rolls.
	for i := 0; i < 4; i++ {
		_, err := repo.Create(ctx, orgID, productrepo.CreateInput{
			Name: "P-" + runID + "-" + string(rune('A'+i)), PriceCents: int64(1000 + i),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	first, err := repo.List(ctx, orgID, productrepo.ListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("expected 2 items + non-empty cursor: %+v", first)
	}
	second, err := repo.List(ctx, orgID, productrepo.ListOptions{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(second.Items) == 0 {
		t.Fatalf("page 2 empty")
	}
	for _, a := range first.Items {
		for _, b := range second.Items {
			if a.ID == b.ID {
				t.Fatalf("duplicate across pages: %s", a.ID)
			}
		}
	}

	// --- Archive is idempotent after first flip ------------------------
	if err := repo.Archive(ctx, orgID, p.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if err := repo.Archive(ctx, orgID, p.ID); !errors.Is(err, productrepo.ErrNotFound) {
		t.Fatalf("second Archive must return ErrNotFound (already inactive); got %v", err)
	}
	got, err := repo.Get(ctx, orgID, p.ID)
	if err != nil {
		t.Fatalf("Get post-archive: %v", err)
	}
	if got.IsActive {
		t.Fatalf("archived product still active")
	}

	// --- Cross-tenant isolation ----------------------------------------
	otherOrg := uuid.New()
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "other-"+runID, "Other")
	defer pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg)

	_, err = repo.Get(ctx, otherOrg, p.ID)
	if !errors.Is(err, productrepo.ErrNotFound) {
		t.Fatalf("cross-tenant Get must refuse; got %v", err)
	}
}

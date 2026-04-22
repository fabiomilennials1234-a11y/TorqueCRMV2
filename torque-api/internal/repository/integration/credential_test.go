// Integration test for S49 integration_credentials (DATABASE_URL gated).
package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	"github.com/milennials/torque-api/internal/service/crypto"
)

func TestCredential_RoundTrip(t *testing.T) {
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

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		t.Fatalf("rand: %v", err)
	}
	cipher, err := crypto.New(base64.StdEncoding.EncodeToString(keyBytes))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	store := integrationrepo.NewStore(pool, cipher)

	runID := uuid.New().String()[:8]
	orgID := uuid.New()
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "intg-"+runID, "IntgOrg "+runID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM integration_credentials WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	exp := time.Now().UTC().Add(1 * time.Hour)
	acct := "user@example.test"
	if err := store.Upsert(ctx, orgID, integrationrepo.ProviderGoogle, integrationrepo.UpsertInput{
		AccessToken:       "access-abc",
		RefreshToken:      "refresh-xyz",
		TokenType:         "Bearer",
		ExpiresAt:         &exp,
		Scopes:            []string{"calendar.events", "email"},
		ExternalAccountID: &acct,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := store.Get(ctx, orgID, integrationrepo.ProviderGoogle)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccessToken != "access-abc" || got.RefreshToken != "refresh-xyz" {
		t.Fatalf("tokens not round-tripped: %+v", got)
	}
	if got.ExternalAccountID == nil || *got.ExternalAccountID != acct {
		t.Fatalf("external account id not persisted")
	}

	if err := store.MarkSuccess(ctx, orgID, integrationrepo.ProviderGoogle); err != nil {
		t.Fatalf("mark success: %v", err)
	}
	if err := store.MarkError(ctx, orgID, integrationrepo.ProviderGoogle, "test failure"); err != nil {
		t.Fatalf("mark error: %v", err)
	}

	meta, err := store.List(ctx, orgID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(meta) != 1 || meta[0].LastErrorText == nil || *meta[0].LastErrorText != "test failure" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	if err := store.Delete(ctx, orgID, integrationrepo.ProviderGoogle); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, orgID, integrationrepo.ProviderGoogle); !errors.Is(err, integrationrepo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCredential_Isolation(t *testing.T) {
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

	keyBytes := make([]byte, 32)
	_, _ = rand.Read(keyBytes)
	cipher, _ := crypto.New(base64.StdEncoding.EncodeToString(keyBytes))
	store := integrationrepo.NewStore(pool, cipher)

	runID := uuid.New().String()[:8]
	orgA, orgB := uuid.New(), uuid.New()
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgA, "iso-a-"+runID, "A "+runID)
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgB, "iso-b-"+runID, "B "+runID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM integration_credentials WHERE organization_id IN ($1,$2)`, orgA, orgB)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	})

	if err := store.Upsert(ctx, orgA, integrationrepo.ProviderTinyERP, integrationrepo.UpsertInput{
		AccessToken: "a-key",
	}); err != nil {
		t.Fatalf("upsert A: %v", err)
	}
	if err := store.Upsert(ctx, orgB, integrationrepo.ProviderTinyERP, integrationrepo.UpsertInput{
		AccessToken: "b-key",
	}); err != nil {
		t.Fatalf("upsert B: %v", err)
	}
	a, err := store.Get(ctx, orgA, integrationrepo.ProviderTinyERP)
	if err != nil || a.AccessToken != "a-key" {
		t.Fatalf("A leak: err=%v a=%+v", err, a)
	}
	b, err := store.Get(ctx, orgB, integrationrepo.ProviderTinyERP)
	if err != nil || b.AccessToken != "b-key" {
		t.Fatalf("B leak: err=%v b=%+v", err, b)
	}
}

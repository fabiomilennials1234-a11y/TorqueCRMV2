// Integration test for the audit service — requires DATABASE_URL.
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/service/audit/...
package audit_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
	auditrepo "github.com/milennials/torque-api/internal/repository/audit"
	auditsvc "github.com/milennials/torque-api/internal/service/audit"
)

func TestAudit_RecordRoundTrip(t *testing.T) {
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
	defer pool.Close()

	runID := uuid.New().String()[:8]
	orgID := uuid.New()
	userID := uuid.New()

	mustExec(t, ctx, pool, `INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "audit-"+runID, "Audit Test "+runID)
	mustExec(t, ctx, pool, `INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','Tester')`,
		userID, "audit-"+runID+"@example.test")

	t.Cleanup(func() {
		cctx, cancelC := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelC()
		_, _ = pool.Exec(cctx, `DELETE FROM audit_log WHERE organization_id = $1 OR target_org_id = $1`, orgID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(cctx, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	svc := auditsvc.New(auditrepo.New(pool))

	sess := domain.Session{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           domain.RoleAdmin,
	}
	if err := svc.Record(ctx, sess, auditsvc.ActionLogin, "", uuid.Nil,
		map[string]string{"ua": "test-agent"}, "req-123"); err != nil {
		t.Fatalf("record: %v", err)
	}

	var (
		actorType string
		action    string
		reqID     *string
		payload   []byte
	)
	err = pool.QueryRow(ctx, `
		SELECT actor_type, action, request_id, payload
		  FROM audit_log
		 WHERE organization_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1
	`, orgID).Scan(&actorType, &action, &reqID, &payload)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if actorType != "admin" {
		t.Errorf("actor_type = %q, want admin", actorType)
	}
	if action != auditsvc.ActionLogin {
		t.Errorf("action = %q, want %q", action, auditsvc.ActionLogin)
	}
	if reqID == nil || *reqID != "req-123" {
		t.Errorf("request_id = %v, want req-123", reqID)
	}
	if len(payload) == 0 || payload[0] != '{' {
		t.Errorf("payload not jsonb: %s", string(payload))
	}

	// Impersonation — distinct row type.
	targetOrg := uuid.New()
	mustExec(t, ctx, pool, `INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		targetOrg, "audit-target-"+runID, "Audit Target "+runID)
	t.Cleanup(func() {
		cctx, cc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cc()
		_, _ = pool.Exec(cctx, `DELETE FROM audit_log WHERE target_org_id = $1`, targetOrg)
		_, _ = pool.Exec(cctx, `DELETE FROM organizations WHERE id = $1`, targetOrg)
	})

	if err := svc.RecordImpersonation(ctx, userID, targetOrg,
		auditsvc.ActionImpersonationStart, map[string]string{"reason": "support ticket 42"}, "req-999"); err != nil {
		t.Fatalf("impersonation: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE target_org_id = $1 AND actor_type = 'master'`, targetOrg).
		Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 impersonation row, got %d", count)
	}
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec %s: %v", sql, err)
	}
}

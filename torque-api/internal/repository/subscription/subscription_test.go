// Integration test for F14 billing (DATABASE_URL gated).
package subscription_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	subrepo "github.com/milennials/torque-api/internal/repository/subscription"
)

func TestSubscription_Lifecycle(t *testing.T) {
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
		orgID, "sub-"+runID, "Sub "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','S')`,
		userID, "sub-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','S')`, memberID, orgID, userID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM billing_events WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM subscriptions WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := subrepo.New(pool)
	now := time.Now().UTC()

	// --- Create pending -----------------------------------------------
	s, err := repo.Create(ctx, subrepo.CreateInput{
		OrganizationID: orgID, PlanID: "free", Provider: "mock",
		AmountCents: 19900, Currency: "BRL",
		ProviderChargeID: "ch_" + runID,
		PixQRCode: "00020126xxx", PixQRCodeImage: "data:image/svg+xml;base64,xxx",
		PixExpiresAt: now.Add(30 * time.Minute), CreatedBy: memberID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.Status != "pending" {
		t.Fatalf("want pending, got %s", s.Status)
	}

	// --- Second Create must fail with ErrAlreadyActive ----------------
	_, err = repo.Create(ctx, subrepo.CreateInput{
		OrganizationID: orgID, PlanID: "free", Provider: "mock",
		AmountCents: 19900, Currency: "BRL",
		ProviderChargeID: "ch_second", PixQRCode: "x", PixQRCodeImage: "y",
		PixExpiresAt: now.Add(30 * time.Minute), CreatedBy: memberID,
	})
	if !errors.Is(err, subrepo.ErrAlreadyActive) {
		t.Fatalf("want ErrAlreadyActive, got %v", err)
	}

	// --- Webhook dedup ------------------------------------------------
	if err := repo.RecordEvent(ctx, &orgID, &s.ID, "mock", "evt_1", "charge.paid",
		json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if err := repo.RecordEvent(ctx, &orgID, &s.ID, "mock", "evt_1", "charge.paid",
		json.RawMessage(`{}`)); !errors.Is(err, subrepo.ErrDuplicateEvent) {
		t.Fatalf("want ErrDuplicateEvent, got %v", err)
	}

	// --- MarkPaid → active + current_period_end stamped ---------------
	periodEnd := now.Add(30 * 24 * time.Hour)
	if err := repo.MarkPaid(ctx, s.ID, periodEnd); err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}
	got, _ := repo.Get(ctx, orgID, s.ID)
	if got.Status != "active" || got.CurrentPeriodEnd == nil {
		t.Fatalf("post-paid state wrong: %+v", got)
	}

	// --- Cancel -------------------------------------------------------
	if err := repo.Cancel(ctx, orgID, s.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := repo.Cancel(ctx, orgID, s.ID); !errors.Is(err, subrepo.ErrNotFound) {
		t.Fatalf("second Cancel must return ErrNotFound; got %v", err)
	}

	// --- After cancel, a new Create succeeds (invariant only blocks
	// concurrent live subs, not historical ones) ----------------------
	s2, err := repo.Create(ctx, subrepo.CreateInput{
		OrganizationID: orgID, PlanID: "free", Provider: "mock",
		AmountCents: 29900, Currency: "BRL",
		ProviderChargeID: "ch_second_" + runID,
		PixQRCode: "y", PixQRCodeImage: "z",
		PixExpiresAt: now.Add(30 * time.Minute), CreatedBy: memberID,
	})
	if err != nil {
		t.Fatalf("Create after cancel: %v", err)
	}

	// --- FindByProviderCharge -----------------------------------------
	got, err = repo.FindByProviderCharge(ctx, "mock", "ch_second_"+runID)
	if err != nil || got.ID != s2.ID {
		t.Fatalf("FindByProviderCharge: %v / mismatch", err)
	}
}

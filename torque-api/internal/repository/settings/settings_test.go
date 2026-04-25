// Integration test for F15 settings (DATABASE_URL gated).
package settings_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	settingsrepo "github.com/milennials/torque-api/internal/repository/settings"
)

func TestSettings_OrgAndWebhook(t *testing.T) {
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
		orgID, "st-"+runID, "St "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','S')`,
		userID, "st-"+runID+"@example.test")
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','S')`, memberID, orgID, userID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM notification_preferences WHERE team_member_id = $1`, memberID)
		_, _ = pool.Exec(c, `DELETE FROM webhook_endpoints WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := settingsrepo.New(pool)

	// --- Update org ---------------------------------------------------
	newName := "Siderúrgica " + runID
	tz := "America/Sao_Paulo"
	o, err := repo.UpdateOrganization(ctx, orgID, settingsrepo.UpdateOrganizationInput{
		Name: &newName, Timezone: &tz,
	})
	if err != nil {
		t.Fatalf("UpdateOrganization: %v", err)
	}
	if o.Name != newName || o.Timezone == nil || *o.Timezone != tz {
		t.Fatalf("patch not applied: %+v", o)
	}

	// --- Webhook: https check ----------------------------------------
	_, err = repo.CreateWebhook(ctx, orgID, settingsrepo.CreateWebhookInput{
		URL: "http://insecure.example.com", Secret: "sixteencharsexactly12345",
		CreatedBy: memberID,
	})
	if !errors.Is(err, settingsrepo.ErrURLScheme) {
		t.Fatalf("want ErrURLScheme, got %v", err)
	}

	// --- Webhook: happy path -----------------------------------------
	w, err := repo.CreateWebhook(ctx, orgID, settingsrepo.CreateWebhookInput{
		URL: "https://example.com/hook", Secret: "sixteencharsexactly12345",
		EventTypes: []string{"lead.*", "pipe_entry.moved"}, CreatedBy: memberID,
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if !w.IsActive || len(w.EventTypes) != 2 {
		t.Fatalf("unexpected webhook: %+v", w)
	}

	// --- Webhook: duplicate url --------------------------------------
	_, err = repo.CreateWebhook(ctx, orgID, settingsrepo.CreateWebhookInput{
		URL: "https://example.com/hook", Secret: "sixteencharsexactly12345",
		CreatedBy: memberID,
	})
	if !errors.Is(err, settingsrepo.ErrURLTaken) {
		t.Fatalf("want ErrURLTaken, got %v", err)
	}

	// --- Webhook: patch + delete --------------------------------------
	inactive := false
	w2, err := repo.UpdateWebhook(ctx, orgID, w.ID, settingsrepo.UpdateWebhookInput{IsActive: &inactive})
	if err != nil || w2.IsActive {
		t.Fatalf("UpdateWebhook did not flip is_active: %+v, err=%v", w2, err)
	}
	if err := repo.DeleteWebhook(ctx, orgID, w.ID); err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}
	if err := repo.DeleteWebhook(ctx, orgID, w.ID); !errors.Is(err, settingsrepo.ErrNotFound) {
		t.Fatalf("second Delete must return ErrNotFound; got %v", err)
	}

	// --- Notification prefs upsert -----------------------------------
	if err := repo.SetPreference(ctx, memberID, "email", "lead.created", false); err != nil {
		t.Fatalf("SetPreference: %v", err)
	}
	if err := repo.SetPreference(ctx, memberID, "email", "lead.created", true); err != nil {
		t.Fatalf("SetPreference upsert: %v", err)
	}
	list, err := repo.ListPreferences(ctx, memberID)
	if err != nil {
		t.Fatalf("ListPreferences: %v", err)
	}
	if len(list) != 1 || !list[0].Enabled {
		t.Fatalf("upsert did not flip enabled: %+v", list)
	}

	// --- Invalid channel refused -------------------------------------
	err = repo.SetPreference(ctx, memberID, "sms", "x", true)
	if err == nil {
		t.Fatalf("want channel validation error")
	}

	// --- Cross-tenant guard ------------------------------------------
	otherOrg := uuid.New()
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "other-"+runID, "Other")
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg) }()

	_, err = repo.GetWebhook(ctx, otherOrg, w.ID)
	if !errors.Is(err, settingsrepo.ErrNotFound) {
		t.Fatalf("cross-tenant GetWebhook must refuse; got %v", err)
	}
}

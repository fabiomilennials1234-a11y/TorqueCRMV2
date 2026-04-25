// Integration test for F10 team members (DATABASE_URL gated).
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/repository/member/...
package member_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
	memberrepo "github.com/milennials/torque-api/internal/repository/member"
)

func TestMember_CRUD_AndOverrides(t *testing.T) {
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
	// Admin (our actor — already a member of the org, created by the auth seed
	// in real life; we bootstrap it here so the tests do not depend on a
	// running auth flow).
	adminUserID := uuid.New()
	adminMemberID := uuid.New()
	// Target user — exists in users but not yet in the org.
	targetUserID := uuid.New()
	targetEmail := "target-" + runID + "@example.test"

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec %s: %v", sql, err)
		}
	}
	mustExec(`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		orgID, "mbr-"+runID, "Mbr "+runID)
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','Admin')`,
		adminUserID, "admin-"+runID+"@example.test")
	mustExec(`INSERT INTO users (id, email, password_hash, display_name) VALUES ($1,$2,'x','Target')`,
		targetUserID, targetEmail)
	mustExec(`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
	          VALUES ($1,$2,$3,'admin','Admin')`, adminMemberID, orgID, adminUserID)

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM member_feature_permissions WHERE team_member_id IN
			(SELECT id FROM team_members WHERE organization_id = $1)`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM team_members WHERE organization_id = $1`, orgID)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE id IN ($1,$2)`, adminUserID, targetUserID)
		_, _ = pool.Exec(c, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	repo := memberrepo.New(pool)

	// --- AddByEmail: happy path -----------------------------------------
	m, err := repo.AddByEmail(ctx, orgID, memberrepo.AddByEmailInput{
		Email: targetEmail, DisplayName: "Target User",
		Role: domain.RoleMember, InvitedBy: adminMemberID,
	})
	if err != nil {
		t.Fatalf("AddByEmail: %v", err)
	}
	if m.UserID != targetUserID || m.Role != domain.RoleMember {
		t.Fatalf("unexpected member: %+v", m)
	}

	// --- AddByEmail: duplicate returns ErrAlreadyMember -----------------
	_, err = repo.AddByEmail(ctx, orgID, memberrepo.AddByEmailInput{
		Email: targetEmail, DisplayName: "dup", Role: domain.RoleMember, InvitedBy: adminMemberID,
	})
	if !errors.Is(err, memberrepo.ErrAlreadyMember) {
		t.Fatalf("want ErrAlreadyMember, got %v", err)
	}

	// --- AddByEmail: unknown email returns ErrUserNotFound --------------
	_, err = repo.AddByEmail(ctx, orgID, memberrepo.AddByEmailInput{
		Email: "nobody-" + runID + "@example.test", DisplayName: "x",
		Role: domain.RoleMember, InvitedBy: adminMemberID,
	})
	if !errors.Is(err, memberrepo.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}

	// --- Update: promote to admin --------------------------------------
	newRole := domain.RoleAdmin
	newName := "Target Admin"
	updated, err := repo.Update(ctx, orgID, m.ID, memberrepo.UpdateInput{
		DisplayName: &newName, Role: &newRole,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Role != domain.RoleAdmin || updated.DisplayName != newName {
		t.Fatalf("patch not applied: %+v", updated)
	}

	// --- Override: set/list/clear --------------------------------------
	if err := repo.SetOverride(ctx, orgID, m.ID, "leads.delete", false, adminMemberID); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	rows, err := repo.ListOverrides(ctx, orgID, m.ID)
	if err != nil {
		t.Fatalf("ListOverrides: %v", err)
	}
	if len(rows) != 1 || rows[0].FeatureKey != "leads.delete" || rows[0].Value != false {
		t.Fatalf("unexpected override rows: %+v", rows)
	}
	// Upsert — same key, different value.
	if err := repo.SetOverride(ctx, orgID, m.ID, "leads.delete", true, adminMemberID); err != nil {
		t.Fatalf("SetOverride upsert: %v", err)
	}
	rows, _ = repo.ListOverrides(ctx, orgID, m.ID)
	if rows[0].Value != true {
		t.Fatalf("upsert did not flip value: %+v", rows)
	}
	if err := repo.ClearOverride(ctx, orgID, m.ID, "leads.delete"); err != nil {
		t.Fatalf("ClearOverride: %v", err)
	}
	rows, _ = repo.ListOverrides(ctx, orgID, m.ID)
	if len(rows) != 0 {
		t.Fatalf("override not cleared: %+v", rows)
	}

	// --- Override rejects unknown feature_key --------------------------
	err = repo.SetOverride(ctx, orgID, m.ID, "bogus.nonexistent", true, adminMemberID)
	if err == nil {
		t.Fatalf("want error for unknown feature_key")
	}

	// --- Cross-tenant guard --------------------------------------------
	otherOrg := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, slug, name, plan_id) VALUES ($1,$2,$3,'free')`,
		otherOrg, "other-"+runID, "Other"); err != nil {
		t.Fatalf("create other org: %v", err)
	}
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = $1`, otherOrg) }()

	_, err = repo.Get(ctx, otherOrg, m.ID)
	if !errors.Is(err, memberrepo.ErrNotFound) {
		t.Fatalf("cross-tenant Get must not find member; got %v", err)
	}
	err = repo.SetOverride(ctx, otherOrg, m.ID, "leads.view", true, adminMemberID)
	if !errors.Is(err, memberrepo.ErrNotFound) {
		t.Fatalf("cross-tenant SetOverride must reject; got %v", err)
	}

	// --- Deactivate ----------------------------------------------------
	if err := repo.Deactivate(ctx, orgID, m.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	// Idempotent: second call returns ErrNotFound (guard is WHERE is_active=true).
	err = repo.Deactivate(ctx, orgID, m.ID)
	if !errors.Is(err, memberrepo.ErrNotFound) {
		t.Fatalf("second Deactivate must return ErrNotFound; got %v", err)
	}
	// List without inactive skips the deactivated member.
	list, err := repo.List(ctx, orgID, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, row := range list {
		if row.ID == m.ID {
			t.Fatalf("deactivated member leaked into active List")
		}
	}
	list, _ = repo.List(ctx, orgID, true)
	seen := false
	for _, row := range list {
		if row.ID == m.ID {
			seen = true
			if row.IsActive {
				t.Fatalf("deactivated member has is_active=true")
			}
		}
	}
	if !seen {
		t.Fatalf("include_inactive List must include deactivated member")
	}
}

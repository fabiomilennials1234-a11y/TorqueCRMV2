// Package permission unit tests.
//
// S69 (D080): cobertura crítica do RBAC. O resolver delega para o
// userrepo.EffectivePermissions; o cascade de admin_only/master_only/
// member_override/default vive na repo (resolve()). Aqui testamos:
//   1. Resolve retorna a permissão correspondente ao key, fail-closed
//      em key desconhecido.
//   2. Bundle aplica master bypass corretamente (todas allowed +
//      Source="master_bypass").
//   3. Bundle preserva o resultado da repo quando NÃO master.
//   4. Erros da repo propagam (wrapped) tanto em Resolve quanto Bundle.
//
// Tests usam fakeRepo (implementa EffectivePermissionsRepo) para evitar
// pgxpool — pattern canônico do projeto (vide trigger/subscriber_test).
package permission

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
)

// --- fakeRepo --------------------------------------------------------

type fakeRepo struct {
	perms []domain.FeaturePermission
	err   error
}

func (f *fakeRepo) EffectivePermissions(_ context.Context, _ uuid.UUID, _ domain.Role) ([]domain.FeaturePermission, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.FeaturePermission, len(f.perms))
	copy(out, f.perms)
	return out, nil
}

// --- helpers ---------------------------------------------------------

func newSession(role domain.Role, isMaster bool) domain.Session {
	return domain.Session{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		TeamMemberID:   uuid.New(),
		Role:           role,
		IsMaster:       isMaster,
	}
}

// --- Resolve ---------------------------------------------------------

func TestResolve_FoundReturnsPermission(t *testing.T) {
	repo := &fakeRepo{
		perms: []domain.FeaturePermission{
			{Key: "leads.view", Allowed: true, Source: "default"},
			{Key: "leads.manage", Allowed: false, Source: "admin_only"},
		},
	}
	r := NewWithRepo(repo)
	got, err := r.Resolve(context.Background(), newSession(domain.RoleMember, false), "leads.view")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Key != "leads.view" || !got.Allowed {
		t.Errorf("expected leads.view allowed, got %+v", got)
	}
}

func TestResolve_UnknownKeyFailClosed(t *testing.T) {
	repo := &fakeRepo{
		perms: []domain.FeaturePermission{
			{Key: "leads.view", Allowed: true, Source: "default"},
		},
	}
	r := NewWithRepo(repo)
	got, err := r.Resolve(context.Background(), newSession(domain.RoleAdmin, false), "feature.does-not-exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Allowed {
		t.Errorf("unknown key must fail closed; got Allowed=true")
	}
	if got.Source != "unknown" {
		t.Errorf("expected Source=unknown for missing key; got %q", got.Source)
	}
	if got.Key != "feature.does-not-exist" {
		t.Errorf("expected Key echoed; got %q", got.Key)
	}
}

func TestResolve_RepoErrorWrapped(t *testing.T) {
	sentinel := errors.New("db down")
	r := NewWithRepo(&fakeRepo{err: sentinel})
	_, err := r.Resolve(context.Background(), newSession(domain.RoleMember, false), "leads.view")
	if err == nil {
		t.Fatal("expected error from repo")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel; got %v", err)
	}
	if !strings.Contains(err.Error(), "leads.view") {
		t.Errorf("expected feature key in error context; got %q", err.Error())
	}
}

// --- Bundle ----------------------------------------------------------

func TestBundle_NonMasterPreservesRepoResult(t *testing.T) {
	repo := &fakeRepo{
		perms: []domain.FeaturePermission{
			{Key: "leads.view", Allowed: true, Source: "default"},
			{Key: "leads.manage", Allowed: false, Source: "admin_only"},
			{Key: "billing.view", Allowed: false, Source: "master_only"},
		},
	}
	r := NewWithRepo(repo)
	got, err := r.Bundle(context.Background(), newSession(domain.RoleMember, false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 perms; got %d", len(got))
	}
	// member must NOT see admin_only or master_only allowed.
	for _, p := range got {
		if p.Key == "leads.manage" && p.Allowed {
			t.Error("admin_only must remain denied for member")
		}
		if p.Key == "billing.view" && p.Allowed {
			t.Error("master_only must remain denied for member")
		}
	}
}

func TestBundle_MasterBypassAllowsAll(t *testing.T) {
	repo := &fakeRepo{
		perms: []domain.FeaturePermission{
			{Key: "leads.view", Allowed: true, Source: "default"},
			{Key: "billing.view", Allowed: false, Source: "master_only"},
			{Key: "settings.manage", Allowed: false, Source: "admin_only"},
		},
	}
	r := NewWithRepo(repo)
	got, err := r.Bundle(context.Background(), newSession(domain.RoleAdmin, true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 perms; got %d", len(got))
	}
	for _, p := range got {
		if !p.Allowed {
			t.Errorf("master must see all allowed; %q got Allowed=false", p.Key)
		}
		if p.Source != "master_bypass" {
			t.Errorf("master Source must be master_bypass; %q got %q", p.Key, p.Source)
		}
	}
}

func TestBundle_MasterEmptyPermsReturnsEmpty(t *testing.T) {
	r := NewWithRepo(&fakeRepo{perms: []domain.FeaturePermission{}})
	got, err := r.Bundle(context.Background(), newSession(domain.RoleAdmin, true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty bundle; got %d", len(got))
	}
}

func TestBundle_RepoErrorWrapped(t *testing.T) {
	sentinel := errors.New("connection refused")
	r := NewWithRepo(&fakeRepo{err: sentinel})
	_, err := r.Bundle(context.Background(), newSession(domain.RoleMember, false))
	if err == nil {
		t.Fatal("expected error from repo")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel; got %v", err)
	}
	if !strings.Contains(err.Error(), "bundle") {
		t.Errorf("expected 'bundle' context in error; got %q", err.Error())
	}
}

// --- Multi-tenant safety: same TeamMemberID different role ----------

func TestResolve_RolePassedDownstream(t *testing.T) {
	// Verifies the role on the session is propagated to the repo. A
	// silent drop here would let a member receive admin perms.
	captured := struct {
		role domain.Role
	}{}
	repo := &fakeRepoCapturing{
		perms: []domain.FeaturePermission{{Key: "x", Allowed: true, Source: "default"}},
		seen:  &captured.role,
	}
	r := NewWithRepo(repo)
	if _, err := r.Resolve(context.Background(), newSession(domain.RoleAdmin, false), "x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.role != domain.RoleAdmin {
		t.Errorf("expected admin role propagated; got %q", captured.role)
	}
}

type fakeRepoCapturing struct {
	perms []domain.FeaturePermission
	seen  *domain.Role
}

func (f *fakeRepoCapturing) EffectivePermissions(_ context.Context, _ uuid.UUID, role domain.Role) ([]domain.FeaturePermission, error) {
	*f.seen = role
	return f.perms, nil
}

package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
)

type stubResolver struct {
	fn func(ctx context.Context, sess domain.Session, key string) (domain.FeaturePermission, error)
}

func (s stubResolver) Resolve(ctx context.Context, sess domain.Session, key string) (domain.FeaturePermission, error) {
	return s.fn(ctx, sess, key)
}

func requestWithSession(sess domain.Session) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	return req.WithContext(mw.WithSession(req.Context(), sess))
}

func TestRequireFeature_MasterBypassesWithoutResolverHit(t *testing.T) {
	t.Parallel()
	called := false
	resolver := stubResolver{fn: func(ctx context.Context, _ domain.Session, _ string) (domain.FeaturePermission, error) {
		called = true
		return domain.FeaturePermission{Allowed: false}, nil
	}}
	h := mw.RequireFeature(resolver, "whatever")(okHandler())

	sess := domain.Session{UserID: uuid.New(), OrganizationID: uuid.New(), Role: domain.RoleMember, IsMaster: true}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(sess))

	if rec.Code != http.StatusOK {
		t.Fatalf("master expected 200, got %d", rec.Code)
	}
	if called {
		t.Fatal("resolver must NOT be hit when session is master")
	}
}

func TestRequireFeature_AllowsWhenResolverSaysYes(t *testing.T) {
	t.Parallel()
	resolver := stubResolver{fn: func(_ context.Context, _ domain.Session, _ string) (domain.FeaturePermission, error) {
		return domain.FeaturePermission{Key: "x", Allowed: true, Source: "default"}, nil
	}}
	h := mw.RequireFeature(resolver, "x")(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleMember}))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireFeature_DeniesWhenResolverSaysNo(t *testing.T) {
	t.Parallel()
	resolver := stubResolver{fn: func(_ context.Context, _ domain.Session, _ string) (domain.FeaturePermission, error) {
		return domain.FeaturePermission{Key: "x", Allowed: false, Source: "admin_only"}, nil
	}}
	h := mw.RequireFeature(resolver, "x")(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleMember}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireFeature_BubblesResolverError(t *testing.T) {
	t.Parallel()
	resolver := stubResolver{fn: func(_ context.Context, _ domain.Session, _ string) (domain.FeaturePermission, error) {
		return domain.FeaturePermission{}, errors.New("boom")
	}}
	h := mw.RequireFeature(resolver, "x")(okHandler())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleMember}))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestRequireFeature_401WithoutSession(t *testing.T) {
	t.Parallel()
	resolver := stubResolver{fn: func(_ context.Context, _ domain.Session, _ string) (domain.FeaturePermission, error) {
		return domain.FeaturePermission{Allowed: true}, nil
	}}
	h := mw.RequireFeature(resolver, "x")(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil) // no session in context
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireRole_AdminPath(t *testing.T) {
	t.Parallel()
	h := mw.RequireRole(domain.RoleAdmin)(okHandler())

	// Member denied.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleMember}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member expected 403, got %d", rec.Code)
	}

	// Admin allowed.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleAdmin}))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin expected 200, got %d", rec.Code)
	}

	// Master bypass.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleMember, IsMaster: true}))
	if rec.Code != http.StatusOK {
		t.Fatalf("master expected 200, got %d", rec.Code)
	}
}

func TestRequireMaster(t *testing.T) {
	t.Parallel()
	h := mw.RequireMaster(okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleAdmin}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin-but-not-master expected 403, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, requestWithSession(domain.Session{UserID: uuid.New(), Role: domain.RoleAdmin, IsMaster: true}))
	if rec.Code != http.StatusOK {
		t.Fatalf("master expected 200, got %d", rec.Code)
	}
}

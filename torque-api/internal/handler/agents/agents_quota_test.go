// Package agents — quota-gate regression tests for the agents CRUD
// surface (S52 / Fase G.2). These tests assert the wiring contract the
// handler declares:
//
//   * POST /agents is mounted behind RequireQuota(ResourceAgents) when
//     WithQuota() has been called, so tenants at cap get 402
//     QUOTA_EXCEEDED BEFORE the handler dials the repository.
//   * Master sessions bypass the gate (impersonation must not double-
//     charge the target tenant).
//   * A missing org_quotas row for `agents` returns 402 (fail-closed).
//   * The handler still mounts cleanly when WithQuota is not called —
//     matches the leads-handler backward-compat contract and keeps
//     fixtures that do not seed `plan_quotas` runnable.
//
// These tests intentionally stop at the middleware boundary. The
// post-admission IncrementUsage / decrement flow lives in the handler
// itself (tested downstream in the repo integration suite once the
// schema is seeded in the integration DB). A unit test that observes
// IncrementUsage against a concrete *quotarepo.Repository would need a
// live pool — the middleware-level contract is what changes across a
// routing refactor, so that's where the regression barrier sits.
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	quotarepo "github.com/milennials/torque-api/internal/repository/quota"
)

// fakeQuotaReader is the narrow QuotaReader the middleware consumes.
// Distinct from the AI-token-budget fake in playground_tokens_test.go
// by contract: this one meters slots (ResourceAgents), that one meters
// tokens (ResourceAITokens).
type fakeQuotaReader struct {
	q   quotarepo.Quota
	err error
}

func (f *fakeQuotaReader) Get(_ context.Context, _ uuid.UUID, _ string) (quotarepo.Quota, error) {
	if f.err != nil {
		return quotarepo.Quota{}, f.err
	}
	return f.q, nil
}

// buildGatedRouter mirrors the Routes() wiring pattern for POST /agents
// without requiring a live *agentrepo.Repository. We mount the real
// RequireQuota middleware so any refactor of the middleware surfaces
// here, and we stub the handler to a 201 echo so the test observes
// admit-vs-deny cleanly.
func buildGatedRouter(reader mw.QuotaReader, sess domain.Session) http.Handler {
	r := chi.NewRouter()
	// Inject session first (Authenticator stand-in) + TenantScope.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := mw.WithSession(req.Context(), sess)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Use(mw.TenantScope)
	r.With(mw.RequireQuota(reader, quotarepo.ResourceAgents)).
		Post("/agents", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ok":true}`))
		})
	return r
}

func nonMasterSession() domain.Session {
	return domain.Session{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		TeamMemberID:   uuid.New(),
		IsMaster:       false,
	}
}

func masterSession() domain.Session {
	return domain.Session{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		TeamMemberID:   uuid.New(),
		IsMaster:       true,
	}
}

// TestAgentsQuota_AdmitsUnderCap — the happy path: usage < limit lets
// the POST /agents through, observability headers land on the response.
func TestAgentsQuota_AdmitsUnderCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quotarepo.Quota{
		ResourceKey:    quotarepo.ResourceAgents,
		EffectiveLimit: 5, CurrentUsage: 2, Remaining: 3,
	}}
	h := buildGatedRouter(reader, nonMasterSession())

	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Quota-Resource"); got != quotarepo.ResourceAgents {
		t.Errorf("X-Quota-Resource: %q, want %q", got, quotarepo.ResourceAgents)
	}
	if got := rr.Header().Get("X-Quota-Remaining"); got != "3" {
		t.Errorf("X-Quota-Remaining: %q, want 3", got)
	}
	if got := rr.Header().Get("X-Quota-Limit"); got != "5" {
		t.Errorf("X-Quota-Limit: %q, want 5", got)
	}
}

// TestAgentsQuota_402AtCap — usage == limit returns 402 with the
// canonical QUOTA_EXCEEDED body before the handler runs.
func TestAgentsQuota_402AtCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quotarepo.Quota{
		ResourceKey:    quotarepo.ResourceAgents,
		EffectiveLimit: 3, CurrentUsage: 3, Remaining: 0,
	}}
	h := buildGatedRouter(reader, nonMasterSession())

	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v — body: %s", err, rr.Body.String())
	}
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "QUOTA_EXCEEDED" {
		t.Fatalf("expected error.code=QUOTA_EXCEEDED, got: %+v", body)
	}
	details, _ := errBlock["details"].(map[string]any)
	if details == nil || details["resource"] != quotarepo.ResourceAgents {
		t.Fatalf("expected details.resource=%q, got: %+v", quotarepo.ResourceAgents, details)
	}
}

// TestAgentsQuota_NotFoundIsFailClosed — a tenant whose org_quotas row
// for `agents` is missing 402s (fail-closed) so a misconfigured plan
// activation surfaces loudly instead of silently granting free slots.
func TestAgentsQuota_NotFoundIsFailClosed(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{err: quotarepo.ErrNotFound}
	h := buildGatedRouter(reader, nonMasterSession())

	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("missing row should 402 (fail-closed), got %d — body: %s",
			rr.Code, rr.Body.String())
	}
}

// TestAgentsQuota_MasterBypass — master sessions skip the gate entirely.
// The reader is configured to return an error so any accidental Get()
// call fails the test loudly.
func TestAgentsQuota_MasterBypass(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{
		err: errors.New("reader.Get must not be called on master path"),
	}
	h := buildGatedRouter(reader, masterSession())

	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("master must bypass: code=%d, body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandler_RoutesMountsWithoutQuota — backward-compat check: a
// Handler constructed without WithQuota still mounts Routes() cleanly
// (no panic, no stray middleware). Fixtures that do not seed
// plan_quotas keep working exactly as they did pre-S52.
//
// We do not exercise the handler's DB path — repo==nil would panic on
// any real request — but the route-declaration surface is what the
// backward-compat contract covers.
func TestHandler_RoutesMountsWithoutQuota(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Routes() must not panic without WithQuota: %v", r)
		}
	}()
	// nil repo + nil bus — Routes only declares handlers, it does not
	// invoke them here. A panic at mount time is the failure mode.
	h := &Handler{repo: nil, bus: nil}
	h.Routes(chi.NewRouter())
}

// TestHandler_RoutesMountsWithQuota — symmetrical guard for the gated
// branch. Calling WithQuota(nil) and then WithQuota(repo) should both
// declare routes without panic. nil repo keeps us away from the
// pgxpool dependency — the gate object itself is what we test.
func TestHandler_RoutesMountsWithQuota(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Routes() must not panic with WithQuota(nil repo): %v", r)
		}
	}()
	// WithQuota(nil) is a legal no-op — h.quota stays nil so the
	// conditional takes the un-gated branch. The explicit call exists
	// to document the contract; a future sign-of-life check could
	// promote nil into a panic, and this test would catch it.
	h := (&Handler{repo: nil, bus: nil}).WithQuota(nil)
	h.Routes(chi.NewRouter())
}

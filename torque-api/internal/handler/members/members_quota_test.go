// Package members — quota-wiring unit tests for the admin handler
// (S52). These exercise ONLY the middleware composition on POST /members
// and the WithQuota builder contract — end-to-end create/deactivate
// paths that touch the DB live in the integration suite gated by
// DATABASE_URL.
//
// Strategy: swap the `quotaGate` behind the handler with a fake so chi
// still sees a non-nil gate and mounts the RequireQuota middleware.
// The repo is left nil because the request is rejected at the gate
// before it reaches the handler on the deny paths — that's the entire
// point of the 402 path. For master-bypass and nil-quota fall-through
// tests we catch any panic past the middleware so the "did not 402"
// assertion is decoupled from the absent repo.
package members

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/repository/quota"
)

// -------- fake quota gate --------------------------------------------

type fakeQuotaGate struct {
	getQ     quota.Quota
	getErr   error
	incCalls []fakeIncCall
}

type fakeIncCall struct {
	Resource string
	Delta    int
}

func (f *fakeQuotaGate) Get(_ context.Context, _ uuid.UUID, _ string) (quota.Quota, error) {
	return f.getQ, f.getErr
}

func (f *fakeQuotaGate) IncrementUsage(_ context.Context, _ uuid.UUID, resource string, delta int) (int, error) {
	f.incCalls = append(f.incCalls, fakeIncCall{Resource: resource, Delta: delta})
	return 0, nil
}

func newFakeGate(q quota.Quota, err error) *fakeQuotaGate {
	return &fakeQuotaGate{getQ: q, getErr: err}
}

// -------- fixtures ---------------------------------------------------

// buildRouter mounts TenantScope (so OrgIDFrom works) + the admin
// handler Routes under chi. Session with the desired master flag is
// injected per-request in authedRequest.
func buildRouter(h *AdminHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(mw.TenantScope)
	h.Routes(r)
	return r
}

func authedRequest(t *testing.T, method, path, body string, master bool) *http.Request {
	t.Helper()
	sess := domain.Session{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		TeamMemberID:   uuid.New(),
		IsMaster:       master,
	}
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(mw.WithSession(req.Context(), sess))
}

// -------- tests ------------------------------------------------------

// TestMembersQuota_GateMountedWhenWithQuotaIsSet verifies the
// conditional routing in Routes(): a handler with quotaGate != nil
// composes RequireQuota in front of POST /members and returns 402
// when the gate says we're at cap.
func TestMembersQuota_GateMountedWhenWithQuotaIsSet(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{
		ResourceKey:    quota.ResourceTeamMembers,
		EffectiveLimit: 5,
		CurrentUsage:   5, // at cap
		Remaining:      0,
	}, nil)
	h := &AdminHandler{quota: gate}

	rr := httptest.NewRecorder()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/members",
		`{"email":"x@y.z","display_name":"X","role":"membro"}`, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 at cap, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "QUOTA_EXCEEDED" {
		t.Fatalf("missing canonical error code: %+v", body)
	}
	if rr.Header().Get("X-Quota-Resource") != quota.ResourceTeamMembers {
		t.Fatalf("missing quota header: %q", rr.Header().Get("X-Quota-Resource"))
	}
	if len(gate.incCalls) != 0 {
		t.Fatalf("no increment must fire on 402 path: %+v", gate.incCalls)
	}
}

// TestMembersQuota_NotFoundIsFailClosed — a tenant without a seeded
// org_quotas row gets 402 (not 500, not 200). Matches the S51 contract
// so operators see misconfigurations as a hard error rather than a
// silent free pass.
func TestMembersQuota_NotFoundIsFailClosed(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{}, quota.ErrNotFound)
	h := &AdminHandler{quota: gate}

	rr := httptest.NewRecorder()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/members",
		`{"email":"x@y.z","display_name":"X","role":"membro"}`, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("missing quota row should 402 fail-closed, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestMembersQuota_MasterBypass — master sessions short-circuit the
// gate (they routinely drive tenants via impersonation and would
// otherwise falsely consume their targets' seats). With repo nil the
// handler will panic past the gate; we recover and assert only the
// middleware-level outcome (not 402).
func TestMembersQuota_MasterBypass(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{EffectiveLimit: 1, CurrentUsage: 1}, nil) // pathological: at cap
	h := &AdminHandler{quota: gate}

	rr := httptest.NewRecorder()
	defer func() { _ = recover() }() // handler with nil repo may panic — expected
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/members",
		`{"email":"x@y.z","display_name":"X","role":"membro"}`, true))

	if rr.Code == http.StatusPaymentRequired {
		t.Fatalf("master must bypass gate, got 402")
	}
}

// TestMembersQuota_NilQuotaFallsThrough — the handler must remain
// functional when WithQuota is never called (e.g., test fixtures, or
// envs where plan_quotas haven't been seeded yet). Regression guard
// for the "collapsed to unbounded" fallback.
func TestMembersQuota_NilQuotaFallsThrough(t *testing.T) {
	t.Parallel()
	h := &AdminHandler{} // quota stays nil

	rr := httptest.NewRecorder()
	defer func() { _ = recover() }() // handler with nil repo may panic — expected past the absent gate
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/members",
		`{"email":"x@y.z","display_name":"X","role":"membro"}`, false))

	if rr.Code == http.StatusPaymentRequired {
		t.Fatalf("nil quota must not gate: got 402")
	}
}

// TestMembersQuota_WithQuotaNilSafeTyped — passing a typed-nil
// *quotarepo.Repository through WithQuota must leave the interface
// field genuinely nil. Without the explicit guard in WithQuota this
// would stash a non-nil interface containing a nil pointer — a subtle
// Go gotcha that turns the nil check in Routes into an always-true.
func TestMembersQuota_WithQuotaNilSafeTyped(t *testing.T) {
	t.Parallel()
	h := NewAdmin(nil, nil).WithQuota(nil)
	if h.quota != nil {
		t.Fatalf("WithQuota(nil) must leave quota nil, got %T", h.quota)
	}
}

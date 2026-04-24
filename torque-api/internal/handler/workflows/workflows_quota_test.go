// Package workflows — quota-wiring unit tests for POST /workflows
// (S52). Mirrors the members_quota_test.go strategy: swap the handler's
// `quotaGate` with a fake so chi mounts the RequireQuota middleware,
// assert on the 402 / admit / bypass paths, and leave the repo nil
// because the 402 paths never reach the handler.
package workflows

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

func buildRouter(h *Handler) http.Handler {
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

// TestWorkflowsQuota_GateMountedWhenWithQuotaIsSet — POST /workflows
// returns 402 QUOTA_EXCEEDED when the gate reports current_usage has
// caught up to effective_limit. Verifies (a) middleware is mounted,
// (b) resource key is "workflows", (c) no IncrementUsage fires on deny.
func TestWorkflowsQuota_GateMountedWhenWithQuotaIsSet(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{
		ResourceKey:    quota.ResourceWorkflows,
		EffectiveLimit: 3,
		CurrentUsage:   3, // at cap
		Remaining:      0,
	}, nil)
	h := &Handler{quota: gate}

	rr := httptest.NewRecorder()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/workflows",
		`{"name":"Onboard","trigger":"manual"}`, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 at cap, got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "QUOTA_EXCEEDED" {
		t.Fatalf("missing canonical error code: %+v", body)
	}
	if rr.Header().Get("X-Quota-Resource") != quota.ResourceWorkflows {
		t.Fatalf("missing quota header: %q", rr.Header().Get("X-Quota-Resource"))
	}
	if len(gate.incCalls) != 0 {
		t.Fatalf("no increment must fire on 402 path: %+v", gate.incCalls)
	}
}

// TestWorkflowsQuota_NotFoundIsFailClosed — tenants without a seeded
// org_quotas row for `workflows` get 402, matching the S51 contract.
func TestWorkflowsQuota_NotFoundIsFailClosed(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{}, quota.ErrNotFound)
	h := &Handler{quota: gate}

	rr := httptest.NewRecorder()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/workflows",
		`{"name":"Onboard","trigger":"manual"}`, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("missing quota row should 402 fail-closed, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestWorkflowsQuota_MasterBypass — master sessions skip the gate.
// Repo is nil so the handler may panic past the middleware; recover
// so the assertion is scoped to "did not 402".
func TestWorkflowsQuota_MasterBypass(t *testing.T) {
	t.Parallel()
	gate := newFakeGate(quota.Quota{EffectiveLimit: 1, CurrentUsage: 1}, nil)
	h := &Handler{quota: gate}

	rr := httptest.NewRecorder()
	defer func() { _ = recover() }()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/workflows",
		`{"name":"Onboard","trigger":"manual"}`, true))

	if rr.Code == http.StatusPaymentRequired {
		t.Fatalf("master must bypass gate, got 402")
	}
}

// TestWorkflowsQuota_NilQuotaFallsThrough — the pre-S52 unbounded
// path must remain available when WithQuota is never called.
func TestWorkflowsQuota_NilQuotaFallsThrough(t *testing.T) {
	t.Parallel()
	h := &Handler{}

	rr := httptest.NewRecorder()
	defer func() { _ = recover() }()
	buildRouter(h).ServeHTTP(rr, authedRequest(t, http.MethodPost, "/workflows",
		`{"name":"Onboard","trigger":"manual"}`, false))

	if rr.Code == http.StatusPaymentRequired {
		t.Fatalf("nil quota must not gate: got 402")
	}
}

// TestWorkflowsQuota_WithQuotaNilSafeTyped — typed-nil repo does not
// pollute the interface field. See members_quota_test for the
// Go-gotcha rationale.
func TestWorkflowsQuota_WithQuotaNilSafeTyped(t *testing.T) {
	t.Parallel()
	h := New(nil, nil).WithQuota(nil)
	if h.quota != nil {
		t.Fatalf("WithQuota(nil) must leave quota nil, got %T", h.quota)
	}
}

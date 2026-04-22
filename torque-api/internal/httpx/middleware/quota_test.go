package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/repository/quota"
)

// fakeQuotaReader is a hand-rolled stub for the QuotaReader interface.
// Lets us exercise RequireQuota without a pgxpool.
type fakeQuotaReader struct {
	q   quota.Quota
	err error
}

func (f *fakeQuotaReader) Get(_ context.Context, _ uuid.UUID, _ string) (quota.Quota, error) {
	if f.err != nil {
		return quota.Quota{}, f.err
	}
	return f.q, nil
}

// withSession builds a request that looks like it just came off the
// TenantScope + auth stack — session + org in context.
func withSession(t *testing.T, master bool) *http.Request {
	t.Helper()
	orgID := uuid.New()
	sess := domain.Session{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		TeamMemberID:   uuid.New(),
		IsMaster:       master,
	}
	req := httptest.NewRequest(http.MethodPost, "/leads", nil)
	ctx := WithSession(req.Context(), sess)
	ctx = context.WithValue(ctx, tenantCtxKey{}, orgID)
	return req.WithContext(ctx)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireQuota_AdmitsUnderCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quota.Quota{
		ResourceKey: quota.ResourceLeads, EffectiveLimit: 100, CurrentUsage: 50, Remaining: 50,
	}}
	mw := RequireQuota(reader, quota.ResourceLeads)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusOK {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-Quota-Resource") != quota.ResourceLeads {
		t.Errorf("missing quota header")
	}
	if rr.Header().Get("X-Quota-Remaining") != "50" {
		t.Errorf("remaining header: %q", rr.Header().Get("X-Quota-Remaining"))
	}
}

func TestRequireQuota_402AtCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quota.Quota{
		ResourceKey: quota.ResourceLeads, EffectiveLimit: 10, CurrentUsage: 10, Remaining: 0,
	}}
	mw := RequireQuota(reader, quota.ResourceLeads)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	// Inspect JSON body for the canonical code.
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "QUOTA_EXCEEDED" {
		t.Fatalf("body: %+v", body)
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "leads") {
		t.Fatalf("body missing resource: %s", rr.Body.String())
	}
}

func TestRequireQuota_NotFoundIsFailClosed(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{err: quota.ErrNotFound}
	mw := RequireQuota(reader, quota.ResourceLeads)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("missing row should 402 (fail closed), got %d", rr.Code)
	}
}

func TestRequireQuota_MasterBypass(t *testing.T) {
	t.Parallel()
	// Pathological reader — should never be called because master bypass
	// short-circuits before the Get().
	reader := &fakeQuotaReader{err: errors.New("reader must not be called on master path")}
	mw := RequireQuota(reader, quota.ResourceLeads)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, true))

	if rr.Code != http.StatusOK {
		t.Fatalf("master must bypass: %d, %s", rr.Code, rr.Body.String())
	}
}

func TestRequireQuota_Unauthenticated(t *testing.T) {
	t.Parallel()
	mw := RequireQuota(&fakeQuotaReader{}, quota.ResourceLeads)(okHandler())

	// Request without session/tenant in ctx.
	req := httptest.NewRequest(http.MethodPost, "/leads", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code: %d", rr.Code)
	}
}

func TestRequireQuota_MissingOrgScope(t *testing.T) {
	t.Parallel()
	mw := RequireQuota(&fakeQuotaReader{}, quota.ResourceLeads)(okHandler())

	sess := domain.Session{
		UserID:         uuid.New(),
		OrganizationID: uuid.Nil,
		IsMaster:       false,
	}
	req := httptest.NewRequest(http.MethodPost, "/leads", nil)
	ctx := WithSession(req.Context(), sess)
	// intentionally NOT setting tenantCtxKey

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req.WithContext(ctx))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing org, got %d", rr.Code)
	}
}

func TestRequireQuota_PanicsOnEmptyResource(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty resource — factory guard")
		}
	}()
	_ = RequireQuota(&fakeQuotaReader{}, "")
}

// Unit tests for the master impersonation cookie swap flow.
//
// These tests exercise Handler.impersonate with fakes for the audit
// appender and the impersonation resolver so we do not require a live
// Postgres. The JWT service is real — it is pure crypto and the exact
// signature / expiry we want to assert on is what the handler will emit.
//
// What we guard here (D062 + ADR-003):
//
//   * audit-first: audit write failure  ⇒ 500 AUDIT_FAILED + NO cookie
//   * success path: 200 + session cookie set with the same attributes
//     as /auth/login (HttpOnly, Secure, SameSite=Strict, Path=/)
//   * Domain attribute respects h.cookieDomain (host-only when empty)
//   * JWT payload: sub == master user_id, org_id == target org,
//     IsMaster stays true, role == admin
//   * resolver "no admin in target org" ⇒ 404 NOT_FOUND, no cookie
package master

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/repository/audit"
	masterrepo "github.com/milennials/torque-api/internal/repository/master"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
)

// ---------- fakes ----------------------------------------------------

type fakeResolver struct {
	memberID uuid.UUID
	userID   uuid.UUID
	err      error
	calls    int
}

func (f *fakeResolver) ImpersonationTarget(_ context.Context, _ uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	f.calls++
	if f.err != nil {
		return uuid.Nil, uuid.Nil, f.err
	}
	return f.memberID, f.userID, nil
}

type fakeAudit struct {
	err     error
	entries []audit.Entry
}

func (f *fakeAudit) Append(_ context.Context, e audit.Entry) error {
	f.entries = append(f.entries, e)
	return f.err
}

// ---------- fixtures -------------------------------------------------

const testSecret = "integration-test-secret-at-least-thirty-two-bytes-long"

type harness struct {
	t            *testing.T
	handler      *Handler
	jwt          *jwtsvc.Service
	resolver     *fakeResolver
	audit        *fakeAudit
	masterUserID uuid.UUID
	masterOrgID  uuid.UUID
	targetOrgID  uuid.UUID
	targetUserID uuid.UUID
	targetTMID   uuid.UUID
}

func newHarness(t *testing.T, cookieSecure bool, cookieDomain string) *harness {
	t.Helper()
	j, err := jwtsvc.New([]byte(testSecret), 15*time.Minute)
	if err != nil {
		t.Fatalf("jwt.New: %v", err)
	}
	h := &harness{
		t:            t,
		jwt:          j,
		resolver:     &fakeResolver{memberID: uuid.New(), userID: uuid.New()},
		audit:        &fakeAudit{},
		masterUserID: uuid.New(),
		masterOrgID:  uuid.New(),
		targetOrgID:  uuid.New(),
	}
	h.targetTMID = h.resolver.memberID
	h.targetUserID = h.resolver.userID
	h.handler = &Handler{
		impResolver:  h.resolver,
		auditRepo:    h.audit,
		jwt:          j,
		cookieSecure: cookieSecure,
		cookieDomain: cookieDomain,
	}
	return h
}

// call builds a chi-routed request with a master session in context and
// returns the recorded response.
func (h *harness) call() *httptest.ResponseRecorder {
	h.t.Helper()
	r := chi.NewRouter()
	r.Post("/master/organizations/{id}/impersonate", h.handler.impersonate)

	body := bytes.NewReader(nil)
	req := httptest.NewRequest(http.MethodPost,
		"/master/organizations/"+h.targetOrgID.String()+"/impersonate", body)
	sess := domain.Session{
		UserID:         h.masterUserID,
		OrganizationID: h.masterOrgID,
		TeamMemberID:   uuid.New(),
		Role:           domain.RoleAdmin,
		IsMaster:       true,
		UIMode:         domain.UIModeManager,
	}
	req = req.WithContext(mw.WithSession(req.Context(), sess))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// ---------- tests ----------------------------------------------------

// Audit write failure: token MUST NOT be minted; cookie MUST NOT be set;
// response MUST be 500 with code AUDIT_FAILED.
func TestImpersonate_AuditFailsRefusesToken(t *testing.T) {
	h := newHarness(t, true, "")
	h.audit.err = errors.New("pg down")

	rec := h.call()
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", resp.StatusCode)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "AUDIT_FAILED" {
		t.Fatalf("code: got %q, want AUDIT_FAILED", body.Code)
	}
	if c := findCookie(resp, mw.SessionCookieName); c != nil {
		t.Fatalf("session cookie MUST NOT be set when audit fails; got %+v", c)
	}
	if h.resolver.calls != 1 {
		t.Fatalf("resolver must have been called exactly once, got %d", h.resolver.calls)
	}
	if len(h.audit.entries) != 1 {
		t.Fatalf("audit.Append must have been attempted, got %d entries", len(h.audit.entries))
	}
}

// Happy path: cookie is set with the exact attributes /auth/login uses,
// and the JWT inside decodes to the impersonated session shape.
func TestImpersonate_HappyPathSetsCookieAndTokenShape(t *testing.T) {
	h := newHarness(t, true, "")

	rec := h.call()
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	cookie := findCookie(resp, mw.SessionCookieName)
	if cookie == nil {
		t.Fatalf("session cookie MUST be set on success")
	}
	if !cookie.HttpOnly {
		t.Fatalf("cookie.HttpOnly must be true")
	}
	if !cookie.Secure {
		t.Fatalf("cookie.Secure must be true when cookieSecure=true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie.SameSite: got %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie.Path: got %q, want %q (must mirror /auth/login)", cookie.Path, "/")
	}
	if cookie.Value == "" {
		t.Fatalf("cookie.Value is empty — JWT not set on the cookie")
	}
	// Cookie expiry should roughly match JWT ttl; give 1s slack.
	if cookie.Expires.Before(time.Now().Add(14*time.Minute)) ||
		cookie.Expires.After(time.Now().Add(16*time.Minute)) {
		t.Fatalf("cookie.Expires outside expected window: %v", cookie.Expires)
	}

	// JWT payload must reflect impersonation: sub = master userID, org =
	// target, role = admin, master flag preserved.
	claims, err := h.jwt.Parse(cookie.Value)
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}
	if claims.Subject != h.masterUserID.String() {
		t.Fatalf("jwt.sub: got %q, want %q (master user_id)", claims.Subject, h.masterUserID)
	}
	if claims.OrgID != h.targetOrgID {
		t.Fatalf("jwt.org_id: got %v, want %v (target org)", claims.OrgID, h.targetOrgID)
	}
	if claims.TeamMemberID != h.targetTMID {
		t.Fatalf("jwt.tm_id: got %v, want %v", claims.TeamMemberID, h.targetTMID)
	}
	if claims.Role != domain.RoleAdmin {
		t.Fatalf("jwt.role: got %v, want admin", claims.Role)
	}
	if !claims.IsMaster {
		t.Fatalf("jwt.master: must be true so RBAC keeps the master bypass")
	}

	// Audit entry must have been written with the right actor/target.
	if len(h.audit.entries) != 1 {
		t.Fatalf("audit entries: got %d, want 1", len(h.audit.entries))
	}
	e := h.audit.entries[0]
	if e.ActorType != "master" {
		t.Fatalf("audit.ActorType: got %q, want master", e.ActorType)
	}
	if e.Action != "master.impersonation_start" {
		t.Fatalf("audit.Action: got %q", e.Action)
	}
	if e.ActorUserID == nil || *e.ActorUserID != h.masterUserID {
		t.Fatalf("audit.ActorUserID: got %v, want %v", e.ActorUserID, h.masterUserID)
	}
	if e.TargetOrgID == nil || *e.TargetOrgID != h.targetOrgID {
		t.Fatalf("audit.TargetOrgID: got %v, want %v", e.TargetOrgID, h.targetOrgID)
	}
}

// Cookie Domain attribute respects the configured h.cookieDomain. Empty
// domain yields a host-only cookie (this is what we want in dev).
func TestImpersonate_CookieDomainRespectsConfig(t *testing.T) {
	t.Run("host-only when cookieDomain is empty", func(t *testing.T) {
		h := newHarness(t, false, "")
		rec := h.call()
		resp := rec.Result()
		defer resp.Body.Close()
		cookie := findCookie(resp, mw.SessionCookieName)
		if cookie == nil {
			t.Fatalf("cookie must be set")
		}
		if cookie.Domain != "" {
			t.Fatalf("cookie.Domain: got %q, want empty (host-only)", cookie.Domain)
		}
		// cookieSecure=false → Secure must be false too
		if cookie.Secure {
			t.Fatalf("cookie.Secure must be false when cookieSecure=false")
		}
	})

	t.Run("explicit domain passed through", func(t *testing.T) {
		// Input traz leading dot (convenção legacy pre-RFC 6265). Go's
		// net/http strip o leading dot on serialize per spec — modern
		// browsers tratam Domain=foo.com como já cobrindo subdomains.
		// Sem o dot input ainda funciona; com dot, Go normaliza out.
		h := newHarness(t, true, ".torque.app")
		rec := h.call()
		resp := rec.Result()
		defer resp.Body.Close()
		cookie := findCookie(resp, mw.SessionCookieName)
		if cookie == nil {
			t.Fatalf("cookie must be set")
		}
		if cookie.Domain != "torque.app" {
			t.Fatalf("cookie.Domain: got %q, want torque.app (Go RFC-strips leading dot on serialize)", cookie.Domain)
		}
	})
}

// Resolver returns ErrNotFound (no admin in target org) → 404 + no cookie.
func TestImpersonate_TargetHasNoAdminReturns404(t *testing.T) {
	h := newHarness(t, true, "")
	h.resolver.err = masterrepo.ErrNotFound

	rec := h.call()
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
	if c := findCookie(resp, mw.SessionCookieName); c != nil {
		t.Fatalf("cookie MUST NOT be set when target has no admin")
	}
	if len(h.audit.entries) != 0 {
		t.Fatalf("audit MUST NOT be written before target is resolved; got %d entries", len(h.audit.entries))
	}
}

// Invalid UUID in the path → 400, no resolver call, no audit, no cookie.
func TestImpersonate_InvalidOrgIDReturns400(t *testing.T) {
	h := newHarness(t, true, "")
	r := chi.NewRouter()
	r.Post("/master/organizations/{id}/impersonate", h.handler.impersonate)

	req := httptest.NewRequest(http.MethodPost,
		"/master/organizations/not-a-uuid/impersonate", nil)
	sess := domain.Session{
		UserID:         h.masterUserID,
		OrganizationID: h.masterOrgID,
		TeamMemberID:   uuid.New(),
		Role:           domain.RoleAdmin,
		IsMaster:       true,
		UIMode:         domain.UIModeManager,
	}
	req = req.WithContext(mw.WithSession(req.Context(), sess))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
	if h.resolver.calls != 0 {
		t.Fatalf("resolver must not be called on invalid id, got %d", h.resolver.calls)
	}
	if len(h.audit.entries) != 0 {
		t.Fatalf("audit must not be written on invalid id, got %d", len(h.audit.entries))
	}
	if c := findCookie(resp, mw.SessionCookieName); c != nil {
		t.Fatalf("cookie must not be set on invalid id")
	}
}

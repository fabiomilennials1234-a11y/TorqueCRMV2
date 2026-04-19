// Package auth_test runs against a real PostgreSQL via DATABASE_URL.
//
//	DATABASE_URL="postgres://torque:torque@localhost:5432/torque?sslmode=disable" \
//	  go test -race ./internal/handler/auth/...
//
// Skipped by default under `go test -short`.
package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	authhandler "github.com/milennials/torque-api/internal/handler/auth"
	preferenceshandler "github.com/milennials/torque-api/internal/handler/preferences"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
	"github.com/milennials/torque-api/internal/service/password"
	"github.com/milennials/torque-api/internal/service/permission"
)

const testJWTSecret = "integration-test-secret-at-least-thirty-two-bytes-long"

// -------- fixture helpers --------------------------------------------

type fixture struct {
	t      *testing.T
	pool   *pgxpool.Pool
	server *httptest.Server
	runID  string

	orgID    uuid.UUID
	userID   uuid.UUID
	memberID uuid.UUID
	email    string
	password string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	runID := uuid.New().String()[:8]
	fx := &fixture{
		t:        t,
		pool:     pool,
		runID:    runID,
		orgID:    uuid.New(),
		userID:   uuid.New(),
		memberID: uuid.New(),
		email:    "it-" + runID + "@example.test",
		password: "correct-horse-battery-staple",
	}

	hash, err := password.Hash(fx.password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	mustExec(t, ctx, pool,
		`INSERT INTO organizations (id, slug, name, plan_id)
		 VALUES ($1, $2, $3, 'free')`,
		fx.orgID, "it-"+runID, "IT Org "+runID)
	mustExec(t, ctx, pool,
		`INSERT INTO users (id, email, password_hash, display_name)
		 VALUES ($1, $2, $3, 'Tester')`,
		fx.userID, fx.email, hash)
	mustExec(t, ctx, pool,
		`INSERT INTO team_members (id, organization_id, user_id, role, display_name)
		 VALUES ($1, $2, $3, 'admin', 'Tester')`,
		fx.memberID, fx.orgID, fx.userID)

	// Cleanup always.
	t.Cleanup(func() {
		cctx, cancelC := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelC()
		_, _ = pool.Exec(cctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, fx.userID)
		_, _ = pool.Exec(cctx, `DELETE FROM team_members WHERE id = $1`, fx.memberID)
		_, _ = pool.Exec(cctx, `DELETE FROM users WHERE id = $1`, fx.userID)
		_, _ = pool.Exec(cctx, `DELETE FROM organizations WHERE id = $1`, fx.orgID)
		pool.Close()
	})

	fx.server = httptest.NewServer(buildRouter(t, pool))
	t.Cleanup(fx.server.Close)
	return fx
}

func buildRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	jsvc, err := jwtsvc.New([]byte(testJWTSecret), 15*time.Minute)
	if err != nil {
		t.Fatalf("jwt new: %v", err)
	}
	users := userrepo.New(pool)
	refresh := refreshrepo.New(pool)
	perms := permission.New(users)

	auth := authhandler.New(authhandler.Options{
		Pool:         pool,
		Users:        users,
		Refresh:      refresh,
		JWT:          jsvc,
		Permission:   perms,
		RefreshTTL:   24 * time.Hour,
		CookieSecure: false, // test over plain HTTP
		CookieDomain: "",
	})

	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.StripOrganizationID)
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Use(mw.Authenticator(jsvc))
		auth.Routes(v1)
		v1.Group(func(priv chi.Router) {
			priv.Use(mw.RequireAuth)
			priv.Use(mw.CSRF)
			auth.MeRoute(priv)
			priv.Group(func(tnt chi.Router) {
				tnt.Use(mw.TenantScope)
				preferenceshandler.New(users).Routes(tnt)
			})
		})
	})
	return r
}

// -------- client helpers ---------------------------------------------

// client returns a fresh http.Client with a cookie jar scoped to this test.
func (f *fixture) client() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		f.t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Jar: jar, Timeout: 10 * time.Second}
}

func (f *fixture) postJSON(c *http.Client, path string, body any, extraHeaders map[string]string) (*http.Response, string) {
	f.t.Helper()
	var rdr io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(http.MethodPost, f.server.URL+path, rdr)
	if err != nil {
		f.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		f.t.Fatalf("post %s: %v", path, err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, string(bodyBytes)
}

func (f *fixture) get(c *http.Client, path string, extraHeaders map[string]string) (*http.Response, string) {
	f.t.Helper()
	req, err := http.NewRequest(http.MethodGet, f.server.URL+path, nil)
	if err != nil {
		f.t.Fatalf("new request: %v", err)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		f.t.Fatalf("get %s: %v", path, err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, string(bodyBytes)
}

// csrfFromJar extracts the __torque_csrf cookie value for X-CSRF-Token echo.
func csrfFromJar(c *http.Client, serverURL string) string {
	u, _ := url.Parse(serverURL)
	for _, cookie := range c.Jar.Cookies(u) {
		if cookie.Name == mw.CSRFCookieName {
			return cookie.Value
		}
	}
	return ""
}

// -------- tests -------------------------------------------------------

func TestLogin_Success_And_Me(t *testing.T) {
	fx := newFixture(t)

	c := fx.client()
	resp, body := fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": fx.email, "password": fx.password}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", resp.StatusCode, body)
	}

	// Cookies set: session, refresh, csrf.
	serverURL, _ := url.Parse(fx.server.URL)
	authURL, _ := url.Parse(fx.server.URL + "/api/v1/auth")
	hasCookie := func(name string) bool {
		for _, ck := range c.Jar.Cookies(serverURL) {
			if ck.Name == name {
				return true
			}
		}
		for _, ck := range c.Jar.Cookies(authURL) {
			if ck.Name == name {
				return true
			}
		}
		return false
	}
	if !hasCookie(mw.SessionCookieName) {
		t.Fatal("missing __torque_session cookie")
	}
	if !hasCookie(mw.CSRFCookieName) {
		t.Fatal("missing __torque_csrf cookie")
	}

	// GET /auth/me with CSRF (me is GET → csrf not required, but header accepted).
	resp, body = fx.get(c, "/api/v1/auth/me", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me expected 200, got %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, fx.email) {
		t.Fatalf("/auth/me body missing email: %s", body)
	}
	if !strings.Contains(body, "\"role\":\"admin\"") {
		t.Fatalf("/auth/me body missing role admin: %s", body)
	}
}

func TestLogin_BadPassword(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	resp, body := fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": fx.email, "password": "wrong"}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.StatusCode, body)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	resp, body := fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": "nobody-" + fx.runID + "@example.test", "password": "whatever"}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.StatusCode, body)
	}
}

func TestMe_RequiresCookie(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	resp, _ := fx.get(c, "/api/v1/auth/me", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}
}

func TestRefreshRotation_Success_ThenReuseRevokesChain(t *testing.T) {
	fx := newFixture(t)

	c := fx.client()
	// Login → cookies.
	resp, body := fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": fx.email, "password": fx.password}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d %s", resp.StatusCode, body)
	}

	// Capture the current refresh cookie for later reuse.
	authURL, _ := url.Parse(fx.server.URL + "/api/v1/auth")
	var oldRefresh *http.Cookie
	for _, ck := range c.Jar.Cookies(authURL) {
		if ck.Name == "__torque_refresh" {
			oldRefresh = ck
			break
		}
	}
	if oldRefresh == nil {
		t.Fatal("no refresh cookie after login")
	}

	// Rotate once — jar now carries the new refresh.
	resp, body = fx.postJSON(c, "/api/v1/auth/refresh", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d body=%s", resp.StatusCode, body)
	}

	// Present the OLD refresh manually. It is one-shot, so now it must be
	// rejected AND the whole chain revoked.
	req, _ := http.NewRequest(http.MethodPost, fx.server.URL+"/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: oldRefresh.Name, Value: oldRefresh.Value})
	bare := &http.Client{Timeout: 10 * time.Second}
	r2, err := bare.Do(req)
	if err != nil {
		t.Fatalf("reuse request: %v", err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reuse expected 401, got %d", r2.StatusCode)
	}

	// The jar's (new) refresh should ALSO be dead now — chain revoked.
	resp, body = fx.postJSON(c, "/api/v1/auth/refresh", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-reuse refresh expected 401 (chain revoked), got %d body=%s", resp.StatusCode, body)
	}
}

func TestLogout_ClearsCookies_AndRevokes(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": fx.email, "password": fx.password}, nil)

	resp, body := fx.postJSON(c, "/api/v1/auth/logout", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout expected 204, got %d body=%s", resp.StatusCode, body)
	}
	// /auth/me must fail now.
	resp, _ = fx.get(c, "/api/v1/auth/me", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout /me expected 401, got %d", resp.StatusCode)
	}
}

func TestPreferences_RequiresCSRF(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	fx.postJSON(c, "/api/v1/auth/login",
		map[string]string{"email": fx.email, "password": fx.password}, nil)

	// Without CSRF → 403.
	resp, body := fx.postPatchJSON(c, "/api/v1/me/preferences",
		map[string]string{"ui_mode": "salesperson"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("without CSRF expected 403, got %d body=%s", resp.StatusCode, body)
	}

	// With CSRF header matching the cookie → 200.
	csrf := csrfFromJar(c, fx.server.URL)
	if csrf == "" {
		t.Fatal("no csrf cookie after login")
	}
	resp, body = fx.postPatchJSON(c, "/api/v1/me/preferences",
		map[string]string{"ui_mode": "salesperson"},
		map[string]string{mw.CSRFHeaderName: csrf})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("with CSRF expected 200, got %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "salesperson") {
		t.Fatalf("response missing salesperson: %s", body)
	}
}

func TestStripOrganizationID_RejectsBodyField(t *testing.T) {
	fx := newFixture(t)
	c := fx.client()
	// No need to log in — strip runs before auth.
	body := map[string]any{"email": fx.email, "password": fx.password, "organization_id": uuid.New()}
	resp, respBody := fx.postJSON(c, "/api/v1/auth/login", body, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 TENANT_FIELD_FORBIDDEN, got %d body=%s", resp.StatusCode, respBody)
	}
	if !strings.Contains(respBody, "TENANT_FIELD_FORBIDDEN") {
		t.Fatalf("expected TENANT_FIELD_FORBIDDEN in body, got %s", respBody)
	}
}

// -------- helpers -----------------------------------------------------

func (f *fixture) postPatchJSON(c *http.Client, path string, body any, extraHeaders map[string]string) (*http.Response, string) {
	f.t.Helper()
	var rdr io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(http.MethodPatch, f.server.URL+path, rdr)
	if err != nil {
		f.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		f.t.Fatalf("patch %s: %v", path, err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, string(bodyBytes)
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	_, err := pool.Exec(ctx, sql, args...)
	if err != nil {
		t.Fatalf("exec %s: %v", sql, err)
	}
}


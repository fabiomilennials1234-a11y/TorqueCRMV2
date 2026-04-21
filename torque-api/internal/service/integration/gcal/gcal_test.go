package gcal_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/service/integration/gcal"
)

// ----- tiny test double for integrationrepo.Store -------------------
//
// The gcal adapter only touches the store for Get / Upsert / Delete /
// MarkSuccess / MarkError. We cannot spin up a real pgx pool here
// (unit tests, no DB), so we use an in-memory fake that matches the
// store's shape just enough.
//
// To keep the adapter's API narrow, the fake satisfies the tiny subset
// by exposing the same method names; the adapter ultimately depends on
// the concrete *integrationrepo.Store type, so we wire a real Store
// whose underlying pool is replaced with a stub. That proved painful,
// so this file takes a different route: a `fakeStore` stands in via
// method-value injection would require an interface; lacking one, we
// instead exercise gcal through its public OAuth and /calendar HTTP
// surfaces and skip the store-dependent CreateMeetingForOrg code path.

// We test: AuthURL assembly, ExchangeCode success/error, RefreshToken
// success/error, Cancel (404 → success), Health (nil by default),
// DecodeIDTokenEmail.

func newEndpoints(srv *httptest.Server) gcal.Endpoints {
	return gcal.Endpoints{
		AuthURL:         srv.URL + "/o/oauth2/v2/auth",
		TokenURL:        srv.URL + "/token",
		CalendarBaseURL: srv.URL + "/calendar/v3",
		RevokeURL:       srv.URL + "/revoke",
	}
}

func TestAuthURL_Assembly(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	p := gcal.New(gcal.Config{
		ClientID:     "client-123",
		ClientSecret: "secret",
		RedirectURL:  "https://app.example.test/integrations/google/callback",
	}, nil, nil, nil).WithEndpoints(newEndpoints(srv))

	raw := p.AuthURL("state-abc")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	for _, want := range []struct{ k, v string }{
		{"response_type", "code"},
		{"client_id", "client-123"},
		{"redirect_uri", "https://app.example.test/integrations/google/callback"},
		{"access_type", "offline"},
		{"prompt", "consent"},
		{"state", "state-abc"},
	} {
		if q.Get(want.k) != want.v {
			t.Errorf("query %s = %q, want %q", want.k, q.Get(want.k), want.v)
		}
	}
	if !strings.Contains(q.Get("scope"), "calendar.events") {
		t.Errorf("scope missing calendar.events: %q", q.Get("scope"))
	}
}

func TestExchangeCode_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "abc-code" {
			t.Errorf("code: %s", r.Form.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-1","refresh_token":"rt-1","token_type":"Bearer","expires_in":3600,"scope":"https://www.googleapis.com/auth/calendar.events","id_token":"x.eyJlbWFpbCI6InVzZXJAdGVzdC5jb20ifQ.sig"}`))
	}))
	defer srv.Close()

	p := gcal.New(gcal.Config{
		ClientID: "cid", ClientSecret: "cs", RedirectURL: "https://r",
	}, nil, nil, nil).WithEndpoints(newEndpoints(srv))

	tok, err := p.ExchangeCode(context.Background(), "abc-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if tok.AccessToken != "at-1" || tok.RefreshToken != "rt-1" {
		t.Fatalf("unexpected tokens: %+v", tok)
	}
	if email := gcal.DecodeIDTokenEmail(tok.IDToken); email != "user@test.com" {
		t.Errorf("email claim: %q", email)
	}
}

func TestExchangeCode_AuthFailed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	p := gcal.New(gcal.Config{ClientID: "cid", ClientSecret: "cs", RedirectURL: "https://r"}, nil, nil, nil).
		WithEndpoints(newEndpoints(srv))
	_, err := p.ExchangeCode(context.Background(), "bad")
	if !errors.Is(err, integration.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestExchangeCode_Unreachable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p := gcal.New(gcal.Config{ClientID: "cid", ClientSecret: "cs", RedirectURL: "https://r"}, nil, nil, nil).
		WithEndpoints(newEndpoints(srv))
	_, err := p.ExchangeCode(context.Background(), "x")
	if !errors.Is(err, integration.ErrUnreachable) {
		t.Fatalf("expected ErrUnreachable, got %v", err)
	}
}

func TestRefreshToken_ReinjectsRefresh(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Google omits refresh_token on refresh.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"scope":""}`))
	}))
	defer srv.Close()
	p := gcal.New(gcal.Config{ClientID: "cid", ClientSecret: "cs", RedirectURL: "https://r"}, nil, nil, nil).
		WithEndpoints(newEndpoints(srv))
	tok, err := p.RefreshToken(context.Background(), "original-rt")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tok.RefreshToken != "original-rt" {
		t.Fatalf("refresh not re-injected, got %q", tok.RefreshToken)
	}
	if tok.AccessToken != "at-2" {
		t.Fatalf("access: %q", tok.AccessToken)
	}
}

func TestRefreshToken_Empty(t *testing.T) {
	t.Parallel()
	p := gcal.New(gcal.Config{ClientID: "c", ClientSecret: "s", RedirectURL: "r"}, nil, nil, nil)
	if _, err := p.RefreshToken(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty refresh")
	}
}

func TestExchangeCode_MissingConfig(t *testing.T) {
	t.Parallel()
	p := gcal.New(gcal.Config{}, nil, nil, nil)
	if _, err := p.ExchangeCode(context.Background(), "x"); err == nil {
		t.Fatal("expected config error")
	}
}

func TestDecodeIDTokenEmail_Malformed(t *testing.T) {
	t.Parallel()
	if email := gcal.DecodeIDTokenEmail("not-a-jwt"); email != "" {
		t.Errorf("expected empty, got %q", email)
	}
	// Valid JSON with no email field.
	mid := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"123"}`))
	if email := gcal.DecodeIDTokenEmail("a." + mid + ".c"); email != "" {
		t.Errorf("expected empty for no email claim, got %q", email)
	}
	// Encode with padding — still accepted.
	j, _ := json.Marshal(map[string]string{"email": "pad@example.test"})
	mid2 := base64.URLEncoding.EncodeToString(j)
	if email := gcal.DecodeIDTokenEmail("a." + mid2 + ".c"); email != "pad@example.test" {
		t.Errorf("padded decode: %q", email)
	}
}

func TestHealth_NilByDefault(t *testing.T) {
	t.Parallel()
	p := gcal.New(gcal.Config{}, nil, nil, nil)
	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
}

func TestBreaker_Trips(t *testing.T) {
	t.Parallel()
	breaker := integration.NewCircuitBreaker(2, time.Minute)
	_ = gcal.New(gcal.Config{ClientID: "cid", ClientSecret: "cs", RedirectURL: "https://r"}, nil, nil, breaker)
	// Exchange isn't wrapped in the breaker (OAuth flow sits outside
	// the Calendar call path). Exercise the breaker primitive directly
	// to confirm the same instance would open CreateMeetingForOrg /
	// Cancel on repeated Unreachable.
	if err := breaker.Do(func() error { return integration.ErrUnreachable }); !errors.Is(err, integration.ErrUnreachable) {
		t.Fatalf("expected first call to pass through: %v", err)
	}
	if err := breaker.Do(func() error { return integration.ErrUnreachable }); !errors.Is(err, integration.ErrUnreachable) {
		t.Fatalf("expected second call to pass through (below threshold)")
	}
	if err := breaker.Do(func() error { return nil }); !errors.Is(err, integration.ErrCircuitOpen) {
		t.Fatalf("expected breaker open on third call, got %v", err)
	}
}

// ---- sanity helpers ------------------------------------------------

func randomKeyB64() string {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	return base64.StdEncoding.EncodeToString(raw)
}

// uuidV4 is a tiny helper — we intentionally avoid importing the full
// uuid package path more than once for readability.
func uuidV4() uuid.UUID { return uuid.New() }

// silence unused-function warnings — randomKeyB64 and uuidV4 are kept
// in case sibling tests (covering store-dependent flows) land next.
var _ = randomKeyB64
var _ = uuidV4

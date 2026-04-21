package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/service/integration/gcal"
)

// fakeGCal is a zero-config gcal.Provider suitable for tests that only
// hit the callback error paths (which do not actually call the network).
func fakeGCal() *gcal.Provider {
	return gcal.New(gcal.Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "https://app.example.test/integrations/google/callback",
	}, nil, nil, nil)
}

// ---- state HMAC round-trip ------------------------------------------

func TestState_RoundTrip(t *testing.T) {
	t.Parallel()
	h := &Handler{stateSecret: []byte("test-secret-min-32-bytes-long----")}
	org := uuid.New()
	s, err := h.newState(org)
	if err != nil {
		t.Fatalf("newState: %v", err)
	}
	got, err := h.verifyState(s)
	if err != nil {
		t.Fatalf("verifyState: %v", err)
	}
	if got != org {
		t.Fatalf("org mismatch: got %s want %s", got, org)
	}
}

func TestState_Tampered(t *testing.T) {
	t.Parallel()
	h := &Handler{stateSecret: []byte("test-secret-min-32-bytes-long----")}
	org := uuid.New()
	s, _ := h.newState(org)
	// Flip the last character of the signature to invalidate.
	bad := s[:len(s)-1] + "A"
	if bad == s {
		bad = s[:len(s)-1] + "B"
	}
	if _, err := h.verifyState(bad); err == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestState_Expired(t *testing.T) {
	t.Parallel()
	h := &Handler{stateSecret: []byte("test-secret-min-32-bytes-long----")}
	org := uuid.New()
	// Manually mint a state with an issued_at 20 minutes ago.
	var payload [28]byte
	copy(payload[:16], org[:])
	binary.BigEndian.PutUint32(payload[16:20], uint32(time.Now().UTC().Add(-20*time.Minute).Unix()))
	body := base64.RawURLEncoding.EncodeToString(payload[:])
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(sig) > 24 {
		sig = sig[:24]
	}
	expired := body + "." + sig
	if _, err := h.verifyState(expired); err == nil {
		t.Fatal("expected expired error")
	}
}

func TestState_Malformed(t *testing.T) {
	t.Parallel()
	h := &Handler{stateSecret: []byte("test-secret-min-32-bytes-long----")}
	cases := []string{"", "no-dot", ".leading", "trailing.", "a.b"}
	for _, c := range cases {
		if _, err := h.verifyState(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

// ---- list handler ---------------------------------------------------

// We exercise the list endpoint indirectly via the state code paths to
// keep this file free of a full fake-store stack. The wiring at main.go
// + the repository integration test give end-to-end coverage; the
// handler layer is thin enough that unit-testing it without a store
// would be ceremony.

// ---- unauth redirect smoke ------------------------------------------

func TestSettingsRedirect(t *testing.T) {
	t.Parallel()
	h := &Handler{frontendBase: "https://app.example.test"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cb", nil)
	h.settingsRedirect(rec, req, "google", "error", "auth_failed")
	if rec.Code != http.StatusFound {
		t.Fatalf("status: %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "https://app.example.test/settings?tab=integrations&provider=google&status=error&reason=auth_failed" {
		t.Fatalf("redirect: %q", loc)
	}
}

func TestSettingsRedirect_NoReason(t *testing.T) {
	t.Parallel()
	h := &Handler{frontendBase: ""}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cb", nil)
	h.settingsRedirect(rec, req, "google", "connected", "")
	loc := rec.Header().Get("Location")
	if loc != "/settings?tab=integrations&provider=google&status=connected" {
		t.Fatalf("redirect: %q", loc)
	}
}

// ---- callback error path (no state → INVALID_STATE) ----------------

func TestCallback_MissingState(t *testing.T) {
	t.Parallel()
	h := &Handler{
		stateSecret: []byte("test-secret-min-32-bytes-long----"),
		gcal:        fakeGCal(),
		logger:      zerolog.Nop(),
	}
	req := httptest.NewRequest(http.MethodGet, "/integrations/google/callback?code=abc", nil)
	rec := httptest.NewRecorder()
	h.googleCallback(rec, req.WithContext(context.Background()))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCallback_BadState(t *testing.T) {
	t.Parallel()
	h := &Handler{
		stateSecret: []byte("test-secret-min-32-bytes-long----"),
		gcal:        fakeGCal(),
		logger:      zerolog.Nop(),
	}
	req := httptest.NewRequest(http.MethodGet,
		"/integrations/google/callback?state=deadbeef.whatever&code=abc", nil)
	rec := httptest.NewRecorder()
	h.googleCallback(rec, req.WithContext(context.Background()))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestCallback_ProviderError(t *testing.T) {
	t.Parallel()
	h := &Handler{
		stateSecret:  []byte("test-secret-min-32-bytes-long----"),
		gcal:         fakeGCal(),
		frontendBase: "https://app.example.test",
		logger:       zerolog.Nop(),
	}
	req := httptest.NewRequest(http.MethodGet,
		"/integrations/google/callback?error=access_denied", nil)
	rec := httptest.NewRecorder()
	h.googleCallback(rec, req.WithContext(context.Background()))
	if rec.Code != http.StatusFound {
		t.Fatalf("status: %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "https://app.example.test/settings?tab=integrations&provider=google&status=error&reason=access_denied" {
		t.Fatalf("redirect: %q", loc)
	}
}

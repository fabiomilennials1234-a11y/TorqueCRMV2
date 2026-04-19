package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mw "github.com/milennials/torque-api/internal/httpx/middleware"
)

func nopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestSecurityHeaders_Defaults(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	mw.SecurityHeaders(nopHandler()).ServeHTTP(rec, req)

	cases := map[string]string{
		"Content-Security-Policy":     "default-src 'none'",
		"Strict-Transport-Security":   "max-age=",
		"X-Frame-Options":             "DENY",
		"X-Content-Type-Options":      "nosniff",
		"Referrer-Policy":             "strict-origin-when-cross-origin",
		"Permissions-Policy":          "camera=()",
		"Cross-Origin-Opener-Policy":  "same-origin",
		"Cross-Origin-Resource-Policy": "same-site",
	}
	for h, substr := range cases {
		if v := rec.Header().Get(h); !strings.Contains(v, substr) {
			t.Errorf("%s missing %q: got %q", h, substr, v)
		}
	}
}

func TestSecurityHeaders_HSTSDisabled(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	mw.SecurityHeadersWith(mw.SecurityHeadersConfig{
		EnableHSTS: false,
	})(nopHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if v := rec.Header().Get("Strict-Transport-Security"); v != "" {
		t.Fatalf("HSTS must be empty when disabled, got %q", v)
	}
	// Other headers still present.
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("X-Frame-Options missing when only HSTS disabled")
	}
}

func TestSecurityHeaders_HSTSPreloadShape(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	mw.SecurityHeadersWith(mw.SecurityHeadersConfig{
		EnableHSTS:  true,
		HSTSMaxAge:  365 * 24 * time.Hour,
		HSTSPreload: true,
	})(nopHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	v := rec.Header().Get("Strict-Transport-Security")
	for _, want := range []string{"max-age=", "includeSubDomains", "preload"} {
		if !strings.Contains(v, want) {
			t.Errorf("HSTS missing %q: %s", want, v)
		}
	}
}

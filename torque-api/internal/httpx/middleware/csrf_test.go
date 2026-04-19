package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mw "github.com/milennials/torque-api/internal/httpx/middleware"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestCSRF_SafeMethodsBypass(t *testing.T) {
	t.Parallel()
	h := mw.CSRF(okHandler())
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req := httptest.NewRequest(m, "/x", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", m, rec.Code)
		}
	}
}

func TestCSRF_MissingCookie(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	rec := httptest.NewRecorder()
	mw.CSRF(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "CSRF_MISSING_COOKIE") {
		t.Fatalf("expected CSRF_MISSING_COOKIE, got %s", rec.Body.String())
	}
}

func TestCSRF_MissingHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: mw.CSRFCookieName, Value: "abc"})
	rec := httptest.NewRecorder()
	mw.CSRF(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "CSRF_MISSING_HEADER") {
		t.Fatalf("expected CSRF_MISSING_HEADER, got %s", rec.Body.String())
	}
}

func TestCSRF_Mismatch(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: mw.CSRFCookieName, Value: "abc"})
	req.Header.Set(mw.CSRFHeaderName, "abd")
	rec := httptest.NewRecorder()
	mw.CSRF(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "CSRF_MISMATCH") {
		t.Fatalf("expected CSRF_MISMATCH, got %s", rec.Body.String())
	}
}

func TestCSRF_Match(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.AddCookie(&http.Cookie{Name: mw.CSRFCookieName, Value: "abc"})
	req.Header.Set(mw.CSRFHeaderName, "abc")
	rec := httptest.NewRecorder()
	mw.CSRF(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

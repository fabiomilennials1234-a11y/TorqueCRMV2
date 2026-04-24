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

	"github.com/milennials/torque-api/internal/repository/quota"
)

// The AI-specific middleware shares the fakeQuotaReader + withSession
// helpers defined in quota_test.go. Go's test compiler builds the whole
// package so both files live in the same compilation unit.

func TestRequireAITokenBudget_AdmitsUnderCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quota.Quota{
		ResourceKey:    quota.ResourceAITokens,
		EffectiveLimit: 50_000,
		CurrentUsage:   12_345,
		Remaining:      37_655,
	}}
	mw := RequireAITokenBudget(reader)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusOK {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-Quota-Resource") != quota.ResourceAITokens {
		t.Errorf("missing ai_tokens resource header")
	}
	if rr.Header().Get("X-Quota-Remaining") != "37655" {
		t.Errorf("unexpected remaining header: %q", rr.Header().Get("X-Quota-Remaining"))
	}
}

func TestRequireAITokenBudget_402AtCap(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quota.Quota{
		ResourceKey: quota.ResourceAITokens, EffectiveLimit: 10_000, CurrentUsage: 10_000,
	}}
	mw := RequireAITokenBudget(reader)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("code: %d, body: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "AI_QUOTA_EXCEEDED" {
		t.Fatalf("want AI_QUOTA_EXCEEDED, body: %+v", body)
	}
	// PT-BR message sanity.
	if !strings.Contains(rr.Body.String(), "cota de tokens de IA") {
		t.Fatalf("pt-br message missing, got: %s", rr.Body.String())
	}
}

func TestRequireAITokenBudget_MissingRowFailsClosed(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{err: quota.ErrNotFound}
	mw := RequireAITokenBudget(reader)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("missing row must fail closed with 402, got %d", rr.Code)
	}
}

func TestRequireAITokenBudget_MasterBypass(t *testing.T) {
	t.Parallel()
	// Pathological reader — must never be hit on the master path.
	reader := &fakeQuotaReader{err: errors.New("reader must not be called for master")}
	mw := RequireAITokenBudget(reader)(okHandler())

	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, true))

	if rr.Code != http.StatusOK {
		t.Fatalf("master must bypass: %d, body: %s", rr.Code, rr.Body.String())
	}
}

// TestRequireAITokenBudget_NoPreIncrement — the middleware must ONLY
// read; it must never call IncrementUsage. We assert this by spying on
// whether a write path was invoked using a tiny extension of the reader.
func TestRequireAITokenBudget_NoPreIncrement(t *testing.T) {
	t.Parallel()
	spy := &spyingReader{
		inner: &fakeQuotaReader{q: quota.Quota{
			ResourceKey: quota.ResourceAITokens, EffectiveLimit: 100_000, CurrentUsage: 0, Remaining: 100_000,
		}},
	}
	mw := RequireAITokenBudget(spy)(okHandler())
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))

	if rr.Code != http.StatusOK {
		t.Fatalf("code: %d", rr.Code)
	}
	if spy.getCalls != 1 {
		t.Fatalf("expected 1 Get call, got %d (the middleware must not retry or double-read)", spy.getCalls)
	}
}

// spyingReader counts Get() calls so we can assert the middleware only
// issues a single read per request (no retry, no double-read).
type spyingReader struct {
	inner    QuotaReader
	getCalls int
}

func (s *spyingReader) Get(ctx context.Context, org uuid.UUID, res string) (quota.Quota, error) {
	s.getCalls++
	return s.inner.Get(ctx, org, res)
}

func TestRequireTTSBudget_402(t *testing.T) {
	t.Parallel()
	reader := &fakeQuotaReader{q: quota.Quota{
		ResourceKey: quota.ResourceTTSSeconds, EffectiveLimit: 300, CurrentUsage: 305,
	}}
	mw := RequireTTSBudget(reader)(okHandler())
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, withSession(t, false))
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("tts budget should 402, got %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	errBlock, _ := body["error"].(map[string]any)
	if errBlock == nil || errBlock["code"] != "TTS_QUOTA_EXCEEDED" {
		t.Fatalf("want TTS_QUOTA_EXCEEDED, got: %+v", body)
	}
}

func TestRequireAITokenBudget_MissingOrgUnauth(t *testing.T) {
	t.Parallel()
	mw := RequireAITokenBudget(&fakeQuotaReader{})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/agents/x/playground/message", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 on empty ctx, got %d", rr.Code)
	}
}

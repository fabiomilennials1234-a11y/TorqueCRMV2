package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mw "github.com/milennials/torque-api/internal/httpx/middleware"
)

// drive a limiter until the first 429, and return how many requests made it through.
func drain(t *testing.T, rl *mw.RateLimiter, req *http.Request) (throughs int, last *httptest.ResponseRecorder) {
	t.Helper()
	h := rl.Middleware(nopHandler())
	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			return throughs, rec
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status %d at i=%d", rec.Code, i)
		}
		throughs++
	}
	return throughs, nil
}

func TestRateLimiter_BurstThenReject(t *testing.T) {
	t.Parallel()
	rl := mw.NewRateLimiter(mw.RateLimitConfig{
		AnonRPS: 1, AnonBurst: 3,
		UserRPS: 1, UserBurst: 3,
		IdleTTL: time.Minute,
	})
	defer rl.Close()

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	through, limited := drain(t, rl, req)
	if through != 3 {
		t.Fatalf("expected burst=3 through, got %d", through)
	}
	if limited == nil {
		t.Fatal("expected a 429 after burst exhausted")
	}
	ra := limited.Header().Get("Retry-After")
	if ra == "" {
		t.Fatal("Retry-After header missing on 429")
	}
	if !strings.Contains(limited.Body.String(), "RATE_LIMITED") {
		t.Fatalf("expected RATE_LIMITED code in body: %s", limited.Body.String())
	}
}

func TestRateLimiter_DistinctIPsHaveDistinctBuckets(t *testing.T) {
	t.Parallel()
	rl := mw.NewRateLimiter(mw.RateLimitConfig{
		AnonRPS: 1, AnonBurst: 2,
		UserRPS: 1, UserBurst: 2,
		IdleTTL: time.Minute,
	})
	defer rl.Close()

	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "10.0.0.1:1"
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "10.0.0.2:1"

	h := rl.Middleware(nopHandler())

	// Exhaust A.
	for i := 0; i < 2; i++ {
		h.ServeHTTP(httptest.NewRecorder(), reqA)
	}
	recA := httptest.NewRecorder()
	h.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusTooManyRequests {
		t.Fatalf("A expected 429 after burst, got %d", recA.Code)
	}
	// B untouched.
	recB := httptest.NewRecorder()
	h.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("B expected 200, got %d — distinct IPs must get distinct buckets", recB.Code)
	}
}

func TestRateLimiter_ForwardedForRespected(t *testing.T) {
	t.Parallel()
	rl := mw.NewRateLimiter(mw.RateLimitConfig{
		AnonRPS: 1, AnonBurst: 1,
		UserRPS: 1, UserBurst: 1,
		IdleTTL:           time.Minute,
		TrustForwardedFor: true,
	})
	defer rl.Close()

	// Two requests from the same proxy (same RemoteAddr) but different
	// X-Forwarded-For should bucket independently.
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "10.0.0.9:1"
	r1.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.9")
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "10.0.0.9:1"
	r2.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.9")

	h := rl.Middleware(nopHandler())
	for _, r := range []*http.Request{r1, r2} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusOK {
			t.Fatalf("first request per XFF must pass, got %d", rec.Code)
		}
	}
}

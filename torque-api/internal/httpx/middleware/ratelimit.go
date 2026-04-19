// Rate limiting middleware — in-memory token bucket per principal.
//
// The principal is:
//   * the authenticated user_id, when a session is attached to the request;
//   * otherwise the client IP (extracted from RemoteAddr, not from
//     X-Forwarded-For, unless cfg.TrustForwardedFor is set).
//
// Anonymous buckets get a smaller budget than authenticated ones. Bursts are
// allowed up to cfg.Burst tokens; sustained traffic drains at cfg.RPS
// requests-per-second.
//
// Rejection sets `429 Too Many Requests` with `Retry-After` (seconds until
// the bucket has at least one token again) and an error envelope.
//
// Memory hygiene: the bucket map grows with distinct principals. A background
// janitor evicts buckets untouched for cfg.IdleTTL. In a distributed setup
// this becomes a Redis token bucket — not an S03 deliverable.
package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig tunes the limiter. Defaults in NewRateLimiter.
type RateLimitConfig struct {
	// AnonRPS is the sustained request rate for anonymous callers.
	AnonRPS float64
	// AnonBurst allows short spikes for anonymous callers.
	AnonBurst int
	// UserRPS is the sustained request rate for authenticated users.
	UserRPS float64
	// UserBurst allows short spikes for authenticated users.
	UserBurst int
	// IdleTTL is how long a bucket survives without traffic before eviction.
	IdleTTL time.Duration
	// TrustForwardedFor takes the left-most X-Forwarded-For entry as the IP.
	// Enable ONLY when the binary sits behind a trusted reverse proxy.
	TrustForwardedFor bool
}

// DefaultRateLimitConfig returns sane starting values.
// Tune per traffic profile; do not guess in prod.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AnonRPS:   5,
		AnonBurst: 10,
		UserRPS:   30,
		UserBurst: 60,
		IdleTTL:   10 * time.Minute,
	}
}

// RateLimiter is the stateful middleware builder. It owns the bucket map and
// a janitor goroutine; create one instance per process and reuse it.
type RateLimiter struct {
	cfg     RateLimitConfig
	buckets sync.Map // key string → *bucket
	stop    chan struct{}
}

type bucket struct {
	lim  *rate.Limiter
	last atomic[time.Time]
}

// atomic is a trivial generic wrapper — go1.22 has no atomic.Pointer[time.Time]
// in std before 1.19, and we want both Load and Store without reflection.
type atomic[T any] struct {
	mu sync.RWMutex
	v  T
}

func (a *atomic[T]) Load() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.v
}

func (a *atomic[T]) Store(v T) {
	a.mu.Lock()
	a.v = v
	a.mu.Unlock()
}

// NewRateLimiter returns a limiter with the given config. A background janitor
// is started; call Close to stop it (tests, graceful shutdown).
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.AnonRPS <= 0 {
		cfg = DefaultRateLimitConfig()
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	rl := &RateLimiter{cfg: cfg, stop: make(chan struct{})}
	go rl.janitor()
	return rl
}

// Close stops the janitor. Safe to call multiple times.
func (rl *RateLimiter) Close() {
	defer func() { _ = recover() }()
	close(rl.stop)
}

func (rl *RateLimiter) janitor() {
	t := time.NewTicker(rl.cfg.IdleTTL)
	defer t.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case now := <-t.C:
			rl.buckets.Range(func(k, v any) bool {
				b := v.(*bucket)
				if now.Sub(b.last.Load()) > rl.cfg.IdleTTL {
					rl.buckets.Delete(k)
				}
				return true
			})
		}
	}
}

// Middleware returns the http.Handler wrapper. Compose BEFORE handlers but
// AFTER Authenticator so authenticated buckets are used where possible.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, rps, burst := rl.principalFor(r)

		b := rl.bucketFor(principal, rps, burst)
		if b.lim.Allow() {
			b.last.Store(time.Now())
			next.ServeHTTP(w, r)
			return
		}
		retryAfter := retryAfterSeconds(b.lim)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		writeError(w, http.StatusTooManyRequests, "RATE_LIMITED",
			fmt.Sprintf("too many requests — retry in %ds", retryAfter))
	})
}

func (rl *RateLimiter) principalFor(r *http.Request) (string, float64, int) {
	if sess, ok := SessionFrom(r.Context()); ok {
		return "u:" + sess.UserID.String(), rl.cfg.UserRPS, rl.cfg.UserBurst
	}
	return "ip:" + rl.clientIP(r), rl.cfg.AnonRPS, rl.cfg.AnonBurst
}

func (rl *RateLimiter) clientIP(r *http.Request) string {
	if rl.cfg.TrustForwardedFor {
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			if comma := strings.IndexByte(v, ','); comma > 0 {
				return strings.TrimSpace(v[:comma])
			}
			return strings.TrimSpace(v)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *RateLimiter) bucketFor(key string, rps float64, burst int) *bucket {
	if v, ok := rl.buckets.Load(key); ok {
		return v.(*bucket)
	}
	b := &bucket{lim: rate.NewLimiter(rate.Limit(rps), burst)}
	b.last.Store(time.Now())
	actual, _ := rl.buckets.LoadOrStore(key, b)
	return actual.(*bucket)
}

// retryAfterSeconds inspects the limiter to suggest a wait that is guaranteed
// to yield at least one token. Uses Reserve() and immediately cancels so the
// budget is not consumed.
func retryAfterSeconds(lim *rate.Limiter) int {
	r := lim.Reserve()
	delay := r.Delay()
	r.Cancel()
	if delay <= 0 {
		return 1
	}
	secs := int(math.Ceil(delay.Seconds()))
	if secs < 1 {
		return 1
	}
	return secs
}


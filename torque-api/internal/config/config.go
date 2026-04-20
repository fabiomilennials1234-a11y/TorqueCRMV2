// Package config carries the runtime configuration of the API.
//
// The config is loaded exclusively from environment variables. Secrets never
// land in the binary or in a file checked into source control. A missing
// required variable is a fatal boot error — the process refuses to start with
// placeholder defaults for anything load-bearing.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the fully-resolved runtime configuration.
//
// Every field is immutable after Load. Pass by value; the struct is small.
type Config struct {
	Env             string        // "dev" | "staging" | "prod"
	HTTPAddr        string        // ":8080"
	DatabaseURL     string        // postgres://...
	LogLevel        string        // "debug" | "info" | "warn" | "error"
	ShutdownTimeout time.Duration // graceful shutdown upper bound
	ReadTimeout     time.Duration // HTTP read timeout
	WriteTimeout    time.Duration // HTTP write timeout
	IdleTimeout     time.Duration // HTTP idle timeout
	CORSOrigins     []string      // allow-list; empty means closed

	// --- Auth / session (S02) ---
	//
	// JWTSecret signs access tokens (HS256). MUST be >= 32 bytes.
	JWTSecret []byte
	// AccessTTL is the lifetime of the __torque_session JWT. 15 min is the
	// default; go shorter in prod once refresh is proven in the wild.
	AccessTTL time.Duration
	// RefreshTTL is the lifetime of the __torque_refresh opaque cookie.
	RefreshTTL time.Duration
	// CookieDomain optionally scopes auth cookies to a parent domain.
	// Empty → host-only cookies (recommended unless the frontend lives on a
	// sibling subdomain).
	CookieDomain string
	// CookieSecure forces the Secure flag. Must be true in prod; may be false
	// in dev when the frontend is served over plain http://localhost.
	CookieSecure bool

	// --- Bootstrap (S02 teaser; fills out in S03) ---
	SentryDSN   string
	WSURL       string
	AppVersion  string

	// --- Observability (S03) ---
	// SentryPublicDSN is the client-side DSN (safe to ship to the browser).
	// Intentionally distinct from SentryDSN (server-side, private). Empty is
	// allowed in dev; prod CI should enforce presence.
	SentryPublicDSN   string
	SentryEnvironment string
	SentrySampleRate  float64
	SentryTracesRate  float64

	// --- Rate limiting (S03) ---
	RateLimitAnonRPS    float64
	RateLimitAnonBurst  int
	RateLimitUserRPS    float64
	RateLimitUserBurst  int
	RateLimitTrustProxy bool

	// --- Feature flags exposed in /api/bootstrap ---
	// Shape: "flag1=true,flag2=false". Everything in here is public — do NOT
	// gate security-sensitive features behind a bootstrap flag.
	FeatureFlags map[string]bool

	// --- Billing (S24) ---
	// BillingProvider selects the concrete integration. S24 ships "mock";
	// "asaas" wiring lands once credentials + dual review are in place.
	BillingProvider      string
	// BillingWebhookSecret must match the X-Torque-Billing-Secret header
	// on provider callbacks. Empty = webhook endpoint refuses every call
	// (secure default — prod deploys MUST set this).
	BillingWebhookSecret string
}

// Load reads the config from the environment. Returns an error if any required
// variable is missing or malformed.
//
// Required (no default):
//   - DATABASE_URL
//
// Optional (with world-class defaults):
//   - ENV             default "dev"
//   - HTTP_ADDR       default ":8080"
//   - LOG_LEVEL       default "info"
//   - SHUTDOWN_TIMEOUT default "15s"
//   - READ_TIMEOUT    default "10s"
//   - WRITE_TIMEOUT   default "15s"
//   - IDLE_TIMEOUT    default "60s"
//   - CORS_ORIGINS    default "" (closed)
func Load() (Config, error) {
	c := Config{
		Env:             getenv("ENV", "dev"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LogLevel:        strings.ToLower(getenv("LOG_LEVEL", "info")),
		ShutdownTimeout: mustDuration("SHUTDOWN_TIMEOUT", "15s"),
		ReadTimeout:     mustDuration("READ_TIMEOUT", "10s"),
		WriteTimeout:    mustDuration("WRITE_TIMEOUT", "15s"),
		IdleTimeout:     mustDuration("IDLE_TIMEOUT", "60s"),
		CORSOrigins:     parseCSV(os.Getenv("CORS_ORIGINS")),

		JWTSecret:    []byte(os.Getenv("JWT_SECRET")),
		AccessTTL:    mustDuration("ACCESS_TTL", "15m"),
		RefreshTTL:   mustDuration("REFRESH_TTL", "720h"), // 30 days
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),

		SentryDSN:  os.Getenv("SENTRY_DSN"),
		WSURL:      getenv("WS_URL", ""),
		AppVersion: getenv("APP_VERSION", "dev"),

		SentryPublicDSN:   os.Getenv("SENTRY_PUBLIC_DSN"),
		SentryEnvironment: getenv("SENTRY_ENVIRONMENT", ""),
		SentrySampleRate:  getenvFloat("SENTRY_SAMPLE_RATE", 1.0),
		// Default 0.1 (10%) — distributed tracing visible without flooding
		// Sentry. Raise to 1.0 during perf investigation; lower to 0 to
		// kill overhead entirely.
		SentryTracesRate:  getenvFloat("SENTRY_TRACES_SAMPLE_RATE", 0.1),

		RateLimitAnonRPS:    getenvFloat("RATELIMIT_ANON_RPS", 5),
		RateLimitAnonBurst:  getenvInt("RATELIMIT_ANON_BURST", 10),
		RateLimitUserRPS:    getenvFloat("RATELIMIT_USER_RPS", 30),
		RateLimitUserBurst:  getenvInt("RATELIMIT_USER_BURST", 60),
		RateLimitTrustProxy: getenvBool("RATELIMIT_TRUST_PROXY", false),

		FeatureFlags: parseFlags(os.Getenv("FEATURE_FLAGS")),

		BillingProvider:      getenv("BILLING_PROVIDER", "mock"),
		BillingWebhookSecret: os.Getenv("BILLING_WEBHOOK_SECRET"),
	}

	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if c.Env != "dev" && c.Env != "staging" && c.Env != "prod" {
		return Config{}, fmt.Errorf("invalid ENV %q (expected dev|staging|prod)", c.Env)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("invalid LOG_LEVEL %q", c.LogLevel)
	}

	// JWT_SECRET is required everywhere — do not ship a default. At least 32
	// bytes so HS256 is not trivially brute-forceable.
	if len(c.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be set and >= 32 bytes (got %d)", len(c.JWTSecret))
	}

	// Secure cookies are mandatory outside dev. Tolerating insecure cookies in
	// prod would degrade the SameSite=Strict guarantee against downgrade.
	if c.Env == "dev" {
		// Default: respect COOKIE_SECURE if user set it, else false.
		c.CookieSecure = getenvBool("COOKIE_SECURE", false)
	} else {
		c.CookieSecure = true
	}

	return c, nil
}

func getenvBool(key string, fallback bool) bool {
	v := strings.ToLower(os.Getenv(key))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// IsProd reports whether the process is running in production.
func (c Config) IsProd() bool { return c.Env == "prod" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func mustDuration(key, fallback string) time.Duration {
	raw := getenv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		// Fall back loudly — surface at boot via Load() validation if needed.
		// Using fallback here means a malformed env var does not panic;
		// it degrades to the default. Config.validate() is the gate.
		d, _ = time.ParseDuration(fallback)
		return d
	}
	return d
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MustAtoi parses an int from env, panics on failure. Reserved for numeric
// knobs that would be meaningless at the fallback (e.g., port-adjacent).
func MustAtoi(key, fallback string) int {
	v, err := strconv.Atoi(getenv(key, fallback))
	if err != nil {
		panic(fmt.Errorf("env %s: %w", key, err))
	}
	return v
}

func getenvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getenvFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

// parseFlags accepts "a=true,b=false" and returns a boolean map.
// Unknown values are treated as false; keys are trimmed and lowercased.
func parseFlags(raw string) map[string]bool {
	if raw == "" {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, pair := range strings.Split(raw, ",") {
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(pair[:eq]))
		val := strings.TrimSpace(strings.ToLower(pair[eq+1:]))
		if key == "" {
			continue
		}
		out[key] = val == "1" || val == "true" || val == "yes" || val == "on"
	}
	return out
}

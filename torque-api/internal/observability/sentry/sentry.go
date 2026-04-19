// Package sentry wires the Sentry Go SDK into the API, with PII scrubbing and
// a panic-recovery middleware.
//
// The Init function is safe to call with an empty DSN (it becomes a no-op —
// events are dropped). This lets dev environments run without Sentry without
// special-casing callers.
package sentry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sentrygo "github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// Config is the minimal surface the caller needs to hand us.
type Config struct {
	DSN         string
	Environment string // "dev" | "staging" | "prod"
	Release     string // app version
	// SampleRate must be in [0,1]. 1.0 captures every event; lower in high-volume prod.
	SampleRate float64
	// TracesSampleRate in [0,1]. Start at 0 (off) and raise deliberately.
	TracesSampleRate float64
}

// Init initializes the global SDK. Empty DSN → no-op (returns nil).
// The caller is expected to `defer sentry.Flush(timeout)` at shutdown.
func Init(cfg Config) error {
	if cfg.DSN == "" {
		return nil // explicit no-op
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 1.0
	}
	return sentrygo.Init(sentrygo.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
		BeforeSend:       scrubPII,
	})
}

// Flush drains pending events. Call from graceful-shutdown.
func Flush(timeout time.Duration) { sentrygo.Flush(timeout) }

// -------- PII scrubbing ----------------------------------------------

// sensitiveKeys lists request/body fields that MUST NEVER leave the server.
// The list is case-insensitive; both "email" and "Email" are redacted.
var sensitiveKeys = []string{
	"password", "token", "access_token", "refresh_token",
	"authorization", "cookie", "csrf", "x-csrf-token",
	"email", "phone", "cpf", "cnpj", "ssn",
	"api_key", "apikey", "secret", "jwt",
}

// scrubPII runs on every event before it hits the network. It redacts
// request headers, cookies, query strings, and body fields that match the
// sensitiveKeys list. Stacktraces and error messages are preserved.
//
// If an event is untrusted (hint reports a panic from a third-party lib), we
// still ship it — the alternative (dropping it) would hide incidents.
func scrubPII(event *sentrygo.Event, _ *sentrygo.EventHint) *sentrygo.Event {
	if event == nil {
		return nil
	}
	if event.Request != nil {
		event.Request.Cookies = "[Scrubbed]"
		event.Request.Headers = scrubMap(event.Request.Headers)
		event.Request.QueryString = scrubQueryString(event.Request.QueryString)
		// Do not ship bodies — we never need them for debugging prod auth flows.
		event.Request.Data = "[Scrubbed]"
	}
	if event.User.Email != "" {
		event.User.Email = "[redacted]"
	}
	if event.User.IPAddress != "" && event.User.IPAddress != "{{auto}}" {
		// Truncate IPv4 to /24, IPv6 to /48 — enough for grouping, not enough
		// to re-identify individual users.
		event.User.IPAddress = truncateIP(event.User.IPAddress)
	}
	// User.Username/ID are intentional fingerprints kept for debugging grouping.
	// Anything placed into extras is the author's responsibility.
	return event
}

func scrubMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if isSensitive(k) {
			out[k] = "[Scrubbed]"
			continue
		}
		out[k] = v
	}
	return out
}

func scrubQueryString(q string) string {
	if q == "" {
		return q
	}
	parts := strings.Split(q, "&")
	for i, p := range parts {
		eq := strings.IndexByte(p, '=')
		if eq < 0 {
			continue
		}
		if isSensitive(p[:eq]) {
			parts[i] = p[:eq] + "=[Scrubbed]"
		}
	}
	return strings.Join(parts, "&")
}

func isSensitive(key string) bool {
	lk := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

func truncateIP(ip string) string {
	if strings.Contains(ip, ":") {
		// IPv6 → keep first three groups (/48).
		parts := strings.Split(ip, ":")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], ":") + "::/48"
		}
		return ip
	}
	// IPv4 → keep first three octets (/24).
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + "." + parts[2] + ".0/24"
	}
	return ip
}

// -------- Recovery middleware ---------------------------------------

// Recovery wraps a handler with a panic catcher that reports to Sentry and
// continues to return 500 via the standard writeError envelope.
//
// This layer replaces the stdlib-only `Recover` middleware when Sentry is
// configured. The stdlib version still works without Sentry; see the
// middleware package for the fallback.
func Recovery(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					err := toError(rec)
					// Capture asynchronously; the default hub is safe for
					// concurrent use.
					if hub := sentrygo.CurrentHub().Clone(); hub != nil {
						hub.Scope().SetRequest(r)
						hub.CaptureException(err)
					}
					logger.Error().
						Err(err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("panic recovered")
					writeInternalError(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func toError(v any) error {
	switch x := v.(type) {
	case error:
		return x
	case string:
		return errors.New(x)
	default:
		return fmt.Errorf("%v", x)
	}
}

// writeInternalError keeps the error envelope shape consistent with the rest
// of the API without importing the full middleware package (cycle).
func writeInternalError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"code":"INTERNAL","message":"internal server error"}`))
}

// Package bootstrap serves GET /api/bootstrap.
//
// Bootstrap is the ONE endpoint the frontend hits before anything else. It
// returns runtime configuration that cannot be baked into the JS bundle —
// public Sentry DSN, WS URL, feature flags. Secrets never belong here.
//
// The endpoint is unauthenticated. The frontend calls it on cold start and
// caches the response with staleTime: Infinity.
package bootstrap

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/milennials/torque-api/internal/httpx"
)

// Config is the public runtime config surface. Every field is safe to ship
// to an unauthenticated client.
//
// Anything sensitive — JWT secret, DB credentials, server-only DSNs — MUST
// NOT be added here. A good sanity check: if you would paste this response
// into a public status page, every field should be safe. Otherwise do not
// include it.
type Config struct {
	AppVersion      string            `json:"app_version"`
	Env             string            `json:"env"`
	WSURL           string            `json:"ws_url"`
	SentryDSN       string            `json:"sentry_dsn"`       // public DSN only (client-side)
	FeatureFlags    map[string]bool   `json:"feature_flags"`
	Extras          map[string]string `json:"extras,omitempty"`
	// ServerTime lets the client detect clock drift on cold start. Formatted
	// as ISO-8601 UTC (Z suffix) per the wire convention (Multi-tenancy.md).
	ServerTime string `json:"server_time"`
}

// Handler serves the bootstrap endpoint.
type Handler struct {
	cfg Config
}

// New captures the current Config. Call after env has been validated; the
// struct is snapshot and not re-read per request.
func New(cfg Config) *Handler { return &Handler{cfg: cfg} }

// Routes mounts GET /api/bootstrap.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/api/bootstrap", h.get)
}

func (h *Handler) get(w http.ResponseWriter, _ *http.Request) {
	// Aggressive public cache: config is stable; client invalidates on deploy.
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=600")
	// Snapshot server_time per request so the client can detect clock drift.
	resp := h.cfg
	resp.ServerTime = time.Now().UTC().Format(time.RFC3339)
	if resp.FeatureFlags == nil {
		resp.FeatureFlags = map[string]bool{}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

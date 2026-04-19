// Package health exposes liveness and readiness probes.
//
//   - GET /healthz  — liveness. Returns 200 as long as the process is up. No
//     dependency checks; this is what orchestrators poll to decide whether to
//     restart the pod.
//   - GET /readyz   — readiness. Returns 200 only if the DB is reachable.
//     Orchestrators use this to decide whether to route traffic.
//
// The split matters: a pod that cannot reach Postgres should stop receiving
// traffic (not-ready) without being restarted (still-alive), because the
// problem is not local.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/milennials/torque-api/internal/db"
)

// Handler bundles the probe handlers.
type Handler struct {
	pool    *db.Pool
	version string
}

// New returns a Handler. Version is surfaced in the response body to aid
// debugging when rolling deployments expose multiple revisions.
func New(pool *db.Pool, version string) *Handler {
	return &Handler{pool: pool, version: version}
}

// Routes mounts the probe routes on the router.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/healthz", h.live)
	r.Get("/readyz", h.ready)
}

type liveResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type readyResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	DB      string `json:"db"`
}

func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, liveResponse{Status: "ok", Version: h.version})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp := readyResponse{Status: "ok", Version: h.version, DB: "ok"}
	if err := h.pool.Ping(ctx); err != nil {
		resp.Status = "degraded"
		resp.DB = "unreachable"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

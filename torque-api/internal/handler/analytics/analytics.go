// Package analytics serves F09 /api/v1/analytics/* endpoints.
//
//   GET  /analytics/leads                 — leads summary (window required)
//   GET  /analytics/messages              — inbound/outbound counts
//   GET  /analytics/tasks                 — task state + overdue
//   GET  /analytics/proposals             — proposal funnel + won amount
//   GET  /analytics/pipes/:id/stages      — Kanban stage volume
//   GET  /analytics/leaderboard           — per-member performance in window
//
// Member-accessible (read-only). The `since`/`until` query params are in
// ISO-8601; `until` defaults to now() when absent but `since` is always
// required — the repo refuses windowed queries without an explicit lower
// bound to prevent accidental full-table scans.
package analytics

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	analyticsrepo "github.com/milennials/torque-api/internal/repository/analytics"
)

type Handler struct {
	repo *analyticsrepo.Repository
}

func New(repo *analyticsrepo.Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/analytics/leads", h.leads)
	r.Get("/analytics/messages", h.messages)
	r.Get("/analytics/tasks", h.tasks)
	r.Get("/analytics/proposals", h.proposals)
	r.Get("/analytics/pipes/{id}/stages", h.stageVolume)
	r.Get("/analytics/leaderboard", h.leaderboard)
}

// -------- helpers ---------------------------------------------------

// parseWindow reads `since` and `until` from the query string. Both are
// ISO-8601. Returns 400 if `since` is missing or malformed.
func parseWindow(w http.ResponseWriter, r *http.Request) (analyticsrepo.Window, bool) {
	q := r.URL.Query()
	sinceRaw := q.Get("since")
	if sinceRaw == "" {
		httpx.WriteError(w, http.StatusBadRequest, "MISSING_WINDOW",
			"since query param is required (ISO-8601)")
		return analyticsrepo.Window{}, false
	}
	since, err := time.Parse(time.RFC3339, sinceRaw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW", "since must be RFC3339")
		return analyticsrepo.Window{}, false
	}
	win := analyticsrepo.Window{Since: since}
	if untilRaw := q.Get("until"); untilRaw != "" {
		until, err := time.Parse(time.RFC3339, untilRaw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW", "until must be RFC3339")
			return analyticsrepo.Window{}, false
		}
		win.Until = until
	}
	return win, true
}

func writeWindowErr(w http.ResponseWriter, err error) bool {
	if errors.Is(err, analyticsrepo.ErrInvalidWindow) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW",
			"until must be strictly greater than since")
		return true
	}
	return false
}

// -------- handlers --------------------------------------------------

func (h *Handler) leads(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	win, ok := parseWindow(w, r)
	if !ok {
		return
	}
	s, err := h.repo.LeadsSummary(r.Context(), orgID, win)
	if writeWindowErr(w, err) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	win, ok := parseWindow(w, r)
	if !ok {
		return
	}
	s, err := h.repo.MessagesSummary(r.Context(), orgID, win)
	if writeWindowErr(w, err) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) tasks(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	win, ok := parseWindow(w, r)
	if !ok {
		return
	}
	s, err := h.repo.TasksSummary(r.Context(), orgID, win)
	if writeWindowErr(w, err) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) proposals(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	win, ok := parseWindow(w, r)
	if !ok {
		return
	}
	s, err := h.repo.ProposalsSummary(r.Context(), orgID, win)
	if writeWindowErr(w, err) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) stageVolume(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	pipeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "pipe id must be a uuid")
		return
	}
	data, err := h.repo.StageVolumeByPipe(r.Context(), orgID, pipeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *Handler) leaderboard(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	win, ok := parseWindow(w, r)
	if !ok {
		return
	}
	data, err := h.repo.Leaderboard(r.Context(), orgID, win)
	if writeWindowErr(w, err) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "analytics failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

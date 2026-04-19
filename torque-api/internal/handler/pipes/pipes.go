// Package pipes serves the /api/v1/pipes endpoints (F01).
//
//   GET  /pipes                          — list all pipes
//   GET  /pipes/:id/stages               — stages of one pipe
//   GET  /pipes/:id/entries              — active entries of one pipe
//   POST /pipes/:id/entries/move         — transition a lead across stages
package pipes

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
	"github.com/milennials/torque-api/internal/ws"
)

// Handler groups the pipe endpoints.
type Handler struct {
	repo *piperepo.Repository
	bus  *event.Bus
}

// New returns a new handler.
func New(repo *piperepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes mounts the endpoints on a tenant-scoped subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/pipes", h.list)
	r.Get("/pipes/{id}/stages", h.stages)
	r.Get("/pipes/{id}/entries", h.entries)
	r.Post("/pipes/{id}/entries/move", h.move)
}

type pipeView struct {
	ID         uuid.UUID `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	IsDefault  bool      `json:"is_default"`
	IsArchived bool      `json:"is_archived"`
	Position   int       `json:"position"`
}

type stageView struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	ColorToken      *string   `json:"color_token,omitempty"`
	Position        int       `json:"position"`
	IsFinalPositive bool      `json:"is_final_positive"`
	IsFinalNegative bool      `json:"is_final_negative"`
}

type entryView struct {
	ID              uuid.UUID `json:"id"`
	StageID         uuid.UUID `json:"stage_id"`
	LeadID          uuid.UUID `json:"lead_id"`
	EnteredStageAt  string    `json:"entered_stage_at"`
}

type moveRequest struct {
	LeadID     uuid.UUID `json:"lead_id"`
	NewStageID uuid.UUID `json:"new_stage_id"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	ps, err := h.repo.ListPipes(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list pipes")
		return
	}
	out := make([]pipeView, len(ps))
	for i, p := range ps {
		out[i] = pipeView{ID: p.ID, Kind: p.Kind, Name: p.Name, IsDefault: p.IsDefault, IsArchived: p.IsArchived, Position: p.Position}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) stages(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	pipeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "pipe id must be a uuid")
		return
	}
	ss, err := h.repo.Stages(r.Context(), orgID, pipeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load stages")
		return
	}
	out := make([]stageView, len(ss))
	for i, s := range ss {
		out[i] = stageView{ID: s.ID, Name: s.Name, ColorToken: s.ColorToken, Position: s.Position, IsFinalPositive: s.IsFinalPositive, IsFinalNegative: s.IsFinalNegative}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) entries(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	pipeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "pipe id must be a uuid")
		return
	}
	es, err := h.repo.Entries(r.Context(), orgID, pipeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load entries")
		return
	}
	out := make([]entryView, len(es))
	for i, e := range es {
		out[i] = entryView{ID: e.ID, StageID: e.StageID, LeadID: e.LeadID, EnteredStageAt: e.EnteredStageAt.UTC().Format(time.RFC3339)}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	pipeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "pipe id must be a uuid")
		return
	}
	var body moveRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	entry, err := h.repo.Move(r.Context(), orgID, pipeID, body.LeadID, body.NewStageID)
	if errors.Is(err, piperepo.ErrStageMismatch) {
		httpx.WriteError(w, http.StatusConflict, "STAGE_MISMATCH", "stage does not belong to this pipe")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not move lead")
		return
	}

	patch, _ := json.Marshal(map[string]any{
		"entry_id":   entry.ID,
		"pipe_id":    entry.PipeID,
		"stage_id":   entry.StageID,
		"lead_id":    entry.LeadID,
	})
	h.bus.Publish(ws.Event{
		Type: "pipe_entry.moved", TenantID: orgID, EntityType: "pipe_entry",
		EntityID: &entry.ID, Patch: json.RawMessage(patch), OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusOK, entryView{ID: entry.ID, StageID: entry.StageID, LeadID: entry.LeadID, EnteredStageAt: entry.EnteredStageAt.UTC().Format(time.RFC3339)})
	_ = domain.PipeEntry{} // retention — ensures import stays valid across refactors
}

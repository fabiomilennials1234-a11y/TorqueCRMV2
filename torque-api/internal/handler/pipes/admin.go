package pipes

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	piperepo "github.com/milennials/torque-api/internal/repository/pipe"
	"github.com/milennials/torque-api/internal/ws"
)

// AdminHandler serves admin-only pipe/stage management. Mounted under the
// RequireRole(admin) subgroup in main.go.
type AdminHandler struct {
	repo *piperepo.Repository
	bus  *event.Bus
}

// NewAdmin binds the admin handler.
func NewAdmin(repo *piperepo.Repository, bus *event.Bus) *AdminHandler {
	return &AdminHandler{repo: repo, bus: bus}
}

// Routes mounts endpoints; all mutations.
func (h *AdminHandler) Routes(r chi.Router) {
	r.Post("/pipes", h.createPipe)
	r.Patch("/pipes/{id}", h.updatePipe)
	r.Delete("/pipes/{id}", h.archivePipe)
	r.Post("/pipes/{id}/stages", h.createStage)
	r.Patch("/pipes/{id}/stages/{stageId}", h.updateStage)
	r.Delete("/pipes/{id}/stages/{stageId}", h.deleteStage)
}

type createPipeReq struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Position  int    `json:"position"`
}

type updatePipeReq struct {
	Name      *string `json:"name,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
	Position  *int    `json:"position,omitempty"`
}

type createStageReq struct {
	Name            string  `json:"name"`
	ColorToken      *string `json:"color_token,omitempty"`
	Position        int     `json:"position"`
	IsFinalPositive bool    `json:"is_final_positive"`
	IsFinalNegative bool    `json:"is_final_negative"`
}

type updateStageReq struct {
	Name            *string `json:"name,omitempty"`
	ColorToken      *string `json:"color_token,omitempty"`
	Position        *int    `json:"position,omitempty"`
	IsFinalPositive *bool   `json:"is_final_positive,omitempty"`
	IsFinalNegative *bool   `json:"is_final_negative,omitempty"`
}

type adminPipeView struct {
	ID         uuid.UUID `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	IsDefault  bool      `json:"is_default"`
	IsArchived bool      `json:"is_archived"`
	Position   int       `json:"position"`
}

type adminStageView struct {
	ID              uuid.UUID `json:"id"`
	PipeID          uuid.UUID `json:"pipe_id"`
	Name            string    `json:"name"`
	ColorToken      *string   `json:"color_token,omitempty"`
	Position        int       `json:"position"`
	IsFinalPositive bool      `json:"is_final_positive"`
	IsFinalNegative bool      `json:"is_final_negative"`
}

func (h *AdminHandler) createPipe(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createPipeReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	p, err := h.repo.CreatePipe(r.Context(), orgID, piperepo.CreatePipeInput{
		Kind: body.Kind, Name: body.Name, IsDefault: body.IsDefault, Position: body.Position,
	})
	if errors.Is(err, piperepo.ErrInvalidKind) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_KIND", err.Error())
		return
	}
	if errors.Is(err, piperepo.ErrNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PIPE", err.Error())
		return
	}
	view := adminPipeView{
		ID: p.ID, Kind: p.Kind, Name: p.Name,
		IsDefault: p.IsDefault, IsArchived: p.IsArchived, Position: p.Position,
	}
	h.publish(orgID, p.ID, "pipe.created", view)
	httpx.WriteJSON(w, http.StatusCreated, view)
}

func (h *AdminHandler) updatePipe(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var body updatePipeReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	p, err := h.repo.UpdatePipe(r.Context(), orgID, id, piperepo.UpdatePipeInput{
		Name: body.Name, IsDefault: body.IsDefault, Position: body.Position,
	})
	if errors.Is(err, piperepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "pipe not found")
		return
	}
	if errors.Is(err, piperepo.ErrNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PIPE", err.Error())
		return
	}
	view := adminPipeView{
		ID: p.ID, Kind: p.Kind, Name: p.Name,
		IsDefault: p.IsDefault, IsArchived: p.IsArchived, Position: p.Position,
	}
	h.publish(orgID, p.ID, "pipe.updated", view)
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (h *AdminHandler) archivePipe(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.repo.ArchivePipe(r.Context(), orgID, id); err != nil {
		if errors.Is(err, piperepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "pipe not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not archive pipe")
		return
	}
	h.publish(orgID, id, "pipe.archived", map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) createStage(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	pipeID, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	var body createStageReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	s, err := h.repo.CreateStage(r.Context(), orgID, pipeID, piperepo.CreateStageInput{
		Name: body.Name, ColorToken: body.ColorToken, Position: body.Position,
		IsFinalPositive: body.IsFinalPositive, IsFinalNegative: body.IsFinalNegative,
	})
	if errors.Is(err, piperepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "pipe not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STAGE", err.Error())
		return
	}
	view := adminStageView{
		ID: s.ID, PipeID: s.PipeID, Name: s.Name, ColorToken: s.ColorToken,
		Position: s.Position, IsFinalPositive: s.IsFinalPositive, IsFinalNegative: s.IsFinalNegative,
	}
	h.publish(orgID, s.ID, "pipe_stage.created", view)
	httpx.WriteJSON(w, http.StatusCreated, view)
}

func (h *AdminHandler) updateStage(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	stageID, ok := parseUUID(w, r, "stageId")
	if !ok {
		return
	}
	var body updateStageReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	s, err := h.repo.UpdateStage(r.Context(), orgID, stageID, piperepo.UpdateStageInput{
		Name: body.Name, ColorToken: body.ColorToken, Position: body.Position,
		IsFinalPositive: body.IsFinalPositive, IsFinalNegative: body.IsFinalNegative,
	})
	if errors.Is(err, piperepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "stage not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STAGE", err.Error())
		return
	}
	view := adminStageView{
		ID: s.ID, PipeID: s.PipeID, Name: s.Name, ColorToken: s.ColorToken,
		Position: s.Position, IsFinalPositive: s.IsFinalPositive, IsFinalNegative: s.IsFinalNegative,
	}
	h.publish(orgID, s.ID, "pipe_stage.updated", view)
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (h *AdminHandler) deleteStage(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	stageID, ok := parseUUID(w, r, "stageId")
	if !ok {
		return
	}
	if err := h.repo.DeleteStage(r.Context(), orgID, stageID); err != nil {
		if errors.Is(err, piperepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "stage not found")
			return
		}
		if errors.Is(err, piperepo.ErrStageHasEntries) {
			httpx.WriteError(w, http.StatusConflict, "STAGE_HAS_ENTRIES", "stage still has active leads")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete stage")
		return
	}
	h.publish(orgID, stageID, "pipe_stage.deleted", map[string]any{"id": stageID})
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers ---------------------------------------------------

func parseUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", name+" must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func decodeErr(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 1 MiB")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}

func (h *AdminHandler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "pipe",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

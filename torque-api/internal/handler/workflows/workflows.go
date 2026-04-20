// Package workflows serves F07 /api/v1/workflows endpoints.
//
//   GET    /workflows                         — list
//   POST   /workflows                         — create (draft)
//   GET    /workflows/:id                     — detail
//   POST   /workflows/:id/publish             — draft|paused → active
//   POST   /workflows/:id/pause               — active → paused
//   POST   /workflows/:id/archive             — * → archived
//
//   GET    /workflows/:id/steps               — list steps
//   POST   /workflows/:id/steps               — create step
//   PUT    /workflows/:id/steps/:sid          — update step
//   DELETE /workflows/:id/steps/:sid          — delete step
//   POST   /workflows/:id/entry/:sid          — set entry step
//
//   POST   /workflows/:id/runs                — enqueue manual run
//   GET    /workflows/:id/runs                — list recent runs
//   POST   /runs/:id/cancel                   — cancel pending/running run
//
// Admin-only subrouter; member-facing view comes with `workflows.view` key
// gating in a future sprint.
package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *workflowrepo.Repository
	bus  *event.Bus
}

func New(repo *workflowrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workflows", h.list)
	r.Post("/workflows", h.create)
	r.Get("/workflows/{id}", h.get)
	r.Post("/workflows/{id}/publish", h.publish)
	r.Post("/workflows/{id}/pause", h.pause)
	r.Post("/workflows/{id}/archive", h.archive)

	r.Get("/workflows/{id}/steps", h.listSteps)
	r.Post("/workflows/{id}/steps", h.createStep)
	r.Put("/workflows/{id}/steps/{sid}", h.updateStep)
	r.Delete("/workflows/{id}/steps/{sid}", h.deleteStep)
	r.Post("/workflows/{id}/entry/{sid}", h.setEntry)

	r.Post("/workflows/{id}/runs", h.enqueueRun)
	r.Get("/workflows/{id}/runs", h.listRuns)
	r.Post("/runs/{id}/cancel", h.cancelRun)
}

// -------- DTOs -------------------------------------------------------

type createReq struct {
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Trigger       string          `json:"trigger"`
	TriggerConfig json.RawMessage `json:"trigger_config,omitempty"`
}
type stepReq struct {
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Config      json.RawMessage `json:"config,omitempty"`
	NextStepIDs []uuid.UUID     `json:"next_step_ids,omitempty"`
	PositionX   *int            `json:"position_x,omitempty"`
	PositionY   *int            `json:"position_y,omitempty"`
}
type runReq struct {
	LeadID        *uuid.UUID      `json:"lead_id,omitempty"`
	TriggerSource string          `json:"trigger_source,omitempty"`
	Input         json.RawMessage `json:"input,omitempty"`
}

type workflowView struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	Trigger       string     `json:"trigger"`
	Status        string     `json:"status"`
	EntryStepID   *uuid.UUID `json:"entry_step_id,omitempty"`
}
type stepView struct {
	ID          uuid.UUID   `json:"id"`
	Kind        string      `json:"kind"`
	Name        string      `json:"name"`
	Config      json.RawMessage `json:"config"`
	NextStepIDs []uuid.UUID `json:"next_step_ids"`
	PositionX   *int        `json:"position_x,omitempty"`
	PositionY   *int        `json:"position_y,omitempty"`
}
type runView struct {
	ID            uuid.UUID  `json:"id"`
	WorkflowID    uuid.UUID  `json:"workflow_id"`
	LeadID        *uuid.UUID `json:"lead_id,omitempty"`
	TriggerSource string     `json:"trigger_source"`
	Status        string     `json:"status"`
	CurrentStepID *uuid.UUID `json:"current_step_id,omitempty"`
	CreatedAt     string     `json:"created_at"`
}

// -------- workflow handlers -----------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.List(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list")
		return
	}
	out := make([]workflowView, len(list))
	for i, v := range list {
		out[i] = toWorkflowView(v)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	creator := sess.TeamMemberID
	wf, err := h.repo.Create(r.Context(), workflowrepo.CreateInput{
		OrganizationID: orgID, Name: body.Name, Description: body.Description,
		Trigger: body.Trigger, TriggerConfig: body.TriggerConfig, CreatedBy: &creator,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WORKFLOW", err.Error())
		return
	}
	h.publishEvent(orgID, wf.ID, "workflow.created", toWorkflowView(wf))
	httpx.WriteJSON(w, http.StatusCreated, toWorkflowView(wf))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	wf, err := h.repo.Get(r.Context(), orgID, id)
	if handleNotFound(w, err, workflowrepo.ErrNotFound) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toWorkflowView(wf))
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "active", "workflow.published")
}
func (h *Handler) pause(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "paused", "workflow.paused")
}
func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "archived", "workflow.archived")
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status, evt string) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	// Publish requires entry_step_id to be set — check before flipping.
	if status == "active" {
		wf, err := h.repo.Get(r.Context(), orgID, id)
		if handleNotFound(w, err, workflowrepo.ErrNotFound) {
			return
		}
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load")
			return
		}
		if wf.EntryStepID == nil {
			httpx.WriteError(w, http.StatusConflict, "NO_ENTRY_STEP", "set an entry step before publishing")
			return
		}
	}
	if err := h.repo.SetStatus(r.Context(), orgID, id, status); err != nil {
		if errors.Is(err, workflowrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not set status")
		return
	}
	h.publishEvent(orgID, id, evt, map[string]string{"status": status})
	w.WriteHeader(http.StatusNoContent)
}

// -------- step handlers ----------------------------------------------

func (h *Handler) listSteps(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	list, err := h.repo.ListSteps(r.Context(), orgID, wfID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list steps")
		return
	}
	out := make([]stepView, len(list))
	for i, s := range list {
		out[i] = toStepView(s)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) createStep(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body stepReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	s, err := h.repo.UpsertStep(r.Context(), workflowrepo.UpsertStepInput{
		OrganizationID: orgID, WorkflowID: wfID,
		Kind: body.Kind, Name: body.Name, Config: body.Config,
		NextStepIDs: body.NextStepIDs, PositionX: body.PositionX, PositionY: body.PositionY,
	})
	if handleNotFound(w, err, workflowrepo.ErrNotFound) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STEP", err.Error())
		return
	}
	h.publishEvent(orgID, s.ID, "workflow_step.upserted", toStepView(s))
	httpx.WriteJSON(w, http.StatusCreated, toStepView(s))
}

func (h *Handler) updateStep(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	sid, ok := parseID(w, r, "sid")
	if !ok {
		return
	}
	var body stepReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	s, err := h.repo.UpsertStep(r.Context(), workflowrepo.UpsertStepInput{
		OrganizationID: orgID, WorkflowID: wfID, ID: &sid,
		Kind: body.Kind, Name: body.Name, Config: body.Config,
		NextStepIDs: body.NextStepIDs, PositionX: body.PositionX, PositionY: body.PositionY,
	})
	if handleNotFound(w, err, workflowrepo.ErrNotFound) {
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STEP", err.Error())
		return
	}
	h.publishEvent(orgID, s.ID, "workflow_step.upserted", toStepView(s))
	httpx.WriteJSON(w, http.StatusOK, toStepView(s))
}

func (h *Handler) deleteStep(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	sid, ok := parseID(w, r, "sid")
	if !ok {
		return
	}
	if err := h.repo.DeleteStep(r.Context(), orgID, wfID, sid); err != nil {
		if errors.Is(err, workflowrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "step not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete step")
		return
	}
	h.publishEvent(orgID, sid, "workflow_step.deleted", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setEntry(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	sid, ok := parseID(w, r, "sid")
	if !ok {
		return
	}
	if err := h.repo.SetEntry(r.Context(), orgID, wfID, sid); err != nil {
		if errors.Is(err, workflowrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow or step not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not set entry")
		return
	}
	h.publishEvent(orgID, wfID, "workflow.entry_set", map[string]uuid.UUID{"entry_step_id": sid})
	w.WriteHeader(http.StatusNoContent)
}

// -------- run handlers -----------------------------------------------

func (h *Handler) enqueueRun(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body runReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	trig := sess.TeamMemberID
	run, err := h.repo.EnqueueRun(r.Context(), workflowrepo.EnqueueRunInput{
		OrganizationID: orgID, WorkflowID: wfID, LeadID: body.LeadID,
		TriggeredBy: &trig, TriggerSource: body.TriggerSource, Input: body.Input,
	})
	if handleNotFound(w, err, workflowrepo.ErrNotFound) {
		return
	}
	if errors.Is(err, workflowrepo.ErrInvalidState) {
		httpx.WriteError(w, http.StatusConflict, "WORKFLOW_NOT_ACTIVE",
			"workflow must be active with an entry_step_id to run")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not enqueue run")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "workflow_run.enqueued", TenantID: orgID, EntityType: "workflow_run",
		EntityID: &run.ID, Patch: toRunView(run), OccurredAt: time.Now().UTC(),
	})
	w.Header().Set("Location", "/api/v1/runs/"+run.ID.String())
	httpx.WriteJSON(w, http.StatusAccepted, toRunView(run))
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	wfID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	list, err := h.repo.ListRuns(r.Context(), orgID, wfID, 50)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list runs")
		return
	}
	out := make([]runView, len(list))
	for i, rn := range list {
		out[i] = toRunView(rn)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) cancelRun(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.repo.CancelRun(r.Context(), orgID, id); err != nil {
		if errors.Is(err, workflowrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusConflict, "NOT_CANCELLABLE",
				"run is terminal or not in tenant")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not cancel run")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "workflow_run.cancelled", TenantID: orgID, EntityType: "workflow_run",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers ----------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", key+" must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func handleNotFound(w http.ResponseWriter, err, sentinel error) bool {
	if errors.Is(err, sentinel) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "not found")
		return true
	}
	return false
}

func decodeError(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "request body exceeds 1 MiB")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}

func (h *Handler) publishEvent(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "workflow",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func toWorkflowView(w workflowrepo.Workflow) workflowView {
	return workflowView{
		ID: w.ID, Name: w.Name, Description: w.Description,
		Trigger: w.Trigger, Status: w.Status, EntryStepID: w.EntryStepID,
	}
}
func toStepView(s workflowrepo.Step) stepView {
	cfg := json.RawMessage(s.Config)
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return stepView{
		ID: s.ID, Kind: s.Kind, Name: s.Name, Config: cfg,
		NextStepIDs: s.NextStepIDs, PositionX: s.PositionX, PositionY: s.PositionY,
	}
}
func toRunView(rn workflowrepo.Run) runView {
	return runView{
		ID: rn.ID, WorkflowID: rn.WorkflowID, LeadID: rn.LeadID,
		TriggerSource: rn.TriggerSource, Status: rn.Status, CurrentStepID: rn.CurrentStepID,
		CreatedAt: rn.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// context import is used via r.Context() in several helpers above; the
// package-level alias keeps the file self-contained if those helpers get
// extracted later.
var _ = context.Canceled

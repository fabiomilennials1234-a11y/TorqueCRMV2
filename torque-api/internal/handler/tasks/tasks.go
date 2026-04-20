// Package tasks serves F05 /api/v1/tasks (ADR-007 unified Task entity).
package tasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	taskrepo "github.com/milennials/torque-api/internal/repository/task"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *taskrepo.Repository
	bus  *event.Bus
}

func New(repo *taskrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/tasks", h.list)
	r.Post("/tasks", h.create)
	r.Get("/tasks/{id}", h.get)
	r.Post("/tasks/{id}/start", h.start)
	r.Post("/tasks/{id}/complete", h.complete)
	r.Post("/tasks/{id}/cancel", h.cancel)
	r.Post("/tasks/{id}/miss", h.miss)
}

type view struct {
	ID          uuid.UUID   `json:"id"`
	LeadID      *uuid.UUID  `json:"lead_id,omitempty"`
	AssignedTo  uuid.UUID   `json:"assigned_to"`
	Kind        string      `json:"kind"`
	Title       string      `json:"title"`
	Description *string     `json:"description,omitempty"`
	Priority    string      `json:"priority"`
	Status      string      `json:"status"`
	DueAt       *string     `json:"due_at,omitempty"`
	StartedAt   *string     `json:"started_at,omitempty"`
	CompletedAt *string     `json:"completed_at,omitempty"`
	Origin      string      `json:"origin"`
	Context     json.RawMessage `json:"context,omitempty"`
	ResultNote  *string     `json:"result_note,omitempty"`
	CreatedAt   string      `json:"created_at"`
}

type createReq struct {
	LeadID      *uuid.UUID      `json:"lead_id,omitempty"`
	AssignedTo  uuid.UUID       `json:"assigned_to"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Priority    string          `json:"priority,omitempty"`
	DueAt       *time.Time      `json:"due_at,omitempty"`
	Context     json.RawMessage `json:"context,omitempty"`
}

type completeReq struct {
	ResultNote *string `json:"result_note,omitempty"`
}
type reasonReq struct {
	Reason *string `json:"reason,omitempty"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	q := r.URL.Query()
	opts := taskrepo.ListOptions{Status: q.Get("status"), Kind: q.Get("kind")}
	if a := q.Get("assigned_to"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "assigned_to must be a uuid")
			return
		}
		opts.AssigneeID = &id
	}
	if l := q.Get("lead_id"); l != "" {
		id, err := uuid.Parse(l)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "lead_id must be a uuid")
			return
		}
		opts.LeadID = &id
	}
	list, err := h.repo.List(r.Context(), orgID, opts)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list tasks")
		return
	}
	out := make([]view, len(list))
	for i, t := range list {
		out[i] = toView(t)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	creator := sess.TeamMemberID
	t, err := h.repo.Create(r.Context(), taskrepo.CreateInput{
		OrganizationID: orgID, LeadID: body.LeadID, AssignedTo: body.AssignedTo,
		CreatedBy: &creator, Kind: body.Kind, Title: body.Title, Description: body.Description,
		Priority: body.Priority, DueAt: body.DueAt, Origin: "manual", Context: body.Context,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TASK", err.Error())
		return
	}
	h.publish(orgID, t.ID, "task.created", toView(t))
	httpx.WriteJSON(w, http.StatusCreated, toView(t))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, taskrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load task")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(t))
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Start(r.Context(), orgID, id); err != nil {
		if errors.Is(err, taskrepo.ErrAssigneeBusy) {
			httpx.WriteError(w, http.StatusConflict, "ASSIGNEE_BUSY", "assignee already has an in_progress task")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not start")
		return
	}
	h.publish(orgID, id, "task.started", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body completeReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := h.repo.Complete(r.Context(), orgID, id, sess.TeamMemberID, body.ResultNote); err != nil {
		if errors.Is(err, taskrepo.ErrInvalidState) {
			httpx.WriteError(w, http.StatusConflict, "INVALID_STATE", "complete legal only from in_progress")
			return
		}
		if errors.Is(err, taskrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not complete")
		return
	}
	h.publish(orgID, id, "task.completed", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "task.cancelled", func(orgID, id uuid.UUID, reason *string) error {
		return h.repo.Cancel(r.Context(), orgID, id, reason)
	})
}
func (h *Handler) miss(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "task.missed", func(orgID, id uuid.UUID, reason *string) error {
		return h.repo.Miss(r.Context(), orgID, id, reason)
	})
}

func (h *Handler) transition(
	w http.ResponseWriter, r *http.Request, evtType string,
	op func(orgID, id uuid.UUID, reason *string) error,
) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body reasonReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := op(orgID, id, body.Reason); err != nil {
		if errors.Is(err, taskrepo.ErrInvalidState) {
			httpx.WriteError(w, http.StatusConflict, "INVALID_STATE", "illegal transition")
			return
		}
		if errors.Is(err, taskrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not transition")
		return
	}
	h.publish(orgID, id, evtType, nil)
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "task",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func toView(t taskrepo.Task) view {
	fmt := func(ti *time.Time) *string {
		if ti == nil {
			return nil
		}
		s := ti.UTC().Format(time.RFC3339)
		return &s
	}
	v := view{
		ID: t.ID, LeadID: t.LeadID, AssignedTo: t.AssignedTo,
		Kind: t.Kind, Title: t.Title, Description: t.Description,
		Priority: t.Priority, Status: t.Status,
		DueAt: fmt(t.DueAt), StartedAt: fmt(t.StartedAt), CompletedAt: fmt(t.CompletedAt),
		Origin: t.Origin, ResultNote: t.ResultNote,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
	}
	if len(t.Context) > 0 {
		v.Context = t.Context
	}
	return v
}

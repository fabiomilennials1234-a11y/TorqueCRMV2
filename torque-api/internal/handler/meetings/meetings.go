// Package meetings serves F13 agenda endpoints (S48).
//
//   GET    /meetings?from=&to=   — list meetings intersecting the window
//   POST   /meetings             — create meeting
//   GET    /meetings/:id         — detail
//   POST   /meetings/:id/status  — update status (scheduled|completed|cancelled|no_show)
//   DELETE /meetings/:id         — delete
package meetings

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	meetingrepo "github.com/milennials/torque-api/internal/repository/meeting"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *meetingrepo.Repository
	bus  *event.Bus
}

func New(repo *meetingrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/meetings", h.list)
	r.Post("/meetings", h.create)
	r.Get("/meetings/{id}", h.get)
	r.Post("/meetings/{id}/status", h.setStatus)
	r.Delete("/meetings/{id}", h.remove)
}

type meetingView struct {
	ID               uuid.UUID  `json:"id"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	StartsAt         time.Time  `json:"starts_at"`
	EndsAt           time.Time  `json:"ends_at"`
	Location         *string    `json:"location,omitempty"`
	LeadID           *uuid.UUID `json:"lead_id,omitempty"`
	OwnerMemberID    *uuid.UUID `json:"owner_member_id,omitempty"`
	Status           string     `json:"status"`
	ExternalProvider *string    `json:"external_provider,omitempty"`
	ExternalID       *string    `json:"external_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type createReq struct {
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        time.Time  `json:"ends_at"`
	Location      *string    `json:"location,omitempty"`
	LeadID        *uuid.UUID `json:"lead_id,omitempty"`
	OwnerMemberID *uuid.UUID `json:"owner_member_id,omitempty"`
}

type setStatusReq struct {
	Status string `json:"status"`
}

func toView(m meetingrepo.Meeting) meetingView {
	return meetingView{
		ID: m.ID, Title: m.Title, Description: m.Description,
		StartsAt: m.StartsAt, EndsAt: m.EndsAt, Location: m.Location,
		LeadID: m.LeadID, OwnerMemberID: m.OwnerMemberID, Status: m.Status,
		ExternalProvider: m.ExternalProvider, ExternalID: m.ExternalID,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7)
	to := now.AddDate(0, 0, 30)
	if raw := r.URL.Query().Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FROM", "from must be RFC3339")
			return
		}
		from = t.UTC()
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_TO", "to must be RFC3339")
			return
		}
		to = t.UTC()
	}
	if !from.Before(to) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW", "from must be before to")
		return
	}
	list, err := h.repo.ListRange(r.Context(), orgID, from, to)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list meetings")
		return
	}
	out := make([]meetingView, len(list))
	for i, m := range list {
		out[i] = toView(m)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "from": from, "to": to})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	m, err := h.repo.Create(r.Context(), meetingrepo.CreateInput{
		OrganizationID: orgID, Title: body.Title, Description: body.Description,
		StartsAt: body.StartsAt, EndsAt: body.EndsAt, Location: body.Location,
		LeadID: body.LeadID, OwnerMemberID: body.OwnerMemberID,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MEETING", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "meeting.created", TenantID: orgID, EntityType: "meeting",
		EntityID: &m.ID, OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, toView(m))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be uuid")
		return
	}
	m, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, meetingrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "meeting not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load meeting")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(m))
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be uuid")
		return
	}
	var body setStatusReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := h.repo.SetStatus(r.Context(), orgID, id, body.Status); err != nil {
		if errors.Is(err, meetingrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "meeting not found")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATUS", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "meeting.updated", TenantID: orgID, EntityType: "meeting",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be uuid")
		return
	}
	if err := h.repo.Delete(r.Context(), orgID, id); err != nil {
		if errors.Is(err, meetingrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "meeting not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete meeting")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "meeting.deleted", TenantID: orgID, EntityType: "meeting",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

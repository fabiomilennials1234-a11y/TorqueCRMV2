// Package leads serves the /api/v1/leads endpoints (F01).
//
//   GET    /leads           — cursor-paginated list
//   POST   /leads           — create
//   GET    /leads/:id       — detail
//   PATCH  /leads/:id       — partial update
//   DELETE /leads/:id       — soft-delete
//
// Every write publishes an event on the bus so connected WebSockets get a
// live patch. Reads are tenant-scoped by the session; the client cannot
// target another tenant.
package leads

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	"github.com/milennials/torque-api/internal/ws"
)

// Handler groups the /leads endpoints.
type Handler struct {
	repo *leadrepo.Repository
	bus  *event.Bus
}

// New returns a new handler.
func New(repo *leadrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes mounts the endpoints. Compose on a tenant-scoped authenticated subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/leads", h.list)
	r.Post("/leads", h.create)
	r.Get("/leads/{id}", h.get)
	r.Patch("/leads/{id}", h.update)
	r.Delete("/leads/{id}", h.softDelete)
}

// -------- DTOs --------------------------------------------------------

type leadView struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Company        *string         `json:"company,omitempty"`
	Phone          *string         `json:"phone,omitempty"`
	Email          *string         `json:"email,omitempty"`
	Position       *string         `json:"position,omitempty"`
	ResponsibleID  *uuid.UUID      `json:"responsible_id,omitempty"`
	Rating         *int16          `json:"rating,omitempty"`
	Score          *int16          `json:"qualification_score,omitempty"`
	Segment        *string         `json:"segment,omitempty"`
	Origin         *string         `json:"origin,omitempty"`
	CustomFields   json.RawMessage `json:"custom_fields,omitempty"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type listEnvelope struct {
	Data []leadView `json:"data"`
	Meta meta       `json:"meta"`
}
type meta struct {
	NextCursor string `json:"next_cursor"`
}

type createRequest struct {
	Name          string     `json:"name"`
	Company       *string    `json:"company,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	Email         *string    `json:"email,omitempty"`
	Position      *string    `json:"position,omitempty"`
	ResponsibleID *uuid.UUID `json:"responsible_id,omitempty"`
	Origin        *string    `json:"origin,omitempty"`
	CustomFields  json.RawMessage `json:"custom_fields,omitempty"`
}

type patchRequest struct {
	Name          *string    `json:"name,omitempty"`
	Company       *string    `json:"company,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	Email         *string    `json:"email,omitempty"`
	Position      *string    `json:"position,omitempty"`
	ResponsibleID *uuid.UUID `json:"responsible_id,omitempty"`
	Rating        *int16     `json:"rating,omitempty"`
	Segment       *string    `json:"segment,omitempty"`
}

// -------- list --------------------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	q := r.URL.Query()

	opts := leadrepo.ListOptions{Cursor: q.Get("cursor"), Search: q.Get("search")}
	if n, err := strconv.Atoi(q.Get("page_size")); err == nil {
		opts.Limit = n
	}
	if raw := q.Get("responsible_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "responsible_id must be a uuid")
			return
		}
		opts.ResponsibleID = &id
	}

	result, err := h.repo.List(r.Context(), orgID, opts)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list leads")
		return
	}
	views := make([]leadView, len(result.Items))
	for i, l := range result.Items {
		views[i] = toView(l)
	}
	httpx.WriteJSON(w, http.StatusOK, listEnvelope{Data: views, Meta: meta{NextCursor: result.NextCursor}})
}

// -------- create ------------------------------------------------------

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_NAME", "name is required")
		return
	}
	in := leadrepo.CreateInput{
		Name: body.Name, Company: body.Company, Phone: body.Phone, Email: body.Email,
		Position: body.Position, ResponsibleID: body.ResponsibleID, Origin: body.Origin,
	}
	if len(body.CustomFields) > 0 {
		in.CustomFields = []byte(body.CustomFields)
	}

	l, err := h.repo.Create(r.Context(), orgID, in)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_LEAD", err.Error())
		return
	}
	h.publish(l, "lead.created")
	w.Header().Set("Location", "/api/v1/leads/"+l.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, toView(l))
}

// -------- get ---------------------------------------------------------

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	l, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, leadrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "lead not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load lead")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(l))
}

// -------- update ------------------------------------------------------

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	var body patchRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	l, err := h.repo.Update(r.Context(), orgID, id, leadrepo.UpdateInput(body))
	if errors.Is(err, leadrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "lead not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_LEAD", err.Error())
		return
	}
	h.publish(l, "lead.updated")
	httpx.WriteJSON(w, http.StatusOK, toView(l))
}

// -------- softDelete --------------------------------------------------

func (h *Handler) softDelete(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	if err := h.repo.SoftDelete(r.Context(), orgID, id); err != nil {
		if errors.Is(err, leadrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "lead not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete lead")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "lead.deleted", TenantID: orgID, EntityType: "lead",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers -----------------------------------------------------

func (h *Handler) publish(l domain.Lead, evtType string) {
	h.bus.Publish(ws.Event{
		Type:       evtType,
		TenantID:   l.OrganizationID,
		EntityType: "lead",
		EntityID:   &l.ID,
		Patch:      toView(l),
		OccurredAt: time.Now().UTC(),
	})
}

func toView(l domain.Lead) leadView {
	v := leadView{
		ID: l.ID, Name: l.Name, Company: l.Company, Phone: l.Phone, Email: l.Email,
		Position: l.Position, ResponsibleID: l.ResponsibleID,
		Rating: l.Rating, Score: l.QualificationScore, Segment: l.Segment, Origin: l.Origin,
		CreatedAt: l.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: l.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if len(l.CustomFields) > 0 {
		v.CustomFields = l.CustomFields
	}
	return v
}

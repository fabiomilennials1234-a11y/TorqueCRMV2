// Package templates serve os endpoints /api/v1/message-templates.
//
//   GET  /message-templates              — list (member)
//   GET  /message-templates/:id          — detail (member)
//   POST /message-templates              — create (admin)
//   PATCH /message-templates/:id         — patch (admin)
//   DELETE /message-templates/:id        — delete (admin)
//
// Admin mutations ficam sob RequireRole(admin) no wiring do main.go.
package templates

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	templaterepo "github.com/milennials/torque-api/internal/repository/template"
)

type ReadHandler struct{ repo *templaterepo.Repository }
type AdminHandler struct{ repo *templaterepo.Repository }

func NewRead(repo *templaterepo.Repository) *ReadHandler   { return &ReadHandler{repo: repo} }
func NewAdmin(repo *templaterepo.Repository) *AdminHandler { return &AdminHandler{repo: repo} }

func (h *ReadHandler) Routes(r chi.Router) {
	r.Get("/message-templates", h.list)
	r.Get("/message-templates/{id}", h.get)
}
func (h *AdminHandler) Routes(r chi.Router) {
	r.Post("/message-templates", h.create)
	r.Patch("/message-templates/{id}", h.update)
	r.Delete("/message-templates/{id}", h.delete)
}

// -------- views ----------------------------------------------------

type templateView struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	Variables []string  `json:"variables"`
	IsActive  bool      `json:"is_active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func toView(t templaterepo.Template) templateView {
	return templateView{
		ID: t.ID, Name: t.Name, Body: t.Body, Variables: t.Variables,
		IsActive: t.IsActive,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// -------- read -----------------------------------------------------

func (h *ReadHandler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	activeOnly := r.URL.Query().Get("active_only") == "1"
	list, err := h.repo.List(r.Context(), orgID, activeOnly)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list templates")
		return
	}
	out := make([]templateView, len(list))
	for i, t := range list {
		out[i] = toView(t)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *ReadHandler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, templaterepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load template")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(t))
}

// -------- admin ----------------------------------------------------

type createReq struct {
	Name      string   `json:"name"`
	Body      string   `json:"body"`
	Variables []string `json:"variables,omitempty"`
}

func (h *AdminHandler) create(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	creator := sess.TeamMemberID
	t, err := h.repo.Create(r.Context(), orgID, templaterepo.CreateInput{
		Name: body.Name, Body: body.Body, Variables: body.Variables, CreatedBy: &creator,
	})
	if errors.Is(err, templaterepo.ErrInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TEMPLATE", "name must be 2-120 chars; body 1-16000 chars")
		return
	}
	if errors.Is(err, templaterepo.ErrNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", "template name already exists")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create template")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toView(t))
}

type updateReq struct {
	Name      *string   `json:"name,omitempty"`
	Body      *string   `json:"body,omitempty"`
	Variables *[]string `json:"variables,omitempty"`
	IsActive  *bool     `json:"is_active,omitempty"`
}

func (h *AdminHandler) update(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body updateReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	t, err := h.repo.Update(r.Context(), orgID, id, templaterepo.UpdateInput{
		Name: body.Name, Body: body.Body, Variables: body.Variables, IsActive: body.IsActive,
	})
	if errors.Is(err, templaterepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	}
	if errors.Is(err, templaterepo.ErrInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TEMPLATE", "name 2-120; body 1-16000")
		return
	}
	if errors.Is(err, templaterepo.ErrNameTaken) {
		httpx.WriteError(w, http.StatusConflict, "NAME_TAKEN", "template name already exists")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update template")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(t))
}

func (h *AdminHandler) delete(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), orgID, id); err != nil {
		if errors.Is(err, templaterepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "template not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers --------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func decodeErr(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body too large")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}

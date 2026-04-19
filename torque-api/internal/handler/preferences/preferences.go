// Package preferences serves PATCH /api/v1/me/preferences.
//
// Today the only preference is `ui_mode` (ADR-007). The endpoint is written
// to grow: new preferences are added to the request DTO and the update path
// extended. Anything persisted here MUST live on users (global), not
// team_members (org-scoped) — preferences follow the person, not the tenant.
package preferences

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
)

// Handler serves the preferences endpoint.
type Handler struct {
	users *userrepo.Repository
}

// New binds the handler to a user repository.
func New(users *userrepo.Repository) *Handler { return &Handler{users: users} }

// Routes mounts the endpoint on the authenticated subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Patch("/me/preferences", h.patch)
}

type patchRequest struct {
	UIMode *domain.UIMode `json:"ui_mode,omitempty"`
}

type patchResponse struct {
	UIMode domain.UIMode `json:"ui_mode"`
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())

	var body patchRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.UIMode == nil {
		httpx.WriteError(w, http.StatusBadRequest, "NOTHING_TO_UPDATE", "no preference provided")
		return
	}
	if !body.UIMode.IsValid() {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_UI_MODE", "ui_mode must be 'manager' or 'salesperson'")
		return
	}

	if err := h.users.UpdateUIMode(r.Context(), sess.UserID, *body.UIMode); err != nil {
		if errors.Is(err, userrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user record missing")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not persist preference")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, patchResponse{UIMode: *body.UIMode})
}

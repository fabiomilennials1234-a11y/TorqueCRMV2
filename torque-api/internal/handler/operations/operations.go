// Package operations serves the /operations endpoints of the 202-Accepted pattern.
//
//   POST   /api/v1/operations           — submit a new async job (tenant-scoped)
//   GET    /api/v1/operations/:id       — poll status (tenant-scoped)
//   DELETE /api/v1/operations/:id       — cancel an in-flight job
//
// Every response a handler ships is safe to cache by the client until the
// status field changes (the WS channel will push the change); no server-side
// caching is done here.
package operations

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
)

// Handler exposes the HTTP surface.
type Handler struct {
	repo *operationrepo.Repository
}

// New binds the handler to a repository.
func New(repo *operationrepo.Repository) *Handler { return &Handler{repo: repo} }

// Routes mounts the endpoints on a tenant-scoped authenticated subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/operations", h.submit)
	r.Get("/operations/{id}", h.get)
	r.Delete("/operations/{id}", h.cancel)
}

// -------- DTOs --------------------------------------------------------

type submitRequest struct {
	Kind           string          `json:"kind"`
	Input          json.RawMessage `json:"input,omitempty"`
	RetryRemaining int             `json:"retry_remaining,omitempty"`
}

type operationView struct {
	ID             uuid.UUID       `json:"id"`
	Kind           string          `json:"kind"`
	Status         string          `json:"status"`
	Progress       *float64        `json:"progress,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          json.RawMessage `json:"error,omitempty"`
	RetryRemaining int             `json:"retry_remaining"`
	CreatedAt      string          `json:"created_at"`
	StartedAt      *string         `json:"started_at,omitempty"`
	EndedAt        *string         `json:"ended_at,omitempty"`
}

// -------- submit ------------------------------------------------------

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())

	var body submitRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.Kind == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_KIND", "kind is required")
		return
	}
	input := []byte(body.Input)
	if len(input) == 0 {
		input = []byte(`{}`)
	}

	op, err := h.repo.Submit(r.Context(), domain.Operation{
		OrganizationID: sess.OrganizationID,
		ActorUserID:    uuidPtr(sess.UserID),
		ActorType:      actorTypeFor(sess),
		Kind:           body.Kind,
		Input:          input,
		RetryRemaining: body.RetryRemaining,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not submit operation")
		return
	}

	// 202 Accepted + Location so curl / browser can follow.
	w.Header().Set("Location", "/api/v1/operations/"+op.ID.String())
	httpx.WriteJSON(w, http.StatusAccepted, toView(op))
}

// -------- get ---------------------------------------------------------

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	op, err := h.repo.Lookup(r.Context(), sess.OrganizationID, id)
	if errors.Is(err, operationrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "operation not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load operation")
		return
	}

	// Hint the client how long to wait before polling again. Terminal = long;
	// running = short.
	if op.Status.IsTerminal() {
		w.Header().Set("Cache-Control", "private, max-age=60")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	httpx.WriteJSON(w, http.StatusOK, toView(op))
}

// -------- cancel ------------------------------------------------------

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	if err := h.repo.Cancel(r.Context(), sess.OrganizationID, id); err != nil {
		if errors.Is(err, operationrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusConflict, "NOT_CANCELLABLE", "operation not cancellable (terminal or missing)")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not cancel operation")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers -----------------------------------------------------

func toView(op domain.Operation) operationView {
	v := operationView{
		ID:             op.ID,
		Kind:           op.Kind,
		Status:         string(op.Status),
		Progress:       op.Progress,
		RetryRemaining: op.RetryRemaining,
		CreatedAt:      op.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if len(op.Result) > 0 && string(op.Result) != "null" {
		v.Result = op.Result
	}
	if len(op.ErrorPayload) > 0 {
		v.Error = op.ErrorPayload
	}
	if op.StartedAt != nil {
		s := op.StartedAt.UTC().Format("2006-01-02T15:04:05Z")
		v.StartedAt = &s
	}
	if op.EndedAt != nil {
		e := op.EndedAt.UTC().Format("2006-01-02T15:04:05Z")
		v.EndedAt = &e
	}
	return v
}

func actorTypeFor(sess domain.Session) string {
	switch {
	case sess.IsMaster:
		return "master"
	case sess.Role == domain.RoleAdmin:
		return "admin"
	case sess.Role == domain.RoleMember:
		return "membro"
	default:
		return "system"
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	copy := id
	return &copy
}

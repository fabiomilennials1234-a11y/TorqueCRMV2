// Package onboarding serves F13 /api/v1/onboarding endpoints.
//
//   GET  /onboarding          — read current user's wizard state
//   POST /onboarding/steps    — body: {step} marks a step complete
//   POST /onboarding/dismiss  — user dismisses the wizard
//   POST /onboarding/reset    — reset back to first step (self-service)
//
// All routes scoped to the caller's team_member_id — the user only reads or
// mutates their own state. Listing other members' onboarding is out of scope;
// master admin impersonation reaches the right row via sess.TeamMemberID on
// impersonation.
package onboarding

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	obrepo "github.com/milennials/torque-api/internal/repository/onboarding"
)

type Handler struct {
	repo *obrepo.Repository
}

func New(repo *obrepo.Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/onboarding", h.get)
	r.Post("/onboarding/steps", h.completeStep)
	r.Post("/onboarding/dismiss", h.dismiss)
	r.Post("/onboarding/reset", h.reset)
}

type statusView struct {
	CurrentStep    string   `json:"current_step"`
	StepsCompleted []string `json:"steps_completed"`
	AllSteps       []string `json:"all_steps"`
	Dismissed      bool     `json:"dismissed"`
	CompletedAt    *string  `json:"completed_at,omitempty"`
}

type stepReq struct {
	Step string `json:"step"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	// EnsureForMember is safe to call on every GET — ON CONFLICT makes it
	// cheap on warm rows and bootstrap-friendly on first load.
	s, err := h.repo.EnsureForMember(r.Context(), orgID, sess.TeamMemberID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load onboarding")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(s))
}

func (h *Handler) completeStep(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body stepReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		if httpx.IsBodyTooLarge(err) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 1 MiB")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.Step == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STEP", "step is required")
		return
	}
	s, err := h.repo.CompleteStep(r.Context(), orgID, sess.TeamMemberID, body.Step)
	if errors.Is(err, obrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "onboarding not initialized — call GET first")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STEP", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(s))
}

func (h *Handler) dismiss(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	if err := h.repo.Dismiss(r.Context(), orgID, sess.TeamMemberID); err != nil {
		if errors.Is(err, obrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "onboarding not initialized")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not dismiss")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	if err := h.repo.Reset(r.Context(), orgID, sess.TeamMemberID); err != nil {
		if errors.Is(err, obrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "onboarding not initialized")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not reset")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toView(s obrepo.Status) statusView {
	v := statusView{
		CurrentStep:    s.CurrentStep,
		StepsCompleted: s.StepsCompleted,
		AllSteps:       append([]string(nil), obrepo.CanonicalSteps...),
		Dismissed:      s.Dismissed,
	}
	if s.CompletedAt != nil {
		str := s.CompletedAt.UTC().Format(time.RFC3339)
		v.CompletedAt = &str
	}
	return v
}

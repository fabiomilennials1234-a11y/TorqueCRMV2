// Package performance serves F09 endpoints (S47).
//
//   GET  /performance/ranking?since=&until=   — aggregated leaderboard
//   GET  /performance/goals                   — list goals
//   POST /performance/goals                   — create goal (admin)
//   GET  /performance/commissions             — list commissions (filter ?member_id)
//   POST /performance/commissions/:id/status  — set status (admin)
//   GET  /performance/awards                  — list awards
//   POST /performance/awards                  — create award (admin)
package performance

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
	performancerepo "github.com/milennials/torque-api/internal/repository/performance"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *performancerepo.Repository
	bus  *event.Bus
}

func New(repo *performancerepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes mounts member-accessible reads. Admin writes ride on a
// subrouter installed by main.go.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/performance/ranking", h.ranking)
	r.Get("/performance/goals", h.listGoals)
	r.Get("/performance/commissions", h.listCommissions)
	r.Get("/performance/awards", h.listAwards)
}

// AdminRoutes are the write paths — called from the admin subrouter.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Post("/performance/goals", h.createGoal)
	r.Post("/performance/commissions/{id}/status", h.setCommissionStatus)
	r.Post("/performance/awards", h.createAward)
}

// -------- ranking ---------------------------------------------------

type rankingView struct {
	MemberID   uuid.UUID `json:"member_id"`
	MemberName string    `json:"member_name"`
	DealsWon   int64     `json:"deals_won"`
	Revenue    int64     `json:"revenue_cents"`
}

func (h *Handler) ranking(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	now := time.Now().UTC()
	since := now.AddDate(0, 0, -30)
	until := now
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_SINCE", "since must be RFC3339")
			return
		}
		since = t.UTC()
	}
	if raw := r.URL.Query().Get("until"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_UNTIL", "until must be RFC3339")
			return
		}
		until = t.UTC()
	}
	if !since.Before(until) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW", "since must be before until")
		return
	}
	entries, err := h.repo.Ranking(r.Context(), orgID, since, until)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not compute ranking")
		return
	}
	out := make([]rankingView, len(entries))
	for i, e := range entries {
		out[i] = rankingView{
			MemberID: e.MemberID, MemberName: e.MemberName,
			DealsWon: e.DealsWon, Revenue: e.Revenue,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "since": since, "until": until})
}

// -------- goals -----------------------------------------------------

type goalView struct {
	ID          uuid.UUID  `json:"id"`
	MemberID    *uuid.UUID `json:"member_id,omitempty"`
	Metric      string     `json:"metric"`
	Target      int64      `json:"target"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	CreatedAt   time.Time  `json:"created_at"`
}

type createGoalReq struct {
	MemberID    *uuid.UUID `json:"member_id,omitempty"`
	Metric      string     `json:"metric"`
	Target      int64      `json:"target"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
}

func (h *Handler) listGoals(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.ListGoals(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list goals")
		return
	}
	out := make([]goalView, len(list))
	for i, g := range list {
		out[i] = goalView{
			ID: g.ID, MemberID: g.MemberID, Metric: g.Metric, Target: g.Target,
			PeriodStart: g.PeriodStart, PeriodEnd: g.PeriodEnd, CreatedAt: g.CreatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) createGoal(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createGoalReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	in := performancerepo.CreateGoalInput{
		OrganizationID: orgID, MemberID: body.MemberID, Metric: body.Metric,
		Target: body.Target, PeriodStart: body.PeriodStart, PeriodEnd: body.PeriodEnd,
	}
	if sess, ok := mw.SessionFrom(r.Context()); ok && sess.TeamMemberID != uuid.Nil {
		tm := sess.TeamMemberID
		in.CreatedBy = &tm
	}
	g, err := h.repo.CreateGoal(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_GOAL", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "goal.created", TenantID: orgID, EntityType: "goal",
		EntityID: &g.ID, OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, goalView{
		ID: g.ID, MemberID: g.MemberID, Metric: g.Metric, Target: g.Target,
		PeriodStart: g.PeriodStart, PeriodEnd: g.PeriodEnd, CreatedAt: g.CreatedAt,
	})
}

// -------- commissions -----------------------------------------------

type commissionView struct {
	ID          uuid.UUID  `json:"id"`
	ProposalID  *uuid.UUID `json:"proposal_id,omitempty"`
	MemberID    uuid.UUID  `json:"member_id"`
	Percentage  float64    `json:"percentage"`
	AmountCents int64      `json:"amount_cents"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	EarnedAt    time.Time  `json:"earned_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

func (h *Handler) listCommissions(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var memberID *uuid.UUID
	if raw := r.URL.Query().Get("member_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_MEMBER", "member_id must be uuid")
			return
		}
		memberID = &id
	}
	list, err := h.repo.ListCommissions(r.Context(), orgID, memberID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list commissions")
		return
	}
	out := make([]commissionView, len(list))
	for i, c := range list {
		out[i] = commissionView{
			ID: c.ID, ProposalID: c.ProposalID, MemberID: c.MemberID,
			Percentage: c.Percentage, AmountCents: c.AmountCents, Currency: c.Currency,
			Status: c.Status, EarnedAt: c.EarnedAt, ApprovedAt: c.ApprovedAt,
			PaidAt: c.PaidAt, Notes: c.Notes,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

type setStatusReq struct {
	Status string `json:"status"`
}

func (h *Handler) setCommissionStatus(w http.ResponseWriter, r *http.Request) {
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
	if err := h.repo.SetCommissionStatus(r.Context(), orgID, id, body.Status); err != nil {
		if errors.Is(err, performancerepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "commission not found")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATUS", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "commission.updated", TenantID: orgID, EntityType: "commission",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// -------- awards ----------------------------------------------------

type awardView struct {
	ID          uuid.UUID       `json:"id"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Criteria    json.RawMessage `json:"criteria"`
	Winners     json.RawMessage `json:"winners"`
	AwardedAt   *time.Time      `json:"awarded_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type createAwardReq struct {
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Criteria    json.RawMessage `json:"criteria,omitempty"`
	Winners     json.RawMessage `json:"winners,omitempty"`
	AwardedAt   *time.Time      `json:"awarded_at,omitempty"`
}

func (h *Handler) listAwards(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.ListAwards(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list awards")
		return
	}
	out := make([]awardView, len(list))
	for i, a := range list {
		out[i] = awardView{
			ID: a.ID, Title: a.Title, Description: a.Description,
			Criteria: a.CriteriaJSON, Winners: a.WinnersJSON,
			AwardedAt: a.AwardedAt, CreatedAt: a.CreatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) createAward(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createAwardReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	in := performancerepo.CreateAwardInput{
		OrganizationID: orgID, Title: body.Title, Description: body.Description,
		Criteria: body.Criteria, Winners: body.Winners, AwardedAt: body.AwardedAt,
	}
	if sess, ok := mw.SessionFrom(r.Context()); ok && sess.TeamMemberID != uuid.Nil {
		tm := sess.TeamMemberID
		in.CreatedBy = &tm
	}
	a, err := h.repo.CreateAward(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_AWARD", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "award.created", TenantID: orgID, EntityType: "award",
		EntityID: &a.ID, OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, awardView{
		ID: a.ID, Title: a.Title, Description: a.Description,
		Criteria: a.CriteriaJSON, Winners: a.WinnersJSON,
		AwardedAt: a.AwardedAt, CreatedAt: a.CreatedAt,
	})
}

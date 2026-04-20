// Package proposals serves F03 /api/v1/proposals endpoints.
//
//   PUT    /proposals/:entryId                — upsert draft
//   GET    /proposals/:entryId                — fetch
//   POST   /proposals/:entryId/send           — draft → sent
//   POST   /proposals/:entryId/viewed         — sent → viewed (pixel + link open)
//   POST   /proposals/:entryId/accept         — sent|viewed → accepted
//   POST   /proposals/:entryId/reject         — sent|viewed → rejected
package proposals

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	proposalrepo "github.com/milennials/torque-api/internal/repository/proposal"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *proposalrepo.Repository
	bus  *event.Bus
}

func New(repo *proposalrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Put("/proposals/{entryId}", h.upsert)
	r.Get("/proposals/{entryId}", h.get)
	r.Post("/proposals/{entryId}/send", h.send)
	r.Post("/proposals/{entryId}/viewed", h.viewed)
	r.Post("/proposals/{entryId}/accept", h.accept)
	r.Post("/proposals/{entryId}/reject", h.reject)
}

type upsertReq struct {
	LeadID         uuid.UUID  `json:"lead_id"`
	Title          string     `json:"title"`
	AmountCents    int64      `json:"amount_cents"`
	Currency       string     `json:"currency,omitempty"`
	AttachmentKey  *string    `json:"attachment_key,omitempty"`
	AttachmentSize *int64     `json:"attachment_size,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type rejectReq struct {
	Reason *string `json:"reason,omitempty"`
}

type view struct {
	PipeEntryID    uuid.UUID `json:"pipe_entry_id"`
	LeadID         uuid.UUID `json:"lead_id"`
	Title          string    `json:"title"`
	AmountCents    int64     `json:"amount_cents"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	AttachmentKey  *string   `json:"attachment_key,omitempty"`
	SentAt         *string   `json:"sent_at,omitempty"`
	FirstViewedAt  *string   `json:"first_viewed_at,omitempty"`
	AcceptedAt     *string   `json:"accepted_at,omitempty"`
	RejectedAt     *string   `json:"rejected_at,omitempty"`
	ExpiresAt      *string   `json:"expires_at,omitempty"`
}

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, ok := parseEntryID(w, r)
	if !ok {
		return
	}
	var body upsertReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	p, err := h.repo.Upsert(r.Context(), proposalrepo.UpsertInput{
		OrganizationID: orgID, PipeEntryID: entryID, LeadID: body.LeadID,
		Title: body.Title, AmountCents: body.AmountCents, Currency: body.Currency,
		AttachmentKey: body.AttachmentKey, AttachmentSize: body.AttachmentSize,
		Notes: body.Notes, ExpiresAt: body.ExpiresAt,
	})
	if errors.Is(err, proposalrepo.ErrInvalidState) {
		httpx.WriteError(w, http.StatusConflict, "PROPOSAL_LOCKED", "proposal is past draft; use transition endpoints")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PROPOSAL", err.Error())
		return
	}
	h.publish(orgID, entryID, "proposal.upserted", toView(p))
	httpx.WriteJSON(w, http.StatusOK, toView(p))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, ok := parseEntryID(w, r)
	if !ok {
		return
	}
	p, err := h.repo.Get(r.Context(), orgID, entryID)
	if errors.Is(err, proposalrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "proposal not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load proposal")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(p))
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "proposal.sent", h.repo.MarkSent)
}

func (h *Handler) viewed(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, ok := parseEntryID(w, r)
	if !ok {
		return
	}
	if err := h.repo.MarkViewed(r.Context(), orgID, entryID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not mark viewed")
		return
	}
	h.publish(orgID, entryID, "proposal.viewed", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "proposal.accepted", h.repo.MarkAccepted)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, ok := parseEntryID(w, r)
	if !ok {
		return
	}
	var body rejectReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	err := h.repo.MarkRejected(r.Context(), orgID, entryID, body.Reason)
	if errors.Is(err, proposalrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "proposal not found")
		return
	}
	if errors.Is(err, proposalrepo.ErrInvalidState) {
		httpx.WriteError(w, http.StatusConflict, "INVALID_STATE", "only sent/viewed proposals can be rejected")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not reject")
		return
	}
	h.publish(orgID, entryID, "proposal.rejected", nil)
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers -----------------------------------------------------

func parseEntryID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "entry id must be a uuid")
		return uuid.Nil, false
	}
	return entryID, true
}

func (h *Handler) transition(
	w http.ResponseWriter, r *http.Request, evtType string,
	op func(ctx context.Context, orgID, entryID uuid.UUID) error,
) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, ok := parseEntryID(w, r)
	if !ok {
		return
	}
	err := op(r.Context(), orgID, entryID)
	if errors.Is(err, proposalrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "proposal not found")
		return
	}
	if errors.Is(err, proposalrepo.ErrInvalidState) {
		httpx.WriteError(w, http.StatusConflict, "INVALID_STATE", "transition not legal from current status")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not transition proposal")
		return
	}
	h.publish(orgID, entryID, evtType, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publish(orgID, entryID uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "proposal",
		EntityID: &entryID, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func toView(p proposalrepo.Proposal) view {
	v := view{
		PipeEntryID: p.PipeEntryID, LeadID: p.LeadID, Title: p.Title,
		AmountCents: p.AmountCents, Currency: p.Currency, Status: string(p.Status),
		AttachmentKey: p.AttachmentKey,
	}
	fmt := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.UTC().Format(time.RFC3339)
		return &s
	}
	v.SentAt = fmt(p.SentAt)
	v.FirstViewedAt = fmt(p.FirstViewedAt)
	v.AcceptedAt = fmt(p.AcceptedAt)
	v.RejectedAt = fmt(p.RejectedAt)
	v.ExpiresAt = fmt(p.ExpiresAt)
	return v
}


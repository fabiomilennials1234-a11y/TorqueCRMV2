// Package confirmations serves the /api/v1/confirmations endpoints (F02).
//
//   PUT    /confirmations/:pipeEntryID          — upsert meeting metadata
//   GET    /confirmations/:pipeEntryID          — fetch
//   POST   /confirmations/:pipeEntryID/confirm  — mark confirmed_at
//   POST   /confirmations/:pipeEntryID/no-show  — mark no_show + reason
//   GET    /confirmations/overdue               — list unconfirmed + past-due
//
// Every write publishes `confirmation.{upserted,confirmed,no_show}` so the
// frontend Kanban and agenda views patch live.
package confirmations

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	confirmationrepo "github.com/milennials/torque-api/internal/repository/confirmation"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	integrationpkg "github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/service/integration/gcal"
	"github.com/milennials/torque-api/internal/ws"
)

// Handler groups the F02 endpoints.
type Handler struct {
	repo   *confirmationrepo.Repository
	bus    *event.Bus
	gcal   *gcal.Provider
	credit *integrationrepo.Store
	logger zerolog.Logger
}

// New binds the handler.
func New(repo *confirmationrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus, logger: zerolog.Nop()}
}

// WithIntegrations attaches the GCal adapter + credential store. nil is
// accepted — the handler collapses back to local-only behavior. Added
// in S50 to close the F02 → Google Calendar sync gap deferred from S49.
func (h *Handler) WithIntegrations(g *gcal.Provider, store *integrationrepo.Store, logger zerolog.Logger) *Handler {
	h.gcal = g
	h.credit = store
	h.logger = logger
	return h
}

// Routes mounts the endpoints on a tenant-scoped subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Put("/confirmations/{entryId}", h.upsert)
	r.Get("/confirmations/{entryId}", h.get)
	r.Post("/confirmations/{entryId}/confirm", h.confirm)
	r.Post("/confirmations/{entryId}/no-show", h.noShow)
	r.Get("/confirmations/overdue", h.overdue)
}

// -------- DTOs --------------------------------------------------------

type upsertRequest struct {
	LeadID         uuid.UUID `json:"lead_id"`
	MeetingAt      time.Time `json:"meeting_at"`
	MeetingChannel *string   `json:"meeting_channel,omitempty"`
	MeetingNotes   *string   `json:"meeting_notes,omitempty"`
}

type noShowRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type view struct {
	PipeEntryID    uuid.UUID  `json:"pipe_entry_id"`
	LeadID         uuid.UUID  `json:"lead_id"`
	MeetingAt      string     `json:"meeting_at"`
	MeetingChannel *string    `json:"meeting_channel,omitempty"`
	MeetingNotes   *string    `json:"meeting_notes,omitempty"`
	ConfirmedAt    *string    `json:"confirmed_at,omitempty"`
	NoShow         bool       `json:"no_show"`
	NoShowReason   *string    `json:"no_show_reason,omitempty"`
}

// -------- handlers ----------------------------------------------------

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "entry id must be a uuid")
		return
	}
	var body upsertRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	c, err := h.repo.Upsert(r.Context(), confirmationrepo.UpsertInput{
		OrganizationID: orgID,
		PipeEntryID:    entryID,
		LeadID:         body.LeadID,
		MeetingAt:      body.MeetingAt,
		MeetingChannel: body.MeetingChannel,
		MeetingNotes:   body.MeetingNotes,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CONFIRMATION", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "confirmation.upserted", TenantID: orgID, EntityType: "confirmation",
		EntityID: &entryID, Patch: toView(c), OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusOK, toView(c))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "entry id must be a uuid")
		return
	}
	c, err := h.repo.Get(r.Context(), orgID, entryID)
	if errors.Is(err, confirmationrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "confirmation not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load confirmation")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(c))
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "entry id must be a uuid")
		return
	}
	// Snapshot BEFORE we flip confirmed_at so the GCal sync has the
	// meeting window + lead id handy for the event body. Absence is
	// fine — the handler still marks the confirmation, the sync
	// short-circuits.
	var snapshot *confirmationrepo.Confirmation
	if c, gerr := h.repo.Get(r.Context(), orgID, entryID); gerr == nil {
		snapshot = &c
	}
	if err := h.repo.MarkConfirmed(r.Context(), orgID, entryID); err != nil {
		if errors.Is(err, confirmationrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "confirmation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not confirm")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "confirmation.confirmed", TenantID: orgID, EntityType: "confirmation",
		EntityID: &entryID, OccurredAt: time.Now().UTC(),
	})
	// S50 — best-effort GCal sync. Detached from the request lifetime
	// so the 204 response does not depend on Google's availability.
	h.maybePushToGCal(orgID, snapshot)
	w.WriteHeader(http.StatusNoContent)
}

// maybePushToGCal spawns a goroutine that creates a Google Calendar
// event mirroring the confirmed meeting. Silent when no gcal
// integration is wired, no google credential exists, or the
// confirmation snapshot is missing required fields.
func (h *Handler) maybePushToGCal(orgID uuid.UUID, snap *confirmationrepo.Confirmation) {
	if h.gcal == nil || h.credit == nil || snap == nil {
		return
	}
	if snap.MeetingAt.IsZero() {
		return
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := h.credit.Get(probeCtx, orgID, integrationrepo.ProviderGoogle); err != nil {
		// No credential — silently skip.
		return
	}

	title := "Reuniao confirmada"
	if snap.MeetingChannel != nil && *snap.MeetingChannel != "" {
		title = "Reuniao confirmada — " + *snap.MeetingChannel
	}
	description := "Confirmada via Torque CRM (F02)."
	if snap.MeetingNotes != nil && *snap.MeetingNotes != "" {
		description += "\n\n" + *snap.MeetingNotes
	}
	// Default duration — 30 minutes. The F02 schema does not carry an
	// explicit end time (the confirmation is about the start moment).
	endAt := snap.MeetingAt.Add(30 * time.Minute)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ctx = domain.WithOrgID(ctx, orgID)

		_, err := h.gcal.CreateMeetingForOrg(ctx, orgID, integrationpkg.MeetingInput{
			Title:       title,
			Description: description,
			StartAt:     snap.MeetingAt,
			EndAt:       endAt,
		})
		if err != nil {
			h.logger.Warn().Err(err).
				Str("org_id", orgID.String()).
				Str("pipe_entry_id", snap.PipeEntryID.String()).
				Msg("gcal confirmation sync failed")
		}
	}()
}

func (h *Handler) noShow(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "entry id must be a uuid")
		return
	}
	var body noShowRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if err := h.repo.MarkNoShow(r.Context(), orgID, entryID, body.Reason); err != nil {
		if errors.Is(err, confirmationrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "confirmation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not mark no-show")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "confirmation.no_show", TenantID: orgID, EntityType: "confirmation",
		EntityID: &entryID, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) overdue(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	limit := 50
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = n
	}
	list, err := h.repo.Overdue(r.Context(), orgID, time.Now().UTC(), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list overdue")
		return
	}
	out := make([]view, len(list))
	for i, c := range list {
		out[i] = toView(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func toView(c confirmationrepo.Confirmation) view {
	v := view{
		PipeEntryID:    c.PipeEntryID,
		LeadID:         c.LeadID,
		MeetingAt:      c.MeetingAt.UTC().Format(time.RFC3339),
		MeetingChannel: c.MeetingChannel,
		MeetingNotes:   c.MeetingNotes,
		NoShow:         c.NoShow,
		NoShowReason:   c.NoShowReason,
	}
	if c.ConfirmedAt != nil {
		s := c.ConfirmedAt.UTC().Format(time.RFC3339)
		v.ConfirmedAt = &s
	}
	return v
}

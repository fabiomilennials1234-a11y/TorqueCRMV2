// Package meetings serves F13 agenda endpoints (S48 local CRUD, S49 GCal sync).
//
//	GET    /meetings?from=&to=   — list meetings intersecting the window
//	POST   /meetings             — create meeting (optionally pushed to GCal)
//	GET    /meetings/:id         — detail
//	POST   /meetings/:id/status  — update status (scheduled|completed|cancelled|no_show)
//	DELETE /meetings/:id         — delete (optionally cancels GCal event)
//
// S49 wiring: when a `google` credential exists for the tenant, POST
// /meetings spawns a detached goroutine that calls GCal and writes the
// resulting provider_event_id back onto the meeting row. DELETE mirrors
// the flow best-effort. HTTP responses never block on GCal — the local
// row is the source of truth; the external event is advisory.
package meetings

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/repository/integration"
	meetingrepo "github.com/milennials/torque-api/internal/repository/meeting"
	integrationpkg "github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/service/integration/gcal"
	"github.com/milennials/torque-api/internal/ws"
)

// Handler groups the F13 endpoints.
type Handler struct {
	repo   *meetingrepo.Repository
	bus    *event.Bus
	gcal   *gcal.Provider
	credit *integration.Store
	logger zerolog.Logger
}

// New constructs the handler with only the local repo wired. GCal +
// credentials are opt-in via WithIntegrations.
func New(repo *meetingrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus, logger: zerolog.Nop()}
}

// WithIntegrations attaches the GCal adapter + credential store. nil is
// accepted — the handler collapses back to local-only behavior.
func (h *Handler) WithIntegrations(g *gcal.Provider, store *integration.Store, logger zerolog.Logger) *Handler {
	h.gcal = g
	h.credit = store
	h.logger = logger
	return h
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
	// AttendeeEmail is passed through to Google Calendar only — never
	// persisted on the meetings table (keeps S48 schema intact).
	AttendeeEmail *string `json:"attendee_email,omitempty"`
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

// emailRE is a deliberately lax RFC5322 shape check — we're not trying
// to validate deliverability, just reject obvious garbage before we
// forward it to Google (which will reject it anyway if malformed).
var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if body.AttendeeEmail != nil {
		email := strings.TrimSpace(*body.AttendeeEmail)
		if email != "" && !emailRE.MatchString(email) {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_ATTENDEE",
				"attendee_email must be a valid email")
			return
		}
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

	// Best-effort GCal sync — detached from the request lifetime.
	h.maybePushToGCal(orgID, m, body)

	httpx.WriteJSON(w, http.StatusCreated, toView(m))
}

// maybePushToGCal spawns a goroutine that creates the event on Google
// Calendar and writes the resulting provider_event_id back to the
// meetings row. Any failure is logged + recorded on the credential
// row; the local meeting is unaffected.
func (h *Handler) maybePushToGCal(orgID uuid.UUID, m meetingrepo.Meeting, body createReq) {
	if h.gcal == nil || h.credit == nil {
		return
	}
	// Fast-path nil check: skip when no google credential exists. We
	// use a short context for the check so a lost goroutine doesn't
	// linger.
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := h.credit.Get(probeCtx, orgID, integration.ProviderGoogle); err != nil {
		// Silent skip — missing credential is expected, not an error.
		return
	}

	attendees := []string{}
	if body.AttendeeEmail != nil {
		if e := strings.TrimSpace(*body.AttendeeEmail); e != "" {
			attendees = append(attendees, e)
		}
	}

	description := ""
	if body.Description != nil {
		description = *body.Description
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ctx = domain.WithOrgID(ctx, orgID)

		res, err := h.gcal.CreateMeetingForOrg(ctx, orgID, integrationpkg.MeetingInput{
			Title:       m.Title,
			Description: description,
			StartAt:     m.StartsAt,
			EndAt:       m.EndsAt,
			Attendees:   attendees,
		})
		if err != nil {
			h.logger.Warn().Err(err).
				Str("org_id", orgID.String()).
				Str("meeting_id", m.ID.String()).
				Msg("gcal event create failed")
			return
		}
		if err := h.repo.SetExternal(ctx, orgID, m.ID, "gcal", res.ProviderEventID); err != nil {
			h.logger.Warn().Err(err).
				Str("org_id", orgID.String()).
				Str("meeting_id", m.ID.String()).
				Msg("persist external id failed")
			return
		}
		// Publish a follow-up patch so the UI updates the external badge.
		h.bus.Publish(ws.Event{
			Type: "meeting.updated", TenantID: orgID, EntityType: "meeting",
			EntityID: &m.ID, OccurredAt: time.Now().UTC(),
		})
	}()
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
	// Snapshot the meeting before delete so we can reach the external
	// event on the way out.
	var snapshot *meetingrepo.Meeting
	if m, err := h.repo.Get(r.Context(), orgID, id); err == nil {
		snapshot = &m
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

	// Best-effort GCal cancel after a successful local delete.
	h.maybeCancelGCal(orgID, snapshot)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) maybeCancelGCal(orgID uuid.UUID, m *meetingrepo.Meeting) {
	if h.gcal == nil || m == nil {
		return
	}
	if m.ExternalProvider == nil || *m.ExternalProvider != "gcal" ||
		m.ExternalID == nil || *m.ExternalID == "" {
		return
	}
	eventID := *m.ExternalID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ctx = domain.WithOrgID(ctx, orgID)
		if err := h.gcal.CancelMeetingForOrg(ctx, orgID, eventID); err != nil {
			h.logger.Warn().Err(err).
				Str("org_id", orgID.String()).
				Str("event_id", eventID).
				Msg("gcal event cancel failed")
		}
	}()
}


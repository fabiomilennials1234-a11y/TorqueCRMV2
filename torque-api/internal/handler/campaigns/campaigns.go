// Package campaigns serves F08 /api/v1/campaigns endpoints.
//
//   GET    /campaigns               — list (optional status filter)
//   POST   /campaigns               — create (draft)
//   GET    /campaigns/:id           — detail + stats
//   POST   /campaigns/:id/launch    — materialize recipients + start running
//   POST   /campaigns/:id/pause     — running → paused
//   POST   /campaigns/:id/resume    — paused → running
//   POST   /campaigns/:id/cancel    — * → cancelled
//   GET    /campaigns/:id/recipients — per-lead delivery ledger
//
// Admin-only. The audience query evaluation (turning `audience_query` into a
// list of leads) currently comes from the request body's `lead_ids` — the
// admin provides it explicitly. A richer audience resolver lands with the
// lead-search DSL in a future sprint.
package campaigns

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	campaignrepo "github.com/milennials/torque-api/internal/repository/campaign"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *campaignrepo.Repository
	bus  *event.Bus
}

func New(repo *campaignrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/campaigns", h.list)
	r.Post("/campaigns", h.create)
	r.Get("/campaigns/{id}", h.get)
	r.Post("/campaigns/{id}/launch", h.launch)
	r.Post("/campaigns/{id}/pause", h.pause)
	r.Post("/campaigns/{id}/resume", h.resume)
	r.Post("/campaigns/{id}/cancel", h.cancel)
	r.Get("/campaigns/{id}/recipients", h.recipients)
}

type createReq struct {
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	ChannelID     *uuid.UUID      `json:"channel_id,omitempty"`
	TemplateBody  string          `json:"template_body"`
	AudienceQuery json.RawMessage `json:"audience_query,omitempty"`
	ScheduledAt   *time.Time      `json:"scheduled_at,omitempty"`
}

type launchReq struct {
	LeadIDs []uuid.UUID `json:"lead_ids"`
}

type campaignView struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ChannelID     *uuid.UUID `json:"channel_id,omitempty"`
	TemplateBody  string     `json:"template_body"`
	Status        string     `json:"status"`
	ScheduledAt   *string    `json:"scheduled_at,omitempty"`
	StartedAt     *string    `json:"started_at,omitempty"`
	EndedAt       *string    `json:"ended_at,omitempty"`
	StatsQueued   int        `json:"stats_queued"`
	StatsSent     int        `json:"stats_sent"`
	StatsFailed   int        `json:"stats_failed"`
	StatsSkipped  int        `json:"stats_skipped"`
}

type recipientView struct {
	ID         uuid.UUID  `json:"id"`
	LeadID     uuid.UUID  `json:"lead_id"`
	Status     string     `json:"status"`
	MessageID  *uuid.UUID `json:"message_id,omitempty"`
	SentAt     *string    `json:"sent_at,omitempty"`
	FailedAt   *string    `json:"failed_at,omitempty"`
}

// -------- handlers --------------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	status := r.URL.Query().Get("status")
	list, err := h.repo.List(r.Context(), orgID, status)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list campaigns")
		return
	}
	out := make([]campaignView, len(list))
	for i, c := range list {
		out[i] = toView(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	creator := sess.TeamMemberID
	c, err := h.repo.Create(r.Context(), campaignrepo.CreateInput{
		OrganizationID: orgID, Name: body.Name, Description: body.Description,
		ChannelID: body.ChannelID, TemplateBody: body.TemplateBody,
		AudienceQuery: body.AudienceQuery, ScheduledAt: body.ScheduledAt,
		CreatedBy: &creator,
	})
	if errors.Is(err, campaignrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "channel not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CAMPAIGN", err.Error())
		return
	}
	h.publish(orgID, c.ID, "campaign.created", toView(c))
	httpx.WriteJSON(w, http.StatusCreated, toView(c))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, campaignrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load campaign")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(c))
}

func (h *Handler) launch(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body launchReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	if len(body.LeadIDs) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "NO_RECIPIENTS", "lead_ids must not be empty")
		return
	}
	if len(body.LeadIDs) > 10_000 {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "TOO_MANY_RECIPIENTS",
			"single launch capped at 10k recipients; batch larger sends")
		return
	}
	queued, err := h.repo.Launch(r.Context(), orgID, id, body.LeadIDs)
	if errors.Is(err, campaignrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
		return
	}
	if errors.Is(err, campaignrepo.ErrInvalidState) {
		httpx.WriteError(w, http.StatusConflict, "INVALID_STATE",
			"campaign must be draft or scheduled to launch")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not launch")
		return
	}
	h.publish(orgID, id, "campaign.launched", map[string]int64{"queued": queued})
	httpx.WriteJSON(w, http.StatusAccepted, map[string]int64{"queued": queued})
}

func (h *Handler) pause(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "paused", "campaign.paused")
}
func (h *Handler) resume(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "running", "campaign.resumed")
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "cancelled", "campaign.cancelled")
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status, evt string) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.SetStatus(r.Context(), orgID, id, status); err != nil {
		if errors.Is(err, campaignrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not set status")
		return
	}
	h.publish(orgID, id, evt, map[string]string{"status": status})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) recipients(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	list, err := h.repo.ListRecipients(r.Context(), orgID, id, 200)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list recipients")
		return
	}
	out := make([]recipientView, len(list))
	for i, rp := range list {
		out[i] = toRecipientView(rp)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// -------- helpers ---------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func decodeError(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 1 MiB")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}

func (h *Handler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "campaign",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func fmtTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func toView(c campaignrepo.Campaign) campaignView {
	return campaignView{
		ID: c.ID, Name: c.Name, Description: c.Description, ChannelID: c.ChannelID,
		TemplateBody: c.TemplateBody, Status: c.Status,
		ScheduledAt: fmtTime(c.ScheduledAt), StartedAt: fmtTime(c.StartedAt),
		EndedAt: fmtTime(c.EndedAt), StatsQueued: c.StatsQueued, StatsSent: c.StatsSent,
		StatsFailed: c.StatsFailed, StatsSkipped: c.StatsSkipped,
	}
}
func toRecipientView(rp campaignrepo.Recipient) recipientView {
	return recipientView{
		ID: rp.ID, LeadID: rp.LeadID, Status: rp.Status, MessageID: rp.MessageID,
		SentAt: fmtTime(rp.SentAt), FailedAt: fmtTime(rp.FailedAt),
	}
}

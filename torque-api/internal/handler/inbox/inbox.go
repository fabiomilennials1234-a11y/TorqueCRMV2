// Package inbox serves the F04 endpoints.
//
//   GET    /api/v1/conversations                    — list (filters)
//   GET    /api/v1/conversations/:id                — detail
//   POST   /api/v1/conversations/:id/read           — zero unread_count
//   POST   /api/v1/conversations/:id/assign         — set assignee (audit on takeover)
//   PATCH  /api/v1/conversations/:id                — change state
//   GET    /api/v1/conversations/:id/messages       — message list
//   POST   /api/v1/conversations/:id/messages       — outbound send (queued)
package inbox

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
	auditrepo "github.com/milennials/torque-api/internal/repository/audit"
	inboxrepo "github.com/milennials/torque-api/internal/repository/inbox"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo  *inboxrepo.Repository
	bus   *event.Bus
	// audit is optional — nil means the handler skips the audit row on
	// takeover. Callers that care about forensics inject a real repo.
	audit *auditrepo.Repository
}

func New(repo *inboxrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// WithAudit wires an audit repo so takeover + state changes generate
// traceable rows. Chain call: `inbox.New(...).WithAudit(auditRepo)`.
func (h *Handler) WithAudit(a *auditrepo.Repository) *Handler {
	h.audit = a
	return h
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/conversations", h.listConversations)
	r.Get("/conversations/{id}", h.getConversation)
	r.Post("/conversations/{id}/read", h.markRead)
	r.Post("/conversations/{id}/assign", h.assign)
	r.Patch("/conversations/{id}", h.patchState)
	r.Get("/conversations/{id}/messages", h.listMessages)
	r.Post("/conversations/{id}/messages", h.sendMessage)
}

type convView struct {
	ID                 uuid.UUID  `json:"id"`
	ChannelID          uuid.UUID  `json:"channel_id"`
	ChannelKind        string     `json:"channel_kind"`
	LeadID             *uuid.UUID `json:"lead_id,omitempty"`
	ContactName        *string    `json:"contact_name,omitempty"`
	ContactHandle      *string    `json:"contact_handle,omitempty"`
	State              string     `json:"state"`
	AssignedTo         *uuid.UUID `json:"assigned_to,omitempty"`
	UnreadCount        int        `json:"unread_count"`
	LastMessageAt      *string    `json:"last_message_at,omitempty"`
	LastMessagePreview *string    `json:"last_message_preview,omitempty"`
}

type msgView struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	Direction      string     `json:"direction"`
	Kind           string     `json:"kind"`
	Body           *string    `json:"body,omitempty"`
	MediaURL       *string    `json:"media_url,omitempty"`
	SentByMemberID *uuid.UUID `json:"sent_by_member_id,omitempty"`
	Status         string     `json:"status"`
	OccurredAt     string     `json:"occurred_at"`
}

type assignReq struct {
	AssignedTo *uuid.UUID `json:"assigned_to"`
}
type patchStateReq struct {
	State string `json:"state"`
}
type sendMsgReq struct {
	Kind     string  `json:"kind"`
	Body     *string `json:"body,omitempty"`
	MediaURL *string `json:"media_url,omitempty"`
}

// -------- list -------------------------------------------------------

func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	q := r.URL.Query()
	opts := inboxrepo.ListConversationsOptions{State: q.Get("state")}
	if a := q.Get("assigned_to"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "assigned_to must be a uuid")
			return
		}
		opts.AssignedTo = &id
	}
	if c := q.Get("channel_id"); c != "" {
		id, err := uuid.Parse(c)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILTER", "channel_id must be a uuid")
			return
		}
		opts.ChannelID = &id
	}
	list, err := h.repo.ListConversations(r.Context(), orgID, opts)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list conversations")
		return
	}
	out := make([]convView, len(list))
	for i, c := range list {
		out[i] = toConvView(c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// -------- detail, read, assign, state --------------------------------

func (h *Handler) getConversation(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := h.repo.GetConversation(r.Context(), orgID, id)
	if errors.Is(err, inboxrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "conversation not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load conversation")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toConvView(c))
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.MarkRead(r.Context(), orgID, id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not mark read")
		return
	}
	h.publish(orgID, id, "conversation.read", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body assignReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}

	// Detect takeover shape BEFORE mutating so we can fetch the previous
	// owner for the audit row. A 200+1 extra select-on-takeover is a
	// cheap price for traceability.
	var previousOwner *uuid.UUID
	isTakeover := false
	if h.audit != nil {
		if prior, err := h.repo.GetConversation(r.Context(), orgID, id); err == nil {
			previousOwner = prior.AssignedTo
			// Takeover = mutating to a non-self owner when the conv already
			// had a different owner.
			if body.AssignedTo != nil && prior.AssignedTo != nil &&
				*prior.AssignedTo != *body.AssignedTo &&
				*body.AssignedTo != sess.TeamMemberID {
				isTakeover = true
			}
		}
	}

	if err := h.repo.AssignConversation(r.Context(), orgID, id, body.AssignedTo); err != nil {
		if errors.Is(err, inboxrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not assign")
		return
	}

	// Audit row for forensic trail on takeover scenarios. Self-assign or
	// first-assign skip the audit write to avoid log spam.
	if isTakeover && h.audit != nil {
		_ = h.audit.Append(r.Context(), auditrepo.Entry{
			ActorType:      string(sess.Role),
			ActorUserID:    &sess.UserID,
			OrganizationID: &orgID,
			Action:         "conversation.takeover",
			EntityType:     ptrStr("conversation"),
			EntityID:       &id,
			Payload: map[string]any{
				"previous_owner": previousOwner,
				"new_owner":      body.AssignedTo,
				"by_member":      sess.TeamMemberID,
			},
		})
	}

	h.publish(orgID, id, "conversation.assigned", map[string]any{
		"assigned_to": body.AssignedTo,
		"takeover":    isTakeover,
	})
	w.WriteHeader(http.StatusNoContent)
}

func ptrStr(s string) *string { return &s }

func (h *Handler) patchState(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body patchStateReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	allowed := map[string]bool{"open": true, "pending": true, "resolved": true, "archived": true}
	if !allowed[body.State] {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATE", "state must be one of open/pending/resolved/archived")
		return
	}
	if err := h.repo.SetConversationState(r.Context(), orgID, id, body.State); err != nil {
		if errors.Is(err, inboxrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not set state")
		return
	}
	h.publish(orgID, id, "conversation.state_changed", map[string]any{"state": body.State})
	w.WriteHeader(http.StatusNoContent)
}

// -------- messages ---------------------------------------------------

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ms, err := h.repo.ListMessages(r.Context(), orgID, id, 100)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list messages")
		return
	}
	out := make([]msgView, len(ms))
	for i, m := range ms {
		out[i] = toMsgView(m)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body sendMsgReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if body.Kind == "" {
		body.Kind = "text"
	}
	member := sess.TeamMemberID
	m, err := h.repo.AppendMessage(r.Context(), inboxrepo.AppendMessageInput{
		OrganizationID: orgID, ConversationID: id,
		Direction: "outbound", Kind: body.Kind,
		Body: body.Body, MediaURL: body.MediaURL,
		SentByMemberID: &member, OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MESSAGE", err.Error())
		return
	}
	patch, _ := json.Marshal(toMsgView(m))
	h.bus.Publish(ws.Event{
		Type: "message.sent", TenantID: orgID, EntityType: "message",
		EntityID: &m.ID, Patch: json.RawMessage(patch), OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusAccepted, toMsgView(m))
}

// -------- helpers ----------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "conversation",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func toConvView(c inboxrepo.Conversation) convView {
	v := convView{
		ID: c.ID, ChannelID: c.ChannelID, ChannelKind: c.ChannelKind, LeadID: c.LeadID,
		ContactName: c.ContactName, ContactHandle: c.ContactHandle,
		State: c.State, AssignedTo: c.AssignedTo, UnreadCount: c.UnreadCount,
		LastMessagePreview: c.LastMessagePreview,
	}
	if c.LastMessageAt != nil {
		s := c.LastMessageAt.UTC().Format(time.RFC3339)
		v.LastMessageAt = &s
	}
	return v
}

func toMsgView(m inboxrepo.Message) msgView {
	return msgView{
		ID: m.ID, ConversationID: m.ConversationID,
		Direction: m.Direction, Kind: m.Kind, Body: m.Body, MediaURL: m.MediaURL,
		SentByMemberID: m.SentByMemberID, Status: m.Status,
		OccurredAt: m.OccurredAt.UTC().Format(time.RFC3339),
	}
}

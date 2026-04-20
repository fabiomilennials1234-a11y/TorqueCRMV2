// Package agents serves F06 Copilot endpoints.
//
//   GET    /api/v1/agents                     — list
//   POST   /api/v1/agents                     — create (draft)
//   GET    /api/v1/agents/:id                 — detail
//   POST   /api/v1/agents/:id/activate        — status draft → active
//   POST   /api/v1/agents/:id/disable         — status → disabled
//   POST   /api/v1/agents/:id/kill-switch     — flip kill_switch
//
//   POST   /api/v1/knowledge/collections      — create collection
//   POST   /api/v1/knowledge/collections/:cid/sources — enqueue source ingest
//
//   POST   /api/v1/agents/:id/sessions        — open session
//   POST   /api/v1/sessions/:id/end           — end session
//   GET    /api/v1/sessions/:id/messages      — history
//
// The LLM runtime itself (prompt assembly, provider call, tool execution)
// is a separate package landing in a follow-up sprint. S15 ships the data
// plane + session lifecycle so the runtime has something to write into.
package agents

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
	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *agentrepo.Repository
	bus  *event.Bus
}

func New(repo *agentrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/agents", h.listAgents)
	r.Post("/agents", h.createAgent)
	r.Get("/agents/{id}", h.getAgent)
	r.Post("/agents/{id}/activate", h.activate)
	r.Post("/agents/{id}/disable", h.disable)
	r.Post("/agents/{id}/kill-switch", h.killSwitch)

	r.Post("/knowledge/collections", h.createCollection)
	r.Post("/knowledge/collections/{cid}/sources", h.enqueueSource)

	r.Post("/agents/{id}/sessions", h.openSession)
	r.Post("/sessions/{id}/end", h.endSession)
	r.Get("/sessions/{id}/messages", h.listMessages)
}

// -------- agents DTOs ------------------------------------------------

type agentView struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	Model           string    `json:"model"`
	Temperature     float64   `json:"temperature"`
	MaxOutputTokens int       `json:"max_output_tokens"`
	ToolsAllowlist  []string  `json:"tools_allowlist"`
	KillSwitch      bool      `json:"kill_switch"`
	Status          string    `json:"status"`
}

type createAgentReq struct {
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	SystemPrompt    string   `json:"system_prompt"`
	Model           string   `json:"model"`
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"max_output_tokens,omitempty"`
	ToolsAllowlist  []string `json:"tools_allowlist,omitempty"`
}

type killSwitchReq struct {
	Enabled bool `json:"enabled"`
}

type createCollectionReq struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type enqueueSourceReq struct {
	Kind  string          `json:"kind"`
	Title string          `json:"title"`
	URI   *string         `json:"uri,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type openSessionReq struct {
	LeadID         *uuid.UUID `json:"lead_id,omitempty"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
}

// -------- agent handlers ---------------------------------------------

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.ListAgents(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list agents")
		return
	}
	out := make([]agentView, len(list))
	for i, a := range list {
		out[i] = toAgentView(a)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) createAgent(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createAgentReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	a, err := h.repo.CreateAgent(r.Context(), agentrepo.CreateAgentInput{
		OrganizationID: orgID, Name: body.Name, Description: body.Description,
		SystemPrompt: body.SystemPrompt, Model: body.Model,
		Temperature: body.Temperature, MaxOutputTokens: body.MaxOutputTokens,
		ToolsAllowlist: body.ToolsAllowlist,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_AGENT", err.Error())
		return
	}
	h.publish(orgID, a.ID, "agent.created", toAgentView(a))
	httpx.WriteJSON(w, http.StatusCreated, toAgentView(a))
}

func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	a, err := h.repo.GetAgent(r.Context(), orgID, id)
	if errors.Is(err, agentrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load agent")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAgentView(a))
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "active", "agent.activated")
}
func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "disabled", "agent.disabled")
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status, evt string) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.repo.SetAgentStatus(r.Context(), orgID, id, status); err != nil {
		if errors.Is(err, agentrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not set status")
		return
	}
	h.publish(orgID, id, evt, map[string]string{"status": status})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) killSwitch(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body killSwitchReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := h.repo.SetKillSwitch(r.Context(), orgID, id, body.Enabled); err != nil {
		if errors.Is(err, agentrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not flip kill switch")
		return
	}
	h.publish(orgID, id, "agent.kill_switch", map[string]bool{"enabled": body.Enabled})
	w.WriteHeader(http.StatusNoContent)
}

// -------- knowledge handlers -----------------------------------------

func (h *Handler) createCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createCollectionReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if body.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_COLLECTION", "name is required")
		return
	}
	id, err := h.repo.CreateCollection(r.Context(), orgID, body.Name, body.Description)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_COLLECTION", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id, "name": body.Name})
}

func (h *Handler) enqueueSource(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	cid, ok := parseID(w, r, "cid")
	if !ok {
		return
	}
	var body enqueueSourceReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	id, err := h.repo.EnqueueSource(r.Context(), agentrepo.EnqueueSourceInput{
		OrganizationID: orgID, CollectionID: cid,
		Kind: body.Kind, Title: body.Title, URI: body.URI, Metadata: body.Metadata,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_SOURCE", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "knowledge.source_enqueued", TenantID: orgID, EntityType: "knowledge_source",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	// 202: the ingest worker picks it up. Actual embedding population
	// happens asynchronously.
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"id": id, "status": "queued"})
}

// -------- session handlers -------------------------------------------

func (h *Handler) openSession(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	agentID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body openSessionReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	s, err := h.repo.OpenSession(r.Context(), orgID, agentID, body.LeadID, body.ConversationID)
	if errors.Is(err, agentrepo.ErrKillSwitch) {
		httpx.WriteError(w, http.StatusForbidden, "AGENT_INACTIVE", "agent is disabled or kill_switch is on")
		return
	}
	if errors.Is(err, agentrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not open session")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "agent_session.opened", TenantID: orgID, EntityType: "agent_session",
		EntityID: &s.ID, OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":          s.ID,
		"agent_id":    s.AgentID,
		"state":       s.State,
		"started_at":  s.StartedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) endSession(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.repo.EndSession(r.Context(), orgID, id); err != nil {
		if errors.Is(err, agentrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "session not found or already ended")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not end session")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "agent_session.ended", TenantID: orgID, EntityType: "agent_session",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ms, err := h.repo.ListMessages(r.Context(), orgID, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list messages")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": ms})
}

// -------- helpers ----------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", key+" must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "agent",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func toAgentView(a agentrepo.Agent) agentView {
	return agentView{
		ID: a.ID, Name: a.Name, Description: a.Description, Model: a.Model,
		Temperature: a.Temperature, MaxOutputTokens: a.MaxOutputTokens,
		ToolsAllowlist: a.ToolsAllowlist, KillSwitch: a.KillSwitch, Status: a.Status,
	}
}

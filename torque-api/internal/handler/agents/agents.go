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
	"github.com/milennials/torque-api/internal/service/ai"
	"github.com/milennials/torque-api/internal/service/knowledge"
	"github.com/milennials/torque-api/internal/ws"
)

type Handler struct {
	repo *agentrepo.Repository
	bus  *event.Bus
	// ingest is optional — wired at boot when the embedder is available.
	// nil ingest = the enqueue endpoint still persists the source row
	// but never flips it to ready. Useful for keeping the CRUD paths
	// operational in a future config where ingest is delegated to a
	// separate service.
	ingest *knowledge.Service
}

func New(repo *agentrepo.Repository, bus *event.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// WithIngest attaches the ingest service. Returns the same handler
// (fluent style) to keep main.go terse.
func (h *Handler) WithIngest(ingest *knowledge.Service) *Handler {
	h.ingest = ingest
	return h
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/agents", h.listAgents)
	r.Post("/agents", h.createAgent)
	r.Get("/agents/{id}", h.getAgent)
	r.Post("/agents/{id}/activate", h.activate)
	r.Post("/agents/{id}/disable", h.disable)
	r.Post("/agents/{id}/kill-switch", h.killSwitch)
	// S39 — bind / unbind the agent's RAG collection.
	r.Put("/agents/{id}/knowledge-collection", h.bindCollection)

	r.Get("/knowledge/collections", h.listCollections)
	r.Post("/knowledge/collections", h.createCollection)
	r.Get("/knowledge/collections/{cid}/sources", h.listSources)
	r.Post("/knowledge/collections/{cid}/sources", h.enqueueSource)

	// S40 — agent triggers (auto-assignment rules).
	r.Get("/agents/{id}/triggers", h.listTriggers)
	r.Post("/agents/{id}/triggers", h.createTrigger)
	r.Patch("/triggers/{tid}", h.updateTrigger)
	r.Delete("/triggers/{tid}", h.deleteTrigger)

	r.Post("/agents/{id}/sessions", h.openSession)
	r.Post("/sessions/{id}/end", h.endSession)
	r.Get("/sessions/{id}/messages", h.listMessages)
}

// -------- agents DTOs ------------------------------------------------

type agentView struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	// SystemPrompt exposed as of S38 so the Playground editor can render
	// + edit the persisted prompt without a separate roundtrip. Admin-only
	// endpoint already guards access; no secret leakage by contract.
	SystemPrompt    string    `json:"system_prompt"`
	Model           string    `json:"model"`
	Temperature     float64   `json:"temperature"`
	MaxOutputTokens int       `json:"max_output_tokens"`
	ToolsAllowlist  []string  `json:"tools_allowlist"`
	KillSwitch      bool      `json:"kill_switch"`
	Status          string    `json:"status"`
	// KnowledgeCollectionID binds the agent to a RAG collection (S39).
	// When non-nil, the playground/production SSE handler runs topK
	// retrieval and prepends context before the LLM call.
	KnowledgeCollectionID *uuid.UUID `json:"knowledge_collection_id,omitempty"`
	// S41 — TTS config. tts_enabled alone is not sufficient; the
	// outbound-audio path also needs tts_voice_id. The UI may set
	// only one of the two mid-flow (e.g. disable while keeping the
	// voice_id remembered) so both are reported independently.
	TTSEnabled bool    `json:"tts_enabled"`
	TTSVoiceID *string `json:"tts_voice_id,omitempty"`
}

type collectionView struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	SourceCount int       `json:"source_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type sourceView struct {
	ID           uuid.UUID  `json:"id"`
	CollectionID uuid.UUID  `json:"collection_id"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	URI          *string    `json:"uri,omitempty"`
	Status       string     `json:"status"`
	Error        *string    `json:"error,omitempty"`
	IngestedAt   *time.Time `json:"ingested_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type bindCollectionReq struct {
	CollectionID *uuid.UUID `json:"collection_id"`
}

type triggerView struct {
	ID          uuid.UUID     `json:"id"`
	AgentID     uuid.UUID     `json:"agent_id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Priority    int           `json:"priority"`
	Filter      ai.FilterSpec `json:"filter"`
	IsActive    bool          `json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type createTriggerReq struct {
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Priority    int           `json:"priority,omitempty"`
	Filter      ai.FilterSpec `json:"filter"`
}

type updateTriggerReq struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Priority    *int           `json:"priority,omitempty"`
	Filter      *ai.FilterSpec `json:"filter,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
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
	Kind     string          `json:"kind"`
	Title    string          `json:"title"`
	URI      *string         `json:"uri,omitempty"`
	// Content is the inline text payload for kind=text|markdown. URL /
	// PDF / DOCX sources defer fetching to a future sprint; passing
	// Content for those kinds is ignored (URL fetch wins when wired).
	Content  string          `json:"content,omitempty"`
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

func (h *Handler) listCollections(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.ListCollections(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list collections")
		return
	}
	out := make([]collectionView, len(list))
	for i, c := range list {
		out[i] = collectionView{
			ID: c.ID, Name: c.Name, Description: c.Description,
			SourceCount: c.SourceCount,
			CreatedAt:   c.CreatedAt, UpdatedAt: c.UpdatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

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
	h.bus.Publish(ws.Event{
		Type: "knowledge.collection_created", TenantID: orgID, EntityType: "knowledge_collection",
		EntityID: &id, OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id, "name": body.Name})
}

func (h *Handler) listSources(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	cid, ok := parseID(w, r, "cid")
	if !ok {
		return
	}
	list, err := h.repo.ListSources(r.Context(), orgID, cid)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list sources")
		return
	}
	out := make([]sourceView, len(list))
	for i, s := range list {
		out[i] = sourceView{
			ID: s.ID, CollectionID: s.CollectionID, Kind: s.Kind, Title: s.Title, URI: s.URI,
			Status: s.Status, Error: s.Error, IngestedAt: s.IngestedAt,
			CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// bindCollection updates agent.knowledge_collection_id. Passing `null`
// unbinds the agent (retrieval disabled next request). Ownership of the
// collection is revalidated at the repo layer — cross-tenant set attempts
// return 404.
func (h *Handler) bindCollection(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body bindCollectionReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := h.repo.SetAgentKnowledgeCollection(r.Context(), orgID, id, body.CollectionID); err != nil {
		if errors.Is(err, agentrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent or collection not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not bind collection")
		return
	}
	a, err := h.repo.GetAgent(r.Context(), orgID, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not reload agent")
		return
	}
	h.publish(orgID, id, "agent.knowledge_bound", toAgentView(a))
	httpx.WriteJSON(w, http.StatusOK, toAgentView(a))
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
	// S39 — for inline kinds (text / markdown) require Content. URL
	// kinds require URI with https scheme (validated in the repo).
	// Rejecting at the handler produces a nicer error than surfacing
	// a later ingest failure.
	switch body.Kind {
	case "text", "markdown":
		if body.Content == "" {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_SOURCE",
				"content is required for text/markdown kinds")
			return
		}
	case "url":
		if body.URI == nil || *body.URI == "" {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_SOURCE",
				"uri is required for url kind")
			return
		}
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
	// Kick off detached ingest for inline kinds. URL fetch is not yet
	// implemented — those sources stay queued until a follow-up sprint
	// wires the fetch + HTTPS allowlist path.
	if h.ingest != nil && (body.Kind == "text" || body.Kind == "markdown") && body.Content != "" {
		h.ingest.RunDetached(orgID, id, body.Content)
	}
	// 202: the ingest runs in the background. The UI polls source
	// status + subscribes to WS for real-time transitions.
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"id": id, "status": "queued"})
}

// -------- trigger handlers -------------------------------------------

func toTriggerView(t agentrepo.AgentTrigger) triggerView {
	return triggerView{
		ID: t.ID, AgentID: t.AgentID, Name: t.Name, Description: t.Description,
		Priority: t.Priority, Filter: t.Filter, IsActive: t.IsActive,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func (h *Handler) listTriggers(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	agentID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	list, err := h.repo.ListTriggersByAgent(r.Context(), orgID, agentID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list triggers")
		return
	}
	out := make([]triggerView, len(list))
	for i, t := range list {
		out[i] = toTriggerView(t)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) createTrigger(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	agentID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body createTriggerReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	in := agentrepo.CreateTriggerInput{
		OrganizationID: orgID, AgentID: agentID,
		Name: body.Name, Description: body.Description, Priority: body.Priority, Filter: body.Filter,
	}
	if sess, ok := mw.SessionFrom(r.Context()); ok && sess.TeamMemberID != uuid.Nil {
		tm := sess.TeamMemberID
		in.CreatedBy = &tm
	}
	t, err := h.repo.CreateTrigger(r.Context(), in)
	if errors.Is(err, agentrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "agent not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TRIGGER", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "agent_trigger.created", TenantID: orgID, EntityType: "agent_trigger",
		EntityID: &t.ID, Patch: toTriggerView(t), OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusCreated, toTriggerView(t))
}

func (h *Handler) updateTrigger(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	tid, ok := parseID(w, r, "tid")
	if !ok {
		return
	}
	var body updateTriggerReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	t, err := h.repo.UpdateTrigger(r.Context(), orgID, tid, agentrepo.UpdateTriggerInput{
		Name: body.Name, Description: body.Description, Priority: body.Priority,
		Filter: body.Filter, IsActive: body.IsActive,
	})
	if errors.Is(err, agentrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "trigger not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TRIGGER", err.Error())
		return
	}
	h.bus.Publish(ws.Event{
		Type: "agent_trigger.updated", TenantID: orgID, EntityType: "agent_trigger",
		EntityID: &t.ID, Patch: toTriggerView(t), OccurredAt: time.Now().UTC(),
	})
	httpx.WriteJSON(w, http.StatusOK, toTriggerView(t))
}

func (h *Handler) deleteTrigger(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	tid, ok := parseID(w, r, "tid")
	if !ok {
		return
	}
	if err := h.repo.DeleteTrigger(r.Context(), orgID, tid); err != nil {
		if errors.Is(err, agentrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "trigger not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete trigger")
		return
	}
	h.bus.Publish(ws.Event{
		Type: "agent_trigger.deleted", TenantID: orgID, EntityType: "agent_trigger",
		EntityID: &tid, OccurredAt: time.Now().UTC(),
	})
	w.WriteHeader(http.StatusNoContent)
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
		ID: a.ID, Name: a.Name, Description: a.Description,
		SystemPrompt: a.SystemPrompt,
		Model: a.Model, Temperature: a.Temperature, MaxOutputTokens: a.MaxOutputTokens,
		ToolsAllowlist: a.ToolsAllowlist, KillSwitch: a.KillSwitch, Status: a.Status,
		KnowledgeCollectionID: a.KnowledgeCollectionID,
		TTSEnabled:            a.TTSEnabled,
		TTSVoiceID:            a.TTSVoiceID,
	}
}

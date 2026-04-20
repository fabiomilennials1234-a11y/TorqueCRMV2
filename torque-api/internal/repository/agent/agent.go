// Package agent persists F06 Copilot configuration + sessions.
//
// Ingest of knowledge sources + embeddings is a worker-scheduled job (kind
// `copilot.ingest_source`). The repository here covers the synchronous
// surface: CRUD on agents, collections, sources; session + message history.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrKillSwitch = errors.New("agent kill_switch is active")
)

type Agent struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Name            string
	Description     *string
	SystemPrompt    string
	Model           string
	Temperature     float64
	MaxOutputTokens int
	ToolsAllowlist  []string
	KillSwitch      bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AgentSession struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	AgentID        uuid.UUID
	LeadID         *uuid.UUID
	ConversationID *uuid.UUID
	State          string
	StartedAt      time.Time
	EndedAt        *time.Time
}

type AgentMessage struct {
	ID           uuid.UUID
	SessionID    uuid.UUID
	Role         string
	Content      string
	ToolName     *string
	ToolPayload  []byte
	TokensInput  *int
	TokensOutput *int
	LatencyMs    *int
	OccurredAt   time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- agents -----------------------------------------------------

type CreateAgentInput struct {
	OrganizationID   uuid.UUID
	Name             string
	Description      *string
	SystemPrompt     string
	Model            string
	Temperature      float64
	MaxOutputTokens  int
	ToolsAllowlist   []string
}

// CreateAgent inserts a draft agent. Status defaults to draft; tenant must
// POST /activate before serving traffic.
func (r *Repository) CreateAgent(ctx context.Context, in CreateAgentInput) (Agent, error) {
	if in.Name == "" || in.SystemPrompt == "" || in.Model == "" {
		return Agent{}, errors.New("name, system_prompt and model are required")
	}
	if in.Temperature == 0 {
		in.Temperature = 0.3
	}
	if in.MaxOutputTokens == 0 {
		in.MaxOutputTokens = 1024
	}
	if in.ToolsAllowlist == nil {
		in.ToolsAllowlist = []string{}
	}
	const q = `
		INSERT INTO agents (
		  organization_id, name, description, system_prompt, model,
		  temperature, max_output_tokens, tools_allowlist
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, status::text, created_at, updated_at
	`
	var a Agent
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.Name, in.Description, in.SystemPrompt, in.Model,
		in.Temperature, in.MaxOutputTokens, in.ToolsAllowlist,
	).Scan(&a.ID, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Agent{}, fmt.Errorf("create agent: %w", err)
	}
	a.OrganizationID = in.OrganizationID
	a.Name = in.Name
	a.Description = in.Description
	a.SystemPrompt = in.SystemPrompt
	a.Model = in.Model
	a.Temperature = in.Temperature
	a.MaxOutputTokens = in.MaxOutputTokens
	a.ToolsAllowlist = in.ToolsAllowlist
	return a, nil
}

// GetAgent returns an agent within the tenant.
func (r *Repository) GetAgent(ctx context.Context, orgID, id uuid.UUID) (Agent, error) {
	const q = `
		SELECT id, organization_id, name, description, system_prompt, model,
		       temperature, max_output_tokens, tools_allowlist,
		       kill_switch, status::text, created_at, updated_at
		  FROM agents
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var a Agent
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&a.ID, &a.OrganizationID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
		&a.Temperature, &a.MaxOutputTokens, &a.ToolsAllowlist,
		&a.KillSwitch, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Agent{}, ErrNotFound
	}
	if err != nil {
		return Agent{}, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

// ListAgents returns every non-deleted agent in the tenant.
func (r *Repository) ListAgents(ctx context.Context, orgID uuid.UUID) ([]Agent, error) {
	const q = `
		SELECT id, organization_id, name, description, system_prompt, model,
		       temperature, max_output_tokens, tools_allowlist,
		       kill_switch, status::text, created_at, updated_at
		  FROM agents
		 WHERE organization_id = $1
		 ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	out := make([]Agent, 0, 4)
	for rows.Next() {
		var a Agent
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
			&a.Temperature, &a.MaxOutputTokens, &a.ToolsAllowlist,
			&a.KillSwitch, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SetAgentStatus transitions the status (draft ↔ active, * → disabled).
func (r *Repository) SetAgentStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE agents SET status = $3::agent_status WHERE organization_id = $1 AND id = $2`,
		orgID, id, status,
	)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetKillSwitch flips the kill_switch flag. When true, all in-flight sessions
// should terminate at their next tool call — the agent runtime checks this
// before every LLM call.
func (r *Repository) SetKillSwitch(ctx context.Context, orgID, id uuid.UUID, enabled bool) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE agents SET kill_switch = $3 WHERE organization_id = $1 AND id = $2`,
		orgID, id, enabled,
	)
	if err != nil {
		return fmt.Errorf("kill switch: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- knowledge base --------------------------------------------

// CreateCollection adds a knowledge collection scoped to the tenant.
func (r *Repository) CreateCollection(ctx context.Context, orgID uuid.UUID, name string, description *string) (uuid.UUID, error) {
	const q = `
		INSERT INTO knowledge_collections (organization_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, orgID, name, description).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("create collection: %w", err)
	}
	return id, nil
}

// EnqueueSource creates a queued knowledge_source. Ingestion is a separate
// worker kind (copilot.ingest_source).
type EnqueueSourceInput struct {
	OrganizationID uuid.UUID
	CollectionID   uuid.UUID
	Kind           string
	Title          string
	URI            *string
	Metadata       json.RawMessage
}

func (r *Repository) EnqueueSource(ctx context.Context, in EnqueueSourceInput) (uuid.UUID, error) {
	meta := in.Metadata
	if len(meta) == 0 {
		meta = []byte(`{}`)
	}
	const q = `
		INSERT INTO knowledge_sources (
		  organization_id, collection_id, kind, title, uri, metadata
		) VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.CollectionID, in.Kind, in.Title, in.URI, meta,
	).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("enqueue source: %w", err)
	}
	return id, nil
}

// -------- sessions + messages ---------------------------------------

// OpenSession creates a new session with an agent. Refuses if kill_switch is on.
func (r *Repository) OpenSession(
	ctx context.Context, orgID, agentID uuid.UUID, leadID, conversationID *uuid.UUID,
) (AgentSession, error) {
	a, err := r.GetAgent(ctx, orgID, agentID)
	if err != nil {
		return AgentSession{}, err
	}
	if a.KillSwitch || a.Status != "active" {
		return AgentSession{}, ErrKillSwitch
	}
	const q = `
		INSERT INTO agent_sessions (organization_id, agent_id, lead_id, conversation_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id, state::text, started_at
	`
	var s AgentSession
	err = r.pool.QueryRow(ctx, q, orgID, agentID, leadID, conversationID).Scan(&s.ID, &s.State, &s.StartedAt)
	if err != nil {
		return AgentSession{}, fmt.Errorf("open session: %w", err)
	}
	s.OrganizationID = orgID
	s.AgentID = agentID
	s.LeadID = leadID
	s.ConversationID = conversationID
	return s, nil
}

// EndSession transitions state → ended and stamps ended_at.
func (r *Repository) EndSession(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE agent_sessions
		    SET state = 'ended', ended_at = now()
		  WHERE organization_id = $1 AND id = $2 AND state = 'open'`,
		orgID, id,
	)
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendMessage adds a turn to the session. Used by the agent runtime for
// both user and assistant + tool call turns. Input/output token counters
// feed billing + observability dashboards.
type AppendMessageInput struct {
	OrganizationID uuid.UUID
	SessionID      uuid.UUID
	Role           string
	Content        string
	ToolName       *string
	ToolPayload    json.RawMessage
	TokensInput    *int
	TokensOutput   *int
	LatencyMs      *int
}

func (r *Repository) AppendMessage(ctx context.Context, in AppendMessageInput) (AgentMessage, error) {
	if in.Role != "user" && in.Role != "assistant" && in.Role != "system" && in.Role != "tool" {
		return AgentMessage{}, fmt.Errorf("invalid role: %s", in.Role)
	}
	var payload any
	if len(in.ToolPayload) > 0 {
		payload = []byte(in.ToolPayload)
	}
	const q = `
		INSERT INTO agent_messages (
		  organization_id, session_id, role, content,
		  tool_name, tool_payload, tokens_input, tokens_output, latency_ms
		) VALUES ($1,$2,$3::agent_message_role,$4,$5,$6,$7,$8,$9)
		RETURNING id, occurred_at
	`
	var m AgentMessage
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.SessionID, in.Role, in.Content,
		in.ToolName, payload, in.TokensInput, in.TokensOutput, in.LatencyMs,
	).Scan(&m.ID, &m.OccurredAt)
	if err != nil {
		return AgentMessage{}, fmt.Errorf("append agent message: %w", err)
	}
	m.SessionID = in.SessionID
	m.Role = in.Role
	m.Content = in.Content
	m.ToolName = in.ToolName
	m.ToolPayload = in.ToolPayload
	m.TokensInput = in.TokensInput
	m.TokensOutput = in.TokensOutput
	m.LatencyMs = in.LatencyMs
	return m, nil
}

// ListMessages returns the session history (oldest first).
func (r *Repository) ListMessages(ctx context.Context, orgID, sessionID uuid.UUID) ([]AgentMessage, error) {
	const q = `
		SELECT id, session_id, role::text, content,
		       tool_name, tool_payload, tokens_input, tokens_output, latency_ms,
		       occurred_at
		  FROM agent_messages
		 WHERE organization_id = $1 AND session_id = $2
		 ORDER BY occurred_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	out := make([]AgentMessage, 0, 16)
	for rows.Next() {
		var m AgentMessage
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.Content,
			&m.ToolName, &m.ToolPayload, &m.TokensInput, &m.TokensOutput, &m.LatencyMs,
			&m.OccurredAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

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
	"strings"
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
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	Name                  string
	Description           *string
	SystemPrompt          string
	Model                 string
	Temperature           float64
	MaxOutputTokens       int
	ToolsAllowlist        []string
	KillSwitch            bool
	Status                string
	KnowledgeCollectionID *uuid.UUID
	TTSEnabled            bool
	TTSVoiceID            *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
		       kill_switch, status::text, knowledge_collection_id,
		       tts_enabled, tts_voice_id,
		       created_at, updated_at
		  FROM agents
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var a Agent
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&a.ID, &a.OrganizationID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
		&a.Temperature, &a.MaxOutputTokens, &a.ToolsAllowlist,
		&a.KillSwitch, &a.Status, &a.KnowledgeCollectionID,
		&a.TTSEnabled, &a.TTSVoiceID,
		&a.CreatedAt, &a.UpdatedAt,
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
		       kill_switch, status::text, knowledge_collection_id,
		       tts_enabled, tts_voice_id,
		       created_at, updated_at
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
			&a.KillSwitch, &a.Status, &a.KnowledgeCollectionID,
			&a.TTSEnabled, &a.TTSVoiceID,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAgentInput is the patch shape for UpdateAgent. Only non-nil fields
// are applied; status + kill_switch have dedicated methods to keep the
// state transitions audit-loud.
type UpdateAgentInput struct {
	Name            *string
	Description     *string
	SystemPrompt    *string
	Model           *string
	Temperature     *float64
	MaxOutputTokens *int
	ToolsAllowlist  *[]string
	// S41 — TTS config. *string with value=="" unsets the voice id
	// (CHECK allows NULL only — callers should pass nil for unset).
	TTSEnabled *bool
	TTSVoiceID *string
}

// UpdateAgent applies a partial patch. Refuses zero-field patches with a
// simple Get round-trip so callers see the current row either way.
func (r *Repository) UpdateAgent(ctx context.Context, orgID, id uuid.UUID, in UpdateAgentInput) (Agent, error) {
	sets := make([]string, 0, 7)
	args := []any{orgID, id}
	idx := 3
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if len(n) < 2 || len(n) > 120 {
			return Agent{}, errors.New("name must be 2-120 chars")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, n)
		idx++
	}
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *in.Description)
		idx++
	}
	if in.SystemPrompt != nil {
		p := strings.TrimSpace(*in.SystemPrompt)
		if len(p) < 10 || len(p) > 16000 {
			return Agent{}, errors.New("system_prompt must be 10-16000 chars")
		}
		sets = append(sets, fmt.Sprintf("system_prompt = $%d", idx))
		args = append(args, p)
		idx++
	}
	if in.Model != nil {
		m := strings.TrimSpace(*in.Model)
		if len(m) < 2 || len(m) > 80 {
			return Agent{}, errors.New("model must be 2-80 chars")
		}
		sets = append(sets, fmt.Sprintf("model = $%d", idx))
		args = append(args, m)
		idx++
	}
	if in.Temperature != nil {
		if *in.Temperature < 0 || *in.Temperature > 2 {
			return Agent{}, errors.New("temperature must be between 0 and 2")
		}
		sets = append(sets, fmt.Sprintf("temperature = $%d", idx))
		args = append(args, *in.Temperature)
		idx++
	}
	if in.MaxOutputTokens != nil {
		if *in.MaxOutputTokens < 16 || *in.MaxOutputTokens > 8192 {
			return Agent{}, errors.New("max_output_tokens must be between 16 and 8192")
		}
		sets = append(sets, fmt.Sprintf("max_output_tokens = $%d", idx))
		args = append(args, *in.MaxOutputTokens)
		idx++
	}
	if in.ToolsAllowlist != nil {
		sets = append(sets, fmt.Sprintf("tools_allowlist = $%d", idx))
		args = append(args, *in.ToolsAllowlist)
		idx++
	}
	if in.TTSEnabled != nil {
		sets = append(sets, fmt.Sprintf("tts_enabled = $%d", idx))
		args = append(args, *in.TTSEnabled)
		idx++
	}
	if in.TTSVoiceID != nil {
		v := strings.TrimSpace(*in.TTSVoiceID)
		if v == "" {
			sets = append(sets, fmt.Sprintf("tts_voice_id = $%d", idx))
			args = append(args, nil)
		} else {
			if len(v) < 2 || len(v) > 80 {
				return Agent{}, errors.New("tts_voice_id must be 2-80 chars")
			}
			sets = append(sets, fmt.Sprintf("tts_voice_id = $%d", idx))
			args = append(args, v)
		}
		// idx not bumped — TTSVoiceID is the last optional field; no further
		// $N placeholders are emitted after this branch.
	}
	if len(sets) == 0 {
		return r.GetAgent(ctx, orgID, id)
	}
	q := fmt.Sprintf(
		`UPDATE agents SET %s WHERE organization_id = $1 AND id = $2`,
		strings.Join(sets, ", "),
	)
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return Agent{}, fmt.Errorf("update agent: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return Agent{}, ErrNotFound
	}
	return r.GetAgent(ctx, orgID, id)
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

// allowedURISchemes — ingest worker may only fetch HTTPS. http is refused
// even in dev to prevent an ingest route from becoming an SSRF/MITM primitive.
// file://, gopher://, dict://, ldap://, localhost/169.254.* are rejected by
// the worker at fetch time; this is the first line of defense.
var allowedURISchemes = map[string]bool{"https": true}

// EnqueueSource verifies collection ownership AND validates the URI scheme
// before writing. A member of org A who probes for a collection_id belonging
// to org B can no longer enqueue a row under the foreign collection.
func (r *Repository) EnqueueSource(ctx context.Context, in EnqueueSourceInput) (uuid.UUID, error) {
	if in.Kind == "" || in.Title == "" {
		return uuid.Nil, errors.New("kind and title are required")
	}
	// Ownership: collection_id must belong to the caller's tenant.
	var ownedBy uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT organization_id FROM knowledge_collections WHERE id = $1 LIMIT 1`,
		in.CollectionID,
	).Scan(&ownedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("collection ownership: %w", err)
	}
	if ownedBy != in.OrganizationID {
		return uuid.Nil, ErrNotFound
	}

	// URI scheme allowlist (when the source is remote).
	if in.URI != nil && *in.URI != "" {
		idx := strings.Index(*in.URI, "://")
		if idx <= 0 {
			return uuid.Nil, errors.New("uri must include an https:// scheme")
		}
		scheme := strings.ToLower((*in.URI)[:idx])
		if !allowedURISchemes[scheme] {
			return uuid.Nil, fmt.Errorf("uri scheme %q not allowed (https only)", scheme)
		}
	}

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

// PlaygroundTurnInput captures the user + assistant turn produced by
// one playground SSE call. Both messages land in a single (ad-hoc)
// agent_session so the /sessions/:id/messages history endpoint can
// replay them, and tokens_input/tokens_output on the assistant row
// feed the S42 metrics + S52 AI-budget meter.
type PlaygroundTurnInput struct {
	OrganizationID uuid.UUID
	AgentID        uuid.UUID
	// LeadID / ConversationID — nil for anonymous playground turns.
	LeadID         *uuid.UUID
	ConversationID *uuid.UUID
	UserContent    string
	AssistantContent string
	TokensInput    int
	TokensOutput   int
	LatencyMs      int
}

// PersistPlaygroundTurn opens a short-lived session, appends the user
// and assistant messages (with the token counters on the assistant
// row), and closes the session. All inside one pgx transaction so a
// partial failure leaves nothing behind. Returns the session id the
// caller surfaces on the terminal SSE frame for traceability.
func (r *Repository) PersistPlaygroundTurn(ctx context.Context, in PlaygroundTurnInput) (uuid.UUID, error) {
	if in.OrganizationID == uuid.Nil || in.AgentID == uuid.Nil {
		return uuid.Nil, errors.New("org and agent required")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // idempotent after commit

	var sessID uuid.UUID
	if err := tx.QueryRow(ctx,
		`INSERT INTO agent_sessions (organization_id, agent_id, lead_id, conversation_id, state)
		 VALUES ($1,$2,$3,$4,'ended')
		 RETURNING id`,
		in.OrganizationID, in.AgentID, in.LeadID, in.ConversationID,
	).Scan(&sessID); err != nil {
		return uuid.Nil, fmt.Errorf("insert session: %w", err)
	}

	// User turn — no token counters (we don't bill input context on
	// the user row; providers report input token count on the
	// assistant turn's terminal frame).
	if _, err := tx.Exec(ctx,
		`INSERT INTO agent_messages (organization_id, session_id, role, content)
		 VALUES ($1,$2,'user',$3)`,
		in.OrganizationID, sessID, in.UserContent,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert user msg: %w", err)
	}

	// Assistant turn — carries the usage counters so aggregate metrics +
	// quota increments have a single source of truth.
	if _, err := tx.Exec(ctx,
		`INSERT INTO agent_messages (
		   organization_id, session_id, role, content,
		   tokens_input, tokens_output, latency_ms
		 ) VALUES ($1,$2,'assistant',$3,$4,$5,$6)`,
		in.OrganizationID, sessID, in.AssistantContent,
		in.TokensInput, in.TokensOutput, in.LatencyMs,
	); err != nil {
		return uuid.Nil, fmt.Errorf("insert assistant msg: %w", err)
	}

	// Stamp ended_at so the session is closed the moment it's persisted
	// (playground turns are stateless — nothing opens a follow-up turn
	// against this same session id).
	if _, err := tx.Exec(ctx,
		`UPDATE agent_sessions SET ended_at = now() WHERE id = $1`, sessID,
	); err != nil {
		return uuid.Nil, fmt.Errorf("close session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit tx: %w", err)
	}
	return sessID, nil
}

// AgentMetrics is the aggregation shape surfaced by GET /agents/:id/metrics.
// Window is closed-open [Since, Until).
type AgentMetrics struct {
	Since           time.Time
	Until           time.Time
	TotalSessions   int
	TotalMessages   int
	TokensInput     int64
	TokensOutput    int64
	AvgLatencyMs    *float64 // nil when no assistant turns in window
	// SessionsByState bundles open / ended / escalated counts so the UI
	// can render a tiny stacked bar without a second round-trip.
	SessionsByState map[string]int
}

// GetAgentMetrics aggregates sessions + messages for an agent over
// a time window. Tenant-scoped. Both tables are indexed on
// (organization_id, created_at|occurred_at DESC) so the window scan
// stays selective.
func (r *Repository) GetAgentMetrics(ctx context.Context, orgID, agentID uuid.UUID, since, until time.Time) (AgentMetrics, error) {
	var m AgentMetrics
	m.Since = since
	m.Until = until
	m.SessionsByState = map[string]int{}

	// Sessions + state breakdown in one pass.
	sessRows, err := r.pool.Query(ctx,
		`SELECT state::text, COUNT(*)
		   FROM agent_sessions
		  WHERE organization_id = $1
		    AND agent_id = $2
		    AND started_at >= $3 AND started_at < $4
		  GROUP BY state`,
		orgID, agentID, since, until,
	)
	if err != nil {
		return AgentMetrics{}, fmt.Errorf("agent metrics sessions: %w", err)
	}
	defer sessRows.Close()
	for sessRows.Next() {
		var state string
		var n int
		if err := sessRows.Scan(&state, &n); err != nil {
			return AgentMetrics{}, err
		}
		m.SessionsByState[state] = n
		m.TotalSessions += n
	}
	if err := sessRows.Err(); err != nil {
		return AgentMetrics{}, err
	}

	// Message-level aggregates. Constrain by session → agent_id to
	// avoid counting messages from other agents in the same tenant.
	var avgLat *float64
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(COALESCE(m.tokens_input, 0)), 0),
		        COALESCE(SUM(COALESCE(m.tokens_output, 0)), 0),
		        AVG(m.latency_ms) FILTER (WHERE m.role = 'assistant')
		   FROM agent_messages m
		   JOIN agent_sessions s ON s.id = m.session_id
		  WHERE m.organization_id = $1
		    AND s.agent_id = $2
		    AND m.occurred_at >= $3 AND m.occurred_at < $4`,
		orgID, agentID, since, until,
	).Scan(&m.TotalMessages, &m.TokensInput, &m.TokensOutput, &avgLat)
	if err != nil {
		return AgentMetrics{}, fmt.Errorf("agent metrics messages: %w", err)
	}
	m.AvgLatencyMs = avgLat
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

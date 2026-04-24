// Action handlers for the workflow engine.
//
// S44 shipped these as stubs; S45 extended the set to 7; S52 turns
// every one of them into a real side-effect backed by a concrete
// repository or provider, with retry classification (ErrTransient /
// ErrNonRetryable) surfaced to the executor.
//
// Each handler accepts a Deps envelope (injected on dispatcher
// construction). A nil repository degrades the handler to a logged
// no-op emitting `wired:false` so tests that don't want a full DB
// wiring still exercise the happy path of the executor. Real
// production always passes a populated Deps.

package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
	inboxrepo "github.com/milennials/torque-api/internal/repository/inbox"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	taskrepo "github.com/milennials/torque-api/internal/repository/task"
	"github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/ws"
)

// ---------- send_message (WhatsApp via Evolution) -------------------

// SendMessageAction sends a real WhatsApp / SZ.Chat message via the
// injected MessagingProvider and records the outbound row in `messages`.
// Config schema:
//
//	{
//	  "channel_id":        "<uuid>",      // optional when channel_external_id provided
//	  "channel_external_id":"<instance>", // provider-side id (e.g. evolution instance)
//	  "template_id":       "<uuid?>",
//	  "body":              "Olá {{lead.name}}",
//	  "to_phone_field":    "phone",       // lead.* field carrying the recipient
//	  "kind":              "text"         // text | image | video | document
//	}
//
// Either body or template_id MUST be present. Placeholders `{{lead.*}}`
// / `{{input.*}}` / `{{prev.<step_id>.*}}` are resolved against the
// StepContext before dispatch.
type SendMessageAction struct {
	logger zerolog.Logger
	deps   Deps
}

// Kind implements ActionHandler.
func (*SendMessageAction) Kind() string { return "send_message" }

// sendMessageConfig is the parsed view of the step config.
type sendMessageConfig struct {
	ChannelID         string `json:"channel_id,omitempty"`
	ChannelExternalID string `json:"channel_external_id,omitempty"`
	TemplateID        string `json:"template_id,omitempty"`
	Body              string `json:"body,omitempty"`
	ToPhoneField      string `json:"to_phone_field,omitempty"`
	Kind              string `json:"kind,omitempty"`
	// Legacy field preserved so pre-S52 configs still parse.
	Template string `json:"template,omitempty"`
	// trigger_kind tags pass-through trigger nodes (lead_created,
	// message_received) that are no-op for send_message.
	TriggerKind string `json:"trigger_kind,omitempty"`
}

// Execute sends the message.
func (a *SendMessageAction) Execute(ctx context.Context, sc StepContext) (StepOutcome, error) {
	var cfg sendMessageConfig
	if err := json.Unmarshal(sc.Config, &cfg); err != nil {
		return failOutput("send_message", ErrNonRetryable, "invalid config: "+err.Error())
	}

	// Pass-through: trigger nodes (lead_created / message_received)
	// reach us as send_message kind because the canvas collapses the
	// node class — flow on without calling the provider.
	if cfg.TriggerKind != "" {
		out, _ := json.Marshal(map[string]any{
			"kind":    "send_message",
			"trigger": cfg.TriggerKind,
			"sent":    false,
			"skipped": "trigger_passthrough",
		})
		return StepOutcome{Output: out}, nil
	}

	if cfg.Body == "" && cfg.TemplateID == "" && cfg.Template == "" {
		return failOutput("send_message", ErrNonRetryable, "body or template_id required")
	}

	// Degrade gracefully when messaging not wired (tests, old boot).
	if a.deps.Messaging == nil {
		a.logger.Warn().
			Str("run_id", sc.RunID).
			Str("step_id", sc.StepID).
			Msg("send_message: messaging provider not wired — noop")
		out, _ := json.Marshal(map[string]any{
			"kind":   "send_message",
			"sent":   false,
			"wired":  false,
			"reason": "messaging_provider_not_configured",
		})
		return StepOutcome{Output: out}, nil
	}

	// Resolve recipient from lead facts.
	recipient, leadID := resolveRecipient(cfg, sc)
	if recipient == "" {
		return failOutput("send_message", ErrNonRetryable, "no recipient resolved from lead")
	}

	// Render placeholders against the full StepContext fact bag.
	body := renderTemplate(cfg.Body, sc)

	kind := cfg.Kind
	if kind == "" {
		kind = "text"
	}

	// Resolve channel external id: explicit config first, else read
	// the channels row by id (tenant-scoped).
	channelExt := cfg.ChannelExternalID
	if channelExt == "" && cfg.ChannelID != "" && a.deps.Pool != nil {
		orgUUID, oerr := uuid.Parse(sc.OrgID)
		chUUID, cerr := uuid.Parse(cfg.ChannelID)
		if oerr == nil && cerr == nil {
			if ext, err := lookupChannelExternalID(ctx, a.deps, orgUUID, chUUID); err == nil {
				channelExt = ext
			}
		}
	}
	if channelExt == "" {
		return failOutput("send_message", ErrNonRetryable, "channel_external_id unresolved")
	}

	res, err := a.deps.Messaging.SendMessage(ctx, integration.OutboundMessage{
		ChannelExternalID: channelExt,
		Recipient:         recipient,
		Kind:              kind,
		Body:              body,
	})
	if err != nil {
		a.logger.Warn().
			Err(err).
			Str("run_id", sc.RunID).
			Str("step_id", sc.StepID).
			Str("provider", a.deps.Messaging.Name()).
			Msg("send_message provider failed")
		return failOutput("send_message", err, err.Error())
	}

	// Persist the outbound row if inbox repo wired and channel id known.
	var messageID, conversationID string
	if a.deps.Inbox != nil && cfg.ChannelID != "" {
		orgUUID, oerr := uuid.Parse(sc.OrgID)
		chUUID, cerr := uuid.Parse(cfg.ChannelID)
		if oerr == nil && cerr == nil {
			mID, cID, perr := a.persistOutbound(ctx, orgUUID, chUUID, recipient, body, res.ProviderMessageID, leadID)
			if perr != nil {
				// Persistence failure is non-fatal for the run (the
				// message already left the provider) but we log loudly.
				a.logger.Error().
					Err(perr).
					Str("run_id", sc.RunID).
					Msg("send_message: outbound persistence failed")
			} else {
				messageID = mID
				conversationID = cID
			}
		}
	}

	out, _ := json.Marshal(map[string]any{
		"kind":                "send_message",
		"sent":                true,
		"provider":            a.deps.Messaging.Name(),
		"provider_message_id": res.ProviderMessageID,
		"message_id":          messageID,
		"conversation_id":     conversationID,
		"recipient":           recipient,
	})
	return StepOutcome{Output: out}, nil
}

// persistOutbound creates or reuses a conversation keyed by
// (channel, recipient) and appends an outbound message row. Called
// after the provider accepted the send so we never write phantom
// "sent" rows for messages that never left.
func (a *SendMessageAction) persistOutbound(
	ctx context.Context, orgID, channelID uuid.UUID,
	recipient, body, providerID, leadID string,
) (string, string, error) {
	contactHandle := recipient
	convID, err := a.deps.Inbox.UpsertConversation(ctx, inboxrepo.UpsertConversationInput{
		OrganizationID:   orgID,
		ChannelID:        channelID,
		ExternalThreadID: recipient,
		ContactHandle:    &contactHandle,
	})
	if err != nil {
		return "", "", fmt.Errorf("upsert conversation: %w", err)
	}
	providerRef := providerID
	bodyRef := body
	msg, err := a.deps.Inbox.AppendMessage(ctx, inboxrepo.AppendMessageInput{
		OrganizationID: orgID,
		ConversationID: convID,
		ExternalID:     ptrIfNotEmpty(providerRef),
		Direction:      "outbound",
		Kind:           "text",
		Body:           ptrIfNotEmpty(bodyRef),
		OccurredAt:     time.Now().UTC(),
	})
	if err != nil {
		return "", "", fmt.Errorf("append message: %w", err)
	}
	return msg.ID.String(), convID.String(), nil
}

// resolveRecipient extracts the phone / handle from the hydrated lead
// fact bag. Field defaults to `phone`.
func resolveRecipient(cfg sendMessageConfig, sc StepContext) (recipient string, leadID string) {
	field := cfg.ToPhoneField
	if field == "" {
		field = "phone"
	}
	var lead map[string]any
	if raw, ok := sc.PreviousOutputs["__lead"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &lead)
	}
	if v, ok := lead[field]; ok {
		recipient = fmt.Sprint(v)
	}
	if id, ok := lead["id"]; ok {
		leadID = fmt.Sprint(id)
	} else {
		leadID = sc.LeadID
	}
	return recipient, leadID
}

// templateRe matches `{{path.segments}}` placeholders. Whitespace
// inside the braces is tolerated (`{{ lead.name }}` also works).
var templateRe = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_.]*)\s*\}\}`)

// renderTemplate substitutes placeholders using the same root grammar
// as branch expressions (lead.*, input.*, prev.<id>.*). Unknown paths
// render as empty string — callers should validate required fields
// before publishing a workflow.
func renderTemplate(body string, sc StepContext) string {
	if body == "" {
		return ""
	}
	return templateRe.ReplaceAllStringFunc(body, func(m string) string {
		sub := templateRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		return resolvePath(sub[1], sc)
	})
}

// lookupChannelExternalID fetches the provider-side instance id for
// a channel row, tenant-scoped. Returns empty string + error on any
// failure so callers can fall back cleanly.
func lookupChannelExternalID(ctx context.Context, deps Deps, orgID, channelID uuid.UUID) (string, error) {
	if deps.Pool == nil {
		return "", errors.New("no pool for channel lookup")
	}
	var ext *string
	err := deps.Pool.QueryRow(ctx,
		`SELECT external_id FROM channels
		  WHERE id = $1 AND organization_id = $2 LIMIT 1`,
		channelID, orgID,
	).Scan(&ext)
	if err != nil {
		return "", fmt.Errorf("channel lookup: %w", err)
	}
	if ext == nil {
		return "", errors.New("channel has no external_id")
	}
	return *ext, nil
}

// ---------- update_lead ---------------------------------------------

// UpdateLeadAction patches lead fields and publishes `lead.updated`.
// Config schema:
//
//	{
//	  "fields": {
//	    "name":           "Bia",
//	    "email":          "bia@acme.com",
//	    "phone":          "+55...",
//	    "responsible_id": "<uuid>",
//	    "rating":         5,
//	    "segment":        "enterprise"
//	  }
//	}
type UpdateLeadAction struct {
	logger zerolog.Logger
	deps   Deps
}

func (*UpdateLeadAction) Kind() string { return "update_lead" }

// updateLeadConfig mirrors lead.UpdateInput with JSON tags.
type updateLeadConfig struct {
	Fields struct {
		Name          *string    `json:"name,omitempty"`
		Company       *string    `json:"company,omitempty"`
		Phone         *string    `json:"phone,omitempty"`
		Email         *string    `json:"email,omitempty"`
		Position      *string    `json:"position,omitempty"`
		ResponsibleID *uuid.UUID `json:"responsible_id,omitempty"`
		Rating        *int16     `json:"rating,omitempty"`
		Segment       *string    `json:"segment,omitempty"`
	} `json:"fields"`
}

// Execute applies the patch through leadrepo.Update + emits a bus event.
func (a *UpdateLeadAction) Execute(ctx context.Context, sc StepContext) (StepOutcome, error) {
	var cfg updateLeadConfig
	if err := json.Unmarshal(sc.Config, &cfg); err != nil {
		return failOutput("update_lead", ErrNonRetryable, "invalid config: "+err.Error())
	}

	if sc.LeadID == "" {
		return failOutput("update_lead", ErrNonRetryable, "lead id absent from run context")
	}
	orgUUID, oerr := uuid.Parse(sc.OrgID)
	leadUUID, lerr := uuid.Parse(sc.LeadID)
	if oerr != nil || lerr != nil {
		return failOutput("update_lead", ErrNonRetryable, "malformed org/lead id")
	}

	if a.deps.Leads == nil {
		a.logger.Warn().
			Str("run_id", sc.RunID).
			Msg("update_lead: leads repo not wired — noop")
		out, _ := json.Marshal(map[string]any{
			"kind":    "update_lead",
			"applied": false,
			"wired":   false,
		})
		return StepOutcome{Output: out}, nil
	}

	patch := leadrepo.UpdateInput{
		Name:          cfg.Fields.Name,
		Company:       cfg.Fields.Company,
		Phone:         cfg.Fields.Phone,
		Email:         cfg.Fields.Email,
		Position:      cfg.Fields.Position,
		ResponsibleID: cfg.Fields.ResponsibleID,
		Rating:        cfg.Fields.Rating,
		Segment:       cfg.Fields.Segment,
	}
	if _, err := a.deps.Leads.Update(ctx, orgUUID, leadUUID, patch); err != nil {
		if errors.Is(err, leadrepo.ErrNotFound) {
			return failOutput("update_lead", ErrNonRetryable, "lead not found")
		}
		return failOutput("update_lead", ErrTransient, err.Error())
	}

	// Publish bus event so inbox / pipe list views refresh.
	if a.deps.Bus != nil {
		a.deps.Bus.Publish(ws.Event{
			Type:       "lead.updated",
			TenantID:   orgUUID,
			EntityType: "lead",
			EntityID:   &leadUUID,
			OccurredAt: time.Now().UTC(),
		})
	}

	out, _ := json.Marshal(map[string]any{
		"kind":    "update_lead",
		"applied": true,
		"lead_id": leadUUID.String(),
	})
	return StepOutcome{Output: out}, nil
}

// ---------- wait (real suspension) ----------------------------------

// WaitAction suspends the run with a scheduled resume. Instead of
// sleeping the runner goroutine (which would block the worker for
// minutes/hours), it returns ErrSuspend and a marker — the executor
// flips the run to `pending` with `next_retry_at = now + duration`
// and the runner reclaims it on schedule.
type WaitAction struct {
	logger zerolog.Logger
}

func (*WaitAction) Kind() string { return "wait" }

// waitConfig caps at 7 days to prevent infinite parking.
type waitConfig struct {
	DurationSeconds int `json:"duration_seconds"`
}

const (
	waitMinSeconds = 1
	waitMaxSeconds = 7 * 24 * 3600 // 7 days
)

// Execute returns ErrSuspend with an output recording the scheduled
// resume. The executor interprets the sentinel and calls SuspendRun.
func (a *WaitAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg waitConfig
	_ = json.Unmarshal(sc.Config, &cfg)
	if cfg.DurationSeconds < waitMinSeconds {
		return failOutput("wait", ErrNonRetryable,
			fmt.Sprintf("duration_seconds must be >= %d", waitMinSeconds))
	}
	if cfg.DurationSeconds > waitMaxSeconds {
		return failOutput("wait", ErrNonRetryable,
			fmt.Sprintf("duration_seconds must be <= %d (7 days)", waitMaxSeconds))
	}
	resumeAt := time.Now().UTC().Add(time.Duration(cfg.DurationSeconds) * time.Second)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Time("resume_at", resumeAt).
		Int("duration_seconds", cfg.DurationSeconds).
		Msg("wait suspending run")
	out, _ := json.Marshal(map[string]any{
		"kind":             "wait",
		"duration_seconds": cfg.DurationSeconds,
		"suspended":        true,
		"resume_at":        resumeAt.Format(time.RFC3339),
	})
	// Encoded marker the executor reads: "__suspend:<rfc3339>".
	return StepOutcome{
		Output:     out,
		NextStepID: "__suspend:" + resumeAt.Format(time.RFC3339),
	}, ErrSuspend
}

// ---------- create_task ---------------------------------------------

// CreateTaskAction inserts a follow-up task scoped to the lead.
// Config schema:
//
//	{
//	  "title":          "Ligar em 1h",
//	  "description":    "...",
//	  "due_in_hours":   24,
//	  "assignee_id":    "<uuid>",    // optional — fallback to lead.responsible_id
//	  "assignee_field": "responsible_id",
//	  "priority":       "normal"
//	}
type CreateTaskAction struct {
	logger zerolog.Logger
	deps   Deps
}

func (*CreateTaskAction) Kind() string { return "create_task" }

type createTaskConfig struct {
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	AssigneeID    *uuid.UUID `json:"assignee_id,omitempty"`
	AssigneeField string     `json:"assignee_field,omitempty"`
	DueInHours    int        `json:"due_in_hours,omitempty"`
	Priority      string     `json:"priority,omitempty"`
}

// Execute creates the task.
func (a *CreateTaskAction) Execute(ctx context.Context, sc StepContext) (StepOutcome, error) {
	var cfg createTaskConfig
	if err := json.Unmarshal(sc.Config, &cfg); err != nil {
		return failOutput("create_task", ErrNonRetryable, "invalid config: "+err.Error())
	}
	if strings.TrimSpace(cfg.Title) == "" {
		return failOutput("create_task", ErrNonRetryable, "title required")
	}
	if a.deps.Tasks == nil {
		a.logger.Warn().Str("run_id", sc.RunID).Msg("create_task: tasks repo not wired — noop")
		out, _ := json.Marshal(map[string]any{
			"kind":    "create_task",
			"created": false,
			"wired":   false,
		})
		return StepOutcome{Output: out}, nil
	}

	orgUUID, oerr := uuid.Parse(sc.OrgID)
	if oerr != nil {
		return failOutput("create_task", ErrNonRetryable, "malformed org id")
	}
	var leadUUID *uuid.UUID
	if sc.LeadID != "" {
		id, lerr := uuid.Parse(sc.LeadID)
		if lerr != nil {
			return failOutput("create_task", ErrNonRetryable, "malformed lead id")
		}
		leadUUID = &id
	}

	// Resolve assignee: explicit id wins; else assignee_field on lead
	// (default `responsible_id`); else caller MUST block — tasks.assignee
	// is NOT NULL and task repo enforces tenant membership.
	assignee := uuid.Nil
	if cfg.AssigneeID != nil {
		assignee = *cfg.AssigneeID
	} else {
		field := cfg.AssigneeField
		if field == "" {
			field = "responsible_id"
		}
		if raw, ok := sc.PreviousOutputs["__lead"]; ok && len(raw) > 0 {
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if v, ok2 := m[field]; ok2 {
				if s, ok3 := v.(string); ok3 {
					if id, err := uuid.Parse(s); err == nil {
						assignee = id
					}
				}
			}
		}
	}
	if assignee == uuid.Nil {
		return failOutput("create_task", ErrNonRetryable, "assignee unresolved")
	}

	title := renderTemplate(cfg.Title, sc)
	var due *time.Time
	if cfg.DueInHours > 0 {
		t := time.Now().UTC().Add(time.Duration(cfg.DueInHours) * time.Hour)
		due = &t
	}
	priority := cfg.Priority
	if priority == "" {
		priority = "normal"
	}

	task, err := a.deps.Tasks.Create(ctx, taskrepo.CreateInput{
		OrganizationID: orgUUID,
		LeadID:         leadUUID,
		AssignedTo:     assignee,
		Title:          title,
		Description:    cfg.Description,
		Priority:       priority,
		Origin:         "workflow",
		DueAt:          due,
	})
	if err != nil {
		if errors.Is(err, taskrepo.ErrNotFound) {
			return failOutput("create_task", ErrNonRetryable, "assignee not in tenant")
		}
		if errors.Is(err, taskrepo.ErrAssigneeBusy) {
			// Busy assignee is a genuine business condition — not
			// retryable (would just busy-loop).
			return failOutput("create_task", ErrNonRetryable, "assignee has active task")
		}
		return failOutput("create_task", ErrTransient, err.Error())
	}

	out, _ := json.Marshal(map[string]any{
		"kind":    "create_task",
		"created": true,
		"task_id": task.ID.String(),
		"title":   title,
	})
	return StepOutcome{Output: out}, nil
}

// ---------- call_agent ----------------------------------------------

// CallAgentAction opens an F06 Copilot agent_session linked to the
// lead + conversation. Config schema:
//
//	{
//	  "agent_id":     "<uuid>",
//	  "greeting":     "Olá!",
//	  "conversation_id": "<uuid?>"
//	}
type CallAgentAction struct {
	logger zerolog.Logger
	deps   Deps
}

func (*CallAgentAction) Kind() string { return "call_agent" }

type callAgentConfig struct {
	AgentID        string `json:"agent_id"`
	Greeting       string `json:"greeting,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
}

// Execute opens the session.
func (a *CallAgentAction) Execute(ctx context.Context, sc StepContext) (StepOutcome, error) {
	var cfg callAgentConfig
	if err := json.Unmarshal(sc.Config, &cfg); err != nil {
		return failOutput("call_agent", ErrNonRetryable, "invalid config: "+err.Error())
	}
	if cfg.AgentID == "" {
		return failOutput("call_agent", ErrNonRetryable, "agent_id required")
	}
	if a.deps.Agents == nil {
		a.logger.Warn().Str("run_id", sc.RunID).Msg("call_agent: agents repo not wired — noop")
		out, _ := json.Marshal(map[string]any{
			"kind":     "call_agent",
			"attached": false,
			"wired":    false,
		})
		return StepOutcome{Output: out}, nil
	}

	orgUUID, oerr := uuid.Parse(sc.OrgID)
	agentUUID, aerr := uuid.Parse(cfg.AgentID)
	if oerr != nil || aerr != nil {
		return failOutput("call_agent", ErrNonRetryable, "malformed org/agent id")
	}
	var leadUUID *uuid.UUID
	if sc.LeadID != "" {
		id, lerr := uuid.Parse(sc.LeadID)
		if lerr == nil {
			leadUUID = &id
		}
	}
	var convUUID *uuid.UUID
	if cfg.ConversationID != "" {
		id, cerr := uuid.Parse(cfg.ConversationID)
		if cerr == nil {
			convUUID = &id
		}
	}

	sess, err := a.deps.Agents.OpenSession(ctx, orgUUID, agentUUID, leadUUID, convUUID)
	if err != nil {
		if errors.Is(err, agentrepo.ErrKillSwitch) {
			// Kill-switch is a tenant policy decision — not retryable.
			return failOutput("call_agent", ErrNonRetryable, "agent kill_switch engaged")
		}
		if errors.Is(err, agentrepo.ErrNotFound) {
			return failOutput("call_agent", ErrNonRetryable, "agent not found")
		}
		return failOutput("call_agent", ErrTransient, err.Error())
	}

	// Optional greeting: first assistant turn logged to the session.
	if cfg.Greeting != "" {
		_, _ = a.deps.Agents.AppendMessage(ctx, agentrepo.AppendMessageInput{
			OrganizationID: orgUUID,
			SessionID:      sess.ID,
			Role:           "assistant",
			Content:        renderTemplate(cfg.Greeting, sc),
		})
	}

	out, _ := json.Marshal(map[string]any{
		"kind":       "call_agent",
		"attached":   true,
		"session_id": sess.ID.String(),
		"agent_id":   agentUUID.String(),
	})
	return StepOutcome{Output: out}, nil
}

// ---------- http ----------------------------------------------------

// HTTPRequestAction calls an external HTTPS endpoint. Hardened against
// SSRF: https-only, private CIDR blocklist, response body capped at
// 1 MiB.
//
// Config schema:
//
//	{
//	  "url":        "https://api.example.com/hook",
//	  "method":     "POST",
//	  "headers":    {"X-Api-Key": "..."},
//	  "body":       "{\"foo\":\"bar\"}",
//	  "timeout_ms": 15000
//	}
type HTTPRequestAction struct {
	logger zerolog.Logger
	client *http.Client
}

func (*HTTPRequestAction) Kind() string { return "http" }

type httpConfig struct {
	URL       string            `json:"url"`
	Method    string            `json:"method,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	TimeoutMS int               `json:"timeout_ms,omitempty"`
}

const (
	httpDefaultTimeout = 15 * time.Second
	httpMaxTimeout     = 30 * time.Second
	httpMaxBodyBytes   = 1 << 20  // 1 MiB request body
	httpMaxRespBytes   = 1 << 20  // 1 MiB response body
	httpLogRespBytes   = 10 << 10 // 10 KiB in output trace
)

// Execute fires the HTTP request.
func (a *HTTPRequestAction) Execute(ctx context.Context, sc StepContext) (StepOutcome, error) {
	var cfg httpConfig
	if err := json.Unmarshal(sc.Config, &cfg); err != nil {
		return failOutput("http", ErrNonRetryable, "invalid config: "+err.Error())
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return failOutput("http", ErrNonRetryable, "method not allowed: "+method)
	}

	// SSRF + scheme guard.
	if err := validateHTTPSURL(cfg.URL); err != nil {
		return failOutput("http", ErrNonRetryable, err.Error())
	}

	// Body size guard.
	body := renderTemplate(cfg.Body, sc)
	if len(body) > httpMaxBodyBytes {
		return failOutput("http", ErrNonRetryable,
			fmt.Sprintf("request body exceeds %d bytes", httpMaxBodyBytes))
	}

	// Timeout.
	timeout := httpDefaultTimeout
	if cfg.TimeoutMS > 0 {
		t := time.Duration(cfg.TimeoutMS) * time.Millisecond
		if t > httpMaxTimeout {
			t = httpMaxTimeout
		}
		timeout = t
	}
	cli := a.client
	if cli == nil {
		// No-redirect policy prevents SSRF-by-redirect (a 302 to
		// http://169.254.169.254 would bypass the validated URL).
		cli = &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else {
		cli.Timeout = timeout
	}

	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, reqBody)
	if err != nil {
		return failOutput("http", ErrNonRetryable, "build request: "+err.Error())
	}
	for k, v := range cfg.Headers {
		// Prevent Host header spoofing (would be ignored by stdlib but
		// cleanly reject here for parity with policy).
		if strings.EqualFold(k, "Host") {
			continue
		}
		req.Header.Set(k, v)
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := cli.Do(req)
	duration := time.Since(start)
	if err != nil {
		// Network-level failures retry. Context cancellations propagate.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return failOutput("http", ErrTransient, "timeout: "+err.Error())
		}
		return failOutput("http", ErrTransient, err.Error())
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, httpMaxRespBytes)
	raw, _ := io.ReadAll(limited)

	status := resp.StatusCode
	// 5xx / 429 retry; 4xx terminal (unless 429).
	if status == http.StatusTooManyRequests || status >= 500 {
		return failOutput("http", ErrTransient,
			fmt.Sprintf("status %d: %s", status, truncateBytes(raw, 400)))
	}
	if status >= 400 {
		return failOutput("http", ErrNonRetryable,
			fmt.Sprintf("status %d: %s", status, truncateBytes(raw, 400)))
	}

	out, _ := json.Marshal(map[string]any{
		"kind":          "http",
		"status_code":   status,
		"duration_ms":   duration.Milliseconds(),
		"method":        method,
		"url":           cfg.URL,
		"response_body": string(truncateBytes(raw, httpLogRespBytes)),
	})
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Int("status", status).
		Dur("duration", duration).
		Msg("http action completed")
	return StepOutcome{Output: out}, nil
}

// executeUnchecked is a test-only entrypoint that skips the SSRF
// validator so the happy-path / 5xx / timeout flows can be exercised
// against httptest loopback servers. Production code MUST NOT call
// this — the public Execute is the only path that reaches the wire.
func (a *HTTPRequestAction) executeUnchecked(ctx context.Context, rawCfg []byte) (StepOutcome, error) {
	var cfg httpConfig
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		return failOutput("http", ErrNonRetryable, "invalid config: "+err.Error())
	}
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodGet
	}
	body := cfg.Body
	if len(body) > httpMaxBodyBytes {
		return failOutput("http", ErrNonRetryable, "body too large")
	}
	timeout := httpDefaultTimeout
	if cfg.TimeoutMS > 0 {
		t := time.Duration(cfg.TimeoutMS) * time.Millisecond
		if t > httpMaxTimeout {
			t = httpMaxTimeout
		}
		timeout = t
	}
	cli := a.client
	if cli == nil {
		cli = &http.Client{Timeout: timeout}
	}
	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, reqBody)
	if err != nil {
		return failOutput("http", ErrNonRetryable, err.Error())
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	start := time.Now()
	resp, err := cli.Do(req)
	duration := time.Since(start)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "deadline exceeded") ||
			strings.Contains(err.Error(), "Client.Timeout exceeded") {
			return failOutput("http", ErrTransient, "timeout: "+err.Error())
		}
		return failOutput("http", ErrTransient, err.Error())
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, httpMaxRespBytes))
	status := resp.StatusCode
	if status == http.StatusTooManyRequests || status >= 500 {
		return failOutput("http", ErrTransient,
			fmt.Sprintf("status %d: %s", status, truncateBytes(raw, 400)))
	}
	if status >= 400 {
		return failOutput("http", ErrNonRetryable,
			fmt.Sprintf("status %d: %s", status, truncateBytes(raw, 400)))
	}
	out, _ := json.Marshal(map[string]any{
		"kind":          "http",
		"status_code":   status,
		"duration_ms":   duration.Milliseconds(),
		"method":        method,
		"url":           cfg.URL,
		"response_body": string(truncateBytes(raw, httpLogRespBytes)),
	})
	return StepOutcome{Output: out}, nil
}

// validateHTTPSURL enforces the full SSRF allowlist:
//   - scheme MUST be https
//   - host MUST resolve to non-private IPs
//   - literal loopback hostnames rejected
//
// The DNS check catches `http://localhost` and
// `http://metadata.google.internal` style attacks; without it, an
// admin could craft a workflow that hits the cloud metadata endpoint.
func validateHTTPSURL(raw string) error {
	if raw == "" {
		return errors.New("url required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("scheme must be https")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("host required")
	}
	lowerHost := strings.ToLower(host)
	for _, deny := range denylistHosts {
		if lowerHost == deny || strings.HasSuffix(lowerHost, "."+deny) {
			return fmt.Errorf("host %q is denylisted", host)
		}
	}
	// Resolve + check each IP. An attacker-controlled hostname that
	// resolves to 10.0.0.1 is blocked here.
	ips, err := net.LookupIP(host)
	if err != nil {
		// DNS failure is NOT a silent allow — reject.
		return fmt.Errorf("dns lookup: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("host %q resolves to blocked range (%s)", host, ip.String())
		}
	}
	return nil
}

// denylistHosts are literal names that must never reach the network,
// regardless of DNS (in case of DNS rebind shenanigans).
var denylistHosts = []string{
	"localhost",
	"metadata.google.internal",
	"metadata.goog",
}

// isBlockedIP blocks RFC1918 / loopback / link-local / carrier-grade
// NAT / unspecified addresses. IPv6 equivalents included.
func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	// Carrier-grade NAT 100.64.0.0/10 (not IsPrivate in stdlib).
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && (ip4[1]&0xc0) == 64 {
			return true
		}
		// AWS metadata 169.254.169.254 is covered by IsLinkLocalUnicast.
	}
	// IPv6 unique-local fc00::/7
	if ip.To4() == nil && len(ip) == net.IPv6len {
		if (ip[0] & 0xfe) == 0xfc {
			return true
		}
	}
	return false
}

// validateHTTPURL is kept for backward compat with existing tests
// (S45 introduced it with a literal prefix check). Its semantics
// match validateHTTPSURL's scheme guard but it returns a boolean.
func validateHTTPURL(raw string) bool {
	return len(raw) >= 9 && raw[:8] == "https://"
}

// ---------- helpers -------------------------------------------------

// failOutput builds a structured trace for a handler that aborted,
// and wraps the error in the appropriate classifier sentinel. If
// classifier is nil the error is returned raw (treated as transient
// by the executor's classifier fallback).
func failOutput(kind string, classifier error, msg string) (StepOutcome, error) {
	out, _ := json.Marshal(map[string]any{
		"kind":   kind,
		"failed": true,
		"error":  msg,
	})
	if classifier == nil {
		return StepOutcome{Output: out}, errors.New(msg)
	}
	if errors.Is(classifier, ErrTransient) {
		return StepOutcome{Output: out}, fmt.Errorf("%w: %s", ErrTransient, msg)
	}
	if errors.Is(classifier, ErrNonRetryable) {
		return StepOutcome{Output: out}, fmt.Errorf("%w: %s", ErrNonRetryable, msg)
	}
	return StepOutcome{Output: out}, fmt.Errorf("%s: %w", msg, classifier)
}

func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func truncateBytes(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}

// Compile-time guards that every action pulls the imports that use it,
// so go vet doesn't yell when deps are nil.
var (
	_ = leadrepo.ErrNotFound
	_ = taskrepo.ErrNotFound
	_ = agentrepo.ErrNotFound
	_ = inboxrepo.ErrNotFound
)

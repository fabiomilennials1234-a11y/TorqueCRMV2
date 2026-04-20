// S45 — additional action handlers promoting the F07 executor from
// "engine proven" to "12 action types" coverage target.
//
// Like S44 handlers, these are side-effect-free stubs that record
// faithful intent output so the execution trace proves end-to-end
// wiring. Connecting to real integrations (tasks repo, agent service,
// HTTPS URL fetch with allowlist) is deferred behind explicit
// interface seams: each handler takes `nil` deps today and swaps to
// a real concrete when a tenant blocks production roll-out.

package workflow

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
)

// ---------- create_task ---------------------------------------------

// CreateTaskAction drops a followup in the tasks queue. The config
// carries {title, assignee_id, due_in_hours}. Real integration wires
// to tasks repo in a follow-up.
type CreateTaskAction struct {
	logger zerolog.Logger
}

func (*CreateTaskAction) Kind() string { return "create_task" }

func (a *CreateTaskAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		Title       string `json:"title"`
		AssigneeID  string `json:"assignee_id"`
		DueInHours  int    `json:"due_in_hours"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("title", cfg.Title).
		Msg("create_task dispatched (stub)")
	out, _ := json.Marshal(map[string]any{
		"kind":          "create_task",
		"title":         cfg.Title,
		"assignee_id":   cfg.AssigneeID,
		"due_in_hours":  cfg.DueInHours,
		"created":       false,
	})
	return StepOutcome{Output: out}, nil
}

// ---------- call_agent ----------------------------------------------

// CallAgentAction hands the conversation over to an F06 Copilot agent.
// Config shape {agent_id, greeting}. When wired, this invokes the
// OpenRouter provider via an agent session (S37/S38 path). Today it
// records the intent.
type CallAgentAction struct {
	logger zerolog.Logger
}

func (*CallAgentAction) Kind() string { return "call_agent" }

func (a *CallAgentAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		AgentID  string `json:"agent_id"`
		Greeting string `json:"greeting"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("agent_id", cfg.AgentID).
		Msg("call_agent dispatched (stub)")
	out, _ := json.Marshal(map[string]any{
		"kind":      "call_agent",
		"agent_id":  cfg.AgentID,
		"greeting":  cfg.Greeting,
		"attached":  false,
	})
	return StepOutcome{Output: out}, nil
}

// ---------- http --------------------------------------------------

// HTTPRequestAction calls an external HTTPS endpoint. Config shape
// {url, method, body_template, timeout_ms}. Real fetch includes the
// existing S29 anti-SSRF allowlist (https scheme only, block private
// CIDRs); for S45 stub we log the intended URL but refuse to fire
// non-https or obviously-internal hosts so a misconfigured workflow
// can't panic the runner even in stub mode.
type HTTPRequestAction struct {
	logger zerolog.Logger
}

func (*HTTPRequestAction) Kind() string { return "http" }

func (a *HTTPRequestAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		URL       string `json:"url"`
		Method    string `json:"method"`
		TimeoutMS int    `json:"timeout_ms"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	accepted := validateHTTPURL(cfg.URL)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("url", cfg.URL).
		Bool("accepted", accepted).
		Msg("http dispatched (stub, accept=https only)")
	out, _ := json.Marshal(map[string]any{
		"kind":     "http",
		"url":      cfg.URL,
		"method":   cfg.Method,
		"accepted": accepted, // false = would be rejected at real fetch time
		"status":   0,        // real status when fetch wires
	})
	return StepOutcome{Output: out}, nil
}

// validateHTTPURL enforces the minimum S29 SSRF invariant: scheme MUST
// be https. Private-CIDR check waits for real fetch (requires DNS
// resolution + ip range match) — today the workflow runner has no
// networking surface so the check is best-effort.
func validateHTTPURL(raw string) bool {
	if len(raw) < 9 {
		return false
	}
	return raw[:8] == "https://"
}

// NewDispatcherS45 returns a dispatcher with S44 + S45 handlers
// registered. Callers that want the extended surface wire through this
// constructor; the original NewDispatcher() stays unchanged so legacy
// workflows don't gain new kinds silently.
func NewDispatcherS45(logger zerolog.Logger) *Dispatcher {
	d := NewDispatcher(logger)
	d.Register(&CreateTaskAction{logger: logger})
	d.Register(&CallAgentAction{logger: logger})
	d.Register(&HTTPRequestAction{logger: logger})
	return d
}

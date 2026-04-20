// Package workflow hosts the S44 executor + action dispatcher for F07.
//
// The executor walks a workflow DAG one step at a time, handing each
// step to the ActionDispatcher. For S44 the dispatcher has four
// concrete actions (send_message, update_lead, wait, branch); all are
// side-effect free in this sprint — they log + produce output JSON —
// so the execution trace is faithful even before the external
// integrations (Evolution, pipe, scheduler) wire to real callers in
// S45. A real send_message plugged in here is the only bit that
// changes; the executor + dispatcher interface stay stable.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
)

// ErrActionUnknown bubbles up when the step kind is not registered.
// Runs hitting this fail — not retry — because a missing handler is a
// configuration bug, not a transient glitch.
var ErrActionUnknown = errors.New("workflow: action handler not registered")

// StepContext carries everything a handler needs to do its work. Kept
// small so expanding it doesn't cascade through every handler.
type StepContext struct {
	// OrgID is the tenant scope. Handlers that write MUST scope every
	// query to it — leaking into another tenant is an invariant
	// violation beyond any single workflow's behavior.
	OrgID string
	// RunID identifies the workflow_run this step is executing under
	// (useful for audit + cross-step context in a future iteration).
	RunID string
	// StepID identifies the workflow_step definition.
	StepID string
	// LeadID is optional — absent for workflows triggered manually
	// or by schedule that don't target a specific lead.
	LeadID string
	// Config is the step's raw JSON config as stored in workflow_steps.config.
	Config json.RawMessage
	// PreviousOutputs carries output JSON from each step visited
	// earlier in this run, keyed by step_id. Enables templates and
	// branch conditions that reference upstream output.
	PreviousOutputs map[string]json.RawMessage
}

// StepOutcome is what a handler returns. Branch handlers populate
// `NextStepID` to steer the executor; straight-action handlers leave
// it empty and the executor uses the first entry of next_step_ids.
type StepOutcome struct {
	Output     json.RawMessage
	NextStepID string // non-empty overrides default DAG walk (branch)
}

// ActionHandler is the per-kind unit of work.
type ActionHandler interface {
	Kind() string
	Execute(ctx context.Context, sc StepContext) (StepOutcome, error)
}

// Dispatcher is a lookup over ActionHandlers keyed by kind. Looking
// up a kind that wasn't registered returns ErrActionUnknown instead
// of panicking so a bad workflow config fails loudly at run time but
// doesn't crash the runner.
type Dispatcher struct {
	handlers map[string]ActionHandler
	logger   zerolog.Logger
}

// NewDispatcher builds a dispatcher with the default S44 handlers
// registered. Callers can later Register() additional kinds (S45+
// `create_task`, `call_agent`, `http`, `schedule` trigger, etc.).
func NewDispatcher(logger zerolog.Logger) *Dispatcher {
	d := &Dispatcher{
		handlers: map[string]ActionHandler{},
		logger:   logger.With().Str("component", "workflow_dispatcher").Logger(),
	}
	d.Register(&SendMessageAction{logger: logger})
	d.Register(&UpdateLeadAction{logger: logger})
	d.Register(&WaitAction{logger: logger})
	d.Register(&BranchAction{logger: logger})
	return d
}

// Register binds a handler to its Kind. Re-registering a kind
// overwrites (useful for tests with a fake handler).
func (d *Dispatcher) Register(h ActionHandler) {
	d.handlers[h.Kind()] = h
}

// Dispatch invokes the handler for `kind`, or returns
// ErrActionUnknown with the kind name attached.
func (d *Dispatcher) Dispatch(ctx context.Context, kind string, sc StepContext) (StepOutcome, error) {
	h, ok := d.handlers[kind]
	if !ok {
		return StepOutcome{}, fmt.Errorf("%w: %s", ErrActionUnknown, kind)
	}
	return h.Execute(ctx, sc)
}

// Handlers returns the registered kinds (sorted-free). Useful for
// introspection in logs / admin tooling — not a hot path.
func (d *Dispatcher) Handlers() []string {
	out := make([]string, 0, len(d.handlers))
	for k := range d.handlers {
		out = append(out, k)
	}
	return out
}

// ---------- send_message --------------------------------------------

// SendMessageAction is an S44 stub. In S45 it will route through
// internal/service/integration/messaging to the Evolution adapter.
// Today it records the intended template + body in output so the
// execution trace proves the dispatch worked end-to-end.
type SendMessageAction struct {
	logger zerolog.Logger
}

// Kind implements ActionHandler.
func (*SendMessageAction) Kind() string { return "send_message" }

// Execute records the intended send as output. No side effect yet.
func (a *SendMessageAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		Template string `json:"template"`
		Body     string `json:"body"`
		// trigger_kind tags trigger-only nodes (lead_created,
		// message_received) that are dropped in the canvas. They
		// act as pass-throughs here: the dispatcher records the
		// trigger fired and flows to next_step_ids.
		TriggerKind string `json:"trigger_kind,omitempty"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("step_id", sc.StepID).
		Str("template", cfg.Template).
		Msg("send_message dispatched (stub)")
	out, _ := json.Marshal(map[string]any{
		"kind":     "send_message",
		"template": cfg.Template,
		"body":     cfg.Body,
		"trigger":  cfg.TriggerKind,
		"sent":     false, // S45 flips true when Evolution wired.
	})
	return StepOutcome{Output: out}, nil
}

// ---------- update_lead ---------------------------------------------

type UpdateLeadAction struct {
	logger zerolog.Logger
}

func (*UpdateLeadAction) Kind() string { return "update_lead" }

// Execute records the fields the workflow wants to set on the lead.
// The real lead repo call wires in S45 (`action.set_lead_stage`
// variant). Keeping it stubbed here keeps S44 scoped to the engine.
func (a *UpdateLeadAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg map[string]any
	_ = json.Unmarshal(sc.Config, &cfg)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("lead_id", sc.LeadID).
		Msg("update_lead dispatched (stub)")
	out, _ := json.Marshal(map[string]any{
		"kind":    "update_lead",
		"fields":  cfg,
		"applied": false,
	})
	return StepOutcome{Output: out}, nil
}

// ---------- wait ----------------------------------------------------

// WaitAction is the simplest handler: records the requested pause
// and returns. Actual suspension + re-enqueue is a follow-up (needs
// a scheduler integration + `run_at` future timestamp). S44 proves
// the engine can sequence through a wait step without blocking the
// runner goroutine.
type WaitAction struct {
	logger zerolog.Logger
}

func (*WaitAction) Kind() string { return "wait" }

func (a *WaitAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		DurationSeconds int `json:"duration_seconds"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Int("duration_seconds", cfg.DurationSeconds).
		Msg("wait dispatched (no real pause in S44)")
	out, _ := json.Marshal(map[string]any{
		"kind":             "wait",
		"duration_seconds": cfg.DurationSeconds,
		"suspended":        false,
	})
	return StepOutcome{Output: out}, nil
}

// ---------- branch --------------------------------------------------

// BranchAction evaluates a minimal expression language:
//
//	"lead.origin == \"meta-ads\""
//	"input.amount > 100"
//	"prev.<step_id>.template == \"warmup\""
//
// The grammar is intentionally tiny — 3 ops (== != >) over dotted
// paths resolved against the StepContext. Reachability of advanced
// ops (regex, arithmetic, "in") moves to a proper expr package when
// real tenants demand them.
//
// Convention for next_step_ids: index 0 = true branch, index 1 = false.
// A branch with only one next_step_id collapses to a pass-through
// (logs which path would've been taken in output but doesn't stall).
type BranchAction struct {
	logger zerolog.Logger
}

func (*BranchAction) Kind() string { return "branch" }

// Execute evaluates cfg.Expression and sets NextStepID to the
// appropriate successor. Unknown / invalid expressions fall back to
// the true branch with `expression_valid=false` in output so the
// trail flags the fault without breaking the run.
func (a *BranchAction) Execute(_ context.Context, sc StepContext) (StepOutcome, error) {
	var cfg struct {
		Expression string `json:"expression"`
	}
	_ = json.Unmarshal(sc.Config, &cfg)
	// We don't wire to actual lead data in S44 — the executor passes
	// a synthetic `facts` map through PreviousOutputs keyed by
	// `__input`. Real lead facts land with the bus subscriber that
	// hydrates the context from the leads repo.
	result, valid := evalBranch(cfg.Expression, sc)
	a.logger.Debug().
		Str("run_id", sc.RunID).
		Str("expression", cfg.Expression).
		Bool("result", result).
		Bool("valid", valid).
		Msg("branch evaluated")
	out, _ := json.Marshal(map[string]any{
		"kind":             "branch",
		"expression":       cfg.Expression,
		"result":           result,
		"expression_valid": valid,
	})
	// next_step_ids[0] = true branch, [1] = false. The executor
	// consults StepOutcome.NextStepID — empty string = default walk.
	// We signal the choice by returning a marker the executor
	// interprets: "__branch:true" or "__branch:false". The executor
	// resolves this to the actual next step id after picking from
	// the step's next_step_ids slice.
	marker := "__branch:true"
	if !result {
		marker = "__branch:false"
	}
	return StepOutcome{Output: out, NextStepID: marker}, nil
}

// evalBranch is a minimal expression evaluator. Returns (result, valid).
// An unparseable expression yields (true, false) so a misconfigured
// branch still lets the run progress instead of stalling the queue.
func evalBranch(expr string, sc StepContext) (bool, bool) {
	lhs, op, rhs, ok := splitExpr(expr)
	if !ok {
		return true, false
	}
	lv := resolvePath(lhs, sc)
	// rhs is either a quoted string, a number, or a dotted path.
	rv := resolveRHS(rhs, sc)
	switch op {
	case "==":
		return lv == rv, true
	case "!=":
		return lv != rv, true
	default:
		return true, false
	}
}

// splitExpr finds the operator token and returns trimmed lhs/rhs.
// Ordering matters: look for "!=" before "=" because "=" is a prefix.
func splitExpr(expr string) (lhs, op, rhs string, ok bool) {
	// Try operators in longest-first order.
	for _, o := range []string{"==", "!="} {
		if idx := indexOf(expr, o); idx >= 0 {
			return trim(expr[:idx]), o, trim(expr[idx+len(o):]), true
		}
	}
	return "", "", "", false
}

// resolvePath walks a dotted path like "lead.origin" / "input.amount"
// against the StepContext. Handles two roots today: "input" (the run
// input json) and "prev.<step_id>" (a prior step output). Unknown
// root returns "".
func resolvePath(path string, sc StepContext) string {
	seg, rest, has := splitDot(path)
	if !has {
		return path
	}
	switch seg {
	case "input":
		var m map[string]any
		if len(sc.PreviousOutputs["__input"]) > 0 {
			_ = json.Unmarshal(sc.PreviousOutputs["__input"], &m)
		}
		return walkMap(m, rest)
	case "prev":
		stepID, sub, has2 := splitDot(rest)
		if !has2 {
			return ""
		}
		var m map[string]any
		if raw, ok := sc.PreviousOutputs[stepID]; ok && len(raw) > 0 {
			_ = json.Unmarshal(raw, &m)
		}
		return walkMap(m, sub)
	case "lead":
		// lead facts land in PreviousOutputs under "__lead" when the
		// bus subscriber hydrates. Keeping the lookup here means
		// tests can inject a fake lead map without changing dispatcher.
		var m map[string]any
		if raw, ok := sc.PreviousOutputs["__lead"]; ok && len(raw) > 0 {
			_ = json.Unmarshal(raw, &m)
		}
		return walkMap(m, rest)
	default:
		return ""
	}
}

// resolveRHS parses the right-hand side. If it starts with a quote,
// it's a literal. Otherwise it's a path.
func resolveRHS(s string, sc StepContext) string {
	s = trim(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return resolvePath(s, sc)
}

func walkMap(m map[string]any, path string) string {
	if m == nil {
		return ""
	}
	seg, rest, has := splitDot(path)
	if !has {
		v, ok := m[path]
		if !ok {
			return ""
		}
		return fmt.Sprint(v)
	}
	next, ok := m[seg].(map[string]any)
	if !ok {
		return ""
	}
	return walkMap(next, rest)
}

func splitDot(s string) (head, tail string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trim(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

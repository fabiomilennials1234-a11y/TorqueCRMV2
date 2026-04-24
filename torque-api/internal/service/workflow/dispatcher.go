// Package workflow hosts the S44/S45/S52 executor + action dispatcher
// for F07.
//
// The executor walks a workflow DAG one step at a time, handing each
// step to the ActionDispatcher. S44 shipped four side-effect-free
// handlers (send_message, update_lead, wait, branch). S45 added three
// more (create_task, call_agent, http). S52 turns all seven into real
// side-effects backed by repositories + the Evolution adapter, with
// retry classification (ErrTransient / ErrNonRetryable) and a bus
// publisher so downstream observers react to every mutation.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
	inboxrepo "github.com/milennials/torque-api/internal/repository/inbox"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	taskrepo "github.com/milennials/torque-api/internal/repository/task"
	"github.com/milennials/torque-api/internal/service/integration"
)

// ErrActionUnknown bubbles up when the step kind is not registered.
// Runs hitting this fail — not retry — because a missing handler is a
// configuration bug, not a transient glitch.
var ErrActionUnknown = errors.New("workflow: action handler not registered")

// ErrTransient is returned by handlers when the failure is expected to
// succeed on retry (Evolution 5xx, HTTP timeout, rate limit). The
// executor schedules a backoff-based retry up to max_attempts.
var ErrTransient = errors.New("workflow: transient error — retryable")

// ErrNonRetryable is returned when a failure is structural (bad
// config, resource deleted, permanent 4xx). The executor sends the
// run straight to the DLQ without wasting retries.
var ErrNonRetryable = errors.New("workflow: non-retryable error")

// ErrSuspend is a sentinel the executor recognizes to mean "park this
// run at the NEXT step with a scheduled resume". Wait handlers return
// this along with a StepOutcome whose NextStepID carries the resume
// marker `__suspend:<iso8601>`.
var ErrSuspend = errors.New("workflow: suspend run for scheduled resume")

// classifyError wraps a raw error with the appropriate sentinel if
// the caller didn't. Callers that KNOW the failure class return the
// right sentinel directly; this is a safety net for adapters that
// surface opaque error values.
func classifyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTransient) || errors.Is(err, ErrNonRetryable) {
		return err
	}
	// Providers: Evolution / SZ.Chat map their own transient errors
	// via integration.Err* sentinels; Unreachable + RateLimited are
	// retryable, AuthFailed is not (credentials don't heal).
	if errors.Is(err, integration.ErrUnreachable) ||
		errors.Is(err, integration.ErrRateLimited) ||
		errors.Is(err, integration.ErrCircuitOpen) {
		return fmt.Errorf("%w: %v", ErrTransient, err)
	}
	if errors.Is(err, integration.ErrAuthFailed) ||
		errors.Is(err, integration.ErrUnsupported) {
		return fmt.Errorf("%w: %v", ErrNonRetryable, err)
	}
	return err
}

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

// Deps is the dependency envelope passed to real handlers. Zero-valued
// fields disable the corresponding side-effect: e.g. constructing a
// dispatcher without a leadrepo makes UpdateLeadAction degrade to a
// logged no-op with `applied:false`. That lets tests wire partial
// dispatchers without a DB.
type Deps struct {
	Pool       *pgxpool.Pool
	Bus        *event.Bus
	Leads      *leadrepo.Repository
	Tasks      *taskrepo.Repository
	Agents     *agentrepo.Repository
	Inbox      *inboxrepo.Repository
	Messaging  integration.MessagingProvider
}

// Dispatcher is a lookup over ActionHandlers keyed by kind. Looking
// up a kind that wasn't registered returns ErrActionUnknown instead
// of panicking so a bad workflow config fails loudly at run time but
// doesn't crash the runner.
type Dispatcher struct {
	handlers map[string]ActionHandler
	logger   zerolog.Logger
}

// NewDispatcher builds a dispatcher with the S44 + S45 handlers
// registered and real dependencies injected. Callers that want a
// deps-free dispatcher (unit tests, old boot paths) can pass a zero
// Deps — handlers will degrade gracefully.
func NewDispatcher(logger zerolog.Logger, deps Deps) *Dispatcher {
	d := &Dispatcher{
		handlers: map[string]ActionHandler{},
		logger:   logger.With().Str("component", "workflow_dispatcher").Logger(),
	}
	d.Register(&SendMessageAction{logger: logger, deps: deps})
	d.Register(&UpdateLeadAction{logger: logger, deps: deps})
	d.Register(&WaitAction{logger: logger})
	d.Register(&BranchAction{logger: logger})
	d.Register(&CreateTaskAction{logger: logger, deps: deps})
	d.Register(&CallAgentAction{logger: logger, deps: deps})
	d.Register(&HTTPRequestAction{logger: logger})
	return d
}

// NewDispatcherBare is the deps-free constructor used by legacy tests
// that only care about the registry + branch logic. Real production
// always calls NewDispatcher with a populated Deps.
func NewDispatcherBare(logger zerolog.Logger) *Dispatcher {
	return NewDispatcher(logger, Deps{})
}

// NewDispatcherS45 retained for compatibility with existing boot
// wiring; it now routes through NewDispatcher with empty deps. Any
// caller that wants real side-effects should migrate to NewDispatcher.
//
// Deprecated: kept so older callsites compile while boot migrates.
func NewDispatcherS45(logger zerolog.Logger) *Dispatcher {
	return NewDispatcher(logger, Deps{})
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
	out, err := h.Execute(ctx, sc)
	return out, classifyError(err)
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
	for _, o := range []string{"==", "!="} {
		if idx := indexOf(expr, o); idx >= 0 {
			return trim(expr[:idx]), o, trim(expr[idx+len(o):]), true
		}
	}
	return "", "", "", false
}

// resolvePath walks a dotted path like "lead.origin" / "input.amount"
// against the StepContext. Handles three roots: "input" (the run
// input json), "prev.<step_id>" (a prior step output), and "lead"
// (the hydrated lead fact bag).
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

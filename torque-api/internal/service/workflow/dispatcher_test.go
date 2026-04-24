package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func newTestDispatcher() *Dispatcher {
	return NewDispatcherBare(zerolog.Nop())
}

// --------------- registry ------------------------------------------

func TestDispatcher_UnknownKindErr(t *testing.T) {
	t.Parallel()
	d := newTestDispatcher()
	_, err := d.Dispatch(context.Background(), "nope", StepContext{})
	if !errors.Is(err, ErrActionUnknown) {
		t.Errorf("want ErrActionUnknown, got %v", err)
	}
}

func TestDispatcher_ListHandlers(t *testing.T) {
	t.Parallel()
	d := newTestDispatcher()
	got := d.Handlers()
	// S52 — dispatcher registers all seven kinds unconditionally;
	// previously S44 shipped four and S45 added three via a separate
	// constructor. Consolidation removes a footgun where a boot path
	// forgot to call NewDispatcherS45.
	want := map[string]bool{
		"send_message": true,
		"update_lead":  true,
		"wait":         true,
		"branch":       true,
		"create_task":  true,
		"call_agent":   true,
		"http":         true,
	}
	for _, k := range got {
		if !want[k] {
			t.Errorf("unexpected handler %q", k)
		}
		delete(want, k)
	}
	if len(want) != 0 {
		t.Errorf("missing handlers: %v", want)
	}
}

// --------------- send_message --------------------------------------

// TestSendMessageAction_NoProviderIsNoop asserts the graceful
// degradation path: without a wired MessagingProvider, the handler
// emits a noop trace (sent=false, wired=false) instead of crashing.
// Real delivery is validated in a gated integration test.
func TestSendMessageAction_NoProviderIsNoop(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"body": "Olá"})
	d := newTestDispatcher()
	out, err := d.Dispatch(context.Background(), "send_message", StepContext{Config: cfg})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["sent"] != false {
		t.Errorf("expected sent=false when provider nil, got %v", parsed)
	}
	if parsed["wired"] != false {
		t.Errorf("expected wired=false when provider nil, got %v", parsed)
	}
}

// --------------- wait ----------------------------------------------

// TestWaitAction_SignalsSuspension confirms wait is a REAL suspension
// now (S52). The handler returns ErrSuspend + a NextStepID marker
// encoding the resume time, and the output still carries the original
// duration so the trace is reproducible.
func TestWaitAction_SignalsSuspension(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]int{"duration_seconds": 300})
	d := newTestDispatcher()
	out, err := d.Dispatch(context.Background(), "wait", StepContext{Config: cfg})
	if !errors.Is(err, ErrSuspend) {
		t.Fatalf("expected ErrSuspend sentinel, got %v", err)
	}
	if !strings.HasPrefix(out.NextStepID, "__suspend:") {
		t.Errorf("expected __suspend: marker, got %q", out.NextStepID)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["duration_seconds"].(float64) != 300 {
		t.Errorf("duration lost: %v", parsed["duration_seconds"])
	}
	if parsed["suspended"] != true {
		t.Errorf("expected suspended=true, got %v", parsed)
	}
}

// TestWaitAction_RejectsZeroDuration covers the non-retryable guard.
func TestWaitAction_RejectsZeroDuration(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]int{"duration_seconds": 0})
	d := newTestDispatcher()
	_, err := d.Dispatch(context.Background(), "wait", StepContext{Config: cfg})
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("expected non-retryable error, got %v", err)
	}
}

// TestWaitAction_RejectsOverMax covers the 7-day cap.
func TestWaitAction_RejectsOverMax(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]int{"duration_seconds": 8 * 24 * 3600})
	d := newTestDispatcher()
	_, err := d.Dispatch(context.Background(), "wait", StepContext{Config: cfg})
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("expected non-retryable error on over-max, got %v", err)
	}
}

// --------------- branch --------------------------------------------

func TestBranchAction_EqTrue(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": `lead.origin == "meta-ads"`})
	leadJSON, _ := json.Marshal(map[string]string{"origin": "meta-ads"})
	d := newTestDispatcher()
	out, err := d.Dispatch(context.Background(), "branch", StepContext{
		Config:          cfg,
		PreviousOutputs: map[string]json.RawMessage{"__lead": leadJSON},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out.NextStepID != "__branch:true" {
		t.Errorf("expected branch:true, got %s", out.NextStepID)
	}
}

func TestBranchAction_EqFalse(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": `lead.origin == "meta-ads"`})
	leadJSON, _ := json.Marshal(map[string]string{"origin": "organic"})
	d := newTestDispatcher()
	out, _ := d.Dispatch(context.Background(), "branch", StepContext{
		Config:          cfg,
		PreviousOutputs: map[string]json.RawMessage{"__lead": leadJSON},
	})
	if out.NextStepID != "__branch:false" {
		t.Errorf("expected branch:false, got %s", out.NextStepID)
	}
}

func TestBranchAction_NeqOp(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": `lead.origin != "meta-ads"`})
	leadJSON, _ := json.Marshal(map[string]string{"origin": "organic"})
	d := newTestDispatcher()
	out, _ := d.Dispatch(context.Background(), "branch", StepContext{
		Config:          cfg,
		PreviousOutputs: map[string]json.RawMessage{"__lead": leadJSON},
	})
	if out.NextStepID != "__branch:true" {
		t.Errorf("neq should be true when values differ; got %s", out.NextStepID)
	}
}

func TestBranchAction_UnparseableFallsToTrue(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": "garbage"})
	d := newTestDispatcher()
	out, err := d.Dispatch(context.Background(), "branch", StepContext{Config: cfg})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out.NextStepID != "__branch:true" {
		t.Errorf("invalid expr should default to true; got %s", out.NextStepID)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["expression_valid"] != false {
		t.Errorf("expected expression_valid=false trace; got %v", parsed)
	}
}

func TestBranchAction_InputPathReference(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": `input.origin == "meta-ads"`})
	inputJSON, _ := json.Marshal(map[string]string{"origin": "meta-ads"})
	d := newTestDispatcher()
	out, _ := d.Dispatch(context.Background(), "branch", StepContext{
		Config:          cfg,
		PreviousOutputs: map[string]json.RawMessage{"__input": inputJSON},
	})
	if out.NextStepID != "__branch:true" {
		t.Errorf("input.* path should resolve; got %s", out.NextStepID)
	}
}

func TestBranchAction_PrevPathReference(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"expression": `prev.step1.template == "warmup"`})
	prevJSON, _ := json.Marshal(map[string]string{"template": "warmup"})
	d := newTestDispatcher()
	out, _ := d.Dispatch(context.Background(), "branch", StepContext{
		Config:          cfg,
		PreviousOutputs: map[string]json.RawMessage{"step1": prevJSON},
	})
	if out.NextStepID != "__branch:true" {
		t.Errorf("prev.<id>.field should resolve; got %s", out.NextStepID)
	}
}

// --------------- register override ---------------------------------

type fakeAction struct{}

func (*fakeAction) Kind() string { return "send_message" }
func (*fakeAction) Execute(_ context.Context, _ StepContext) (StepOutcome, error) {
	return StepOutcome{Output: []byte(`{"overridden":true}`)}, nil
}

func TestDispatcher_RegisterOverrides(t *testing.T) {
	t.Parallel()
	d := newTestDispatcher()
	d.Register(&fakeAction{})
	out, _ := d.Dispatch(context.Background(), "send_message", StepContext{})
	if string(out.Output) != `{"overridden":true}` {
		t.Errorf("expected override output, got %s", out.Output)
	}
}

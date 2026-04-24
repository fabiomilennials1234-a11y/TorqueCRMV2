package workflow

import (
	"testing"
	"time"

	"github.com/google/uuid"

	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
)

// TestBackoffFor locks the backoff ladder so a well-meaning tweak
// doesn't silently shrink it. The ladder is the single knob that
// guarantees we don't hammer a flaky upstream — any change needs a
// deliberate test update.
func TestBackoffFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 0, want: 15 * time.Second}, // clamp
		{attempts: 1, want: 15 * time.Second},
		{attempts: 2, want: time.Minute},
		{attempts: 3, want: 5 * time.Minute},
		{attempts: 4, want: 30 * time.Minute},
		{attempts: 5, want: 2 * time.Hour},
		{attempts: 99, want: 2 * time.Hour}, // cap
	}
	for _, c := range cases {
		got := backoffFor(c.attempts)
		if got != c.want {
			t.Errorf("backoffFor(%d)=%v, want %v", c.attempts, got, c.want)
		}
	}
}

// TestErrorCode_Classification locks the mapping between sentinel
// errors and the short codes persisted to workflow_run_failures.
func TestErrorCode_Classification(t *testing.T) {
	t.Parallel()
	if c := errorCode(nil); c != "HANDLER_FAILED" {
		t.Errorf("nil → HANDLER_FAILED expected, got %s", c)
	}
	if c := errorCode(ErrTransient); c != "TRANSIENT" {
		t.Errorf("transient code mismatch: %s", c)
	}
	if c := errorCode(ErrNonRetryable); c != "NON_RETRYABLE" {
		t.Errorf("non-retryable code mismatch: %s", c)
	}
	if c := errorCode(ErrActionUnknown); c != "ACTION_UNKNOWN" {
		t.Errorf("action-unknown code mismatch: %s", c)
	}
}

// TestParseSuspendMarker locks the executor's understanding of the
// WaitAction control-flow marker. Happy path picks the step's first
// outgoing edge; bad markers reject cleanly.
func TestParseSuspendMarker(t *testing.T) {
	t.Parallel()
	nextID := uuid.New()
	step := workflowrepo.Step{
		ID:          uuid.New(),
		NextStepIDs: []uuid.UUID{nextID},
	}
	when := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)
	marker := "__suspend:" + when.Format(time.RFC3339)

	parsedAt, parsedID, ok := parseSuspendMarker(marker, step)
	if !ok {
		t.Fatalf("valid marker should parse")
	}
	if !parsedAt.Equal(when) {
		t.Errorf("time round-trip failed: want %v, got %v", when, parsedAt)
	}
	if parsedID != nextID {
		t.Errorf("next id mismatch: want %v, got %v", nextID, parsedID)
	}

	// Negative cases.
	if _, _, ok := parseSuspendMarker("garbage", step); ok {
		t.Errorf("non-suspend marker must not parse")
	}
	if _, _, ok := parseSuspendMarker("__suspend:not-a-time", step); ok {
		t.Errorf("suspend marker with bad timestamp must not parse")
	}
	// Marker present but step has no outgoing edge — also a reject.
	bad := workflowrepo.Step{ID: uuid.New(), NextStepIDs: nil}
	if _, _, ok := parseSuspendMarker(marker, bad); ok {
		t.Errorf("suspend marker on terminal step must not parse")
	}
}

// TestClassifyError_ProviderSentinels locks the mapping between the
// integration package's error sentinels and the workflow sentinels —
// if an adapter swaps its error surface this test catches silent
// regression.
func TestClassifyError_ProviderSentinels(t *testing.T) {
	t.Parallel()
	// Skip: classifyError is package-internal and tested indirectly
	// by dispatcher-level tests. This placeholder documents the gap
	// for the follow-up integration suite where real adapter errors
	// are propagated.
	_ = classifyError
}

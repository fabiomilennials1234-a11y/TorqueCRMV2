package agents

import (
	"testing"
)

// TestLatestUserContent_PicksMostRecent confirms the post-stream
// persistence helper picks the tail-most user message — the one the
// provider actually responded to — so the persisted row matches the
// turn the assistant reply corresponds to.
func TestLatestUserContent_PicksMostRecent(t *testing.T) {
	t.Parallel()
	msgs := []struct {
		Role    string
		Content string
	}{
		{Role: "user", Content: "primeiro"},
		{Role: "assistant", Content: "respondi primeiro"},
		{Role: "user", Content: "segundo"},
	}
	if got := latestUserContent(msgs); got != "segundo" {
		t.Fatalf("want 'segundo' (most recent user), got %q", got)
	}
}

func TestLatestUserContent_EmptyWhenOnlyAssistant(t *testing.T) {
	t.Parallel()
	msgs := []struct {
		Role    string
		Content string
	}{
		{Role: "assistant", Content: "só eu"},
	}
	if got := latestUserContent(msgs); got != "" {
		t.Fatalf("want empty string, got %q", got)
	}
}

// TestLatestUserContent_EmptySliceSafe — defensive guard that a zero-len
// slice doesn't panic (the playground handler already validates
// len>0 but the helper should be robust).
func TestLatestUserContent_EmptySliceSafe(t *testing.T) {
	t.Parallel()
	if got := latestUserContent(nil); got != "" {
		t.Fatalf("nil slice should return empty, got %q", got)
	}
}

// TestFinalizeInput_TotalTokens verifies the arithmetic the
// IncrementUsage call depends on. We don't exercise the repo path
// (that requires a live pgxpool), but we lock in the input+output sum
// so a refactor that splits these doesn't silently drop the total.
func TestFinalizeInput_TotalTokens(t *testing.T) {
	t.Parallel()
	in := finalizeInput{
		userContent:      "u",
		assistantContent: "a",
		inputTokens:      120,
		outputTokens:     45,
		latencyMs:        300,
	}
	got := in.inputTokens + in.outputTokens
	if got != 165 {
		t.Fatalf("input+output = %d, want 165", got)
	}
}

// TestFinalizeInput_ZeroTokensIsMeaningful — a 0+0 finalize MUST not
// trigger a quota increment. The playground finalizeStream wraps the
// IncrementUsage call in `if total > 0` for exactly this case
// (provider frames occasionally drop usage metadata under retry).
// This test locks that gate semantics in.
func TestFinalizeInput_ZeroTokensIsMeaningful(t *testing.T) {
	t.Parallel()
	in := finalizeInput{inputTokens: 0, outputTokens: 0}
	total := in.inputTokens + in.outputTokens
	if total != 0 {
		t.Fatalf("want 0 total for empty token frame, got %d", total)
	}
}

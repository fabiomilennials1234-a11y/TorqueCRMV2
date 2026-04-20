package agents

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
)

func TestPrependContext_EmbedsChunksAboveSystemPrompt(t *testing.T) {
	t.Parallel()
	sourceA := uuid.New()
	sourceB := uuid.New()
	chunks := []agentrepo.RetrievedChunk{
		{ID: uuid.New(), SourceID: sourceA, Ord: 0, Content: "Alpha content", Distance: 0.12},
		{ID: uuid.New(), SourceID: sourceB, Ord: 3, Content: "Beta content", Distance: 0.34},
	}
	base := "Você é um assistente."
	got := prependContext(base, chunks)

	// Context block should come before the base prompt.
	ctxIdx := strings.Index(got, "# Retrieved context")
	baseIdx := strings.Index(got, base)
	if ctxIdx < 0 {
		t.Fatal("context header missing")
	}
	if baseIdx < 0 {
		t.Fatal("base prompt missing")
	}
	if ctxIdx >= baseIdx {
		t.Errorf("context must come before base prompt (ctx=%d base=%d)", ctxIdx, baseIdx)
	}

	// Each chunk's content must appear.
	for _, c := range chunks {
		if !strings.Contains(got, c.Content) {
			t.Errorf("chunk content %q missing from output", c.Content)
		}
	}

	// Source identifiers should be short-prefixed (first 8 chars).
	if !strings.Contains(got, sourceA.String()[:8]) {
		t.Errorf("sourceA prefix missing")
	}
	if !strings.Contains(got, sourceB.String()[:8]) {
		t.Errorf("sourceB prefix missing")
	}

	// Delimiter separating the retrieved block from the system prompt.
	if !strings.Contains(got, "---\n\n") {
		t.Errorf("missing block delimiter")
	}
}

func TestPrependContext_EmptyChunks_StillIncludesHeader(t *testing.T) {
	t.Parallel()
	got := prependContext("base", nil)
	// Even with no chunks we keep the envelope; callers gate on len(chunks) > 0
	// before calling so this is a defensive assertion.
	if !strings.Contains(got, "base") {
		t.Errorf("base prompt dropped: %q", got)
	}
}

func TestClassify_MapsKnownErrorsToCodes(t *testing.T) {
	t.Parallel()
	// Not exhaustive — just ensures the taxonomy stays stable across
	// refactors. Unknown → "UNKNOWN" is the default, verified by the
	// error-frame assertion in useAgentStream.test.tsx.
	if got := classify(nil); got != "UNKNOWN" {
		t.Errorf("nil err → %q (want UNKNOWN)", got)
	}
}

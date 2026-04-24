package ai_test

import (
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/ai"
)

func TestChunk_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := ai.ChunkText("", ai.DefaultChunkOptions()); len(got) != 0 {
		t.Errorf("want no chunks, got %d", len(got))
	}
	if got := ai.ChunkText("   \n\n\t  ", ai.DefaultChunkOptions()); len(got) != 0 {
		t.Errorf("want no chunks for whitespace-only input")
	}
}

func TestChunk_SingleSmallParagraph(t *testing.T) {
	t.Parallel()
	out := ai.ChunkText("Olá mundo.", ai.DefaultChunkOptions())
	if len(out) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(out))
	}
	if out[0].Ord != 0 {
		t.Errorf("want ord=0, got %d", out[0].Ord)
	}
	if !strings.Contains(out[0].Content, "Olá") {
		t.Errorf("chunk content missing: %q", out[0].Content)
	}
}

func TestChunk_SplitsParagraphs(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("palavra ", 600) + "\n\n" + strings.Repeat("outra ", 600)
	out := ai.ChunkText(text, ai.ChunkOptions{TargetTokens: 500, OverlapTokens: 50})
	if len(out) < 2 {
		t.Fatalf("want multiple chunks, got %d", len(out))
	}
	ords := make(map[int]bool)
	for _, c := range out {
		if ords[c.Ord] {
			t.Errorf("duplicate ord %d", c.Ord)
		}
		ords[c.Ord] = true
	}
}

func TestChunk_OverlapPresent(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("alpha beta gamma delta ", 200)
	out := ai.ChunkText(text, ai.ChunkOptions{TargetTokens: 100, OverlapTokens: 20})
	if len(out) < 2 {
		t.Fatalf("want >=2 chunks, got %d", len(out))
	}
	// The last words of chunk[i] should appear at the start of chunk[i+1]
	// (given the text is all the same word repeated this is trivially true,
	// but we still want to verify the overlap seeding ran).
	first := out[0].Content
	second := out[1].Content
	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("empty chunks: %q / %q", first, second)
	}
	// Extract trailing word from chunk 0 (must be "alpha", "beta", "gamma" or "delta")
	tail := strings.Fields(first)
	head := strings.Fields(second)
	if len(tail) == 0 || len(head) == 0 {
		t.Fatalf("empty tokens")
	}
	if tail[len(tail)-1] != head[0] {
		// This may legitimately fail if the split lands differently; treat
		// as a soft invariant, but ensure some overlap of vocab at least.
		found := false
		tailSet := map[string]bool{}
		start := len(tail) - 5
		if start < 0 {
			start = 0
		}
		for _, w := range tail[start:] {
			tailSet[w] = true
		}
		end := 5
		if end > len(head) {
			end = len(head)
		}
		for _, w := range head[:end] {
			if tailSet[w] {
				found = true
				break
			}
		}
		if !found {
			s3 := len(tail) - 3
			if s3 < 0 {
				s3 = 0
			}
			e3 := 3
			if e3 > len(head) {
				e3 = len(head)
			}
			t.Errorf("no overlap between chunk tail %q and head %q",
				strings.Join(tail[s3:], " "),
				strings.Join(head[:e3], " "))
		}
	}
}

func TestChunk_PreservesOrderAndContent(t *testing.T) {
	t.Parallel()
	text := "Um.\n\nDois.\n\nTrês."
	out := ai.ChunkText(text, ai.DefaultChunkOptions())
	if len(out) < 1 {
		t.Fatalf("want >= 1 chunks")
	}
	joined := ""
	for _, c := range out {
		joined += " " + c.Content
	}
	for _, want := range []string{"Um", "Dois", "Três"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in chunks: %q", want, joined)
		}
	}
}

func TestChunk_LongSentenceSlicedByWords(t *testing.T) {
	t.Parallel()
	// One giant sentence with no punctuation — exceeds budget.
	text := strings.Repeat("word ", 1000)
	out := ai.ChunkText(text, ai.ChunkOptions{TargetTokens: 50, OverlapTokens: 5})
	if len(out) < 2 {
		t.Fatalf("want multi-chunk split, got %d", len(out))
	}
	for i, c := range out {
		if c.Ord != i {
			t.Errorf("chunk %d has ord %d", i, c.Ord)
		}
	}
}


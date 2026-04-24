package ai

import (
	"strings"
	"unicode"
)

// TextChunk is a piece of text small enough to fit in a retrieval context
// window. `Ord` preserves source order so rebuilding the original
// document after ingest is trivial.
//
// Renamed from `Chunk` in the CI repair sprint to disambiguate from
// `ai.Chunk` (LLM stream frame, openrouter.go) which lives in the same
// package and would otherwise redeclare the identifier.
type TextChunk struct {
	Ord            int
	Content        string
	TokensEstimate int
}

// ChunkOptions tunes the splitter.
type ChunkOptions struct {
	// TargetTokens is the soft upper bound on tokens per chunk.
	// ~500 is a good default — small enough to fit 4-6 chunks in a
	// typical 4k-token retrieval budget, large enough that paragraphs
	// stay coherent.
	TargetTokens int
	// OverlapTokens is the count of trailing tokens from chunk N-1
	// that we prepend to chunk N. Keeps continuity across boundaries
	// so queries that sit near a split still hit both sides.
	OverlapTokens int
}

// DefaultChunkOptions returns the S39 production defaults.
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{TargetTokens: 500, OverlapTokens: 50}
}

// ChunkText splits `text` into overlapping, paragraph-aware TextChunks.
//
// Tokens are approximated with a word count scaled by a tokens-per-word
// constant — empirically ~1.3 for English + Portuguese mixed text. We
// never call the tokenizer here because:
//  1. Chunking is CPU-bound on ingest and should not add a model round
//     trip.
//  2. The approximation is stable enough for target sizing; the actual
//     prompt budget check happens later when we concat chunks into the
//     system prompt.
//
// The splitter walks paragraphs first (blank line separated), then
// sentences, then words. It prefers natural boundaries but will slice
// mid-sentence if a single paragraph exceeds TargetTokens.
func ChunkText(text string, opts ChunkOptions) []TextChunk {
	if opts.TargetTokens <= 0 {
		opts = DefaultChunkOptions()
	}
	if opts.OverlapTokens < 0 {
		opts.OverlapTokens = 0
	}
	if opts.OverlapTokens >= opts.TargetTokens {
		opts.OverlapTokens = opts.TargetTokens / 5
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)

	var chunks []TextChunk
	var current []string
	currentTokens := 0
	ord := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		content := strings.TrimSpace(strings.Join(current, " "))
		if content == "" {
			current = current[:0]
			currentTokens = 0
			return
		}
		chunks = append(chunks, TextChunk{
			Ord:            ord,
			Content:        content,
			TokensEstimate: currentTokens,
		})
		ord++
		// Seed next chunk with the last OverlapTokens worth of words.
		if opts.OverlapTokens > 0 {
			overlap := trailingTokens(content, opts.OverlapTokens)
			if overlap != "" {
				current = []string{overlap}
				currentTokens = estimateTokens(overlap)
				return
			}
		}
		current = current[:0]
		currentTokens = 0
	}

	for _, para := range paragraphs {
		paraTokens := estimateTokens(para)
		if currentTokens+paraTokens <= opts.TargetTokens {
			current = append(current, para)
			currentTokens += paraTokens
			continue
		}
		// Flush what we have, then decide.
		if currentTokens > 0 {
			flush()
		}
		if paraTokens <= opts.TargetTokens {
			current = append(current, para)
			currentTokens += paraTokens
			continue
		}
		// Paragraph alone exceeds budget — split by sentences.
		for _, sentence := range splitSentences(para) {
			sentTokens := estimateTokens(sentence)
			if currentTokens+sentTokens > opts.TargetTokens && currentTokens > 0 {
				flush()
			}
			if sentTokens > opts.TargetTokens {
				// Even a single sentence is too long — slice by words.
				words := strings.Fields(sentence)
				for _, w := range words {
					wTokens := estimateTokens(w)
					if currentTokens+wTokens > opts.TargetTokens && currentTokens > 0 {
						flush()
					}
					current = append(current, w)
					currentTokens += wTokens
				}
				continue
			}
			current = append(current, sentence)
			currentTokens += sentTokens
		}
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitSentences is a pragmatic regex-less splitter. Treats '.', '!',
// '?' as terminators, keeps them attached, skips empties. Good enough
// for chunk boundaries where exact sentence counts are irrelevant.
func splitSentences(p string) []string {
	var out []string
	var cur strings.Builder
	for _, r := range p {
		cur.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			sent := strings.TrimSpace(cur.String())
			if sent != "" {
				out = append(out, sent)
			}
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		sent := strings.TrimSpace(cur.String())
		if sent != "" {
			out = append(out, sent)
		}
	}
	if len(out) == 0 {
		return []string{p}
	}
	return out
}

// estimateTokens converts a word count to an approximate token count.
// The 1.3 multiplier matches the average OpenAI / Gemini tokenizer
// ratio for mixed English / Portuguese text. Exact enough for sizing.
func estimateTokens(s string) int {
	words := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			words++
			inWord = true
		}
	}
	est := int(float64(words) * 1.3)
	if est < 1 && words > 0 {
		est = 1
	}
	return est
}

// trailingTokens returns the last approximately `tokens` worth of
// words from `s`. Used to seed the next chunk with overlap context.
func trailingTokens(s string, tokens int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	// Reverse-walk: accumulate word estimates until we cover ~tokens.
	taken := 0
	cut := len(words)
	for i := len(words) - 1; i >= 0; i-- {
		taken += estimateTokens(words[i])
		if taken >= tokens {
			cut = i
			break
		}
		cut = i
	}
	return strings.Join(words[cut:], " ")
}

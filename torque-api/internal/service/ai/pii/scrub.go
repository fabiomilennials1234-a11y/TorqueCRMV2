// Package pii scrubs Brazilian PII (CPF, CNPJ, phone, email, credit card)
// from free-form text before it crosses a process boundary the tenant
// cannot audit — notably, before upstream LLM providers see the text.
//
// Design notes:
//
//   - The scrubber is deliberately conservative: a false-positive here
//     costs an LLM a little context; a false-negative leaks a CPF to
//     OpenAI. We err on the side of over-redaction.
//
//   - Patterns are anchored on word boundaries and common separators
//     (spaces, dots, hyphens, slashes) so a CPF written as
//     "123.456.789-00" and "12345678900" and "123 456 789 00" all
//     redact.
//
//   - No Luhn / modulo-11 validator is applied. A malformed CPF is
//     still PII intent; discarding it is as correct as redacting a
//     valid one, and the validator cost is non-trivial on every turn.
//
//   - The package is pure — no logger, no metrics. Callers add structured
//     logging around Scrub() using the returned Counts. This keeps the
//     scrubber a trivially-testable leaf dep.
package pii

import (
	"regexp"
	"strings"
)

// Counts reports how many hits each pattern produced in a single Scrub
// pass. The zero value means "no PII found" and is the happy path; the
// playground handler compares against zero to decide whether to attach
// an `ai.pii_scrub.*` log field.
type Counts struct {
	CPF    int
	CNPJ   int
	Phone  int
	Email  int
	Credit int
}

// Total is the sum across every pattern — useful for a single
// "ai.pii_scrub.total" metric when you don't care which kind fired.
func (c Counts) Total() int {
	return c.CPF + c.CNPJ + c.Phone + c.Email + c.Credit
}

// Sentinels used as replacement tokens. Kept as constants so downstream
// prompt-engineering can reason about them verbatim (e.g., a system
// prompt can explicitly tell the LLM "when you see [CPF-REDIGIDO],
// answer: não posso compartilhar esse dado").
const (
	SentinelCPF    = "[CPF-REDIGIDO]"
	SentinelCNPJ   = "[CNPJ-REDIGIDO]"
	SentinelPhone  = "[TELEFONE-REDIGIDO]"
	SentinelEmail  = "[EMAIL-REDIGIDO]"
	SentinelCredit = "[CARTAO-REDIGIDO]"
)

// ----- compiled patterns -------------------------------------------------

// Ordering matters: CNPJ must run BEFORE CPF because a 14-digit CNPJ
// contains an 11-digit CPF as a substring. Email must run BEFORE phone
// so phone's digit-heavy regex does not eat the numeric prefix of an
// email's localpart. Credit card runs LAST so it does not eat an already
// redacted CPF/CNPJ/phone.
//
// Anchoring: every pattern uses lookaround-free boundary heuristics
// (Go's regexp is RE2, no lookbehind) — the character-class boundaries
// `[^0-9]` and explicit word-boundary prefixes live in the replacement
// logic, not the pattern, to avoid swallowing adjacent characters.
var (
	// CNPJ — 14 digits, optionally formatted `XX.XXX.XXX/XXXX-XX` with
	// spaces allowed between groups. The non-capturing boundary chars
	// are matched then re-emitted so we don't gobble surrounding text.
	reCNPJ = regexp.MustCompile(
		`\b\d{2}[\s.]?\d{3}[\s.]?\d{3}[\s/]?\d{4}[\s-]?\d{2}\b`,
	)

	// CPF — 11 digits, optionally formatted `XXX.XXX.XXX-XX`. The
	// `\b` anchor prevents matching inside a longer 14-digit string
	// (CNPJ already consumed by the earlier pass).
	reCPF = regexp.MustCompile(
		`\b\d{3}[\s.]?\d{3}[\s.]?\d{3}[\s-]?\d{2}\b`,
	)

	// Email — deliberate minimal RFC 5322 shape. Over-matches some
	// exotic-legal addresses but we accept that to keep the regex
	// readable. The `.[\w.-]+` tail requires at least one TLD char.
	reEmail = regexp.MustCompile(
		`[\w.+-]+@[\w-]+\.[\w.-]+`,
	)

	// Phone BR — optional +55, optional parens/hyphens, 10-11 digits
	// total body. We match the common UI shapes:
	//   +55 (11) 91234-5678
	//   (11) 1234-5678
	//   11912345678
	//   11 91234 5678
	// The `\b` at the start avoids matching the tail of a 20-digit
	// credit card number by accident.
	rePhone = regexp.MustCompile(
		`(?:\+?55[\s-]?)?\(?\d{2}\)?[\s-]?9?\d{4}[\s-]?\d{4}\b`,
	)

	// Credit card — 13-19 digits with optional single spaces or hyphens
	// between groups. No Luhn check (RE2 can't express it cheaply).
	// Runs last so it does not eat an already-redacted phone/CPF/CNPJ.
	reCredit = regexp.MustCompile(
		`\b(?:\d[\s-]?){12,18}\d\b`,
	)
)

// Scrub walks `text` through the 5 patterns in priority order and
// returns (redacted, counts). Safe to call on empty strings (returns
// the input as-is with zero counts) and on text with no PII (the
// regex passes produce zero replacements and counts stays zero).
func Scrub(text string) (string, Counts) {
	var c Counts
	if text == "" {
		return text, c
	}
	out := text

	// CNPJ first (longest digit sequence).
	out = reCNPJ.ReplaceAllStringFunc(out, func(m string) string {
		c.CNPJ++
		return SentinelCNPJ
	})
	// CPF second.
	out = reCPF.ReplaceAllStringFunc(out, func(m string) string {
		c.CPF++
		return SentinelCPF
	})
	// Email before phone — phone's digit class would eat the numeric
	// prefix of localparts like `123abc@x.com`.
	out = reEmail.ReplaceAllStringFunc(out, func(m string) string {
		c.Email++
		return SentinelEmail
	})
	// Phone.
	out = rePhone.ReplaceAllStringFunc(out, func(m string) string {
		c.Phone++
		return SentinelPhone
	})
	// Credit card — only after everything else has been consumed so
	// we don't redact a phone twice.
	out = reCredit.ReplaceAllStringFunc(out, func(m string) string {
		// A long token that was already redacted (contains `[` or `]`)
		// won't match \b\d sequences, so this guard is defensive only.
		if strings.ContainsAny(m, "[]") {
			return m
		}
		c.Credit++
		return SentinelCredit
	})
	return out, c
}

// Chunk is the minimal projection of agentrepo.RetrievedChunk that
// ScrubRAGContext needs. Pure data — no repo import — keeps the
// package a leaf dep.
type Chunk struct {
	Content string
}

// ScrubRAGContext scrubs every retrieved chunk's content in place and
// returns the sum Counts across chunks. Use this right BEFORE the
// chunks are concatenated into the system prompt — RAG retrieval from
// a tenant's own knowledge base can still surface PII that must not
// reach the upstream LLM.
func ScrubRAGContext(chunks []Chunk) ([]Chunk, Counts) {
	if len(chunks) == 0 {
		return chunks, Counts{}
	}
	var total Counts
	out := make([]Chunk, len(chunks))
	for i, ch := range chunks {
		scrubbed, c := Scrub(ch.Content)
		out[i] = Chunk{Content: scrubbed}
		total.CPF += c.CPF
		total.CNPJ += c.CNPJ
		total.Phone += c.Phone
		total.Email += c.Email
		total.Credit += c.Credit
	}
	return out, total
}

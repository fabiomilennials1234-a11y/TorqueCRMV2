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

// Ordering matters. Final order pos-fix dos test fails do CI repair:
//
//   1. Email (anchored em '@', mais especifico).
//   2. CNPJ (14 digitos formatted).
//   3. Phone (10-13 digitos com optional +55 prefix). DEVE rodar ANTES de
//      Credit porque "+5511987654321" (13 digitos com prefix) matches AMBOS
//      Phone (E.164) e Credit (13+ digitos seq). Phone wins por ser uso
//      conversacional mais comum. rePhone tem `\b` prefix anchor pra evitar
//      consumir parte de strings mais longas (e.g. nao engole 10 trailing
//      digitos de um CPF unformatted "12345678900").
//   4. Credit card (13-19 digitos sequencia continua). Roda antes de CPF
//      pra capturar 16-digitos sem separator (e.g. 4111111111111111) que
//      conteriam 11-digit CPF substrings.
//   5. CPF (11 digitos formatted ou unformatted) — captura 11 digitos que
//      nao satisfazem o "9 prefix trigger" do rePhone (e.g. "12345678900").
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
	// Brazil phone shape com 2 alternatives pra distinguir cartoes de
	// credito 13+ digitos sem separator:
	//   A) E.164 com mandatory "+55" prefix — relaxado internamente
	//      porque o "+" e marker inequivoco.
	//   B) Domestic — DDD + (mandatory "9" mobile prefix OU mandatory
	//      separator entre DDD e os 4 primeiros digitos do landline).
	// Pattern original `(?:\+?55[\s-]?)?...9?...` aceitava "5555-4444-..."
	// como "55" prefix + DDD + 4 + 4, engolindo cartao 13-19 digitos.
	rePhone = regexp.MustCompile(
		`(?:\+55[\s-]?\(?\d{2}\)?[\s-]?9?\d{4}[\s-]?\d{4}|\b\(?\d{2}\)?(?:[\s-]?9\d{4}|[\s-]\d{4})[\s-]?\d{4})\b`,
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

	// 1. Email primeiro (anchored em '@', mais especifico).
	out = reEmail.ReplaceAllStringFunc(out, func(m string) string {
		c.Email++
		return SentinelEmail
	})
	// 2. CNPJ (14 digitos formatted).
	out = reCNPJ.ReplaceAllStringFunc(out, func(m string) string {
		c.CNPJ++
		return SentinelCNPJ
	})
	// 3. Phone antes de Credit — E.164 "+5511987654321" matches ambos;
	// phone wins por ser uso conversacional. \b prefix evita engolir
	// parte de CPF unformatted como "12345678900".
	out = rePhone.ReplaceAllStringFunc(out, func(m string) string {
		c.Phone++
		return SentinelPhone
	})
	// 4. Credit card antes de CPF — cartoes 16-digitos sem separator
	// conteriam 11-digit CPF substrings se CPF rodasse antes.
	out = reCredit.ReplaceAllStringFunc(out, func(m string) string {
		if strings.ContainsAny(m, "[]") {
			return m
		}
		c.Credit++
		return SentinelCredit
	})
	// 5. CPF captura 11-digit sequences que nao satisfizeram o "9 prefix
	// trigger" do phone regex (e.g. "12345678900").
	out = reCPF.ReplaceAllStringFunc(out, func(m string) string {
		c.CPF++
		return SentinelCPF
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

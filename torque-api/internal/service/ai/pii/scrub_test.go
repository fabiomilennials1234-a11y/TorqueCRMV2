package pii

import (
	"strings"
	"testing"
)

// TestScrub exercises every pattern against realistic inputs. Table is
// split by category so a regression in one pattern fails a single
// sub-test with a descriptive name rather than a single mega-test.
func TestScrub(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  string // substring that MUST appear in the output
		notIn string // substring that MUST NOT appear (original PII)
		wantCounts Counts
	}{
		// ----- CPF ------------------------------------------------------
		{
			name:       "cpf_formatted",
			in:         "Meu CPF é 123.456.789-00, pode confirmar?",
			want:       SentinelCPF,
			notIn:      "123.456.789",
			wantCounts: Counts{CPF: 1},
		},
		{
			name:       "cpf_unformatted",
			in:         "CPF 12345678900 do cliente",
			want:       SentinelCPF,
			notIn:      "12345678900",
			wantCounts: Counts{CPF: 1},
		},
		{
			name:       "cpf_spaced",
			in:         "123 456 789 00",
			want:       SentinelCPF,
			notIn:      "123 456 789 00",
			wantCounts: Counts{CPF: 1},
		},
		{
			name:       "cpf_two_in_one_line",
			in:         "Primeiro 111.222.333-44 e segundo 555.666.777-88",
			want:       SentinelCPF,
			notIn:      "111.222.333-44",
			wantCounts: Counts{CPF: 2},
		},

		// ----- CNPJ -----------------------------------------------------
		{
			name:       "cnpj_formatted",
			in:         "CNPJ 12.345.678/0001-99 vencendo",
			want:       SentinelCNPJ,
			notIn:      "12.345.678",
			wantCounts: Counts{CNPJ: 1},
		},
		{
			name:       "cnpj_unformatted",
			in:         "cnpj: 12345678000199",
			want:       SentinelCNPJ,
			notIn:      "12345678000199",
			wantCounts: Counts{CNPJ: 1},
		},

		// ----- Phone BR -------------------------------------------------
		{
			name:       "phone_e164",
			in:         "Ligue para +55 11 91234-5678 hoje",
			want:       SentinelPhone,
			notIn:      "91234-5678",
			wantCounts: Counts{Phone: 1},
		},
		{
			name:       "phone_parens",
			in:         "Telefone (11) 91234-5678",
			want:       SentinelPhone,
			notIn:      "(11) 91234-5678",
			wantCounts: Counts{Phone: 1},
		},
		{
			name:       "phone_no_9",
			in:         "Comercial (11) 3344-5566",
			want:       SentinelPhone,
			notIn:      "3344-5566",
			wantCounts: Counts{Phone: 1},
		},
		{
			name:       "phone_unformatted",
			in:         "Whatsapp 11912345678",
			want:       SentinelPhone,
			notIn:      "11912345678",
			wantCounts: Counts{Phone: 1},
		},

		// ----- Email ----------------------------------------------------
		{
			name:       "email_basic",
			in:         "Chame no ana.silva@empresa.com.br por favor",
			want:       SentinelEmail,
			notIn:      "ana.silva@empresa.com.br",
			wantCounts: Counts{Email: 1},
		},
		{
			name:       "email_plus_tag",
			in:         "meu email: user+tag@domain.io",
			want:       SentinelEmail,
			notIn:      "user+tag@domain.io",
			wantCounts: Counts{Email: 1},
		},

		// ----- Credit card ----------------------------------------------
		{
			name:       "credit_16_spaced",
			in:         "Cartão 4111 1111 1111 1111 validade 12/28",
			want:       SentinelCredit,
			notIn:      "4111 1111 1111 1111",
			wantCounts: Counts{Credit: 1},
		},
		{
			name:       "credit_16_hyphen",
			in:         "5555-4444-3333-2222",
			want:       SentinelCredit,
			notIn:      "5555-4444-3333-2222",
			wantCounts: Counts{Credit: 1},
		},
		{
			name:       "credit_15_amex",
			in:         "378282246310005 amex",
			want:       SentinelCredit,
			notIn:      "378282246310005",
			wantCounts: Counts{Credit: 1},
		},

		// ----- Mixed ----------------------------------------------------
		{
			name:       "mixed_cpf_and_email",
			in:         "CPF 123.456.789-00 email joao@corp.com",
			want:       SentinelCPF,
			notIn:      "123.456.789-00",
			wantCounts: Counts{CPF: 1, Email: 1},
		},
		{
			name: "mixed_all_kinds",
			in: "Cliente João, CPF 111.222.333-44, CNPJ 12.345.678/0001-99, " +
				"telefone (11) 98765-4321, email j@c.com, cartão 4111111111111111",
			want:       SentinelCPF,
			notIn:      "111.222.333-44",
			wantCounts: Counts{CPF: 1, CNPJ: 1, Phone: 1, Email: 1, Credit: 1},
		},

		// ----- Negatives ------------------------------------------------
		{
			name:       "no_pii",
			in:         "olá, tudo bem por aí?",
			want:       "olá, tudo bem por aí?",
			notIn:      "REDIGIDO",
			wantCounts: Counts{},
		},
		{
			name:       "short_numbers_are_safe",
			in:         "O total foi 42 itens em 3 pedidos",
			want:       "42",
			notIn:      "REDIGIDO",
			wantCounts: Counts{},
		},
		{
			name:       "empty_string",
			in:         "",
			want:       "",
			notIn:      "REDIGIDO",
			wantCounts: Counts{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, counts := Scrub(tc.in)
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("Scrub output does not contain %q.\n got: %s", tc.want, got)
			}
			if tc.notIn != "" && tc.wantCounts.Total() > 0 && strings.Contains(got, tc.notIn) {
				t.Fatalf("Scrub output STILL contains raw PII %q.\n got: %s", tc.notIn, got)
			}
			if counts != tc.wantCounts {
				t.Fatalf("Counts mismatch.\n want: %+v\n  got: %+v\n input: %q\n output: %q",
					tc.wantCounts, counts, tc.in, got)
			}
		})
	}
}

// TestScrubRAGContext exercises the chunk-slice path: counts accumulate
// across chunks, Content is redacted in place on a COPY (no aliasing).
func TestScrubRAGContext(t *testing.T) {
	chunks := []Chunk{
		{Content: "Cliente CPF 111.222.333-44 tem plano Growth."},
		{Content: "Entrar em contato: anapaula@empresa.com.br"},
		{Content: "sem pii aqui"},
	}
	out, total := ScrubRAGContext(chunks)
	if len(out) != len(chunks) {
		t.Fatalf("expected %d output chunks, got %d", len(chunks), len(out))
	}
	if total.CPF != 1 || total.Email != 1 {
		t.Fatalf("expected counts{CPF:1, Email:1, ...}, got %+v", total)
	}
	if strings.Contains(out[0].Content, "111.222.333-44") {
		t.Fatalf("CPF leaked through into output chunk 0: %q", out[0].Content)
	}
	if strings.Contains(out[1].Content, "anapaula@empresa.com.br") {
		t.Fatalf("Email leaked through into output chunk 1: %q", out[1].Content)
	}
	if out[2].Content != chunks[2].Content {
		t.Fatalf("clean chunk was mutated: %q vs %q", out[2].Content, chunks[2].Content)
	}
	// Confirm we returned a new slice (no aliasing that could surprise callers).
	if &out[0] == &chunks[0] {
		t.Fatalf("ScrubRAGContext must return a new slice header, not alias the input")
	}
}

// TestScrubEmpty — defensive check that an empty chunk slice returns
// zero counts without a nil-deref.
func TestScrubEmptyChunks(t *testing.T) {
	out, total := ScrubRAGContext(nil)
	if out != nil {
		t.Fatalf("expected nil passthrough, got %v", out)
	}
	if total != (Counts{}) {
		t.Fatalf("expected zero counts, got %+v", total)
	}
}

// TestCountsTotal — sanity on the Total() summary.
func TestCountsTotal(t *testing.T) {
	c := Counts{CPF: 1, CNPJ: 2, Phone: 3, Email: 4, Credit: 5}
	if c.Total() != 15 {
		t.Fatalf("Total() = %d; want 15", c.Total())
	}
}

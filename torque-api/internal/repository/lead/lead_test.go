package lead

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursor_RoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 19, 12, 34, 56, 789_000_000, time.UTC)
	id := uuid.New()
	tok := encodeCursor(now, id)
	if tok == "" {
		t.Fatal("empty cursor")
	}
	gotT, gotID, err := decodeCursor(tok)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotT.Equal(now) {
		t.Fatalf("time drift: %v != %v", gotT, now)
	}
	if gotID != id {
		t.Fatalf("id drift: %v != %v", gotID, id)
	}
}

func TestDecodeCursor_RejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []string{"", "not-base64", "bm90LWEtcGlwZXM", "====", "aW52YWxpZA"}
	for _, c := range cases {
		if _, _, err := decodeCursor(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

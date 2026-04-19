package password_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/password"
)

func TestHashAndVerify_RoundTrip(t *testing.T) {
	t.Parallel()
	h, err := password.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if h == "" {
		t.Fatal("empty hash")
	}
	if err := password.Verify(h, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("verify same: %v", err)
	}
}

func TestVerify_MismatchReturnsSentinel(t *testing.T) {
	t.Parallel()
	h, _ := password.Hash("one")
	err := password.Verify(h, "two")
	if !errors.Is(err, password.ErrMismatch) {
		t.Fatalf("expected ErrMismatch, got %v", err)
	}
}

func TestHash_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := password.Hash(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestHash_RejectsBcryptCap(t *testing.T) {
	t.Parallel()
	if _, err := password.Hash(strings.Repeat("a", 73)); err == nil {
		t.Fatal("expected error for >72-byte password")
	}
}

func TestNeedsRehash_FalseForCurrentCost(t *testing.T) {
	t.Parallel()
	h, _ := password.Hash("x")
	if password.NeedsRehash(h) {
		t.Fatal("current-cost hash should not need rehash")
	}
}

func TestNeedsRehash_TrueForMalformed(t *testing.T) {
	t.Parallel()
	if !password.NeedsRehash("not-a-bcrypt-hash") {
		t.Fatal("malformed hash must signal rehash")
	}
}

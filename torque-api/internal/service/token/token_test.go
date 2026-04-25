package token_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/token"
)

func TestGenerate_URLSafeHighEntropy(t *testing.T) {
	t.Parallel()
	a := token.Generate()
	b := token.Generate()
	if a == b {
		t.Fatal("token collisions signal broken rand source")
	}
	if len(a) < 40 {
		t.Fatalf("unexpectedly short token: %d chars", len(a))
	}
	if strings.ContainsAny(a, "+/=") {
		t.Fatalf("expected url-safe base64, got %q", a)
	}
}

func TestHash_Deterministic(t *testing.T) {
	t.Parallel()
	h1 := token.Hash("abc")
	h2 := token.Hash("abc")
	if h1 != h2 {
		t.Fatal("hash must be deterministic")
	}
	if token.Hash("abc") == token.Hash("abd") {
		t.Fatal("different inputs must produce different hashes")
	}
	if len(token.Hash("abc")) != 64 {
		t.Fatal("sha256 hex must be 64 chars")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	t.Parallel()
	if err := token.ConstantTimeEqual("foo", "foo"); err != nil {
		t.Fatalf("equal: %v", err)
	}
	if err := token.ConstantTimeEqual("foo", "bar"); !errors.Is(err, token.ErrMismatch) {
		t.Fatalf("expected ErrMismatch, got %v", err)
	}
	if err := token.ConstantTimeEqual("foo", "foos"); !errors.Is(err, token.ErrMismatch) {
		t.Fatalf("expected ErrMismatch for different length, got %v", err)
	}
}

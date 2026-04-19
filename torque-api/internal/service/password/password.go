// Package password wraps bcrypt with Torque's chosen cost factor.
//
// Cost 12 is the 2026 sweet spot on commodity hardware (~250ms per hash on
// a modern server). Raise to 13 once CPUs get ~2x faster or profiling shows
// sub-100ms hashes dominate login budget.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Cost is the bcrypt work factor. Exported so tests may read it; do not mutate.
const Cost = 12

// ErrMismatch is returned by Verify when the hash does not match the password.
// Callers MUST NOT distinguish this from "no such user" in responses — that
// is how user enumeration begins.
var ErrMismatch = errors.New("password mismatch")

// Hash derives a bcrypt hash from a plaintext password.
//
// Returns an error on empty input; bcrypt itself caps inputs at 72 bytes and
// silently truncates, which we explicitly refuse so callers cannot store a
// hash of a truncated password by accident.
func Hash(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("password is empty")
	}
	if len(plaintext) > 72 {
		return "", fmt.Errorf("password length %d exceeds bcrypt 72-byte cap", len(plaintext))
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), Cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(h), nil
}

// Verify compares a plaintext password against a stored hash in constant time.
// Returns ErrMismatch on mismatch, nil on success, or an unwrapped error if
// the hash itself is malformed (non-bcrypt, corrupted, or wrong cost).
func Verify(hash, plaintext string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	switch {
	case err == nil:
		return nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrMismatch
	default:
		return fmt.Errorf("bcrypt verify: %w", err)
	}
}

// NeedsRehash reports whether a stored hash uses a lower cost than the
// current target. Call this after Verify succeeds to opportunistically upgrade.
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		// Malformed hash — signal "yes rehash" so the caller replaces it.
		return true
	}
	return cost < Cost
}

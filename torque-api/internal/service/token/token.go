// Package token generates and fingerprints opaque bearer tokens (refresh,
// CSRF). Every value returned by Generate is cryptographically random and
// URL-safe; nothing here involves the database.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// defaultBytes is 32 bytes ≈ 256 bits of entropy. Well above any practical
// brute-force or birthday bound; base64url encodes to 43 chars (no padding).
const defaultBytes = 32

// Generate returns a URL-safe random string of 256-bit entropy.
// Panics only if the OS entropy source fails, which is a fatal system state.
func Generate() string {
	b := make([]byte, defaultBytes)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Hash computes the sha256 hex digest of a raw token. Stored in DB.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ErrMismatch is returned by ConstantTimeEqual when two values differ.
var ErrMismatch = errors.New("token mismatch")

// ConstantTimeEqual compares two token strings in constant time. Returns
// ErrMismatch on inequality. Used for CSRF double-submit validation so a
// timing attack cannot discover a valid token byte-by-byte.
func ConstantTimeEqual(a, b string) error {
	if len(a) != len(b) {
		return ErrMismatch
	}
	if subtle.ConstantTimeCompare([]byte(a), []byte(b)) != 1 {
		return ErrMismatch
	}
	return nil
}

// Package crypto wraps AES-256-GCM in a minimal authenticated-cipher API
// for use by integration_credentials storage.
//
// Semantics:
//   - The key is exactly 32 bytes (AES-256).
//   - Encrypt returns nonce||ciphertext||tag. Nonce is 12 bytes fresh random
//     per call (required for GCM — nonce reuse is catastrophic).
//   - Decrypt splits the envelope and verifies the tag; tampered or
//     truncated inputs return an error and never leak a partial plaintext.
//
// Nothing here ever logs plaintext or the key. Errors surface as generic
// "decrypt failed" / "envelope too short" strings; callers that need more
// context wrap with additional metadata (without the secret material).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidKey is returned when the supplied key is not exactly 32 bytes
// after base64 decoding.
var ErrInvalidKey = errors.New("crypto: key must be 32 bytes after base64 decoding")

// ErrShortEnvelope means the supplied ciphertext is shorter than the minimum
// (12-byte nonce + 16-byte GCM tag).
var ErrShortEnvelope = errors.New("crypto: envelope truncated")

// ErrDecrypt is the opaque decrypt-failure error. Callers receive only this
// — we never distinguish "wrong key" from "tampered tag" from "tampered
// ciphertext" since any of those is a crypto-level invariant break.
var ErrDecrypt = errors.New("crypto: decrypt failed")

// Cipher is a reusable AES-256-GCM wrapper.
type Cipher struct {
	gcm cipher.AEAD
}

// New builds a Cipher from a base64-encoded key. Accepts either standard or
// URL-safe base64 (with or without padding). Returns ErrInvalidKey when the
// decoded byte length is not exactly 32.
func New(keyBase64 string) (*Cipher, error) {
	raw, err := decodeKey(keyBase64)
	if err != nil {
		return nil, err
	}
	if len(raw) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: gcm: %w", err)
	}
	return &Cipher{gcm: gcm}, nil
}

// Encrypt seals plaintext. The returned envelope is nonce||ciphertext||tag.
// A fresh 12-byte random nonce is drawn per call via crypto/rand.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	if c == nil || c.gcm == nil {
		return nil, errors.New("crypto: cipher not initialized")
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	// gcm.Seal(dst, nonce, plaintext, ad) → dst || ciphertext || tag
	out := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return out, nil
}

// Decrypt opens an envelope produced by Encrypt. Returns ErrShortEnvelope
// when too small to contain nonce+tag, ErrDecrypt on any other failure.
func (c *Cipher) Decrypt(envelope []byte) ([]byte, error) {
	if c == nil || c.gcm == nil {
		return nil, errors.New("crypto: cipher not initialized")
	}
	ns := c.gcm.NonceSize()
	if len(envelope) < ns+c.gcm.Overhead() {
		return nil, ErrShortEnvelope
	}
	nonce := envelope[:ns]
	ct := envelope[ns:]
	pt, err := c.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}

// decodeKey tries URL-safe base64 first, then standard base64 (both in raw
// and padded form). A key that happens to be literal 32 bytes is rejected
// upstream by the length check — callers must use base64.
func decodeKey(s string) ([]byte, error) {
	var last error
	for _, enc := range []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	} {
		raw, err := enc.DecodeString(s)
		if err == nil {
			return raw, nil
		}
		last = err
	}
	return nil, fmt.Errorf("crypto: base64 decode: %w", last)
}

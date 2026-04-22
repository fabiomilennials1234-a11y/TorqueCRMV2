package crypto_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/milennials/torque-api/internal/service/crypto"
)

func makeKeyB64(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestCipher_RoundTrip(t *testing.T) {
	t.Parallel()
	c, err := crypto.New(makeKeyB64(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	plain := []byte("hello torque — this is a refresh_token_abc123")
	env, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(env, plain) {
		t.Fatalf("envelope leaks plaintext")
	}
	got, err := c.Decrypt(env)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("roundtrip mismatch: got %q", got)
	}
}

func TestCipher_EmptyPlaintextRoundTrip(t *testing.T) {
	t.Parallel()
	c, err := crypto.New(makeKeyB64(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	env, err := c.Encrypt(nil)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := c.Decrypt(env)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty plaintext, got %d bytes", len(got))
	}
}

func TestCipher_Tampered(t *testing.T) {
	t.Parallel()
	c, err := crypto.New(makeKeyB64(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	env, _ := c.Encrypt([]byte("secret"))
	// Flip one bit in the ciphertext area (after the 12-byte nonce).
	tampered := append([]byte{}, env...)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := c.Decrypt(tampered); !errors.Is(err, crypto.ErrDecrypt) {
		t.Fatalf("expected ErrDecrypt, got %v", err)
	}
}

func TestCipher_Truncated(t *testing.T) {
	t.Parallel()
	c, err := crypto.New(makeKeyB64(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := c.Decrypt([]byte{1, 2, 3}); !errors.Is(err, crypto.ErrShortEnvelope) {
		t.Fatalf("expected ErrShortEnvelope, got %v", err)
	}
}

func TestCipher_WrongKeySize(t *testing.T) {
	t.Parallel()
	// 16 bytes base64-encoded — AES-128 not AES-256.
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := crypto.New(short); !errors.Is(err, crypto.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey for short key, got %v", err)
	}
	// 64 bytes — also wrong.
	long := base64.StdEncoding.EncodeToString(make([]byte, 64))
	if _, err := crypto.New(long); !errors.Is(err, crypto.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey for long key, got %v", err)
	}
}

func TestCipher_KeyFormats(t *testing.T) {
	t.Parallel()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	std := base64.StdEncoding.EncodeToString(raw)
	url := base64.URLEncoding.EncodeToString(raw)
	rawURL := base64.RawURLEncoding.EncodeToString(raw)

	for _, s := range []string{std, url, rawURL} {
		if _, err := crypto.New(s); err != nil {
			t.Fatalf("accepted format failed: %v", err)
		}
	}
}

func TestCipher_WrongKeyRejectsOtherCiphertext(t *testing.T) {
	t.Parallel()
	a, _ := crypto.New(makeKeyB64(t))
	b, _ := crypto.New(makeKeyB64(t))
	env, _ := a.Encrypt([]byte("payload"))
	if _, err := b.Decrypt(env); !errors.Is(err, crypto.ErrDecrypt) {
		t.Fatalf("expected ErrDecrypt when opening with a different key, got %v", err)
	}
}

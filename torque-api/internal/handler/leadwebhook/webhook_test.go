package leadwebhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// verifyHMAC is the only piece of the handler that can be exercised
// without a live pgx pool. The end-to-end path (record → create lead →
// attach → publish) is covered by an integration test that spins the
// real repositories against a test Postgres — see the CI suite.

func TestVerifyHMAC_HappyPath(t *testing.T) {
	t.Parallel()
	secret := []byte("super-secret-key-at-least-32-chars")
	body := []byte(`{"external_id":"abc","name":"Ana","phone":"11999"}`)
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !verifyHMAC(secret, body, sig) {
		t.Fatal("valid signature rejected")
	}
}

func TestVerifyHMAC_RejectsMissingPrefix(t *testing.T) {
	t.Parallel()
	secret := []byte("s")
	body := []byte("x")
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	// Plain hex without "sha256=" prefix must be rejected — defeats
	// a downgrade attack where a bystander swaps the prefix.
	if verifyHMAC(secret, body, hex.EncodeToString(mac.Sum(nil))) {
		t.Fatal("unprefixed hash accepted")
	}
}

func TestVerifyHMAC_RejectsTampered(t *testing.T) {
	t.Parallel()
	secret := []byte("secret")
	body := []byte(`{"external_id":"abc"}`)
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	// Flip the last byte of the body — signature should no longer match.
	tampered := append([]byte{}, body...)
	tampered[len(tampered)-2] ^= 0x01
	if verifyHMAC(secret, tampered, sig) {
		t.Fatal("tampered body validated as intact")
	}
}

func TestVerifyHMAC_RejectsWrongSecret(t *testing.T) {
	t.Parallel()
	body := []byte("payload")
	mac := hmac.New(sha256.New, []byte("secretA"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if verifyHMAC([]byte("secretB"), body, sig) {
		t.Fatal("wrong secret accepted")
	}
}

func TestVerifyHMAC_RejectsMalformedHex(t *testing.T) {
	t.Parallel()
	if verifyHMAC([]byte("s"), []byte("x"), "sha256=zz") {
		t.Fatal("non-hex suffix accepted")
	}
	if verifyHMAC([]byte("s"), []byte("x"), "sha256=") {
		t.Fatal("empty hex accepted")
	}
}

func TestVerifyHMAC_ConstantTimeIsNotShortCircuit(t *testing.T) {
	t.Parallel()
	// Signatures of different lengths must not match — and must not
	// panic. Subtle.ConstantTimeCompare treats unequal lengths as a
	// guaranteed mismatch, so this is really a regression guard.
	body := []byte("x")
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write(body)
	// Produce a valid 32-byte hash then truncate it.
	full := hex.EncodeToString(mac.Sum(nil))
	if verifyHMAC([]byte("s"), body, "sha256="+full[:10]) {
		t.Fatal("truncated signature accepted")
	}
}

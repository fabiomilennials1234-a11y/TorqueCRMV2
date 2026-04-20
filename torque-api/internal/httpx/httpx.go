// Package httpx hosts shared request/response helpers used by handlers.
// Intentionally tiny — each helper is boring, predictable, and test-free.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Error is the canonical JSON error envelope. Every 4xx/5xx response shares it.
type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// WriteJSON serializes v as JSON with the given status. Silently discards
// encoding errors; JSON encoding of known-safe server-owned types does not fail.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError emits a structured Error body with status.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, Error{Code: code, Message: message})
}

// MaxBodyBytes bounds any JSON body we decode. 1 MiB covers every known
// payload today (longest is `agents.system_prompt` capped at 16k).
// Endpoints that need to receive larger payloads (future file uploads) must
// wrap their own handler in a fresh `http.MaxBytesReader` with a higher cap.
const MaxBodyBytes int64 = 1 << 20

// DecodeJSON strictly parses a request body into v. Rejects unknown fields
// so a misspelled key surfaces at the boundary rather than silently drops.
// The body is capped at MaxBodyBytes to prevent memory-amplification DoS;
// oversized bodies yield an explicit error that handlers map to 413.
func DecodeJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// IsBodyTooLarge reports whether err came from MaxBytesReader.
// Handlers use it to map to 413 Payload Too Large with a stable code.
func IsBodyTooLarge(err error) bool {
	var max *http.MaxBytesError
	return err != nil && errors.As(err, &max)
}

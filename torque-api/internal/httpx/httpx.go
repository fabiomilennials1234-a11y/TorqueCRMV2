// Package httpx hosts shared request/response helpers used by handlers.
// Intentionally tiny — each helper is boring, predictable, and test-free.
package httpx

import (
	"encoding/json"
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

// DecodeJSON strictly parses a request body into v. Rejects unknown fields
// so a misspelled key surfaces at the boundary rather than silently drops.
func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

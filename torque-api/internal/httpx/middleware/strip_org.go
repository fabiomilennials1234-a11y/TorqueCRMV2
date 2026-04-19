package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// StripOrganizationID enforces the invariant that the client never sends
// `organization_id` in a mutation body (Multi-tenancy.md §"Invariante absoluta").
//
// Only POST/PATCH/PUT with a JSON body are inspected. If the top-level JSON
// object contains `organization_id`, the request is rejected with 400.
//
// Why strict rejection instead of silent strip:
//   - Silent strip would mask bugs in the client and risk a tenant leak if the
//     middleware ever runs after a handler has decoded the body.
//   - A 400 surfaces the problem loudly in development and blocks the request
//     in production. Frontend already strips preemptively at src/api/client.
//
// This middleware must run BEFORE any body-consuming handler.
func StripOrganizationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPatch && r.Method != http.MethodPut {
			next.ServeHTTP(w, r)
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") || r.ContentLength == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Peek at the body without consuming it.
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_BODY", "could not read request body")
			return
		}
		_ = r.Body.Close()

		// Attempt decode; if malformed JSON, let the handler produce its own error.
		var top map[string]json.RawMessage
		if json.Unmarshal(buf, &top) == nil {
			if _, has := top["organization_id"]; has {
				writeError(w, http.StatusBadRequest, "TENANT_FIELD_FORBIDDEN",
					"organization_id must not be sent in request body; it is derived from the session")
				return
			}
		}

		// Restore body for downstream handlers.
		r.Body = io.NopCloser(bytes.NewReader(buf))
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"code":"` + code + `","message":"` + msg + `"}`))
}

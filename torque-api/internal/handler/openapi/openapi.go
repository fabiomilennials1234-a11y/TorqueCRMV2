// Package openapi serves the OpenAPI specification.
//
// Two endpoints:
//
//   GET /openapi.yaml   — source of truth (hand-curated in api/openapi.yaml)
//   GET /openapi.json   — same content, parsed and re-serialized as JSON for
//                         tooling that cannot consume YAML (openapi-typescript
//                         on the frontend, Stoplight, etc.)
//
// The spec is read from disk ONCE at boot and cached in memory. Reloads require
// a deploy; the surface is small enough that hot-reload would only add risk.
package openapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// Handler serves the spec in both YAML and JSON.
type Handler struct {
	yamlBody []byte
	jsonBody []byte
}

// New reads the spec file and returns a handler. Returns an error if the file
// is missing or unparseable — the caller should fail the boot.
//
// `path` is relative to the working directory; typical value is
// "api/openapi.yaml" assuming the process runs from the module root.
func New(path string) (*Handler, error) {
	if path == "" {
		path = filepath.Join("api", "openapi.yaml")
	}
	yamlBody, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(yamlBody, &doc); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	// Re-serialize to JSON. `gopkg.in/yaml.v3` produces map[string]any which
	// json.Marshal handles natively.
	jsonBody, err := yaml.Marshal(doc) // keep canonical YAML
	if err != nil {
		return nil, fmt.Errorf("re-marshal yaml: %w", err)
	}
	_ = jsonBody // keep formatter happy if we flip to yaml v2 later
	jsonBytes, err := mapToJSON(doc)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}
	return &Handler{yamlBody: yamlBody, jsonBody: jsonBytes}, nil
}

// Routes mounts the two endpoints on the root router (not under /api/v1).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/openapi.yaml", h.yaml)
	r.Get("/openapi.json", h.json)
}

func (h *Handler) yaml(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(h.yamlBody)
}

func (h *Handler) json(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(h.jsonBody)
}

// mapToJSON is a tiny helper that encodes a decoded-YAML map[string]any into
// JSON. We do it by hand (instead of json.Marshal) because yaml.v3 produces
// `map[interface{}]interface{}` for nested maps with non-string keys, which
// json.Marshal refuses. We coerce keys to strings on the fly.
func mapToJSON(doc any) ([]byte, error) {
	converted := coerceKeys(doc)
	return jsonMarshal(converted)
}

func coerceKeys(v any) any {
	switch m := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(m))
		for k, vv := range m {
			out[k] = coerceKeys(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, vv := range m {
			out[fmt.Sprintf("%v", k)] = coerceKeys(vv)
		}
		return out
	case []any:
		out := make([]any, len(m))
		for i, vv := range m {
			out[i] = coerceKeys(vv)
		}
		return out
	default:
		return v
	}
}

// jsonMarshal is a tiny indirection so the import list stays explicit. Kept
// separate so future callers can swap for a streaming encoder if needed.
func jsonMarshal(v any) ([]byte, error) {
	b, err := jsonMarshalCanonical(v)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("empty json encoding")
	}
	return b, nil
}

// jsonMarshalCanonical wraps encoding/json; extracted so the package surface
// stays small and callers do not need a second import.
func jsonMarshalCanonical(v any) ([]byte, error) {
	return jsonEncode(v)
}

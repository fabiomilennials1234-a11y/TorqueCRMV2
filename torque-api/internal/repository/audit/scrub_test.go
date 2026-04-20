// Unit test for audit payload scrubbing (no DB needed).
package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScrubPayload_RedactsSensitiveKeys(t *testing.T) {
	t.Parallel()
	input := map[string]any{
		"action":         "login",
		"email":          "user@example.com",
		"password":       "s3cret",
		"api_key":        "sk-abc",
		"nested": map[string]any{
			"Authorization": "Bearer xyz",
			"visible":       "ok",
		},
		"arr": []any{
			map[string]any{"token": "t1", "keep": 1},
			"leave as is",
		},
	}

	out := scrubPayload(input).(map[string]any)
	if out["email"] != "[Scrubbed]" {
		t.Errorf("email must be scrubbed")
	}
	if out["password"] != "[Scrubbed]" {
		t.Errorf("password must be scrubbed")
	}
	if out["api_key"] != "[Scrubbed]" {
		t.Errorf("api_key must be scrubbed")
	}
	if out["action"] != "login" {
		t.Errorf("non-sensitive keys must pass through; got %v", out["action"])
	}
	nested := out["nested"].(map[string]any)
	if nested["Authorization"] != "[Scrubbed]" {
		t.Errorf("nested Authorization must be scrubbed (case-insensitive)")
	}
	if nested["visible"] != "ok" {
		t.Errorf("nested non-sensitive keys must pass through")
	}
	arr := out["arr"].([]any)
	first := arr[0].(map[string]any)
	if first["token"] != "[Scrubbed]" {
		t.Errorf("token inside array element must be scrubbed")
	}
	if first["keep"] != 1 {
		t.Errorf("siblings preserved; got %v", first["keep"])
	}

	// Serialization smoke test — the output must be valid JSON.
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal scrubbed payload: %v", err)
	}
	if !strings.Contains(string(raw), "[Scrubbed]") {
		t.Fatalf("expected redacted markers in JSON output")
	}
}

func TestScrubPayload_PreservesNonMapTypes(t *testing.T) {
	t.Parallel()
	if got := scrubPayload("string"); got != "string" {
		t.Errorf("string preserved; got %v", got)
	}
	if got := scrubPayload(42); got != 42 {
		t.Errorf("number preserved; got %v", got)
	}
	if got := scrubPayload(nil); got != nil {
		t.Errorf("nil preserved; got %v", got)
	}
}

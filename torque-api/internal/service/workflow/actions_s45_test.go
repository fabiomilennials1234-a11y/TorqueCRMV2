package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// ------ dispatcher surface -----------------------------------------

func TestDispatcher_RegistersSevenKinds(t *testing.T) {
	t.Parallel()
	d := NewDispatcherBare(zerolog.Nop())
	want := map[string]bool{
		"send_message": true,
		"update_lead":  true,
		"wait":         true,
		"branch":       true,
		"create_task":  true,
		"call_agent":   true,
		"http":         true,
	}
	got := d.Handlers()
	if len(got) != len(want) {
		t.Fatalf("want %d handlers, got %d (%v)", len(want), len(got), got)
	}
	for _, k := range got {
		if !want[k] {
			t.Errorf("unexpected handler %q", k)
		}
	}
}

// ------ create_task --------------------------------------------------

// TestCreateTaskAction_NoRepoIsNoop exercises the graceful path when
// the tasks repository isn't wired. In production we always wire it;
// the noop exists so unit tests don't need a DB.
func TestCreateTaskAction_NoRepoIsNoop(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]any{"title": "Ligar", "due_in_hours": 24})
	d := NewDispatcherBare(zerolog.Nop())
	out, err := d.Dispatch(context.Background(), "create_task", StepContext{Config: cfg})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["created"] != false {
		t.Errorf("expected created=false when repo nil; got %v", parsed)
	}
	if parsed["wired"] != false {
		t.Errorf("expected wired=false when repo nil; got %v", parsed)
	}
}

// TestCreateTaskAction_RequiresTitle locks the non-retryable guard.
func TestCreateTaskAction_RequiresTitle(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]any{"due_in_hours": 24})
	d := NewDispatcherBare(zerolog.Nop())
	_, err := d.Dispatch(context.Background(), "create_task", StepContext{Config: cfg})
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("expected non-retryable when title missing, got %v", err)
	}
}

// ------ call_agent --------------------------------------------------

func TestCallAgentAction_NoRepoIsNoop(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"agent_id": "00000000-0000-0000-0000-000000000001", "greeting": "Olá"})
	d := NewDispatcherBare(zerolog.Nop())
	out, _ := d.Dispatch(context.Background(), "call_agent", StepContext{Config: cfg})
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["attached"] != false {
		t.Errorf("expected attached=false with nil deps, got %v", parsed)
	}
}

func TestCallAgentAction_RequiresAgentID(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"greeting": "oi"})
	d := NewDispatcherBare(zerolog.Nop())
	_, err := d.Dispatch(context.Background(), "call_agent", StepContext{Config: cfg})
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("expected non-retryable when agent_id missing, got %v", err)
	}
}

// ------ http / SSRF -------------------------------------------------

// TestHTTPRequestAction_SSRF covers the scheme guard + private-range
// blocks. The public path is exercised separately with httptest.
func TestHTTPRequestAction_SSRF(t *testing.T) {
	t.Parallel()
	d := NewDispatcherBare(zerolog.Nop())
	cases := []struct {
		name string
		url  string
	}{
		{"rejects_http", "http://example.com/hook"},
		{"rejects_ftp", "ftp://example.com/f"},
		{"rejects_empty", ""},
		{"rejects_loopback_literal", "https://localhost/x"},
		{"rejects_private_literal", "https://10.0.0.1/x"},
		{"rejects_aws_metadata", "https://169.254.169.254/latest/meta-data/"},
		{"rejects_gcp_metadata", "https://metadata.google.internal/computeMetadata/"},
		{"rejects_rfc1918", "https://192.168.1.1/ping"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cfg, _ := json.Marshal(map[string]string{"url": c.url, "method": "POST"})
			_, err := d.Dispatch(context.Background(), "http", StepContext{Config: cfg})
			if !errors.Is(err, ErrNonRetryable) {
				t.Errorf("url %q: expected NonRetryable SSRF rejection, got %v", c.url, err)
			}
		})
	}
}

// TestHTTPRequestAction_RejectsBadMethod locks the method allowlist.
func TestHTTPRequestAction_RejectsBadMethod(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"url": "https://example.com/", "method": "TRACE"})
	d := NewDispatcherBare(zerolog.Nop())
	_, err := d.Dispatch(context.Background(), "http", StepContext{Config: cfg})
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("expected non-retryable on bad method, got %v", err)
	}
}

// TestHTTPRequestAction_HappyPath uses httptest over https to exercise
// the full round-trip without needing external DNS. Because our
// validator blocks loopback IPs, we override the client for this case.
func TestHTTPRequestAction_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	// Build the handler directly so we can bypass the public-only
	// validator for this loopback URL. SSRF is still covered by the
	// table above; this test proves the wiring & response shape.
	h := &HTTPRequestAction{logger: zerolog.Nop(), client: srv.Client()}
	cfg, _ := json.Marshal(map[string]any{
		"url":     srv.URL + "/hook",
		"method":  "POST",
		"body":    `{"x":1}`,
	})
	// Because srv.URL resolves to 127.0.0.1 the validator blocks it;
	// the point of this test is the 2xx path, so we call Execute
	// through a wrapper that short-circuits the validator.
	out, err := h.executeUnchecked(context.Background(), cfg)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["status_code"].(float64) != 200 {
		t.Errorf("want 200, got %v", parsed["status_code"])
	}
	if !strings.Contains(parsed["response_body"].(string), "ok") {
		t.Errorf("expected response body passthrough, got %v", parsed)
	}
}

// TestHTTPRequestAction_RetriesOn5xx proves status=500 maps to transient.
func TestHTTPRequestAction_RetriesOn5xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	h := &HTTPRequestAction{logger: zerolog.Nop(), client: srv.Client()}
	cfg, _ := json.Marshal(map[string]any{"url": srv.URL + "/x", "method": "GET"})
	_, err := h.executeUnchecked(context.Background(), cfg)
	if !errors.Is(err, ErrTransient) {
		t.Errorf("5xx must map to transient, got %v", err)
	}
}

// TestHTTPRequestAction_FailsOn4xx proves status=404 maps to non-retryable.
func TestHTTPRequestAction_FailsOn4xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	h := &HTTPRequestAction{logger: zerolog.Nop(), client: srv.Client()}
	cfg, _ := json.Marshal(map[string]any{"url": srv.URL + "/x", "method": "GET"})
	_, err := h.executeUnchecked(context.Background(), cfg)
	if !errors.Is(err, ErrNonRetryable) {
		t.Errorf("4xx must map to non-retryable, got %v", err)
	}
}

// TestHTTPRequestAction_TimeoutIsTransient confirms a context-timeout
// is treated as a retryable signal (the upstream might be slow, not dead).
func TestHTTPRequestAction_TimeoutIsTransient(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep past the client's patience.
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli := srv.Client()
	cli.Timeout = 20 * time.Millisecond
	h := &HTTPRequestAction{logger: zerolog.Nop(), client: cli}
	cfg, _ := json.Marshal(map[string]any{
		"url":        srv.URL + "/slow",
		"method":     "GET",
		"timeout_ms": 20,
	})
	_, err := h.executeUnchecked(context.Background(), cfg)
	if !errors.Is(err, ErrTransient) {
		t.Errorf("timeout must map to transient, got %v", err)
	}
}

// ------ template rendering ------------------------------------------

func TestRenderTemplate_LeadPlaceholders(t *testing.T) {
	t.Parallel()
	leadJSON, _ := json.Marshal(map[string]string{"name": "Marina", "phone": "+5511999999999"})
	sc := StepContext{
		PreviousOutputs: map[string]json.RawMessage{"__lead": leadJSON},
	}
	body := renderTemplate("Olá {{ lead.name }}! seu numero: {{lead.phone}}", sc)
	if !strings.Contains(body, "Marina") || !strings.Contains(body, "+5511999999999") {
		t.Errorf("placeholders lost: %q", body)
	}
}

func TestRenderTemplate_UnknownPathRendersEmpty(t *testing.T) {
	t.Parallel()
	sc := StepContext{PreviousOutputs: map[string]json.RawMessage{}}
	body := renderTemplate("Olá {{lead.name}}!", sc)
	if body != "Olá !" {
		t.Errorf("expected empty placeholder substitution, got %q", body)
	}
}

// ------ validateHTTPURL backward compatibility ---------------------

func TestValidateHTTPURL_LiteralPrefix(t *testing.T) {
	t.Parallel()
	if !validateHTTPURL("https://x.example.com") {
		t.Errorf("expected https accepted by literal guard")
	}
	if validateHTTPURL("http://x.example.com") {
		t.Errorf("http must reject")
	}
	if validateHTTPURL("HTTPS://X") {
		t.Errorf("uppercase scheme rejected (literal prefix guard)")
	}
}

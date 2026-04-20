package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestS45Dispatcher_RegistersSevenKinds(t *testing.T) {
	t.Parallel()
	d := NewDispatcherS45(zerolog.Nop())
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
		t.Fatalf("want %d handlers, got %d", len(want), len(got))
	}
	for _, k := range got {
		if !want[k] {
			t.Errorf("unexpected handler %q", k)
		}
	}
}

func TestCreateTaskAction_OutputShape(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]any{"title": "Ligar", "due_in_hours": 24})
	d := NewDispatcherS45(zerolog.Nop())
	out, err := d.Dispatch(context.Background(), "create_task", StepContext{Config: cfg})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["title"] != "Ligar" {
		t.Errorf("title lost: %v", parsed)
	}
	if parsed["created"] != false {
		t.Errorf("stub should report created=false in S45")
	}
}

func TestCallAgentAction_OutputShape(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"agent_id": "a1", "greeting": "Olá"})
	d := NewDispatcherS45(zerolog.Nop())
	out, _ := d.Dispatch(context.Background(), "call_agent", StepContext{Config: cfg})
	var parsed map[string]any
	_ = json.Unmarshal(out.Output, &parsed)
	if parsed["agent_id"] != "a1" {
		t.Errorf("agent_id lost: %v", parsed)
	}
	if parsed["attached"] != false {
		t.Errorf("stub should report attached=false in S45")
	}
}

func TestHTTPRequestAction_AcceptsHTTPSOnly(t *testing.T) {
	t.Parallel()
	d := NewDispatcherS45(zerolog.Nop())
	cases := []struct {
		url      string
		accepted bool
	}{
		{"https://api.example.com/hook", true},
		{"http://api.example.com/hook", false},
		{"ftp://file.example.com", false},
		{"", false},
	}
	for _, c := range cases {
		cfg, _ := json.Marshal(map[string]string{"url": c.url, "method": "POST"})
		out, _ := d.Dispatch(context.Background(), "http", StepContext{Config: cfg})
		var parsed map[string]any
		_ = json.Unmarshal(out.Output, &parsed)
		if parsed["accepted"] != c.accepted {
			t.Errorf("url %q: want accepted=%v, got %v", c.url, c.accepted, parsed["accepted"])
		}
	}
}

func TestValidateHTTPURL_EdgeCases(t *testing.T) {
	t.Parallel()
	if validateHTTPURL("https://x") == false {
		t.Errorf("expected https://x valid")
	}
	if validateHTTPURL("http://x") {
		t.Errorf("http must reject")
	}
	if validateHTTPURL("HTTPS://X") { // minimum S29 check is literal prefix — uppercase rejected
		t.Errorf("uppercase scheme rejected (literal prefix guard)")
	}
}

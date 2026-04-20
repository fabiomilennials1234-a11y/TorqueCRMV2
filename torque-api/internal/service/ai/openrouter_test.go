package ai_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/milennials/torque-api/internal/service/ai"
)

func TestNewOpenRouter_RejectsBadConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  ai.Config
	}{
		{"empty base", ai.Config{APIKey: "k"}},
		{"http base", ai.Config{BaseURL: "http://x", APIKey: "k"}},
		{"no api key", ai.Config{BaseURL: "https://x"}},
	}
	for _, c := range cases {
		if _, err := ai.NewOpenRouter(c.cfg); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestOpenRouter_Name(t *testing.T) {
	t.Parallel()
	p, err := ai.NewOpenRouter(ai.Config{BaseURL: "https://x", APIKey: "k"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if p.Name() != "openrouter" {
		t.Errorf("name: %s", p.Name())
	}
}

func TestOpenRouter_Chat_SSEHappyPath(t *testing.T) {
	t.Parallel()
	// Simulated OpenRouter streaming 3 delta frames + [DONE].
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing bearer")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw, _ := w.(http.Flusher)
		frames := []string{
			`{"choices":[{"delta":{"content":"Olá"}}]}`,
			`{"choices":[{"delta":{"content":", mundo"}}]}`,
			`{"choices":[{"delta":{"content":"!"}}],"usage":{"prompt_tokens":10,"completion_tokens":3}}`,
		}
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			if fw != nil {
				fw.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if fw != nil {
			fw.Flush()
		}
	}))
	defer srv.Close()

	p, err := ai.NewOpenRouter(ai.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "secret",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	out := make(chan ai.Chunk, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = p.Chat(ctx, ai.ChatRequest{
		Model:       "test/model",
		Messages:    []ai.Message{{Role: ai.RoleUser, Content: "oi"}},
		Temperature: 0.3,
		MaxTokens:   128,
	}, out)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	var deltas []string
	var terminal ai.Chunk
	for chunk := range out {
		if chunk.Done {
			terminal = chunk
			continue
		}
		if chunk.Delta != "" {
			deltas = append(deltas, chunk.Delta)
		}
	}
	joined := strings.Join(deltas, "")
	if joined != "Olá, mundo!" {
		t.Fatalf("joined deltas: %q", joined)
	}
	if !terminal.Done {
		t.Fatalf("terminal chunk missing Done=true")
	}
	if terminal.InputTokens != 10 || terminal.OutputTokens != 3 {
		t.Fatalf("usage not relayed: %+v", terminal)
	}
}

func TestOpenRouter_Chat_AuthFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	p, err := ai.NewOpenRouter(ai.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "wrong",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	out := make(chan ai.Chunk, 2)
	err = p.Chat(context.Background(), ai.ChatRequest{
		Model: "x", Messages: []ai.Message{{Role: ai.RoleUser, Content: "x"}},
	}, out)
	if !errors.Is(err, ai.ErrAuthFailed) {
		t.Fatalf("want ErrAuthFailed, got %v", err)
	}
}

func TestOpenRouter_Chat_RateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p, _ := ai.NewOpenRouter(ai.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "k", HTTPClient: srv.Client(),
	})
	out := make(chan ai.Chunk, 2)
	err := p.Chat(context.Background(), ai.ChatRequest{
		Model: "x", Messages: []ai.Message{{Role: ai.RoleUser, Content: "x"}},
	}, out)
	if !errors.Is(err, ai.ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestOpenRouter_Chat_ProviderUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p, _ := ai.NewOpenRouter(ai.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "k", HTTPClient: srv.Client(),
	})
	out := make(chan ai.Chunk, 2)
	err := p.Chat(context.Background(), ai.ChatRequest{
		Model: "x", Messages: []ai.Message{{Role: ai.RoleUser, Content: "x"}},
	}, out)
	if !errors.Is(err, ai.ErrProviderUnavailable) {
		t.Fatalf("want ErrProviderUnavailable, got %v", err)
	}
}

func TestOpenRouter_Chat_ClosesOutWithoutDone(t *testing.T) {
	t.Parallel()
	// Server that streams ONE frame then hangs up without [DONE]. The
	// adapter must still emit a terminal Done chunk so consumers know
	// the stream is over.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fw, _ := w.(http.Flusher)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"hi"}}]}`+"\n\n")
		if fw != nil {
			fw.Flush()
		}
	}))
	defer srv.Close()

	p, _ := ai.NewOpenRouter(ai.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "k", HTTPClient: srv.Client(),
	})
	out := make(chan ai.Chunk, 4)
	err := p.Chat(context.Background(), ai.ChatRequest{
		Model: "x", Messages: []ai.Message{{Role: ai.RoleUser, Content: "x"}},
	}, out)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	var sawDelta, sawDone bool
	for chunk := range out {
		if chunk.Delta != "" {
			sawDelta = true
		}
		if chunk.Done {
			sawDone = true
		}
	}
	if !sawDelta {
		t.Errorf("expected delta chunk")
	}
	if !sawDone {
		t.Errorf("expected terminal Done chunk even without [DONE] sentinel")
	}
}

// Package ai wraps the LLM provider that the Copilot (F06) talks to.
//
// OpenRouter is the concrete provider today — it's a thin gateway over
// dozens of models (Anthropic, OpenAI, Google, open-weight), which means
// one integration surface covers the entire model menu. If we swap to
// a direct Anthropic / OpenAI client later, it implements the same
// `Provider` interface and nothing above this package changes.
//
// Streaming is SSE (`text/event-stream`). The adapter parses the
// server-sent `data: {...}` frames and emits Go channel messages so
// callers can relay to HTTP SSE or a WebSocket without bespoke parsing.
//
// Security posture:
//   - BaseURL must be https:// (rejected at construction otherwise).
//   - API key never logged; redacted via Sentry scrub list.
//   - Per-request context + hard ctx timeout (default 30s) prevents
//     goroutine leak on a hung upstream.
//   - 401/403 short-circuit as ErrAuthFailed; 429 as ErrRateLimited;
//     5xx as ErrProviderUnavailable; the rest of 4xx surfaces the
//     provider message truncated at 200 chars.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------- error taxonomy ------------------------------------------

var (
	ErrAuthFailed          = errors.New("ai: provider refused the API key")
	ErrRateLimited         = errors.New("ai: provider rate-limited us")
	ErrProviderUnavailable = errors.New("ai: provider returned 5xx")
	ErrKillSwitchTripped   = errors.New("ai: kill switch active")
	ErrBadRequest          = errors.New("ai: provider rejected the request")
)

// ---------- message shape (provider-agnostic) -----------------------

// Role is the OpenAI-style conversation role. Tool calls are out of
// scope in S37; they arrive with F06.5+ when we wire function calling.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn in the conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is what callers hand us; we forward it to OpenRouter
// after a light normalization.
type ChatRequest struct {
	Model       string    // e.g. "anthropic/claude-sonnet-4"
	Messages    []Message // ordered, oldest first; system MUST be first if present
	Temperature float64   // 0..2, clamped by the backend constraint
	MaxTokens   int       // 16..8192, clamped
}

// Chunk is one piece of the streamed reply.
type Chunk struct {
	// Delta is the text this chunk added. May be empty on lifecycle
	// frames (stream start, stream end) — callers should ignore empty
	// deltas rather than flushing them as literal "".
	Delta string
	// Done is true on the terminal frame; no more Deltas will arrive.
	Done bool
	// InputTokens / OutputTokens are populated on the terminal frame
	// when the provider reports them. Zero means "unknown".
	InputTokens  int
	OutputTokens int
}

// ---------- provider interface --------------------------------------

// Provider is the surface the handler depends on. The concrete
// OpenRouter struct implements it; tests inject a fake.
type Provider interface {
	// Name is the audit string (e.g. "openrouter", "anthropic-direct").
	Name() string
	// Chat streams chunks over `out` until the model finishes or ctx
	// cancels. `out` is closed by the provider when streaming ends.
	Chat(ctx context.Context, req ChatRequest, out chan<- Chunk) error
	// Health is a shallow GET against a safe endpoint for readiness.
	Health(ctx context.Context) error
}

// ---------- OpenRouter implementation -------------------------------

// Config carries everything the adapter needs to boot. All fields are
// required except HTTPClient (defaults) and Timeout (default 30s).
type Config struct {
	BaseURL    string        // e.g. "https://openrouter.ai/api/v1"
	APIKey     string        // openrouter key
	Timeout    time.Duration // default 30s
	HTTPClient *http.Client  // default &http.Client{Timeout: Timeout}
	// Referer + Title are OpenRouter-specific headers that drive their
	// per-app analytics. Empty strings are accepted.
	Referer string
	Title   string
}

// OpenRouter is the concrete Provider.
type OpenRouter struct {
	cfg    Config
	client *http.Client
}

// NewOpenRouter validates config and builds the adapter.
func NewOpenRouter(cfg Config) (*OpenRouter, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		return nil, errors.New("openrouter: BaseURL required")
	}
	if !strings.HasPrefix(base, "https://") {
		return nil, errors.New("openrouter: BaseURL must use https://")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("openrouter: APIKey required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	cfg.BaseURL = base
	return &OpenRouter{cfg: cfg, client: client}, nil
}

// Name implements Provider.
func (*OpenRouter) Name() string { return "openrouter" }

// Health pings /models — cheap and always available.
func (p *OpenRouter) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	res, err := p.client.Do(req)
	if err != nil {
		return ErrProviderUnavailable
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden:
		return ErrAuthFailed
	case res.StatusCode >= 500:
		return ErrProviderUnavailable
	}
	return nil
}

// Chat streams the provider's reply. The caller MUST consume `out` or
// ctx cancellation is the only way to unblock the send path.
func (p *OpenRouter) Chat(ctx context.Context, req ChatRequest, out chan<- Chunk) error {
	defer close(out)

	body, err := json.Marshal(map[string]any{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build chat request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.cfg.Referer != "" {
		httpReq.Header.Set("HTTP-Referer", p.cfg.Referer)
	}
	if p.cfg.Title != "" {
		httpReq.Header.Set("X-Title", p.cfg.Title)
	}

	res, err := p.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return ErrProviderUnavailable
	}
	defer res.Body.Close()

	// Map non-2xx to a typed error. We read up to 200 bytes of body to
	// preserve the provider's error hint without leaking secrets.
	switch {
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden:
		return ErrAuthFailed
	case res.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case res.StatusCode >= 500:
		return ErrProviderUnavailable
	case res.StatusCode >= 400:
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 200))
		return fmt.Errorf("%w: %s", ErrBadRequest, strings.TrimSpace(string(raw)))
	}

	return parseSSE(ctx, res.Body, out)
}

// parseSSE reads OpenRouter's `text/event-stream` wire format and emits
// Chunks on `out`. The grammar we accept:
//
//	data: {...JSON...}\n\n
//	data: [DONE]\n\n
//
// The last-line `[DONE]` sentinel is emitted as `{Done:true}`; any
// chunks received before it are relayed as deltas. Usage totals (input
// / output tokens) ride on the last JSON frame when present.
func parseSSE(ctx context.Context, r io.Reader, out chan<- Chunk) error {
	sc := bufio.NewScanner(r)
	// OpenRouter sometimes emits long messages; lift the default 64K limit.
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var lastInput, lastOutput int

	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := sc.Text()
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			select {
			case out <- Chunk{Done: true, InputTokens: lastInput, OutputTokens: lastOutput}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}

		var frame struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			// Malformed frame — swallow rather than kill the stream.
			// OpenRouter occasionally ships heartbeat comments that
			// look like data:{}; we don't want those to fail the call.
			continue
		}
		if frame.Usage != nil {
			lastInput = frame.Usage.PromptTokens
			lastOutput = frame.Usage.CompletionTokens
		}
		for _, c := range frame.Choices {
			if c.Delta.Content == "" {
				continue
			}
			select {
			case out <- Chunk{Delta: c.Delta.Content}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read sse stream: %w", err)
	}
	// Some providers close without [DONE]. Emit the terminal chunk.
	select {
	case out <- Chunk{Done: true, InputTokens: lastInput, OutputTokens: lastOutput}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

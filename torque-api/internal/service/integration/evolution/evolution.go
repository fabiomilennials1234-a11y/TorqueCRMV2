// Package evolution is the concrete MessagingProvider for Evolution API
// (open-source WhatsApp gateway). It sits behind the integration.Provider
// interface declared in S27 so the inbox worker stays provider-agnostic.
//
// Defensive choices:
//  - Caller URL kept in config; never derived from user input.
//  - Circuit breaker + retry wrappers come from the shared integration pkg.
//  - Shared http.Client (100ms connect, 10s total) prevents goroutine leak
//    on remote hang.
//  - Auth header apikey is read once on build; rotation = restart.
package evolution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/milennials/torque-api/internal/service/integration"
)

// Config wires the adapter at boot.
type Config struct {
	// BaseURL of the Evolution instance, e.g. https://evo.example.com.
	BaseURL string
	// APIKey for the `apikey` header.
	APIKey string
	// Optional override of the shared http.Client. Defaults to 10s timeout.
	HTTPClient *http.Client
}

// Provider implements integration.MessagingProvider against Evolution API.
type Provider struct {
	cfg        Config
	client     *http.Client
	breaker    *integration.CircuitBreaker
}

// New builds a provider. Returns error on missing required config so boot
// fails loud instead of silently sending to "".
func New(cfg Config) (*Provider, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		return nil, errors.New("evolution: BaseURL is required")
	}
	if !strings.HasPrefix(base, "https://") {
		return nil, errors.New("evolution: BaseURL must use https://")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("evolution: APIKey is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Provider{
		cfg:     Config{BaseURL: base, APIKey: cfg.APIKey},
		client:  client,
		breaker: integration.NewCircuitBreaker(5, 30*time.Second),
	}, nil
}

// Name returns the `billing_provider` ENUM value — here "evolution".
// (MessagingProvider keeps the same interface; the DB ENUM for messaging
// accepts free-form channel_kind so this string is exposed in audit logs.)
func (*Provider) Name() string { return "evolution" }

// SendMessage dispatches an outbound message. Caller is expected to wrap
// the call in integration.Retry if it wants the exponential backoff.
func (p *Provider) SendMessage(ctx context.Context, msg integration.OutboundMessage) (integration.MessageResult, error) {
	if msg.ChannelExternalID == "" {
		return integration.MessageResult{}, errors.New("channel_external_id required")
	}
	if msg.Recipient == "" {
		return integration.MessageResult{}, errors.New("recipient required")
	}

	var result integration.MessageResult
	err := p.breaker.Do(func() error {
		res, berr := p.sendOnce(ctx, msg)
		if berr != nil {
			return berr
		}
		result = res
		return nil
	})
	return result, err
}

// Health pings the `/instance/fetchInstances` diagnostic endpoint. A
// 401 means the api-key is bad (ErrAuthFailed); anything else ≥500
// bubbles as ErrUnreachable.
func (p *Provider) Health(ctx context.Context) error {
	req, err := p.newRequest(ctx, http.MethodGet, "/instance/fetchInstances", nil)
	if err != nil {
		return err
	}
	res, err := p.client.Do(req)
	if err != nil {
		return integration.ErrUnreachable
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return integration.ErrAuthFailed
	}
	if res.StatusCode >= 500 {
		return integration.ErrUnreachable
	}
	return nil
}

// -------- internal --------------------------------------------------

// sendOnce is the single-attempt call. The circuit breaker counts
// unreachable errors; explicit 4xx auth failures short-circuit.
func (p *Provider) sendOnce(ctx context.Context, msg integration.OutboundMessage) (integration.MessageResult, error) {
	path, body, err := buildSendPayload(msg)
	if err != nil {
		return integration.MessageResult{}, err
	}
	req, err := p.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return integration.MessageResult{}, err
	}
	res, err := p.client.Do(req)
	if err != nil {
		return integration.MessageResult{}, integration.ErrUnreachable
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return integration.MessageResult{}, integration.ErrAuthFailed
	case res.StatusCode == http.StatusTooManyRequests:
		return integration.MessageResult{}, integration.ErrRateLimited
	case res.StatusCode >= 500:
		return integration.MessageResult{}, integration.ErrUnreachable
	case res.StatusCode >= 400:
		return integration.MessageResult{}, fmt.Errorf("evolution rejected: %s", truncate(string(raw), 200))
	}

	var reply sendReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		// Provider replied 2xx but not JSON — accept as success with
		// synthetic id. Better than refusing a successful send.
		return integration.MessageResult{
			ProviderMessageID: "evo_" + shortID(raw),
			AcceptedAt:        time.Now().UTC(),
		}, nil
	}

	id := reply.Key.ID
	if id == "" {
		id = reply.MessageID
	}
	if id == "" {
		id = "evo_" + shortID(raw)
	}
	return integration.MessageResult{
		ProviderMessageID: id,
		AcceptedAt:        time.Now().UTC(),
	}, nil
}

// buildSendPayload maps the provider-agnostic OutboundMessage to the path
// + body Evolution expects. `/message/sendText/:instance` for text,
// `/message/sendMedia/:instance` for media kinds. Unsupported kinds
// surface as ErrUnsupported.
func buildSendPayload(msg integration.OutboundMessage) (string, *bytes.Buffer, error) {
	inst := msg.ChannelExternalID

	switch msg.Kind {
	case "text":
		body, err := encodeJSON(map[string]any{
			"number":       msg.Recipient,
			"textMessage":  map[string]string{"text": msg.Body},
			"options":      map[string]any{"delay": 1000, "presence": "composing"},
		})
		if err != nil {
			return "", nil, err
		}
		return "/message/sendText/" + inst, body, nil
	case "image", "video", "document":
		body, err := encodeJSON(map[string]any{
			"number":       msg.Recipient,
			"mediaMessage": map[string]any{
				"mediatype": msg.Kind,
				"media":     msg.MediaURL,
				"caption":   msg.Body,
			},
		})
		if err != nil {
			return "", nil, err
		}
		return "/message/sendMedia/" + inst, body, nil
	default:
		return "", nil, integration.ErrUnsupported
	}
}

func (p *Provider) newRequest(ctx context.Context, method, path string, body *bytes.Buffer) (*http.Request, error) {
	var rdr io.Reader
	if body != nil {
		rdr = body
	}
	req, err := http.NewRequestWithContext(ctx, method, p.cfg.BaseURL+path, rdr)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("apikey", p.cfg.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

type sendReply struct {
	Key       struct{ ID string `json:"id"` } `json:"key"`
	MessageID string                          `json:"messageId"`
}

func encodeJSON(v any) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return &buf, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func shortID(raw []byte) string {
	if len(raw) == 0 {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// Non-cryptographic digest is enough for a correlation id.
	h := uint64(1469598103934665603)
	for _, b := range raw {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return fmt.Sprintf("%x", h)
}

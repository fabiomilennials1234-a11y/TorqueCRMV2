// Package szchat is the SZ.Chat MessagingProvider for Torque (S50 / F.2).
//
// SZ.Chat is a Brazil-first omnichannel gateway (WhatsApp/Instagram/
// Messenger/SMS) with a simple REST surface. Auth is a per-tenant
// API key stored encrypted in integration_credentials under
// provider='szchat' (access_token_encrypted). Outbound message POST is
// idempotent against the caller-supplied Recipient + Body pair — we
// surface the provider's message id back to the caller.
//
// Inbound messages come via SZ.Chat's own webhook, delivered into the
// existing /webhooks/lead path (S50 Lead Webhook) when configured as a
// lead source, or the future /webhooks/message path for conversations.
package szchat

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

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	"github.com/milennials/torque-api/internal/service/integration"
)

const (
	defaultBaseURL = "https://api.szchat.com/v1"
	bodyCap        = 1 << 20
)

// Config wires the adapter at boot. Per-tenant API keys live in the
// credential store.
type Config struct {
	BaseURL string
}

// Provider implements integration.MessagingProvider against SZ.Chat.
type Provider struct {
	cfg     Config
	http    *http.Client
	store   *integrationrepo.Store
	breaker *integration.CircuitBreaker
}

// New builds a Provider with sane defaults.
func New(cfg Config, store *integrationrepo.Store, httpClient *http.Client, breaker *integration.CircuitBreaker) *Provider {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if breaker == nil {
		breaker = integration.NewCircuitBreaker(5, 30*time.Second)
	}
	return &Provider{
		cfg:     Config{BaseURL: base},
		http:    httpClient,
		store:   store,
		breaker: breaker,
	}
}

// Name returns the DB provider tag.
func (*Provider) Name() string { return integrationrepo.ProviderSZChat }

// StoreAPIKey persists a tenant's SZ.Chat API key + optional channel id.
func (p *Provider) StoreAPIKey(ctx context.Context, orgID uuid.UUID, apiKey string, channelID *string) error {
	if n := len(apiKey); n < 16 || n > 200 {
		return fmt.Errorf("szchat: api_key length must be 16-200 chars (got %d)", n)
	}
	return p.store.Upsert(ctx, orgID, integrationrepo.ProviderSZChat, integrationrepo.UpsertInput{
		AccessToken:       apiKey,
		TokenType:         "ApiKey",
		ExternalAccountID: channelID,
	})
}

// SendMessage satisfies the MessagingProvider interface. OrgID comes
// from ctx.
func (p *Provider) SendMessage(ctx context.Context, msg integration.OutboundMessage) (integration.MessageResult, error) {
	orgID, ok := domain.OrgIDFrom(ctx)
	if !ok || orgID == uuid.Nil {
		return integration.MessageResult{}, errors.New("szchat: no org id in context")
	}
	return p.SendMessageForOrg(ctx, orgID, msg)
}

// SendMessageForOrg is the explicit-org variant.
func (p *Provider) SendMessageForOrg(ctx context.Context, orgID uuid.UUID, msg integration.OutboundMessage) (integration.MessageResult, error) {
	if msg.Recipient == "" {
		return integration.MessageResult{}, errors.New("szchat: recipient required")
	}
	if msg.Body == "" && msg.MediaURL == "" {
		return integration.MessageResult{}, errors.New("szchat: body or media_url required")
	}
	cred, err := p.store.Get(ctx, orgID, integrationrepo.ProviderSZChat)
	if err != nil {
		return integration.MessageResult{}, err
	}

	payload := outboundBody{
		Channel:   msg.ChannelExternalID,
		Recipient: msg.Recipient,
		Kind:      normalizeKind(msg.Kind),
		Body:      msg.Body,
		MediaURL:  msg.MediaURL,
	}
	if payload.Channel == "" && cred.ExternalAccountID != nil {
		payload.Channel = *cred.ExternalAccountID
	}

	var out integration.MessageResult
	berr := p.breaker.Do(func() error {
		res, callErr := p.postJSON(ctx, cred.AccessToken, "/messages", payload)
		if callErr != nil {
			return callErr
		}
		out = res
		return nil
	})
	if berr != nil {
		p.recordError(ctx, orgID, berr)
		return integration.MessageResult{}, berr
	}
	_ = p.store.MarkSuccess(ctx, orgID, integrationrepo.ProviderSZChat)
	return out, nil
}

// Health is a no-op probe in dev — the credential store reflects
// per-tenant health via last_error_*.
func (p *Provider) Health(_ context.Context) error { return nil }

// ---------------- internals -----------------------------------------

type outboundBody struct {
	Channel   string `json:"channel,omitempty"`
	Recipient string `json:"to"`
	Kind      string `json:"kind"`
	Body      string `json:"body,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
}

type outboundResponse struct {
	ID        string    `json:"id"`
	AcceptedAt time.Time `json:"accepted_at"`
	Error     *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *Provider) postJSON(ctx context.Context, apiKey, path string, body outboundBody) (integration.MessageResult, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return integration.MessageResult{}, fmt.Errorf("szchat: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return integration.MessageResult{}, fmt.Errorf("szchat: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return integration.MessageResult{}, integration.ErrUnreachable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, bodyCap))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// fall through to body parse
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return integration.MessageResult{}, integration.ErrAuthFailed
	case resp.StatusCode == http.StatusTooManyRequests:
		return integration.MessageResult{}, integration.ErrRateLimited
	case resp.StatusCode >= 500:
		return integration.MessageResult{}, integration.ErrUnreachable
	default:
		return integration.MessageResult{}, fmt.Errorf("szchat: http %d: %s",
			resp.StatusCode, truncate(string(raw), 200))
	}

	var out outboundResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return integration.MessageResult{}, fmt.Errorf("szchat: decode: %w", err)
	}
	if out.Error != nil {
		return integration.MessageResult{}, fmt.Errorf("szchat: %s: %s",
			out.Error.Code, truncate(out.Error.Message, 200))
	}
	if out.ID == "" {
		return integration.MessageResult{}, errors.New("szchat: empty message id")
	}
	accepted := out.AcceptedAt
	if accepted.IsZero() {
		accepted = time.Now().UTC()
	}
	return integration.MessageResult{
		ProviderMessageID: out.ID,
		AcceptedAt:        accepted,
	}, nil
}

func (p *Provider) recordError(ctx context.Context, orgID uuid.UUID, err error) {
	if err == nil || errors.Is(err, integration.ErrCircuitOpen) {
		return
	}
	_ = p.store.MarkError(ctx, orgID, integrationrepo.ProviderSZChat, err.Error())
}

// normalizeKind maps the provider-agnostic kind to SZ.Chat's wire
// vocabulary. Unknown kinds default to "text" so the provider is the
// one that rejects invalid payloads, not Torque.
func normalizeKind(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "", "text":
		return "text"
	case "image":
		return "image"
	case "audio":
		return "audio"
	case "video":
		return "video"
	case "document", "file":
		return "document"
	default:
		return "text"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

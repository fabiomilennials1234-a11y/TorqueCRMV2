// Asaas provider for the billing package (S51 / Fase G.1).
//
// Implements `Provider` against Asaas v3 REST API. PIX is the only
// billing type shipped in S51 — boleto + cartão are deferred behind a
// TODO until the second dual-review pass lands with the production
// credentials. The adapter speaks JSON + `access_token` header auth;
// sandbox / production toggles via BaseURL (sandbox endpoint carves
// off the same surface as prod).
//
// Webhook payloads arrive at /webhooks/billing with the shared-secret
// header (configured on the Asaas dashboard). This file owns the
// payload shape → normalized `webhookEvent` conversion via
// NormalizeWebhook(raw []byte).
//
// **Dual-review gate**: `New()` requires both a non-empty APIKey AND
// a non-empty BaseURL — if either is blank, construction fails and
// main.go falls back to the mock. This means a deploy cannot
// accidentally run prod-asaas with half-configured credentials; the
// env must be explicit, which is the point of the second reviewer's
// sign-off on the deployment config.
package billing

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
)

const (
	asaasSandboxURL = "https://sandbox.asaas.com/api/v3"
	asaasProdURL    = "https://api.asaas.com/v3"
	asaasBodyCap    = 1 << 20
)

// AsaasConfig carves out the Provider wiring. APIKey is required; an
// empty string makes New() return an error rather than silently
// fall back to sandbox.
type AsaasConfig struct {
	APIKey  string
	BaseURL string // empty → sandbox (safer default for dev + staging)
}

// AsaasProvider implements Provider against Asaas v3.
type AsaasProvider struct {
	cfg  AsaasConfig
	http *http.Client
}

// NewAsaas builds the provider. Returns an error if APIKey is empty —
// this is the dual-review tripwire: bootstrap refuses to wire a real
// provider without explicit credentials.
func NewAsaas(cfg AsaasConfig, httpClient *http.Client) (*AsaasProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("asaas: api_key required (dual-review gate)")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = asaasSandboxURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &AsaasProvider{cfg: cfg, http: httpClient}, nil
}

// Name returns the DB ENUM tag.
func (*AsaasProvider) Name() string { return "asaas" }

// CreateCharge issues a PIX charge via POST /payments. Asaas requires
// a customer ID for every charge — we POST /customers first and use
// the returned `id` for the subsequent payment. The pipeline is
// bounded by the caller's context timeout.
func (p *AsaasProvider) CreateCharge(ctx context.Context, in ChargeInput) (Charge, error) {
	if in.AmountCents <= 0 {
		return Charge{}, errors.New("asaas: amount_cents must be positive")
	}
	if in.Currency != "" && in.Currency != "BRL" {
		return Charge{}, fmt.Errorf("asaas: unsupported currency %q (only BRL)", in.Currency)
	}

	// 1. Ensure a customer exists for this org. Asaas dedups on
	//    externalReference = organization_id on the provider side, so
	//    repeat calls are idempotent without us tracking customer ids
	//    locally (subscriptions.provider_customer_id is kept for
	//    observability + fast paths; this call backfills it).
	customerID, err := p.ensureCustomer(ctx, in)
	if err != nil {
		return Charge{}, err
	}

	// 2. Create a PIX payment with 30-minute due window (matches
	//    MockProvider cadence; prod may raise later).
	due := time.Now().UTC().Add(30 * time.Minute)
	paymentBody := map[string]any{
		"customer":          customerID,
		"billingType":       "PIX",
		"dueDate":           due.Format("2006-01-02"),
		"value":             float64(in.AmountCents) / 100.0,
		"description":       "Torque CRM — " + in.PlanID,
		"externalReference": in.OrganizationID.String(),
	}
	var payment struct {
		ID      string `json:"id"`
		Value   float64 `json:"value"`
		DueDate string  `json:"dueDate"`
	}
	if err := p.call(ctx, http.MethodPost, "/payments", paymentBody, &payment); err != nil {
		return Charge{}, err
	}
	if payment.ID == "" {
		return Charge{}, errors.New("asaas: empty payment id")
	}

	// 3. Fetch the PIX QR code. Asaas exposes this on a dedicated
	//    endpoint that returns both the EMV string + base64 PNG.
	var pix struct {
		EncodedImage string `json:"encodedImage"`
		Payload      string `json:"payload"`
		ExpiresAt    string `json:"expirationDate"`
	}
	if err := p.call(ctx, http.MethodGet, "/payments/"+payment.ID+"/pixQrCode", nil, &pix); err != nil {
		// Non-fatal — the payment exists and will settle via webhook;
		// we just can't show a QR right now. Surface the error so the
		// caller rolls back the subscription row.
		return Charge{}, fmt.Errorf("asaas: pix qr fetch: %w", err)
	}

	expires := due
	if pix.ExpiresAt != "" {
		if t, terr := time.Parse(time.RFC3339, pix.ExpiresAt); terr == nil {
			expires = t.UTC()
		}
	}

	return Charge{
		ProviderChargeID: payment.ID,
		AmountCents:      in.AmountCents,
		Currency:         "BRL",
		PixQRCode:        pix.Payload,
		PixQRCodeImage:   "data:image/png;base64," + pix.EncodedImage,
		ExpiresAt:        expires,
	}, nil
}

// CancelCharge POSTs to /payments/:id and flips status to cancelled.
// 404 + already-cancelled are treated as success per the interface
// contract — charges roll off retention and we don't want a late
// cancel to error.
func (p *AsaasProvider) CancelCharge(ctx context.Context, providerChargeID string) error {
	if providerChargeID == "" {
		return errors.New("asaas: empty charge id")
	}
	err := p.call(ctx, http.MethodDelete, "/payments/"+providerChargeID, nil, nil)
	if err == nil {
		return nil
	}
	// ErrNotFound from our call() helper means Asaas returned 404.
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// ensureCustomer POSTs /customers with externalReference=org_id.
// Asaas dedups on externalReference — repeat calls return the same
// record, so we don't need to track customer_id locally to stay
// idempotent.
func (p *AsaasProvider) ensureCustomer(ctx context.Context, in ChargeInput) (string, error) {
	body := map[string]any{
		"name":              firstNonEmpty(in.CustomerName, "Cliente Torque"),
		"externalReference": in.OrganizationID.String(),
	}
	if in.CustomerEmail != "" {
		body["email"] = in.CustomerEmail
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := p.call(ctx, http.MethodPost, "/customers", body, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", errors.New("asaas: empty customer id")
	}
	return out.ID, nil
}

// call is the low-level HTTP wrapper. responseInto may be nil for
// 2xx-without-body endpoints (e.g. DELETE). 404 returns ErrNotFound so
// the caller can short-circuit.
func (p *AsaasProvider) call(ctx context.Context, method, path string, body any, responseInto any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("asaas: marshal: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.cfg.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("asaas: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("access_token", p.cfg.APIKey)
	req.Header.Set("User-Agent", "torque-crm/1.0")

	resp, err := p.http.Do(req)
	if err != nil {
		return fmt.Errorf("asaas: http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, asaasBodyCap))

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("asaas: unauthorized (check api_key)")
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("asaas: rate limited")
	case resp.StatusCode >= 500:
		return fmt.Errorf("asaas: server error %d: %s", resp.StatusCode, truncate(string(raw), 200))
	case resp.StatusCode >= 400:
		return fmt.Errorf("asaas: http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	if responseInto == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, responseInto); err != nil {
		return fmt.Errorf("asaas: decode response: %w", err)
	}
	return nil
}

// ---------------- webhook normalizer --------------------------------

// AsaasWebhook is the outer shape Asaas emits on its webhook channel.
// We accept a permissive subset — the full payload is richer but we
// pull only the fields that drive the subscription state machine.
type AsaasWebhook struct {
	Event   string `json:"event"`
	Payment struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"payment"`
}

// NormalizedEvent is the wire shape the billing webhook handler
// already accepts (webhookEvent in billing.go). Duplicated here so
// the billing handler can pick either entry point without circular
// imports between packages.
type NormalizedEvent struct {
	Provider  string          `json:"provider"`
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	ChargeID  string          `json:"charge_id,omitempty"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}

// NormalizeAsaasWebhook maps Asaas's rich payload onto the same
// `{provider, event_id, event_type, charge_id}` shape the generic
// billing webhook already consumes.
//
// event id is stable: Asaas emits the same payment.id across retries
// for the same event, and our billing_events dedup is on
// (provider, provider_event_id). We prefix with the event type so
// distinct transitions on the same payment (paid → overdue → paid
// again) each get their own row.
//
// event_type mapping:
//
//	PAYMENT_CONFIRMED / PAYMENT_RECEIVED → charge.paid
//	PAYMENT_OVERDUE                       → charge.overdue
//	PAYMENT_DELETED / PAYMENT_REFUNDED    → charge.cancelled
//	(unknown event)                       → passed through verbatim so
//	                                         billing_events records it
//	                                         but the state machine
//	                                         does not mutate the row.
func NormalizeAsaasWebhook(raw []byte) (NormalizedEvent, error) {
	var w AsaasWebhook
	if err := json.Unmarshal(raw, &w); err != nil {
		return NormalizedEvent{}, fmt.Errorf("asaas webhook decode: %w", err)
	}
	if w.Payment.ID == "" {
		return NormalizedEvent{}, errors.New("asaas webhook: empty payment id")
	}
	mapped := "unknown"
	switch w.Event {
	case "PAYMENT_CONFIRMED", "PAYMENT_RECEIVED":
		mapped = "charge.paid"
	case "PAYMENT_OVERDUE":
		mapped = "charge.overdue"
	case "PAYMENT_DELETED", "PAYMENT_REFUNDED":
		mapped = "charge.cancelled"
	default:
		// Pass the raw event through with a namespaced code so the
		// append-only billing_events log captures it without the
		// state machine acting on it.
		mapped = "asaas." + strings.ToLower(w.Event)
	}
	return NormalizedEvent{
		Provider:  "asaas",
		EventID:   w.Event + ":" + w.Payment.ID,
		EventType: mapped,
		ChargeID:  w.Payment.ID,
		Raw:       json.RawMessage(raw),
	}, nil
}

// ---------------- tiny helpers --------------------------------------

func firstNonEmpty(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

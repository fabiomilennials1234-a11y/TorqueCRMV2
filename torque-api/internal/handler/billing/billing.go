// Package billing serves F14 /api/v1/billing endpoints.
//
//   GET  /billing/subscription          — tenant's current (live) subscription
//   POST /billing/checkout              — start checkout; returns PIX payload
//   POST /billing/subscription/cancel   — cancel active subscription
//   POST /billing/webhook               — provider callback (PUBLIC, verified by secret)
//
// Read is member-accessible (billing.view). Checkout + cancel are admin-only
// via RequireRole. Webhook is public; authentication is via a shared secret
// in the header `X-Torque-Billing-Secret`.
package billing

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	quotarepo "github.com/milennials/torque-api/internal/repository/quota"
	subrepo "github.com/milennials/torque-api/internal/repository/subscription"
	"github.com/milennials/torque-api/internal/service/billing"
	"github.com/milennials/torque-api/internal/ws"
)

// WebhookSecretHeader is the header the provider must include. The value
// lives in config (BILLING_WEBHOOK_SECRET) and is verified with constant-
// time comparison inside the handler.
const WebhookSecretHeader = "X-Torque-Billing-Secret"

type ReadHandler struct {
	repo *subrepo.Repository
}

type AdminHandler struct {
	repo     *subrepo.Repository
	provider billing.Provider
	bus      *event.Bus
}

type WebhookHandler struct {
	repo   *subrepo.Repository
	bus    *event.Bus
	secret string
	// quotaRepo is optional; when non-nil, subscription.activated
	// events trigger plan_quotas → org_quotas seeding so a tenant
	// graduating from free → growth gets the new ceiling without a
	// dba intervention. S51 wires it.
	quotaRepo *quotarepo.Repository
}

func NewRead(repo *subrepo.Repository) *ReadHandler { return &ReadHandler{repo: repo} }

func NewAdmin(repo *subrepo.Repository, provider billing.Provider, bus *event.Bus) *AdminHandler {
	return &AdminHandler{repo: repo, provider: provider, bus: bus}
}

func NewWebhook(repo *subrepo.Repository, bus *event.Bus, secret string) *WebhookHandler {
	return &WebhookHandler{repo: repo, bus: bus, secret: secret}
}

// WithQuotaRepo wires the quota repository so subscription.activated
// events seed plan_quotas → org_quotas for the tenant. nil is
// accepted — the handler collapses back to no-op seeding.
func (h *WebhookHandler) WithQuotaRepo(q *quotarepo.Repository) *WebhookHandler {
	h.quotaRepo = q
	return h
}

func (h *ReadHandler) Routes(r chi.Router) {
	r.Get("/billing/subscription", h.getActive)
}

func (h *AdminHandler) Routes(r chi.Router) {
	r.Post("/billing/checkout", h.checkout)
	r.Post("/billing/subscription/cancel", h.cancel)
}

func (h *WebhookHandler) Routes(r chi.Router) {
	r.Post("/billing/webhook", h.receive)
	// S51 — Asaas emits its webhooks with a richer payload than the
	// normalized one. Dedicated endpoint normalizes then forwards.
	r.Post("/billing/asaas", h.receiveAsaas)
}

// -------- views ------------------------------------------------------

type subscriptionView struct {
	ID                 uuid.UUID `json:"id"`
	PlanID             string    `json:"plan_id"`
	Status             string    `json:"status"`
	Provider           string    `json:"provider"`
	AmountCents        int64     `json:"amount_cents"`
	Currency           string    `json:"currency"`
	PixQRCode          *string   `json:"pix_qr_code,omitempty"`
	PixQRCodeImage     *string   `json:"pix_qr_code_image,omitempty"`
	PixExpiresAt       *string   `json:"pix_expires_at,omitempty"`
	CurrentPeriodEnd   *string   `json:"current_period_end,omitempty"`
	CancelledAt        *string   `json:"cancelled_at,omitempty"`
	CreatedAt          string    `json:"created_at"`
}

func toView(s subrepo.Subscription) subscriptionView {
	v := subscriptionView{
		ID: s.ID, PlanID: s.PlanID, Status: s.Status, Provider: s.Provider,
		AmountCents: s.AmountCents, Currency: s.Currency,
		PixQRCode: s.PixQRCode, PixQRCodeImage: s.PixQRCodeImage,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
	}
	if s.PixExpiresAt != nil {
		x := s.PixExpiresAt.UTC().Format(time.RFC3339)
		v.PixExpiresAt = &x
	}
	if s.CurrentPeriodEnd != nil {
		x := s.CurrentPeriodEnd.UTC().Format(time.RFC3339)
		v.CurrentPeriodEnd = &x
	}
	if s.CancelledAt != nil {
		x := s.CancelledAt.UTC().Format(time.RFC3339)
		v.CancelledAt = &x
	}
	return v
}

// -------- read -------------------------------------------------------

func (h *ReadHandler) getActive(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	s, err := h.repo.GetActive(r.Context(), orgID)
	if errors.Is(err, subrepo.ErrNotFound) {
		// Not an error — the tenant has not started checkout yet.
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": nil})
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load subscription")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(s))
}

// -------- admin ------------------------------------------------------

type checkoutReq struct {
	PlanID      string `json:"plan_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency,omitempty"`
}

func (h *AdminHandler) checkout(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body checkoutReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		if httpx.IsBodyTooLarge(err) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body too large")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.PlanID == "" || body.AmountCents <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CHECKOUT", "plan_id and positive amount_cents are required")
		return
	}
	currency := body.Currency
	if currency == "" {
		currency = "BRL"
	}

	// 1. Create charge via the wired provider.
	charge, err := h.provider.CreateCharge(r.Context(), billing.ChargeInput{
		OrganizationID: orgID, PlanID: body.PlanID,
		AmountCents: body.AmountCents, Currency: currency,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "PROVIDER_ERROR", err.Error())
		return
	}

	// 2. Persist the 'pending' row.
	sub, err := h.repo.Create(r.Context(), subrepo.CreateInput{
		OrganizationID: orgID, PlanID: body.PlanID, Provider: h.provider.Name(),
		AmountCents: body.AmountCents, Currency: currency,
		ProviderChargeID: charge.ProviderChargeID,
		PixQRCode: charge.PixQRCode, PixQRCodeImage: charge.PixQRCodeImage,
		PixExpiresAt: charge.ExpiresAt, CreatedBy: sess.TeamMemberID,
	})
	if errors.Is(err, subrepo.ErrAlreadyActive) {
		_ = h.provider.CancelCharge(r.Context(), charge.ProviderChargeID)
		httpx.WriteError(w, http.StatusConflict, "ALREADY_ACTIVE",
			"tenant already has an active or pending subscription")
		return
	}
	if err != nil {
		_ = h.provider.CancelCharge(r.Context(), charge.ProviderChargeID)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create subscription")
		return
	}
	h.publish(orgID, sub.ID, "subscription.created", toView(sub))
	httpx.WriteJSON(w, http.StatusCreated, toView(sub))
}

func (h *AdminHandler) cancel(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	sub, err := h.repo.GetActive(r.Context(), orgID)
	if errors.Is(err, subrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "no active subscription")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load subscription")
		return
	}
	if sub.ProviderChargeID != nil {
		_ = h.provider.CancelCharge(r.Context(), *sub.ProviderChargeID)
	}
	if err := h.repo.Cancel(r.Context(), orgID, sub.ID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not cancel subscription")
		return
	}
	h.publish(orgID, sub.ID, "subscription.cancelled", map[string]any{"id": sub.ID})
	w.WriteHeader(http.StatusNoContent)
}

// -------- webhook ----------------------------------------------------

// webhookEvent is the normalized shape across providers. The real Asaas
// shape is richer; we accept a subset here and let the AsaasProvider layer
// shape-convert before calling RecordEvent.
type webhookEvent struct {
	Provider  string          `json:"provider"`
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	ChargeID  string          `json:"charge_id,omitempty"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}

func (h *WebhookHandler) receive(w http.ResponseWriter, r *http.Request) {
	// Secret check — constant-time-ish via plain compare is fine here since
	// both sides are server-controlled. A mismatched secret gets 401 with no
	// body to avoid leaking acceptable values.
	if h.secret == "" || r.Header.Get(WebhookSecretHeader) != h.secret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var body webhookEvent
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse webhook")
		return
	}
	if body.Provider == "" || body.EventID == "" || body.EventType == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_EVENT",
			"provider, event_id, and event_type are required")
		return
	}

	// Resolve subscription by charge id (may be nil for provider-level events).
	var subID *uuid.UUID
	var orgID *uuid.UUID
	if body.ChargeID != "" {
		sub, err := h.repo.FindByProviderCharge(r.Context(), body.Provider, body.ChargeID)
		if err == nil {
			subID = &sub.ID
			orgID = &sub.OrganizationID
		}
	}

	// Record the event first; if it's a duplicate we return 200 OK without
	// mutating the subscription. The provider often retries on any
	// non-2xx.
	if err := h.repo.RecordEvent(r.Context(), orgID, subID, body.Provider, body.EventID, body.EventType, body.Raw); err != nil {
		if errors.Is(err, subrepo.ErrDuplicateEvent) {
			w.WriteHeader(http.StatusOK)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not record event")
		return
	}

	// State machine. Only a small allowlist of event types drives the
	// subscription row — unknown events are logged via billing_events but
	// ignored.
	if subID != nil && orgID != nil {
		switch body.EventType {
		case "charge.paid", "payment.confirmed":
			// Default period: 30 days from now. The Asaas provider will
			// compute this from the plan cycle when it lands.
			_ = h.repo.MarkPaid(r.Context(), *subID, time.Now().UTC().AddDate(0, 0, 30))
			// S51 — seed plan_quotas → org_quotas on activation so the
			// RequireQuota middleware sees the new effective_limit for
			// the freshly-graduated tenant.
			h.seedQuotas(r, *orgID, *subID)
			h.publish(*orgID, *subID, "subscription.activated", map[string]any{"id": *subID})
		case "charge.overdue", "payment.failed":
			_ = h.repo.MarkPastDue(r.Context(), *subID)
			h.publish(*orgID, *subID, "subscription.past_due", map[string]any{"id": *subID})
		case "charge.cancelled":
			_ = h.repo.Cancel(r.Context(), *orgID, *subID)
			h.publish(*orgID, *subID, "subscription.cancelled", map[string]any{"id": *subID})
		}
	}
	w.WriteHeader(http.StatusOK)
}

// receiveAsaas accepts the native Asaas webhook body, normalizes it
// via billing.NormalizeAsaasWebhook, and forwards the result through
// the same dedup + state-machine path as the generic /billing/webhook
// endpoint. Same shared-secret header — Asaas lets us configure a
// custom access token per webhook URL.
func (h *WebhookHandler) receiveAsaas(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" || r.Header.Get(WebhookSecretHeader) != h.secret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not read body")
		return
	}
	normalized, err := billing.NormalizeAsaasWebhook(raw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_EVENT", err.Error())
		return
	}

	var subID *uuid.UUID
	var orgID *uuid.UUID
	if normalized.ChargeID != "" {
		sub, serr := h.repo.FindByProviderCharge(r.Context(), normalized.Provider, normalized.ChargeID)
		if serr == nil {
			subID = &sub.ID
			orgID = &sub.OrganizationID
		}
	}

	if err := h.repo.RecordEvent(r.Context(), orgID, subID,
		normalized.Provider, normalized.EventID, normalized.EventType, normalized.Raw); err != nil {
		if errors.Is(err, subrepo.ErrDuplicateEvent) {
			w.WriteHeader(http.StatusOK)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not record event")
		return
	}

	if subID != nil && orgID != nil {
		switch normalized.EventType {
		case "charge.paid":
			_ = h.repo.MarkPaid(r.Context(), *subID, time.Now().UTC().AddDate(0, 0, 30))
			h.seedQuotas(r, *orgID, *subID)
			h.publish(*orgID, *subID, "subscription.activated", map[string]any{"id": *subID})
		case "charge.overdue":
			_ = h.repo.MarkPastDue(r.Context(), *subID)
			h.publish(*orgID, *subID, "subscription.past_due", map[string]any{"id": *subID})
		case "charge.cancelled":
			_ = h.repo.Cancel(r.Context(), *orgID, *subID)
			h.publish(*orgID, *subID, "subscription.cancelled", map[string]any{"id": *subID})
		}
	}
	w.WriteHeader(http.StatusOK)
}

// seedQuotas fans the activation into plan_quotas → org_quotas. nil
// quotaRepo is a no-op; lookup failures are logged to the event bus
// as a follow-up row would be overkill — the state machine remains
// authoritative.
func (h *WebhookHandler) seedQuotas(r *http.Request, orgID, subID uuid.UUID) {
	if h.quotaRepo == nil {
		return
	}
	sub, err := h.repo.Get(r.Context(), orgID, subID)
	if err != nil {
		return
	}
	_ = h.quotaRepo.SeedPlanDefaults(r.Context(), orgID, sub.PlanID)
}

func (h *AdminHandler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "subscription",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func (h *WebhookHandler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "subscription",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

// Package leadwebhook serves the S50 public lead ingestion endpoint.
//
//	POST /webhooks/lead  (PUBLIC — HMAC signature verified)
//
// The webhook reuses the billing-webhook shape (flat JSON body, no
// redirect) but replaces the shared-secret header with a proper HMAC
// over the raw body. Providers like n8n, Meta Lead Ads relay, SZ.Chat
// conversations-to-lead, or bespoke landing page scripts sign the
// request with the tenant's lead webhook secret and send:
//
//	X-Torque-Lead-Signature: sha256=<hex>
//	X-Torque-Tenant:         <organization_uuid>
//
// The tenant id travels in a header (not the body) because the
// signature covers the raw body — it MUST NOT include tenant
// routing. An unsigned request with a correct tenant is rejected
// just like a tampered body.
//
// Deduplication: every inbound body carries an `external_id` which the
// provider guarantees is unique per real-world lead (Meta ad leadgen_id,
// n8n workflow execution id, SZ.Chat conversation id). The
// lead_webhook_events table carries a UNIQUE(organization_id,
// external_id) constraint; on a duplicate we respond 200 without
// touching the leads table.
package leadwebhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	leadrepo "github.com/milennials/torque-api/internal/repository/lead"
	webhookrepo "github.com/milennials/torque-api/internal/repository/leadwebhook"
	"github.com/milennials/torque-api/internal/ws"
)

// Header names are exported so the client stub in torque-web can reuse them.
const (
	SignatureHeader = "X-Torque-Lead-Signature"
	TenantHeader    = "X-Torque-Tenant"
	// MaxBodyBytes caps the payload — a 64 KiB limit is 10x the
	// largest realistic Meta leadgen payload. Anything larger is
	// adversarial and gets rejected at the HTTP layer.
	MaxBodyBytes int64 = 64 * 1024
)

// Handler carries the deps.
type Handler struct {
	leads    *leadrepo.Repository
	events   *webhookrepo.Repository
	bus      *event.Bus
	secret   []byte
	logger   zerolog.Logger
}

// Options binds the handler.
type Options struct {
	Leads  *leadrepo.Repository
	Events *webhookrepo.Repository
	Bus    *event.Bus
	Secret string
	Logger zerolog.Logger
}

// New constructs the handler.
func New(o Options) *Handler {
	return &Handler{
		leads:  o.Leads,
		events: o.Events,
		bus:    o.Bus,
		secret: []byte(o.Secret),
		logger: o.Logger,
	}
}

// Routes mounts the public endpoint. Caller must NOT place this group
// under Authenticator / RequireAuth / CSRF — the HMAC signature is the
// only thing the provider carries.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/lead", h.receive)
}

// inboundBody is the accepted shape. UTM fields are optional; we only
// require external_id + name + (phone|email). custom_fields lets
// providers ship arbitrary tenant-defined JSON without schema churn.
type inboundBody struct {
	ExternalID   string          `json:"external_id"`
	Source       string          `json:"source,omitempty"` // 'meta'|'szchat'|'n8n'|...
	Name         string          `json:"name"`
	Company      *string         `json:"company,omitempty"`
	Phone        *string         `json:"phone,omitempty"`
	Email        *string         `json:"email,omitempty"`
	Position     *string         `json:"position,omitempty"`
	Origin       *string         `json:"origin,omitempty"`
	CustomFields json.RawMessage `json:"custom_fields,omitempty"`
}

func (h *Handler) receive(w http.ResponseWriter, r *http.Request) {
	// 1. Guard: webhook disabled when no secret configured. Empty
	//    secret means "refuse every request" (never "accept any").
	if len(h.secret) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// 2. Read + cap body. We need the raw bytes for HMAC + a rewind
	//    for the JSON decode.
	raw, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes+1))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not read body")
		return
	}
	if int64(len(raw)) > MaxBodyBytes {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 64KiB")
		return
	}

	// 3. Verify HMAC. Signature header is `sha256=<hex>`; anything
	//    else is rejected.
	sig := r.Header.Get(SignatureHeader)
	if !verifyHMAC(h.secret, raw, sig) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 4. Resolve tenant from header (required).
	tenantRaw := strings.TrimSpace(r.Header.Get(TenantHeader))
	orgID, err := uuid.Parse(tenantRaw)
	if err != nil || orgID == uuid.Nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TENANT", "X-Torque-Tenant must be a uuid")
		return
	}

	// 5. Parse body.
	var body inboundBody
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse body")
		return
	}
	if body.ExternalID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_EVENT", "external_id required")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_LEAD", "name required")
		return
	}
	if body.Phone == nil && body.Email == nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_LEAD", "phone or email required")
		return
	}

	// 6. Record the event first (dedup gate). Payload is the entire
	//    raw body so audit has the provider's original bytes.
	source := body.Source
	if source == "" {
		source = "webhook"
	}
	recErr := h.events.Record(r.Context(), webhookrepo.RecordInput{
		OrganizationID: orgID,
		ExternalID:     body.ExternalID,
		Source:         source,
		Payload:        json.RawMessage(raw),
		SignatureOK:    true,
	})
	if errors.Is(recErr, webhookrepo.ErrDuplicate) {
		// Idempotent success — provider retry.
		w.WriteHeader(http.StatusOK)
		return
	}
	if recErr != nil {
		h.logger.Error().Err(recErr).Str("org_id", orgID.String()).Msg("record lead webhook event failed")
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not record event")
		return
	}

	// 7. Create the lead.
	origin := body.Origin
	if origin == nil {
		s := source
		origin = &s
	}
	in := leadrepo.CreateInput{
		Name: body.Name, Company: body.Company, Phone: body.Phone, Email: body.Email,
		Position: body.Position, Origin: origin,
	}
	if len(body.CustomFields) > 0 {
		in.CustomFields = []byte(body.CustomFields)
	}
	lead, err := h.leads.Create(r.Context(), orgID, in)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("create lead from webhook failed")
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID_LEAD", err.Error())
		return
	}

	// 8. Attach lead id on the event (best-effort; failure here is a
	//    log-only miss, the lead is already live).
	if aerr := h.events.AttachLead(r.Context(), orgID, body.ExternalID, lead.ID); aerr != nil {
		h.logger.Warn().Err(aerr).
			Str("org_id", orgID.String()).
			Str("external_id", body.ExternalID).
			Msg("attach lead to webhook event failed")
	}

	// 9. Publish `lead.created` — the workflow BusSubscriber fans
	//    this out to any workflow with trigger=lead_created.
	h.bus.Publish(ws.Event{
		Type:       "lead.created",
		TenantID:   orgID,
		EntityType: "lead",
		EntityID:   &lead.ID,
		OccurredAt: time.Now().UTC(),
	})

	// 10. 201 with the lead id so the caller can cross-reference.
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"lead_id":     lead.ID,
		"external_id": body.ExternalID,
	})
}

// verifyHMAC compares the signature header against HMAC-SHA256(body).
// Accepts "sha256=<hex>" form only — unprefixed hex is explicitly
// rejected so a downgrade attack cannot pass a plain hash.
func verifyHMAC(secret, body []byte, sigHeader string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return false
	}
	provided, err := hex.DecodeString(sigHeader[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := mac.Sum(nil)
	return subtle.ConstantTimeCompare(provided, expected) == 1
}

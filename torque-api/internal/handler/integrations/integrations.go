// Package integrations serves S49 /integrations endpoints.
//
//	GET  /integrations                         — list connected providers (member)
//	GET  /integrations/google/connect          — 302 → Google consent (admin)
//	GET  /integrations/google/callback         — OAuth redirect landing (admin — CSRF-exempt)
//	POST /integrations/google/disconnect       — revoke + delete (admin)
//	POST /integrations/tinyerp                 — store API key (admin)
//	POST /integrations/tinyerp/disconnect      — delete (admin)
//	POST /integrations/tinyerp/push-order      — create order for proposal (admin)
//
// The callback MUST be mounted on a route group that has Authenticator +
// RequireAuth + TenantScope but NOT CSRF — the browser arrives via a
// full-page 302 from Google and cannot carry X-CSRF-Token. State HMAC
// (below) closes the CSRF gap.
//
// State format:
//
//	base64url(org_uuid_16B | ts_4B_BE | nonce_8B) + "." +
//	base64url(hmac_sha256(secret, above))[:16]
//
// TTL is 10 minutes and the HMAC is constant-time compared. State binds
// the OAuth flow to the originating org so a master impersonator cannot
// hijack a connection started from a different tenant.
package integrations

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	"github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/service/integration/gcal"
	"github.com/milennials/torque-api/internal/service/integration/tinyerp"
)

// stateTTL is how long an unconsumed state value remains valid.
const stateTTL = 10 * time.Minute

// Handler groups S49 endpoints.
type Handler struct {
	store       *integrationrepo.Store
	gcal        *gcal.Provider
	tiny        *tinyerp.Provider
	stateSecret []byte
	logger      zerolog.Logger
	// frontendBase is the origin the /connected / /error redirect lands on.
	// Empty = same origin (relative redirect).
	frontendBase string
}

// Options binds the handler.
type Options struct {
	Store        *integrationrepo.Store
	GCal         *gcal.Provider
	Tiny         *tinyerp.Provider
	StateSecret  []byte
	Logger       zerolog.Logger
	FrontendBase string
}

// New constructs the handler.
func New(o Options) *Handler {
	return &Handler{
		store:        o.Store,
		gcal:         o.GCal,
		tiny:         o.Tiny,
		stateSecret:  o.StateSecret,
		logger:       o.Logger,
		frontendBase: strings.TrimRight(o.FrontendBase, "/"),
	}
}

// MemberRoutes mounts the member-safe read endpoint (GET /integrations).
// Must sit inside the authenticated + TenantScope group.
func (h *Handler) MemberRoutes(r chi.Router) {
	r.Get("/integrations", h.list)
}

// AdminRoutes mounts admin-only mutations + OAuth connect. Callback is
// NOT mounted here — see CallbackRoute for the CSRF-exempt subgroup.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/integrations/google/connect", h.googleConnect)
	r.Post("/integrations/google/disconnect", h.googleDisconnect)
	r.Post("/integrations/tinyerp", h.tinyConnect)
	r.Post("/integrations/tinyerp/disconnect", h.tinyDisconnect)
	r.Post("/integrations/tinyerp/push-order", h.tinyPushOrder)
}

// CallbackRoute mounts the OAuth callback on a CSRF-exempt group. The
// caller is expected to keep Authenticator + RequireAuth + TenantScope
// upstream; the HMAC in the state verifies origin.
func (h *Handler) CallbackRoute(r chi.Router) {
	r.Get("/integrations/google/callback", h.googleCallback)
}

// ---------------- list -----------------------------------------------

type credentialView struct {
	Provider          string     `json:"provider"`
	Connected         bool       `json:"connected"`
	ExternalAccountID *string    `json:"external_account_id,omitempty"`
	Scopes            []string   `json:"scopes,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	LastSuccessAt     *time.Time `json:"last_success_at,omitempty"`
	LastErrorText     *string    `json:"last_error_text,omitempty"`
	LastErrorAt       *time.Time `json:"last_error_at,omitempty"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	metas, err := h.store.List(r.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("list integrations failed")
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list integrations")
		return
	}
	out := make([]credentialView, 0, len(metas))
	for _, m := range metas {
		out = append(out, credentialView{
			Provider:          m.Provider,
			Connected:         true,
			ExternalAccountID: m.ExternalAccountID,
			Scopes:            m.Scopes,
			ExpiresAt:         m.ExpiresAt,
			LastSuccessAt:     m.LastSuccessAt,
			LastErrorText:     m.LastErrorText,
			LastErrorAt:       m.LastErrorAt,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// ---------------- google connect / callback / disconnect -------------

func (h *Handler) googleConnect(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if h.gcal == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "GOOGLE_OAUTH_DISABLED",
			"google oauth is not configured")
		return
	}
	state, err := h.newState(orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("state mint failed")
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "state mint failed")
		return
	}
	http.Redirect(w, r, h.gcal.AuthURL(state), http.StatusFound)
}

func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if h.gcal == nil {
		h.settingsRedirect(w, r, "google", "error", "oauth_disabled")
		return
	}

	rawState := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		h.settingsRedirect(w, r, "google", "error", errParam)
		return
	}
	stateOrg, err := h.verifyState(rawState)
	if err != nil {
		h.logger.Warn().Err(err).Str("org_id", orgID.String()).Msg("invalid oauth state")
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATE", "state invalid or expired")
		return
	}
	if stateOrg != orgID {
		h.logger.Warn().
			Str("state_org_id", stateOrg.String()).
			Str("session_org_id", orgID.String()).
			Msg("oauth state org mismatch")
		httpx.WriteError(w, http.StatusBadRequest, "STATE_ORG_MISMATCH", "state mismatch")
		return
	}
	if code == "" {
		httpx.WriteError(w, http.StatusBadRequest, "MISSING_CODE", "code required")
		return
	}

	tok, err := h.gcal.ExchangeCode(r.Context(), code)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("google code exchange failed")
		reason := "exchange_failed"
		if errors.Is(err, integration.ErrAuthFailed) {
			reason = "auth_failed"
		} else if errors.Is(err, integration.ErrUnreachable) {
			reason = "unreachable"
		}
		h.settingsRedirect(w, r, "google", "error", reason)
		return
	}

	email := gcal.DecodeIDTokenEmail(tok.IDToken)
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}
	exp := time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second)
	scopes := strings.Fields(tok.Scope)
	if len(scopes) == 0 {
		scopes = nil
	}

	if err := h.store.Upsert(r.Context(), orgID, integrationrepo.ProviderGoogle,
		integrationrepo.UpsertInput{
			AccessToken:       tok.AccessToken,
			RefreshToken:      tok.RefreshToken,
			TokenType:         tok.TokenType,
			ExpiresAt:         &exp,
			Scopes:            scopes,
			ExternalAccountID: emailPtr,
		}); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("persist google credential failed")
		h.settingsRedirect(w, r, "google", "error", "persist_failed")
		return
	}
	_ = h.store.MarkSuccess(r.Context(), orgID, integrationrepo.ProviderGoogle)
	h.settingsRedirect(w, r, "google", "connected", "")
}

func (h *Handler) googleDisconnect(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if h.gcal == nil {
		if err := h.store.Delete(r.Context(), orgID, integrationrepo.ProviderGoogle); err != nil &&
			!errors.Is(err, integrationrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "disconnect failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.gcal.Disconnect(r.Context(), orgID); err != nil &&
		!errors.Is(err, integrationrepo.ErrNotFound) {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("gcal disconnect failed")
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "disconnect failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------- tinyerp --------------------------------------------

type tinyConnectReq struct {
	APIKey string `json:"api_key"`
}

func (h *Handler) tinyConnect(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if h.tiny == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "TINYERP_DISABLED",
			"tinyerp adapter not configured")
		return
	}
	var body tinyConnectReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	if err := h.tiny.StoreAPIKey(r.Context(), orgID, body.APIKey); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_API_KEY", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) tinyDisconnect(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if err := h.store.Delete(r.Context(), orgID, integrationrepo.ProviderTinyERP); err != nil &&
		!errors.Is(err, integrationrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "disconnect failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type pushOrderItem struct {
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitCents   int64  `json:"unit_cents"`
}

type pushOrderReq struct {
	CustomerEmail string          `json:"customer_email"`
	CustomerName  string          `json:"customer_name"`
	Items         []pushOrderItem `json:"items"`
	TotalCents    int64           `json:"total_cents"`
	Currency      string          `json:"currency"`
}

type pushOrderView struct {
	ProviderOrderID string    `json:"provider_order_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func (h *Handler) tinyPushOrder(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	if h.tiny == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "TINYERP_DISABLED",
			"tinyerp adapter not configured")
		return
	}
	var body pushOrderReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse")
		return
	}
	items := make([]integration.OrderItem, 0, len(body.Items))
	for _, it := range body.Items {
		items = append(items, integration.OrderItem{
			SKU:         it.SKU,
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitCents:   it.UnitCents,
		})
	}
	res, err := h.tiny.CreateOrderForOrg(r.Context(), orgID, integration.OrderInput{
		CustomerEmail: body.CustomerEmail,
		CustomerName:  body.CustomerName,
		Items:         items,
		TotalCents:    body.TotalCents,
		Currency:      body.Currency,
	})
	if errors.Is(err, integrationrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusPreconditionFailed, "TINYERP_NOT_CONNECTED",
			"tinyerp credentials missing for this organization")
		return
	}
	if errors.Is(err, integration.ErrAuthFailed) {
		httpx.WriteError(w, http.StatusBadGateway, "TINYERP_AUTH", "tinyerp refused credentials")
		return
	}
	if errors.Is(err, integration.ErrRateLimited) {
		httpx.WriteError(w, http.StatusTooManyRequests, "TINYERP_RATE_LIMITED",
			"tinyerp rate-limited the request")
		return
	}
	if errors.Is(err, integration.ErrUnreachable) {
		httpx.WriteError(w, http.StatusBadGateway, "TINYERP_UNREACHABLE",
			"tinyerp unreachable")
		return
	}
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("push order failed")
		httpx.WriteError(w, http.StatusBadGateway, "TINYERP_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, pushOrderView{
		ProviderOrderID: res.ProviderOrderID,
		CreatedAt:       res.CreatedAt,
	})
}

// ---------------- state HMAC -----------------------------------------

// newState mints a signed state binding (orgID, issuedAt, nonce).
func (h *Handler) newState(orgID uuid.UUID) (string, error) {
	var payload [16 + 4 + 8]byte
	copy(payload[:16], orgID[:])
	ts := uint32(time.Now().UTC().Unix())
	binary.BigEndian.PutUint32(payload[16:20], ts)
	if _, err := rand.Read(payload[20:]); err != nil {
		return "", fmt.Errorf("state nonce: %w", err)
	}
	bodyB64 := base64.RawURLEncoding.EncodeToString(payload[:])
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write([]byte(bodyB64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	// Trim signature to 16 bytes (24 base64url chars); 128 bits is
	// overkill for a 10-minute TTL but matches the spec.
	if len(sig) > 24 {
		sig = sig[:24]
	}
	return bodyB64 + "." + sig, nil
}

// verifyState parses, HMAC-checks (constant-time), and TTL-checks the state.
// Returns the org uuid encoded in the payload on success.
func (h *Handler) verifyState(raw string) (uuid.UUID, error) {
	dot := strings.IndexByte(raw, '.')
	if dot <= 0 || dot == len(raw)-1 {
		return uuid.Nil, errors.New("state malformed")
	}
	bodyB64 := raw[:dot]
	sig := raw[dot+1:]

	// Recompute expected signature.
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write([]byte(bodyB64))
	expSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(expSig) > 24 {
		expSig = expSig[:24]
	}
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expSig)) != 1 {
		return uuid.Nil, errors.New("state signature mismatch")
	}

	payload, err := base64.RawURLEncoding.DecodeString(bodyB64)
	if err != nil || len(payload) != 28 {
		return uuid.Nil, errors.New("state payload malformed")
	}
	var orgID uuid.UUID
	copy(orgID[:], payload[:16])
	ts := int64(binary.BigEndian.Uint32(payload[16:20]))
	issued := time.Unix(ts, 0).UTC()
	if time.Since(issued) > stateTTL {
		return uuid.Nil, errors.New("state expired")
	}
	if time.Until(issued) > time.Minute {
		// Clock skew guard — reject states from the far future.
		return uuid.Nil, errors.New("state from future")
	}
	return orgID, nil
}

// ---------------- settings redirect helper ---------------------------

func (h *Handler) settingsRedirect(w http.ResponseWriter, r *http.Request, provider, status, reason string) {
	target := h.frontendBase + "/settings?tab=integrations&provider=" + provider + "&status=" + status
	if reason != "" {
		target += "&reason=" + reason
	}
	http.Redirect(w, r, target, http.StatusFound)
}

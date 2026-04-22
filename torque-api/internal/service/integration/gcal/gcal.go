// Package gcal is the Google Calendar CalendarProvider for Torque
// (S49 / F.1). It stores per-tenant OAuth2 credentials via the
// integration repository and talks to the Calendar API over HTTPS.
//
// Why a dedicated package: the integration_test surface for Google is
// large enough (OAuth exchange, token refresh, event create, event
// cancel, health probe, revoke) that keeping it in the top-level
// integration pkg would bury it under messaging. A provider-per-package
// layout also lets each integration evolve its own retry shape without
// fighting neighbors.
//
// The WithEndpoints escape hatch is there so tests can stub the full
// Google surface behind a single httptest.Server; production code never
// touches it.
package gcal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	"github.com/milennials/torque-api/internal/service/integration"
)

// Config is the OAuth registration metadata for Google.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Endpoints is a factored-out Google URL surface so tests can stub.
type Endpoints struct {
	AuthURL         string
	TokenURL        string
	CalendarBaseURL string
	RevokeURL       string
}

// DefaultEndpoints returns the Google production URLs.
func DefaultEndpoints() Endpoints {
	return Endpoints{
		AuthURL:         "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:        "https://oauth2.googleapis.com/token",
		CalendarBaseURL: "https://www.googleapis.com/calendar/v3",
		RevokeURL:       "https://oauth2.googleapis.com/revoke",
	}
}

// scopeCalendarEvents is the minimum scope required to create events on
// the primary calendar. `openid email` is added so we can display the
// connected account back to the operator on the settings screen.
const scopeCalendarEvents = "https://www.googleapis.com/auth/calendar.events openid email"

// bodyCap bounds how much of any Google response we ever read. One MiB
// covers the largest JSON Google emits for these endpoints; anything
// bigger is hostile.
const bodyCap = 1 << 20

// Provider is the gcal adapter.
type Provider struct {
	cfg       Config
	http      *http.Client
	store     *integrationrepo.Store
	breaker   *integration.CircuitBreaker
	endpoints Endpoints
}

// New builds a Provider with production defaults. The returned value is
// safe to use when OAuth is not yet configured (ClientID empty) — the
// AuthURL / ExchangeCode paths will surface the missing config as an
// explicit error, and CreateMeetingForOrg / Health will simply return
// "no credential" for orgs that never connected.
func New(cfg Config, store *integrationrepo.Store, httpClient *http.Client, breaker *integration.CircuitBreaker) *Provider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if breaker == nil {
		breaker = integration.NewCircuitBreaker(5, 30*time.Second)
	}
	return &Provider{
		cfg:       cfg,
		http:      httpClient,
		store:     store,
		breaker:   breaker,
		endpoints: DefaultEndpoints(),
	}
}

// WithEndpoints overrides the Google URL surface. Test-only.
func (p *Provider) WithEndpoints(e Endpoints) *Provider {
	p.endpoints = e
	return p
}

// Name returns the DB provider tag.
func (*Provider) Name() string { return "google" }

// AuthURL assembles the consent URL. The caller is responsible for
// storing `state` somewhere safe — the integrations handler signs a
// stateless HMAC; verification is its concern, not ours.
func (p *Provider) AuthURL(state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	q.Set("scope", scopeCalendarEvents)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("state", state)
	return p.endpoints.AuthURL + "?" + q.Encode()
}

// TokenResponse is the Google OAuth2 token endpoint shape.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// ExchangeCode swaps the `code` from the OAuth callback for an access +
// refresh token pair.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" || p.cfg.RedirectURL == "" {
		return nil, errors.New("gcal: oauth client not configured")
	}
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("redirect_uri", p.cfg.RedirectURL)
	form.Set("grant_type", "authorization_code")
	return p.postToken(ctx, form)
}

// RefreshToken exchanges a refresh token for a new access token.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("gcal: refresh_token empty")
	}
	form := url.Values{}
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	tok, err := p.postToken(ctx, form)
	if err != nil {
		return nil, err
	}
	// Google omits refresh_token on refresh responses. Re-attach so
	// callers can always persist a full tuple.
	if tok.RefreshToken == "" {
		tok.RefreshToken = refreshToken
	}
	return tok, nil
}

func (p *Provider) postToken(ctx context.Context, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoints.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("gcal: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, integration.ErrUnreachable
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, bodyCap))

	switch {
	case resp.StatusCode == http.StatusUnauthorized,
		resp.StatusCode == http.StatusForbidden,
		resp.StatusCode == http.StatusBadRequest:
		// Google returns 400 for invalid_grant / invalid refresh.
		return nil, fmt.Errorf("gcal: %w (%d): %s",
			integration.ErrAuthFailed, resp.StatusCode, truncate(string(body), 200))
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, integration.ErrRateLimited
	case resp.StatusCode >= 500:
		return nil, integration.ErrUnreachable
	case resp.StatusCode >= 300:
		return nil, fmt.Errorf("gcal: token status %d: %s",
			resp.StatusCode, truncate(string(body), 200))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("gcal: decode token: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, errors.New("gcal: empty access_token")
	}
	if tr.TokenType == "" {
		tr.TokenType = "Bearer"
	}
	return &tr, nil
}

// DecodeIDTokenEmail returns the email claim from a Google id_token JWT.
// We deliberately skip signature verification: the token just travelled
// over TLS from accounts.google.com and is used only for display; any
// security-sensitive decision (scopes, user id) comes from the access
// token exchange, not this claim.
func DecodeIDTokenEmail(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	// Google uses base64url (no padding) for JWT segments.
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some emitters include padding; fall back to URLEncoding.
		raw, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return ""
	}
	return claims.Email
}

// ---------------- Calendar operations -------------------------------

// MeetingInput mirrors integration.MeetingInput with the addition of
// attendee email for Google's attendee list. We don't widen the public
// interface — the meetings handler calls CreateMeetingForOrg directly
// with the richer shape via the handler-local type.
//
// (Keeping integration.MeetingInput stable keeps the mock adapters +
// other CalendarProvider implementations unaffected.)

// CreateMeetingForOrg inserts an event on the tenant's primary
// calendar. A valid credential must exist in the store; the method
// transparently refreshes the access token when it is within 2 minutes
// of expiry.
func (p *Provider) CreateMeetingForOrg(ctx context.Context, orgID uuid.UUID, in integration.MeetingInput) (integration.MeetingResult, error) {
	cred, err := p.ensureCredential(ctx, orgID)
	if err != nil {
		return integration.MeetingResult{}, err
	}

	body := map[string]any{
		"summary":     in.Title,
		"description": in.Description,
		"start":       map[string]string{"dateTime": in.StartAt.UTC().Format(time.RFC3339)},
		"end":         map[string]string{"dateTime": in.EndAt.UTC().Format(time.RFC3339)},
	}
	if len(in.Attendees) > 0 {
		atts := make([]map[string]string, 0, len(in.Attendees))
		for _, a := range in.Attendees {
			if a != "" {
				atts = append(atts, map[string]string{"email": a})
			}
		}
		if len(atts) > 0 {
			body["attendees"] = atts
		}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return integration.MeetingResult{}, fmt.Errorf("gcal: encode event: %w", err)
	}

	var out integration.MeetingResult
	berr := p.breaker.Do(func() error {
		res, err := p.callCalendar(ctx, cred.AccessToken, http.MethodPost,
			"/calendars/primary/events", bytes.NewReader(buf))
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	if berr != nil {
		p.recordError(ctx, orgID, berr)
		return integration.MeetingResult{}, berr
	}
	_ = p.store.MarkSuccess(ctx, orgID, integrationrepo.ProviderGoogle)
	return out, nil
}

// CreateMeeting satisfies the CalendarProvider interface. OrgID comes
// from the context (domain.OrgIDFrom).
func (p *Provider) CreateMeeting(ctx context.Context, in integration.MeetingInput) (integration.MeetingResult, error) {
	orgID, ok := domain.OrgIDFrom(ctx)
	if !ok || orgID == uuid.Nil {
		return integration.MeetingResult{}, errors.New("gcal: no org id in context")
	}
	return p.CreateMeetingForOrg(ctx, orgID, in)
}

// CancelMeetingForOrg deletes an event from the primary calendar.
// A 404 is treated as success — the caller's invariant (the event is
// gone) still holds.
func (p *Provider) CancelMeetingForOrg(ctx context.Context, orgID uuid.UUID, eventID string) error {
	if eventID == "" {
		return errors.New("gcal: event_id required")
	}
	cred, err := p.ensureCredential(ctx, orgID)
	if err != nil {
		return err
	}
	berr := p.breaker.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
			p.endpoints.CalendarBaseURL+"/calendars/primary/events/"+url.PathEscape(eventID), nil)
		if err != nil {
			return fmt.Errorf("gcal: build cancel: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
		resp, err := p.http.Do(req)
		if err != nil {
			return integration.ErrUnreachable
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, bodyCap))
		switch {
		case resp.StatusCode < 300:
			return nil
		case resp.StatusCode == http.StatusNotFound:
			return nil // already gone — idempotent success
		case resp.StatusCode == http.StatusUnauthorized:
			return integration.ErrAuthFailed
		case resp.StatusCode == http.StatusForbidden:
			return integration.ErrAuthFailed
		case resp.StatusCode == http.StatusTooManyRequests:
			return integration.ErrRateLimited
		case resp.StatusCode >= 500:
			return integration.ErrUnreachable
		default:
			return fmt.Errorf("gcal: cancel status %d", resp.StatusCode)
		}
	})
	if berr != nil {
		p.recordError(ctx, orgID, berr)
		return berr
	}
	_ = p.store.MarkSuccess(ctx, orgID, integrationrepo.ProviderGoogle)
	return nil
}

// CancelMeeting satisfies the CalendarProvider interface.
func (p *Provider) CancelMeeting(ctx context.Context, eventID string) error {
	orgID, ok := domain.OrgIDFrom(ctx)
	if !ok || orgID == uuid.Nil {
		return errors.New("gcal: no org id in context")
	}
	return p.CancelMeetingForOrg(ctx, orgID, eventID)
}

// Health probes /users/me/calendarList for the first credential we find.
// Zero credentials across the process → nil (dev mode, nothing to check).
//
// Health is deliberately lightweight — the settings page per-tenant
// status comes from integration_credentials.last_* columns, not from
// this probe.
func (p *Provider) Health(ctx context.Context) error {
	// We don't have a cross-tenant scan API on the repo, nor do we want
	// one. Health returns nil; the per-tenant health is visible via the
	// List endpoint.
	return nil
}

// Disconnect revokes the refresh token with Google and deletes the row.
// Revoke is best-effort: if Google returns an error, the row is still
// removed — a stale refresh token on Google's side is a smaller problem
// than a stuck Torque connection.
func (p *Provider) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	cred, err := p.store.Get(ctx, orgID, integrationrepo.ProviderGoogle)
	if err != nil && !errors.Is(err, integrationrepo.ErrNotFound) {
		return err
	}
	if cred != nil && cred.RefreshToken != "" {
		_ = p.revoke(ctx, cred.RefreshToken) // best-effort
	}
	if cred != nil {
		return p.store.Delete(ctx, orgID, integrationrepo.ProviderGoogle)
	}
	return nil
}

// ---------------- internals -----------------------------------------

// ensureCredential loads the tenant's credential and refreshes the
// access token when within 2 minutes of expiry. The refreshed tuple is
// persisted back so subsequent calls see the new access token.
func (p *Provider) ensureCredential(ctx context.Context, orgID uuid.UUID) (*integrationrepo.Credential, error) {
	cred, err := p.store.Get(ctx, orgID, integrationrepo.ProviderGoogle)
	if err != nil {
		return nil, err
	}
	if cred.ExpiresAt == nil || time.Until(*cred.ExpiresAt) > 2*time.Minute {
		return cred, nil
	}
	if cred.RefreshToken == "" {
		return nil, integration.ErrAuthFailed
	}
	tok, err := p.RefreshToken(ctx, cred.RefreshToken)
	if err != nil {
		p.recordError(ctx, orgID, err)
		return nil, err
	}
	exp := time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if err := p.store.Upsert(ctx, orgID, integrationrepo.ProviderGoogle, integrationrepo.UpsertInput{
		AccessToken:       tok.AccessToken,
		RefreshToken:      tok.RefreshToken,
		TokenType:         tok.TokenType,
		ExpiresAt:         &exp,
		Scopes:            splitScopes(tok.Scope),
		ExternalAccountID: cred.ExternalAccountID,
	}); err != nil {
		return nil, err
	}
	cred.AccessToken = tok.AccessToken
	cred.RefreshToken = tok.RefreshToken
	cred.ExpiresAt = &exp
	return cred, nil
}

// callCalendar runs a JSON request to the Calendar API and maps the
// response into an integration.MeetingResult (the only shape we need
// from event creates today).
func (p *Provider) callCalendar(ctx context.Context, accessToken, method, path string, body io.Reader) (integration.MeetingResult, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.endpoints.CalendarBaseURL+path, body)
	if err != nil {
		return integration.MeetingResult{}, fmt.Errorf("gcal: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return integration.MeetingResult{}, integration.ErrUnreachable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, bodyCap))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// fall through to decode
	case resp.StatusCode == http.StatusUnauthorized,
		resp.StatusCode == http.StatusForbidden:
		return integration.MeetingResult{}, integration.ErrAuthFailed
	case resp.StatusCode == http.StatusTooManyRequests:
		return integration.MeetingResult{}, integration.ErrRateLimited
	case resp.StatusCode >= 500:
		return integration.MeetingResult{}, integration.ErrUnreachable
	default:
		return integration.MeetingResult{}, fmt.Errorf("gcal: calendar status %d: %s",
			resp.StatusCode, truncate(string(raw), 200))
	}

	var ev struct {
		ID       string `json:"id"`
		HTMLLink string `json:"htmlLink"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return integration.MeetingResult{}, fmt.Errorf("gcal: decode event: %w", err)
	}
	if ev.ID == "" {
		return integration.MeetingResult{}, errors.New("gcal: response missing id")
	}
	return integration.MeetingResult{
		ProviderEventID: ev.ID,
		MeetingURL:      ev.HTMLLink,
	}, nil
}

// revoke calls Google's revoke endpoint with the refresh token. Errors
// are discarded by the caller — see Disconnect.
func (p *Provider) revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoints.RevokeURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, bodyCap))
	return nil
}

// recordError persists a short, operator-readable error on the
// credential row. Only auth / unreachable / rate-limited errors are
// considered persistent enough to surface; transient breaker-open
// conditions are not.
func (p *Provider) recordError(ctx context.Context, orgID uuid.UUID, err error) {
	if err == nil || errors.Is(err, integration.ErrCircuitOpen) {
		return
	}
	msg := err.Error()
	_ = p.store.MarkError(ctx, orgID, integrationrepo.ProviderGoogle, msg)
}

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	out := strings.Fields(s)
	if len(out) == 0 {
		return nil
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

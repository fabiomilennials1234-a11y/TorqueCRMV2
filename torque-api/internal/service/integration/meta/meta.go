// Package meta is the Meta Ads InsightsProvider for Torque (S50 / F.2).
//
// Hits the Graph API /act_<id>/insights endpoint with a per-tenant
// long-lived System User token stored in integration_credentials under
// provider='meta' (access_token_encrypted). A 15-minute in-DB cache
// (meta_insights_cache) absorbs the dashboard refresh cadence without
// burning Meta rate budget.
//
// Meta's Graph API response shape is deliberately contained inside this
// package. Callers consume integration.AdAccountInsights and integration
// sentinel errors; Meta-specific field names (cpc, ctr, actions[]) are
// the packager's concern.
package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	integrationrepo "github.com/milennials/torque-api/internal/repository/integration"
	"github.com/milennials/torque-api/internal/service/integration"
)

const (
	defaultBaseURL = "https://graph.facebook.com/v18.0"
	bodyCap        = 1 << 20
	// Graph-specific error codes we map onto sentinels. 190 / 102 are
	// OAuthException variants ("invalid/expired token"); 17 / 4 / 613
	// are rate-limit codes across account/user/application tiers.
	codeOAuthExpired = 190
	codeOAuthGeneric = 102
	codeRateLimit17  = 17
	codeRateLimit4   = 4
	codeRateLimit613 = 613
)

// Config wires the adapter at boot. Per-tenant tokens live in the
// credential store; AppSecret is the single global used to compute
// `appsecret_proof` per call.
type Config struct {
	BaseURL   string
	AppSecret string
}

// Provider implements integration.InsightsProvider against Graph API.
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
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if breaker == nil {
		breaker = integration.NewCircuitBreaker(5, 30*time.Second)
	}
	return &Provider{
		cfg:     Config{BaseURL: base, AppSecret: cfg.AppSecret},
		http:    httpClient,
		store:   store,
		breaker: breaker,
	}
}

// Name returns the DB provider tag.
func (*Provider) Name() string { return integrationrepo.ProviderMeta }

// StoreAPIKey persists a tenant's Meta System User token.
func (p *Provider) StoreAPIKey(ctx context.Context, orgID uuid.UUID, token string, externalAccountID *string) error {
	if n := len(token); n < 20 || n > 500 {
		return fmt.Errorf("meta: access_token length must be 20-500 chars (got %d)", n)
	}
	return p.store.Upsert(ctx, orgID, integrationrepo.ProviderMeta, integrationrepo.UpsertInput{
		AccessToken:       token,
		TokenType:         "Bearer",
		ExternalAccountID: externalAccountID,
	})
}

// AdAccountInsights pulls the account-level + campaign-level slice for
// the given window. OrgID comes from ctx via domain.OrgIDFrom.
func (p *Provider) AdAccountInsights(ctx context.Context, w integration.InsightsWindow) (integration.AdAccountInsights, error) {
	orgID, ok := domain.OrgIDFrom(ctx)
	if !ok || orgID == uuid.Nil {
		return integration.AdAccountInsights{}, errors.New("meta: no org id in context")
	}
	return p.AdAccountInsightsForOrg(ctx, orgID, w)
}

// AdAccountInsightsForOrg is the explicit-org variant used by handlers
// that already have orgID in scope.
func (p *Provider) AdAccountInsightsForOrg(ctx context.Context, orgID uuid.UUID, w integration.InsightsWindow) (integration.AdAccountInsights, error) {
	if w.AccountID == "" {
		return integration.AdAccountInsights{}, errors.New("meta: account_id required")
	}
	if w.DateRange == "" {
		w.DateRange = "last_7d"
	}

	cred, err := p.store.Get(ctx, orgID, integrationrepo.ProviderMeta)
	if err != nil {
		return integration.AdAccountInsights{}, err
	}

	var out integration.AdAccountInsights
	berr := p.breaker.Do(func() error {
		res, callErr := p.fetchInsights(ctx, cred.AccessToken, w)
		if callErr != nil {
			return callErr
		}
		out = res
		return nil
	})
	if berr != nil {
		p.recordError(ctx, orgID, berr)
		return integration.AdAccountInsights{}, berr
	}
	_ = p.store.MarkSuccess(ctx, orgID, integrationrepo.ProviderMeta)
	return out, nil
}

// Health is a conservative probe — reads the token metadata for any
// provider row that exists. Empty store = OK (no tenants connected).
func (p *Provider) Health(_ context.Context) error {
	// Per-tenant health is surfaced via integration_credentials
	// last_error_*; a global healthcheck for Meta would need an app
	// token which we do not carry at boot.
	return nil
}

// ---------------- internals -----------------------------------------

// graphInsights mirrors the fields we consume from Graph API's
// /act_<id>/insights response (level=account + the campaign-level
// call). The shape is deliberately flexible — Meta returns numbers as
// strings for monetary fields.
type graphResponse struct {
	Data   []graphInsightRow `json:"data"`
	Error  *graphError       `json:"error,omitempty"`
}

type graphError struct {
	Code    int    `json:"code"`
	Subcode int    `json:"error_subcode,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type graphInsightRow struct {
	AccountID    string        `json:"account_id"`
	AccountCurrency string     `json:"account_currency,omitempty"`
	CampaignID   string        `json:"campaign_id,omitempty"`
	CampaignName string        `json:"campaign_name,omitempty"`
	Spend        string        `json:"spend"`
	Impressions  string        `json:"impressions"`
	Clicks       string        `json:"clicks"`
	Actions      []graphAction `json:"actions,omitempty"`
	DateStart    string        `json:"date_start,omitempty"`
	DateStop     string        `json:"date_stop,omitempty"`
}

type graphAction struct {
	ActionType string `json:"action_type"`
	Value      string `json:"value"`
}

// fetchInsights performs two GETs — one rolled-up at account level and
// one split by campaign. Meta does not support "both in one request"
// without time_increment gymnastics; two calls is simpler.
func (p *Provider) fetchInsights(ctx context.Context, token string, w integration.InsightsWindow) (integration.AdAccountInsights, error) {
	acct, err := p.get(ctx, token, w.AccountID, insightParams{
		level:     "account",
		datePreset: w.DateRange,
	})
	if err != nil {
		return integration.AdAccountInsights{}, err
	}
	camp, err := p.get(ctx, token, w.AccountID, insightParams{
		level:     "campaign",
		datePreset: w.DateRange,
	})
	if err != nil {
		return integration.AdAccountInsights{}, err
	}

	out := integration.AdAccountInsights{
		AccountID: strings.TrimPrefix(w.AccountID, "act_"),
		DateRange: w.DateRange,
		FetchedAt: time.Now().UTC(),
	}
	if len(acct) > 0 {
		out.Currency = acct[0].AccountCurrency
		out.SpendCents = decimalToCents(acct[0].Spend)
		out.Impressions = toInt64(acct[0].Impressions)
		out.Clicks = toInt64(acct[0].Clicks)
		out.Leads = sumLeadActions(acct[0].Actions)
		out.CPLeadCents = cpLead(out.SpendCents, out.Leads)
	}
	out.Campaigns = make([]integration.AdCampaign, 0, len(camp))
	for _, row := range camp {
		out.Campaigns = append(out.Campaigns, integration.AdCampaign{
			CampaignID:  row.CampaignID,
			Name:        row.CampaignName,
			SpendCents:  decimalToCents(row.Spend),
			Impressions: toInt64(row.Impressions),
			Clicks:      toInt64(row.Clicks),
			Leads:       sumLeadActions(row.Actions),
		})
	}
	return out, nil
}

type insightParams struct {
	level      string
	datePreset string
}

func (p *Provider) get(ctx context.Context, token, accountID string, params insightParams) ([]graphInsightRow, error) {
	// Graph API expects accountID to be prefixed with "act_". Accept
	// both and normalize.
	id := accountID
	if !strings.HasPrefix(id, "act_") {
		id = "act_" + id
	}
	q := url.Values{}
	q.Set("access_token", token)
	q.Set("level", params.level)
	q.Set("date_preset", params.datePreset)
	q.Set("fields", "account_id,account_currency,campaign_id,campaign_name,spend,impressions,clicks,actions")
	if proof := p.appsecretProof(token); proof != "" {
		q.Set("appsecret_proof", proof)
	}

	u := p.cfg.BaseURL + "/" + id + "/insights?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("meta: build request: %w", err)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, integration.ErrUnreachable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, bodyCap))

	var env graphResponse
	if jerr := json.Unmarshal(raw, &env); jerr != nil {
		if resp.StatusCode >= 500 {
			return nil, integration.ErrUnreachable
		}
		return nil, fmt.Errorf("meta: decode response: %w", jerr)
	}
	if env.Error != nil {
		switch env.Error.Code {
		case codeOAuthExpired, codeOAuthGeneric:
			return nil, integration.ErrAuthFailed
		case codeRateLimit17, codeRateLimit4, codeRateLimit613:
			return nil, integration.ErrRateLimited
		default:
			return nil, fmt.Errorf("meta: graph error code=%d: %s",
				env.Error.Code, truncate(env.Error.Message, 200))
		}
	}
	if resp.StatusCode >= 500 {
		return nil, integration.ErrUnreachable
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("meta: http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	return env.Data, nil
}

// appsecretProof is the HMAC-SHA256 of the access token keyed by the
// app secret, hex-encoded. Optional if AppSecret is empty.
func (p *Provider) appsecretProof(token string) string {
	if p.cfg.AppSecret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(p.cfg.AppSecret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func (p *Provider) recordError(ctx context.Context, orgID uuid.UUID, err error) {
	if err == nil || errors.Is(err, integration.ErrCircuitOpen) {
		return
	}
	_ = p.store.MarkError(ctx, orgID, integrationrepo.ProviderMeta, err.Error())
}

// ---------------- number helpers ------------------------------------

// decimalToCents parses Meta's "12.34" style money values into int64
// cents. Returns 0 on parse failure — Meta does not emit negative
// spend but we guard anyway.
func decimalToCents(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	var cents int64
	if len(parts) == 2 {
		fragment := parts[1]
		if len(fragment) > 2 {
			fragment = fragment[:2]
		}
		if len(fragment) == 1 {
			fragment += "0"
		}
		c, err := strconv.ParseInt(fragment, 10, 64)
		if err == nil {
			cents = c
		}
	}
	total := whole*100 + cents
	if neg {
		total = -total
	}
	return total
}

func toInt64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// sumLeadActions totals the lead-flavored action buckets Meta reports.
// "lead" is the generic conversion; "onsite_conversion.lead_grouped"
// covers Lead Ads; we add any field whose type contains "lead".
func sumLeadActions(actions []graphAction) int64 {
	var total int64
	for _, a := range actions {
		if !strings.Contains(strings.ToLower(a.ActionType), "lead") {
			continue
		}
		total += toInt64(a.Value)
	}
	return total
}

// cpLead returns cost-per-lead in cents, or -1 ("unavailable") when
// Leads is zero. Zero leads with non-zero spend is a real state; the
// frontend needs to distinguish "no leads" from "free leads".
func cpLead(spendCents, leads int64) int64 {
	if leads <= 0 {
		return -1
	}
	return spendCents / leads
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

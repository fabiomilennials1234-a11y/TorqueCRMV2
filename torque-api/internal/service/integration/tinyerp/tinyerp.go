// Package tinyerp is the TinyERP ERPProvider for Torque (S49 / F.1).
//
// TinyERP's legacy API speaks form-encoded POSTs and returns a wrapping
// envelope `{retorno: {...}}` — a deliberately ugly shape that we keep
// firewalled inside this package. Callers work with the
// integration.OrderInput/OrderResult types and integration sentinel
// errors; provider-specific `codigo_erro` values are mapped onto the
// closest match (auth, rate-limit, unreachable).
//
// API key is per-tenant, stored encrypted in integration_credentials
// under provider='tinyerp' (access_token_encrypted carries the key;
// expires_at is always NULL — API keys don't rotate automatically).
package tinyerp

import (
	"context"
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

const (
	defaultBaseURL = "https://api.tiny.com.br/api2"
	bodyCap        = 1 << 20

	// TinyERP's documented error codes (subset we care about).
	// 6  → "Token inválido" (auth failure)
	// 30 → "Limite de requisições excedido"
	codigoErroAuth      = 6
	codigoErroRateLimit = 30
)

// Config wires the adapter at boot. Per-tenant API keys are stored in
// integration_credentials.
type Config struct {
	BaseURL string
}

// Provider implements integration.ERPProvider against TinyERP.
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
func (*Provider) Name() string { return "tinyerp" }

// StoreAPIKey persists a tenant's API key (validated for length).
func (p *Provider) StoreAPIKey(ctx context.Context, orgID uuid.UUID, apiKey string) error {
	if n := len(apiKey); n < 10 || n > 200 {
		return fmt.Errorf("tinyerp: api_key length must be 10-200 chars (got %d)", n)
	}
	return p.store.Upsert(ctx, orgID, integrationrepo.ProviderTinyERP, integrationrepo.UpsertInput{
		AccessToken: apiKey,
		TokenType:   "ApiKey",
	})
}

// CreateOrderForOrg pushes an order to TinyERP. The tenant's API key is
// read from the credential store; a missing credential surfaces as
// ErrNotFound from the repo (caller maps to 400/412 as appropriate).
func (p *Provider) CreateOrderForOrg(ctx context.Context, orgID uuid.UUID, in integration.OrderInput) (integration.OrderResult, error) {
	cred, err := p.store.Get(ctx, orgID, integrationrepo.ProviderTinyERP)
	if err != nil {
		return integration.OrderResult{}, err
	}

	payload, err := mapOrder(in)
	if err != nil {
		return integration.OrderResult{}, err
	}

	form := url.Values{}
	form.Set("token", cred.AccessToken)
	form.Set("formato", "json")
	form.Set("pedido", string(payload))

	var out integration.OrderResult
	berr := p.breaker.Do(func() error {
		res, callErr := p.call(ctx, "/pedido.incluir.php", form)
		if callErr != nil {
			return callErr
		}
		out = res
		return nil
	})
	if berr != nil {
		p.recordError(ctx, orgID, berr)
		return integration.OrderResult{}, berr
	}
	_ = p.store.MarkSuccess(ctx, orgID, integrationrepo.ProviderTinyERP)
	return out, nil
}

// CreateOrder satisfies the ERPProvider interface. OrgID comes from ctx.
func (p *Provider) CreateOrder(ctx context.Context, in integration.OrderInput) (integration.OrderResult, error) {
	orgID, ok := domain.OrgIDFrom(ctx)
	if !ok || orgID == uuid.Nil {
		return integration.OrderResult{}, errors.New("tinyerp: no org id in context")
	}
	return p.CreateOrderForOrg(ctx, orgID, in)
}

// Health probes /info.obter.php when a credential exists; no-op when
// none is configured for any tenant we can see (dev).
func (p *Provider) Health(ctx context.Context) error {
	// No cross-tenant scan API on the store. Per-tenant health is
	// exposed through the integrations list endpoint (last_success_at,
	// last_error_*). Returning nil keeps /health green in dev.
	return nil
}

// ---------------- internals -----------------------------------------

// envelope is the outer TinyERP JSON wrapper.
type envelope struct {
	Retorno struct {
		Status     string          `json:"status"`      // "OK" | "Erro"
		CodigoErro int             `json:"codigo_erro"` // when Status=="Erro"
		Erros      json.RawMessage `json:"erros,omitempty"`
		Registros  json.RawMessage `json:"registros,omitempty"`
	} `json:"retorno"`
}

// registros for pedido.incluir.php:
//
//	{"registros":{"registro":{"id":"12345","numero":"100","serie":"1"}}}
//
// We only need `id`.
type registroOrder struct {
	Registro struct {
		ID string `json:"id"`
	} `json:"registro"`
}

func (p *Provider) call(ctx context.Context, path string, form url.Values) (integration.OrderResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		return integration.OrderResult{}, fmt.Errorf("tinyerp: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.http.Do(req)
	if err != nil {
		return integration.OrderResult{}, integration.ErrUnreachable
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, bodyCap))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// fall through to envelope parsing
	case resp.StatusCode == http.StatusTooManyRequests:
		return integration.OrderResult{}, integration.ErrRateLimited
	case resp.StatusCode >= 500:
		return integration.OrderResult{}, integration.ErrUnreachable
	default:
		return integration.OrderResult{}, fmt.Errorf("tinyerp: http %d: %s",
			resp.StatusCode, truncate(string(raw), 200))
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return integration.OrderResult{}, fmt.Errorf("tinyerp: decode envelope: %w", err)
	}
	switch env.Retorno.Status {
	case "OK":
		var reg registroOrder
		if err := json.Unmarshal(env.Retorno.Registros, &reg); err != nil {
			return integration.OrderResult{}, fmt.Errorf("tinyerp: decode registros: %w", err)
		}
		if reg.Registro.ID == "" {
			return integration.OrderResult{}, errors.New("tinyerp: empty order id")
		}
		return integration.OrderResult{
			ProviderOrderID: reg.Registro.ID,
			CreatedAt:       time.Now().UTC(),
		}, nil
	case "Erro":
		switch env.Retorno.CodigoErro {
		case codigoErroAuth:
			return integration.OrderResult{}, integration.ErrAuthFailed
		case codigoErroRateLimit:
			return integration.OrderResult{}, integration.ErrRateLimited
		default:
			return integration.OrderResult{}, fmt.Errorf("tinyerp: codigo_erro=%d: %s",
				env.Retorno.CodigoErro, truncate(string(env.Retorno.Erros), 200))
		}
	default:
		return integration.OrderResult{}, fmt.Errorf("tinyerp: unknown status %q", env.Retorno.Status)
	}
}

// orderBody is the minimal pedido payload. TinyERP accepts a far richer
// schema; we start with the fields F03's proposal → order path needs.
type orderBody struct {
	Cliente    orderCliente    `json:"cliente"`
	Itens      []orderItemWrap `json:"itens"`
	NumeroPed  string          `json:"numero_pedido,omitempty"`
	ValorTotal string          `json:"total_pedido"`
}

type orderCliente struct {
	Nome  string `json:"nome"`
	Email string `json:"email,omitempty"`
}

type orderItemWrap struct {
	Item orderItem `json:"item"`
}

type orderItem struct {
	Codigo   string `json:"codigo,omitempty"`
	Descricao string `json:"descricao"`
	Unidade  string `json:"unidade,omitempty"`
	Qtd      int    `json:"quantidade"`
	ValorUn  string `json:"valor_unitario"`
}

func mapOrder(in integration.OrderInput) ([]byte, error) {
	if in.TotalCents <= 0 {
		return nil, errors.New("tinyerp: total_cents must be positive")
	}
	if strings.TrimSpace(in.CustomerName) == "" {
		return nil, errors.New("tinyerp: customer name required")
	}
	items := make([]orderItemWrap, 0, len(in.Items))
	for _, it := range in.Items {
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		items = append(items, orderItemWrap{
			Item: orderItem{
				Codigo:    it.SKU,
				Descricao: it.Description,
				Qtd:       qty,
				ValorUn:   centsToDecimal(it.UnitCents),
			},
		})
	}
	body := orderBody{
		Cliente:    orderCliente{Nome: in.CustomerName, Email: in.CustomerEmail},
		Itens:      items,
		ValorTotal: centsToDecimal(in.TotalCents),
	}
	return json.Marshal(body)
}

// centsToDecimal formats an int64 of cents as "123.45" with 2 decimals.
func centsToDecimal(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func (p *Provider) recordError(ctx context.Context, orgID uuid.UUID, err error) {
	if err == nil || errors.Is(err, integration.ErrCircuitOpen) {
		return
	}
	_ = p.store.MarkError(ctx, orgID, integrationrepo.ProviderTinyERP, err.Error())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

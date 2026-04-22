package tinyerp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/integration"
)

// We test the HTTP + envelope layer via the unexported `call` method so
// we don't need a real *integrationrepo.Store (which requires a pgxpool)
// to exercise the TinyERP wire protocol. The store-dependent path is
// covered at the handler level / repo integration tests.

func newProvider(url string) *Provider {
	return New(Config{BaseURL: url}, nil, nil, nil)
}

func TestCall_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pedido.incluir.php" {
			t.Errorf("path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if form.Get("token") == "" || form.Get("formato") != "json" {
			t.Errorf("missing form fields: %v", form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"OK","registros":{"registro":{"id":"98765"}}}}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	form := url.Values{}
	form.Set("token", "k")
	form.Set("formato", "json")
	form.Set("pedido", `{"x":1}`)
	res, err := p.call(context.Background(), "/pedido.incluir.php", form)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if res.ProviderOrderID != "98765" {
		t.Fatalf("id: %q", res.ProviderOrderID)
	}
}

func TestCall_AuthFailedByCodigoErro(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"Erro","codigo_erro":6,"erros":[{"erro":"Token inválido."}]}}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	_, err := p.call(context.Background(), "/pedido.incluir.php", url.Values{})
	if !errors.Is(err, integration.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestCall_RateLimitedByCodigoErro(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"Erro","codigo_erro":30}}`))
	}))
	defer srv.Close()
	p := newProvider(srv.URL)
	_, err := p.call(context.Background(), "/pedido.incluir.php", url.Values{})
	if !errors.Is(err, integration.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestCall_UnreachableOn5xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	p := newProvider(srv.URL)
	_, err := p.call(context.Background(), "/pedido.incluir.php", url.Values{})
	if !errors.Is(err, integration.ErrUnreachable) {
		t.Fatalf("expected ErrUnreachable, got %v", err)
	}
}

func TestCall_RateLimitedOn429(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	p := newProvider(srv.URL)
	_, err := p.call(context.Background(), "/pedido.incluir.php", url.Values{})
	if !errors.Is(err, integration.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestMapOrder_Valid(t *testing.T) {
	t.Parallel()
	raw, err := mapOrder(integration.OrderInput{
		CustomerName: "Cliente Teste",
		Items: []integration.OrderItem{
			{SKU: "ABC-1", Description: "Plano Torque", Quantity: 1, UnitCents: 29900},
		},
		TotalCents: 29900,
	})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if !strings.Contains(string(raw), `"total_pedido":"299.00"`) {
		t.Fatalf("total not formatted: %s", string(raw))
	}
	if !strings.Contains(string(raw), `"valor_unitario":"299.00"`) {
		t.Fatalf("unit not formatted: %s", string(raw))
	}
}

func TestMapOrder_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := mapOrder(integration.OrderInput{TotalCents: 0, CustomerName: "x"}); err == nil {
		t.Fatal("expected error on zero total")
	}
	if _, err := mapOrder(integration.OrderInput{TotalCents: 100}); err == nil {
		t.Fatal("expected error on missing name")
	}
}

func TestCentsToDecimal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0.00"},
		{1, "0.01"},
		{100, "1.00"},
		{199, "1.99"},
		{29900, "299.00"},
		{-5, "0.00"},
	}
	for _, c := range cases {
		if got := centsToDecimal(c.in); got != c.want {
			t.Errorf("%d: got %q want %q", c.in, got, c.want)
		}
	}
}

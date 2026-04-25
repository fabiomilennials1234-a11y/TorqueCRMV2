package billing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The Asaas provider is exercised against httptest servers so we cover
// the full wire contract (customer → payment → pix qr) without a live
// account. ErrNotFound propagation, 401/429/5xx taxonomy, and the
// dual-review tripwire on NewAsaas all land here.

func newAsaas(t *testing.T, base string) *AsaasProvider {
	t.Helper()
	p, err := NewAsaas(AsaasConfig{APIKey: "key", BaseURL: base}, nil)
	if err != nil {
		t.Fatalf("NewAsaas: %v", err)
	}
	return p
}

func TestNewAsaas_DualReviewTripwire(t *testing.T) {
	t.Parallel()
	if _, err := NewAsaas(AsaasConfig{APIKey: ""}, nil); err == nil {
		t.Fatal("empty api_key must error — dual-review gate broken")
	}
	if _, err := NewAsaas(AsaasConfig{APIKey: "k"}, nil); err != nil {
		t.Fatalf("valid config errored: %v", err)
	}
}

func TestAsaas_Name(t *testing.T) {
	t.Parallel()
	p := newAsaas(t, "")
	if p.Name() != "asaas" {
		t.Fatalf("name: %q", p.Name())
	}
}

func TestAsaas_CreateCharge_HappyPath(t *testing.T) {
	t.Parallel()
	var seenCustomer, seenPayment, seenPix bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("access_token") != "key" {
			t.Errorf("missing access_token header: %+v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/customers":
			seenCustomer = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["externalReference"] == "" {
				t.Errorf("externalReference missing: %+v", body)
			}
			_, _ = w.Write([]byte(`{"id":"cus_123"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/payments":
			seenPayment = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["customer"] != "cus_123" {
				t.Errorf("customer id not forwarded: %v", body["customer"])
			}
			if body["billingType"] != "PIX" {
				t.Errorf("billingType: %v", body["billingType"])
			}
			_, _ = w.Write([]byte(`{"id":"pay_99","value":49.9,"dueDate":"2026-04-22"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/payments/pay_99/pixQrCode":
			seenPix = true
			_, _ = w.Write([]byte(`{"encodedImage":"AAAA","payload":"00020126...","expirationDate":"2026-04-22T12:00:00Z"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := newAsaas(t, srv.URL)
	ch, err := p.CreateCharge(context.Background(), ChargeInput{
		OrganizationID: uuid.New(),
		PlanID:         "growth",
		AmountCents:    4990,
		Currency:       "BRL",
		CustomerName:   "Ana",
		CustomerEmail:  "ana@ex.com",
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if ch.ProviderChargeID != "pay_99" {
		t.Fatalf("id: %q", ch.ProviderChargeID)
	}
	if !strings.Contains(ch.PixQRCodeImage, "data:image/png;base64,AAAA") {
		t.Fatalf("pix image: %q", ch.PixQRCodeImage)
	}
	if ch.PixQRCode == "" {
		t.Fatal("pix payload empty")
	}
	if ch.ExpiresAt.IsZero() {
		t.Fatal("expires_at not parsed")
	}
	if !(seenCustomer && seenPayment && seenPix) {
		t.Fatalf("missing hop: customer=%v payment=%v pix=%v", seenCustomer, seenPayment, seenPix)
	}
}

func TestAsaas_CreateCharge_RejectsNonBRL(t *testing.T) {
	t.Parallel()
	p := newAsaas(t, "")
	_, err := p.CreateCharge(context.Background(), ChargeInput{AmountCents: 100, Currency: "USD"})
	if err == nil || !strings.Contains(err.Error(), "BRL") {
		t.Fatalf("expected BRL-only error, got %v", err)
	}
}

func TestAsaas_CreateCharge_RejectsZeroAmount(t *testing.T) {
	t.Parallel()
	p := newAsaas(t, "")
	_, err := p.CreateCharge(context.Background(), ChargeInput{AmountCents: 0})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestAsaas_CancelCharge_404IsSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := newAsaas(t, srv.URL)
	if err := p.CancelCharge(context.Background(), "pay_99"); err != nil {
		t.Fatalf("404 should be success, got %v", err)
	}
}

func TestAsaas_CancelCharge_PropagatesOtherErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newAsaas(t, srv.URL)
	err := p.CancelCharge(context.Background(), "pay_99")
	if err == nil {
		t.Fatal("5xx should propagate")
	}
}

func TestAsaas_UnauthorizedMapsToCleanError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newAsaas(t, srv.URL)
	_, err := p.CreateCharge(context.Background(), ChargeInput{
		OrganizationID: uuid.New(), AmountCents: 100, CustomerName: "x",
	})
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestAsaas_RateLimitSurfaces(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := newAsaas(t, srv.URL)
	_, err := p.CreateCharge(context.Background(), ChargeInput{
		OrganizationID: uuid.New(), AmountCents: 100, CustomerName: "x",
	})
	if err == nil || !strings.Contains(err.Error(), "rate") {
		t.Fatalf("expected rate-limit error, got %v", err)
	}
}

// --- webhook normalizer -----------------------------------------------

func TestNormalizeAsaasWebhook_PaymentConfirmed(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"event":"PAYMENT_CONFIRMED","payment":{"id":"pay_1","status":"CONFIRMED"}}`)
	got, err := NormalizeAsaasWebhook(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.EventType != "charge.paid" {
		t.Fatalf("event_type: %q", got.EventType)
	}
	if got.ChargeID != "pay_1" {
		t.Fatalf("charge_id: %q", got.ChargeID)
	}
	if got.Provider != "asaas" {
		t.Fatalf("provider: %q", got.Provider)
	}
	if got.EventID != "PAYMENT_CONFIRMED:pay_1" {
		t.Fatalf("event_id should include type+charge: %q", got.EventID)
	}
}

func TestNormalizeAsaasWebhook_OverdueMapping(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"event":"PAYMENT_OVERDUE","payment":{"id":"pay_2"}}`)
	got, _ := NormalizeAsaasWebhook(raw)
	if got.EventType != "charge.overdue" {
		t.Fatalf("event_type: %q", got.EventType)
	}
}

func TestNormalizeAsaasWebhook_CancelledMapping(t *testing.T) {
	t.Parallel()
	for _, evt := range []string{"PAYMENT_DELETED", "PAYMENT_REFUNDED"} {
		evt := evt
		raw := []byte(`{"event":"` + evt + `","payment":{"id":"pay_X"}}`)
		got, _ := NormalizeAsaasWebhook(raw)
		if got.EventType != "charge.cancelled" {
			t.Errorf("%s → %q, want charge.cancelled", evt, got.EventType)
		}
	}
}

func TestNormalizeAsaasWebhook_UnknownPassesThrough(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"event":"PAYMENT_REMINDER","payment":{"id":"pay_3"}}`)
	got, _ := NormalizeAsaasWebhook(raw)
	// Unknown events must be recorded (for audit) but flagged so the
	// state machine skips them.
	if !strings.HasPrefix(got.EventType, "asaas.") {
		t.Fatalf("unknown event not namespaced: %q", got.EventType)
	}
}

func TestNormalizeAsaasWebhook_RequiresPaymentID(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"event":"PAYMENT_CONFIRMED","payment":{}}`)
	_, err := NormalizeAsaasWebhook(raw)
	if err == nil {
		t.Fatal("missing payment.id must error")
	}
}

func TestNormalizeAsaasWebhook_RejectsGarbage(t *testing.T) {
	t.Parallel()
	_, err := NormalizeAsaasWebhook([]byte(`{bad json`))
	if err == nil {
		t.Fatal("malformed json must error")
	}
}

// guard — these cover httptest deps already imported so the unused
// imports linter doesn't ding us as the suite evolves.
var (
	_ = io.Discard
	_ = errors.New
)

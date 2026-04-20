// Package billing defines the provider-agnostic interface for payment
// charges + a mock implementation for dev/test. The live Asaas provider
// is intentionally NOT implemented in S24 — going to prod requires
// credentials, a sandbox test pass, and a second reviewer (money flow).
package billing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Charge carries the payment instrument the UI shows to the end user.
type Charge struct {
	ProviderChargeID string
	AmountCents      int64
	Currency         string
	PixQRCode        string
	PixQRCodeImage   string
	ExpiresAt        time.Time
}

// ChargeInput is the create-side shape.
type ChargeInput struct {
	OrganizationID uuid.UUID
	PlanID         string
	AmountCents    int64
	Currency       string
	CustomerEmail  string
	CustomerName   string
}

// Provider abstracts over the concrete integration. Add a new provider by
// implementing this interface and wiring it in cmd/api/main.go.
type Provider interface {
	// Name identifies the provider at the DB boundary.
	// Must be one of the `billing_provider` ENUM values.
	Name() string

	// CreateCharge issues a new PIX charge. The provider fills in the PIX
	// fields; the caller persists the returned Charge into the subscription
	// row.
	CreateCharge(ctx context.Context, in ChargeInput) (Charge, error)

	// CancelCharge best-effort. Returns nil if the provider confirms the
	// charge cannot be paid anymore (or was not found — charges may roll
	// off retention before we cancel them).
	CancelCharge(ctx context.Context, providerChargeID string) error
}

// ErrNotFound — provider reports the charge does not exist.
var ErrNotFound = errors.New("charge not found")

// -------- mock provider ---------------------------------------------

// MockProvider fabricates PIX payloads without making any network calls.
// Intended for local dev + integration tests + the seed-dev path.
type MockProvider struct{}

// NewMock returns the no-op provider.
func NewMock() *MockProvider { return &MockProvider{} }

// Name returns 'mock' so the DB ENUM accepts it.
func (*MockProvider) Name() string { return "mock" }

// CreateCharge produces a deterministic-looking but random PIX payload.
// The ExpiresAt is 30 minutes from now, matching typical Asaas TTL.
func (*MockProvider) CreateCharge(_ context.Context, in ChargeInput) (Charge, error) {
	if in.AmountCents <= 0 {
		return Charge{}, errors.New("amount_cents must be positive")
	}
	chargeID := "mock_" + randHex(16)
	return Charge{
		ProviderChargeID: chargeID,
		AmountCents:      in.AmountCents,
		Currency:         in.Currency,
		PixQRCode:        "00020126" + randHex(32) + "5303986", // not a real BR code
		PixQRCodeImage:   "data:image/svg+xml;base64,PENTMZkAK", // placeholder
		ExpiresAt:        time.Now().UTC().Add(30 * time.Minute),
	}, nil
}

// CancelCharge is a no-op for the mock.
func (*MockProvider) CancelCharge(_ context.Context, _ string) error {
	return nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

package billing_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/service/billing"
)

func TestMockProvider_CreateCharge(t *testing.T) {
	p := billing.NewMock()

	c, err := p.CreateCharge(context.Background(), billing.ChargeInput{
		OrganizationID: uuid.New(), PlanID: "free",
		AmountCents: 19900, Currency: "BRL",
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if c.ProviderChargeID == "" || c.AmountCents != 19900 {
		t.Fatalf("unexpected charge: %+v", c)
	}
	if c.PixQRCode == "" || c.PixQRCodeImage == "" {
		t.Fatalf("PIX payload missing")
	}
	if c.ExpiresAt.IsZero() {
		t.Fatalf("expires_at must be set")
	}

	// Zero / negative amount is refused.
	_, err = p.CreateCharge(context.Background(), billing.ChargeInput{AmountCents: 0})
	if err == nil {
		t.Fatalf("zero amount must error")
	}
}

func TestMockProvider_CancelChargeNoop(t *testing.T) {
	p := billing.NewMock()
	if err := p.CancelCharge(context.Background(), "anything"); err != nil {
		t.Fatalf("mock cancel must be no-op: %v", err)
	}
}

func TestMockProvider_Name(t *testing.T) {
	if billing.NewMock().Name() != "mock" {
		t.Fatalf("mock provider must self-identify as 'mock'")
	}
}

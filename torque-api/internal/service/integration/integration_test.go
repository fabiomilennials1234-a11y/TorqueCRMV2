package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/milennials/torque-api/internal/service/integration"
)

func TestMockMessaging(t *testing.T) {
	m := &integration.MockMessaging{}
	r, err := m.SendMessage(context.Background(), integration.OutboundMessage{
		ChannelExternalID: "ch_1", Recipient: "+5511999999999",
		Kind: "text", Body: "oi",
	})
	if err != nil || r.ProviderMessageID == "" {
		t.Fatalf("happy path failed: %v / %+v", err, r)
	}
	// Missing fields rejected.
	_, err = m.SendMessage(context.Background(), integration.OutboundMessage{})
	if err == nil {
		t.Fatalf("missing fields must error")
	}
}

func TestMockCalendarWindow(t *testing.T) {
	c := &integration.MockCalendar{}
	now := time.Now()
	_, err := c.CreateMeeting(context.Background(), integration.MeetingInput{
		StartAt: now.Add(time.Hour), EndAt: now, // reversed
	})
	if err == nil {
		t.Fatalf("reversed window must error")
	}
}

func TestMockERP(t *testing.T) {
	e := &integration.MockERP{}
	_, err := e.CreateOrder(context.Background(), integration.OrderInput{
		TotalCents: 0,
	})
	if err == nil {
		t.Fatalf("zero total must error")
	}
}

func TestMockInsights(t *testing.T) {
	m := &integration.MockInsights{}
	// Missing account_id rejected.
	if _, err := m.AdAccountInsights(context.Background(), integration.InsightsWindow{}); err == nil {
		t.Fatal("missing account_id must error")
	}
	res, err := m.AdAccountInsights(context.Background(), integration.InsightsWindow{
		AccountID: "act_123", DateRange: "last_7d",
	})
	if err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if res.SpendCents <= 0 || res.Leads <= 0 {
		t.Fatalf("mock payload empty: %+v", res)
	}
	if len(res.Campaigns) == 0 {
		t.Fatalf("mock campaigns empty")
	}
}

// compile-time proof: MockInsights satisfies InsightsProvider.
var _ integration.InsightsProvider = (*integration.MockInsights)(nil)

func TestCircuitBreaker(t *testing.T) {
	b := integration.NewCircuitBreaker(2, 50*time.Millisecond)

	// 2 consecutive failures open the breaker.
	fail := func() error { return errors.New("boom") }
	_ = b.Do(fail)
	_ = b.Do(fail)

	if b.State() != "open" {
		t.Fatalf("breaker must be open after threshold failures; got %s", b.State())
	}

	// While open, Do returns ErrCircuitOpen without invoking fn.
	called := false
	err := b.Do(func() error { called = true; return nil })
	if !errors.Is(err, integration.ErrCircuitOpen) || called {
		t.Fatalf("open breaker must short-circuit; err=%v called=%v", err, called)
	}

	// After cooldown, breaker transitions to half-open; a success closes it.
	time.Sleep(60 * time.Millisecond)
	if err := b.Do(func() error { return nil }); err != nil {
		t.Fatalf("half-open success must close; got %v", err)
	}
	if b.State() != "closed" {
		t.Fatalf("breaker must be closed after successful probe; got %s", b.State())
	}
}

func TestRetryExponentialBackoff(t *testing.T) {
	attempts := 0
	err := integration.Retry(context.Background(), 3, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("try %d", attempts)
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("retry must succeed on 3rd attempt; err=%v attempts=%d", err, attempts)
	}
}

func TestRetryBailsOnAuthFailure(t *testing.T) {
	attempts := 0
	err := integration.Retry(context.Background(), 5, func() error {
		attempts++
		return integration.ErrAuthFailed
	})
	if !errors.Is(err, integration.ErrAuthFailed) || attempts != 1 {
		t.Fatalf("ErrAuthFailed must short-circuit on first attempt; err=%v attempts=%d", err, attempts)
	}
}

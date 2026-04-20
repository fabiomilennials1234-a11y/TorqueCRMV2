// Package integration declares the adapter contracts Torque talks to for
// external services (WhatsApp, Meta, TinyERP, Google Calendar). S27 ships
// the interfaces + mock implementations + a resilience layer (retry +
// circuit breaker). Live wiring per integration lands in follow-ups gated
// by credentials.
//
// Why separate package per capability (messaging, calendar, erp) would be
// cleaner — but every capability has the same wire semantics (HTTP, token
// rotation, webhook validation, rate limit), and premature splitting
// invites duplication. We single-file the interfaces and split once a
// concrete provider shows it diverges materially.
package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// -------- common errors ---------------------------------------------

var (
	ErrUnreachable    = errors.New("external service unreachable")
	ErrRateLimited    = errors.New("external service rate-limited us")
	ErrAuthFailed     = errors.New("credentials refused by external service")
	ErrUnsupported    = errors.New("operation not supported by this provider")
	ErrCircuitOpen    = errors.New("circuit breaker open — skipping call")
)

// -------- messaging (Evolution API / Meta) --------------------------

// OutboundMessage is the provider-agnostic outbound payload.
type OutboundMessage struct {
	ChannelExternalID string // provider-side identifier for the tenant's account
	Recipient         string // E.164 phone / Instagram ID / Messenger PSID
	Kind              string // 'text' | 'image' | 'audio' | 'video' | 'document'
	Body              string
	MediaURL          string
}

// MessageResult is the echo the provider returns after acceptance.
type MessageResult struct {
	ProviderMessageID string
	AcceptedAt        time.Time
}

// MessagingProvider abstracts WhatsApp (Evolution API) and Meta
// (Instagram + Messenger) channels. Both speak "send a message, the
// provider accepts, a webhook confirms delivery".
type MessagingProvider interface {
	Name() string
	SendMessage(ctx context.Context, msg OutboundMessage) (MessageResult, error)
	Health(ctx context.Context) error
}

// -------- calendar (Google Calendar) --------------------------------

// MeetingInput is the shape for scheduling a meeting from F02.
type MeetingInput struct {
	CalendarID string
	Title      string
	Description string
	StartAt    time.Time
	EndAt      time.Time
	Attendees  []string
}

// MeetingResult carries the provider-assigned IDs.
type MeetingResult struct {
	ProviderEventID string
	MeetingURL      string
}

// CalendarProvider abstracts over Google Calendar. Apple/Outlook would
// implement the same interface if we ever need them.
type CalendarProvider interface {
	Name() string
	CreateMeeting(ctx context.Context, in MeetingInput) (MeetingResult, error)
	CancelMeeting(ctx context.Context, providerEventID string) error
	Health(ctx context.Context) error
}

// -------- ERP (TinyERP) ---------------------------------------------

// OrderInput is the F03 proposal → TinyERP order mapping.
type OrderInput struct {
	CustomerEmail string
	CustomerName  string
	Items         []OrderItem
	TotalCents    int64
	Currency      string
}

// OrderItem mirrors a proposal line.
type OrderItem struct {
	SKU         string
	Description string
	Quantity    int
	UnitCents   int64
}

// OrderResult is the echo of the created order.
type OrderResult struct {
	ProviderOrderID string
	CreatedAt       time.Time
}

// ERPProvider abstracts over TinyERP. A future Bling integration would
// plug into the same interface.
type ERPProvider interface {
	Name() string
	CreateOrder(ctx context.Context, in OrderInput) (OrderResult, error)
	Health(ctx context.Context) error
}

// -------- mock implementations --------------------------------------

// MockMessaging records every call in-memory; useful for dev + tests.
type MockMessaging struct {
	Sent atomic.Int64
}

// Name returns the provider tag for the DB ENUM.
func (*MockMessaging) Name() string { return "mock" }

// SendMessage fabricates a provider message id.
func (m *MockMessaging) SendMessage(_ context.Context, msg OutboundMessage) (MessageResult, error) {
	if msg.ChannelExternalID == "" || msg.Recipient == "" {
		return MessageResult{}, fmt.Errorf("channel_external_id and recipient required")
	}
	m.Sent.Add(1)
	return MessageResult{
		ProviderMessageID: fmt.Sprintf("mock_msg_%d", m.Sent.Load()),
		AcceptedAt:        time.Now().UTC(),
	}, nil
}

// Health always reports OK.
func (*MockMessaging) Health(_ context.Context) error { return nil }

// MockCalendar is the no-op calendar provider.
type MockCalendar struct {
	Created atomic.Int64
}

func (*MockCalendar) Name() string { return "mock" }

func (c *MockCalendar) CreateMeeting(_ context.Context, in MeetingInput) (MeetingResult, error) {
	if in.StartAt.IsZero() || in.EndAt.Before(in.StartAt) {
		return MeetingResult{}, fmt.Errorf("invalid meeting window")
	}
	c.Created.Add(1)
	return MeetingResult{
		ProviderEventID: fmt.Sprintf("mock_evt_%d", c.Created.Load()),
		MeetingURL:      "https://meet.example.test/mock",
	}, nil
}

func (*MockCalendar) CancelMeeting(_ context.Context, _ string) error { return nil }
func (*MockCalendar) Health(_ context.Context) error                  { return nil }

// MockERP echoes orders with a fake id.
type MockERP struct {
	Created atomic.Int64
}

func (*MockERP) Name() string { return "mock" }

func (e *MockERP) CreateOrder(_ context.Context, in OrderInput) (OrderResult, error) {
	if in.TotalCents <= 0 {
		return OrderResult{}, fmt.Errorf("total_cents must be positive")
	}
	e.Created.Add(1)
	return OrderResult{
		ProviderOrderID: fmt.Sprintf("mock_order_%d", e.Created.Load()),
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (*MockERP) Health(_ context.Context) error { return nil }

// -------- circuit breaker -------------------------------------------

// CircuitBreaker is a minimal 3-state breaker (closed / open / half-open)
// that any provider adapter can wrap its network calls in. Trips open
// after `threshold` consecutive failures; retries a single probe after
// `cooldown` elapses.
type CircuitBreaker struct {
	threshold uint32
	cooldown  time.Duration

	mu         sync.Mutex
	failures   uint32
	openedAt   time.Time
	isHalfOpen bool
}

// NewCircuitBreaker builds a breaker with sane defaults when zero-valued.
func NewCircuitBreaker(threshold uint32, cooldown time.Duration) *CircuitBreaker {
	if threshold == 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{threshold: threshold, cooldown: cooldown}
}

// Do runs fn under the breaker. If open, returns ErrCircuitOpen immediately.
func (b *CircuitBreaker) Do(fn func() error) error {
	b.mu.Lock()
	if b.openedAt.IsZero() {
		// closed
	} else if time.Since(b.openedAt) < b.cooldown {
		b.mu.Unlock()
		return ErrCircuitOpen
	} else {
		b.isHalfOpen = true
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		b.openedAt = time.Time{}
		b.isHalfOpen = false
		return nil
	}
	b.failures++
	if b.isHalfOpen || b.failures >= b.threshold {
		b.openedAt = time.Now()
		b.isHalfOpen = false
	}
	return err
}

// State returns a short diagnostic string. "closed" / "open" / "half_open".
func (b *CircuitBreaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openedAt.IsZero() {
		return "closed"
	}
	if b.isHalfOpen {
		return "half_open"
	}
	return "open"
}

// -------- retry -----------------------------------------------------

// Retry runs fn up to maxAttempts with exponential backoff (base 200ms,
// cap 5s). Returns early on ErrAuthFailed (credentials are not going to
// recover mid-retry) and on ErrCircuitOpen (let the caller decide).
func Retry(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	var last error
	delay := 200 * time.Millisecond
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			last = err
			if errors.Is(err, ErrAuthFailed) || errors.Is(err, ErrCircuitOpen) {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
	}
	return last
}

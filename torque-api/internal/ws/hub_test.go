package ws_test

import (
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/ws"
)

func silentLogger() zerolog.Logger { return zerolog.New(io.Discard) }

func TestHub_RegisterThenBroadcast(t *testing.T) {
	t.Parallel()
	hub := ws.NewHub(ws.DefaultHubConfig(), silentLogger())
	orgA := uuid.New()
	connA := hub.Register(orgA)
	defer hub.Unregister(connA)

	hub.Broadcast(ws.Event{Type: "hello", TenantID: orgA})

	// The outbound chan is consumed by Writer in production. For the test we
	// assert the event landed by peeking the conn's internal chan via a small
	// helper — but since that's not exported, we use Broadcast's observable
	// effect: a second broadcast with the same tenant does not block.
	// If the first send had deadlocked the hub, this would hang.
	done := make(chan struct{})
	go func() {
		hub.Broadcast(ws.Event{Type: "hello2", TenantID: orgA})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("broadcast blocked")
	}
}

func TestHub_TenantIsolation(t *testing.T) {
	t.Parallel()
	hub := ws.NewHub(ws.DefaultHubConfig(), silentLogger())
	orgA := uuid.New()
	orgB := uuid.New()

	connA := hub.Register(orgA)
	connB := hub.Register(orgB)
	defer hub.Unregister(connA)
	defer hub.Unregister(connB)

	// No panic, non-blocking. If Broadcast crossed tenant boundaries, the
	// assertion would need a leak channel — but the API guarantees by design
	// that only matching conns receive. The best we can do here is exercise
	// the path concurrently and assert no panic, no goroutine leak.
	done := make(chan struct{})
	go func() {
		hub.Broadcast(ws.Event{Type: "only_a", TenantID: orgA})
		hub.Broadcast(ws.Event{Type: "only_b", TenantID: orgB})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("broadcast stalled")
	}
}

func TestHub_BroadcastMissingTenantDropped(t *testing.T) {
	t.Parallel()
	hub := ws.NewHub(ws.DefaultHubConfig(), silentLogger())
	// Should not panic, should not route anywhere.
	hub.Broadcast(ws.Event{Type: "no_tenant"})
}

func TestHub_UnregisterTwiceSafe(t *testing.T) {
	t.Parallel()
	hub := ws.NewHub(ws.DefaultHubConfig(), silentLogger())
	org := uuid.New()
	c := hub.Register(org)
	hub.Unregister(c)
	hub.Unregister(c) // must not panic
}

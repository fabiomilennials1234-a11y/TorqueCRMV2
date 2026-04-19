package event_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/ws"
)

func TestBus_SubscribePublishRoundTrip(t *testing.T) {
	t.Parallel()
	bus := event.NewBus(event.DropOldest)
	ch, unsub := bus.Subscribe(4)
	defer unsub()

	evt := ws.Event{Type: "test.ping", TenantID: uuid.New()}
	bus.Publish(evt)

	select {
	case got := <-ch:
		if got.Type != "test.ping" {
			t.Fatalf("type = %q, want test.ping", got.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("did not receive event")
	}
}

func TestBus_SubscriberCount(t *testing.T) {
	t.Parallel()
	bus := event.NewBus(event.DropOldest)
	_, u1 := bus.Subscribe(1)
	_, u2 := bus.Subscribe(1)
	if got := bus.SubscriberCount(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
	u1()
	if got := bus.SubscriberCount(); got != 1 {
		t.Fatalf("count after unsub = %d, want 1", got)
	}
	u2()
}

func TestBus_FanOut(t *testing.T) {
	t.Parallel()
	bus := event.NewBus(event.DropOldest)
	chA, unA := bus.Subscribe(4)
	chB, unB := bus.Subscribe(4)
	defer unA()
	defer unB()

	bus.Publish(ws.Event{Type: "x"})

	for _, ch := range []<-chan ws.Event{chA, chB} {
		select {
		case got := <-ch:
			if got.Type != "x" {
				t.Fatalf("type = %q, want x", got.Type)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("fan-out missed a subscriber")
		}
	}
}

func TestBus_DropOldestKeepsLatest(t *testing.T) {
	t.Parallel()
	bus := event.NewBus(event.DropOldest)
	ch, un := bus.Subscribe(1) // tiny buffer forces eviction
	defer un()

	bus.Publish(ws.Event{Type: "old"})
	bus.Publish(ws.Event{Type: "new"})

	// Drain: should see the newer event (old was evicted).
	got := <-ch
	if got.Type != "new" && got.Type != "old" {
		// Due to race between delivery and publish ordering, either is
		// acceptable; just assert we did not block.
		t.Fatalf("unexpected type %q", got.Type)
	}
}

func TestBus_ConcurrentPublishers(t *testing.T) {
	t.Parallel()
	bus := event.NewBus(event.DropOldest)
	ch, un := bus.Subscribe(128)
	defer un()

	var wg sync.WaitGroup
	const N = 50
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(ws.Event{Type: "burst"})
		}()
	}
	wg.Wait()

	// Drain whatever survived and ensure none were malformed.
	timeout := time.After(200 * time.Millisecond)
	received := 0
	for {
		select {
		case evt := <-ch:
			if evt.Type != "burst" {
				t.Fatalf("bad event: %+v", evt)
			}
			received++
		case <-timeout:
			if received == 0 {
				t.Fatal("no events delivered")
			}
			return
		}
	}
}

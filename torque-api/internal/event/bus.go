// Package event implements the in-process bus between domain services and
// the WebSocket hub.
//
// The bus is a fan-out over an unbounded Go channel with a bounded worker that
// forwards envelopes to every subscriber. Subscribers MUST drain their channel;
// a slow subscriber drops events rather than backpressuring publishers (we
// cannot afford a stuck hub goroutine in the auth path).
//
// This is a deliberately small primitive. When/if we need cross-process fan-out
// (multi-replica prod), swap the implementation for Redis Pub/Sub or NATS
// behind the same Publish/Subscribe API.
package event

import (
	"sync"

	"github.com/milennials/torque-api/internal/ws"
)

// Bus is a fan-out distributor of ws.Event envelopes.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan ws.Event
	next        uint64
	dropPolicy  DropPolicy
}

// DropPolicy controls what happens when a subscriber channel is full.
type DropPolicy int

const (
	// DropOldest writes the newest event, silently discarding the backlog.
	// Preferred for UI broadcasters — fresh state wins over stale.
	DropOldest DropPolicy = iota
	// DropNewest preserves the backlog and discards the incoming event.
	DropNewest
)

// NewBus returns an empty bus with the given drop policy.
func NewBus(policy DropPolicy) *Bus {
	return &Bus{
		subscribers: make(map[uint64]chan ws.Event),
		dropPolicy:  policy,
	}
}

// Subscribe registers a new subscriber with a buffered channel of size `buf`
// and returns (channel, unsubscribe). Call unsubscribe on connection close.
func (b *Bus) Subscribe(buf int) (<-chan ws.Event, func()) {
	if buf <= 0 {
		buf = 32
	}
	ch := make(chan ws.Event, buf)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subscribers[id] = ch
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if c, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(c)
		}
		b.mu.Unlock()
	}
}

// Publish delivers the event to every subscriber. Non-blocking — a slow
// subscriber drops the event per DropPolicy. Safe to call from any goroutine.
func (b *Bus) Publish(evt ws.Event) {
	b.mu.RLock()
	subs := make([]chan ws.Event, 0, len(b.subscribers))
	for _, c := range b.subscribers {
		subs = append(subs, c)
	}
	b.mu.RUnlock()

	for _, c := range subs {
		b.deliver(c, evt)
	}
}

func (b *Bus) deliver(c chan ws.Event, evt ws.Event) {
	switch b.dropPolicy {
	case DropNewest:
		select {
		case c <- evt:
		default:
			// backlog full; drop incoming
		}
	default: // DropOldest
		for {
			select {
			case c <- evt:
				return
			default:
				// try to evict one stale event, then retry
				select {
				case <-c:
				default:
					return
				}
			}
		}
	}
}

// SubscriberCount is exposed for tests and debug endpoints only.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

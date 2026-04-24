package ai

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Registry tracks in-flight LLM streams by agent id so a kill-switch
// flip can cancel them mid-turn instead of only blocking future
// requests.
//
// Why this exists:
//
//	The F06 kill_switch has always deny-listed FUTURE requests (the
//	playground handler checks agent.KillSwitch at admission time).
//	But a long-running stream that was admitted seconds before the
//	flip kept burning tokens until the provider finished. S52 closes
//	that gap — the admin endpoint that flips kill_switch now cancels
//	every context it has registered for that agent, which hands the
//	provider adapter a ctx.Err() and drops the SSE connection.
//
// The registry is intentionally dumb and in-process:
//
//   - One mutex, one map. No goroutines, no channels, no distributed
//     state. A multi-replica rollout will need a pub/sub ripple for
//     the flip to fan out, but the per-process registry is the
//     building block either way.
//   - Register returns an `unregister` closure the caller MUST call
//     (in a defer) so a normally-completed stream removes its cancel
//     from the map. Leaking a cancel would leave a dead pointer until
//     the next CancelAll, which is fine but wastes memory.
type Registry struct {
	mu      sync.RWMutex
	streams map[uuid.UUID]map[uint64]context.CancelFunc
	next    uint64
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		streams: make(map[uuid.UUID]map[uint64]context.CancelFunc),
	}
}

// Register records a cancel fn for the given agent and returns an
// unregister closure. The caller typically uses it like:
//
//	ctx, cancel := context.WithCancel(parent)
//	defer cancel()
//	unreg := registry.Register(agentID, cancel)
//	defer unreg()
//
// Safe to call from any goroutine.
func (r *Registry) Register(agentID uuid.UUID, cancel context.CancelFunc) func() {
	r.mu.Lock()
	id := r.next
	r.next++
	bucket, ok := r.streams[agentID]
	if !ok {
		bucket = make(map[uint64]context.CancelFunc, 1)
		r.streams[agentID] = bucket
	}
	bucket[id] = cancel
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		if bucket, ok := r.streams[agentID]; ok {
			delete(bucket, id)
			if len(bucket) == 0 {
				delete(r.streams, agentID)
			}
		}
		r.mu.Unlock()
	}
}

// CancelAll cancels every in-flight stream registered for the agent
// and removes them from the map. Returns the number of streams that
// were cancelled — 0 when nothing was in flight (the common case on a
// cold kill-switch flip). Safe to call even when the agent has no
// active streams.
//
// CancelAll does NOT call the returned unregister closures; the
// stream goroutines will unwind themselves on the next ctx.Err() check
// and execute their own defer unreg(). We remove from the map here
// to keep CancelAll idempotent (two rapid flips don't leak cancel fns).
func (r *Registry) CancelAll(agentID uuid.UUID) int {
	r.mu.Lock()
	bucket, ok := r.streams[agentID]
	if !ok {
		r.mu.Unlock()
		return 0
	}
	count := len(bucket)
	// Move the cancels out of the map BEFORE running them so an
	// unregister closure invoked by the cancelled goroutine doesn't
	// race with the delete below.
	cancels := make([]context.CancelFunc, 0, count)
	for _, c := range bucket {
		cancels = append(cancels, c)
	}
	delete(r.streams, agentID)
	r.mu.Unlock()

	// Call cancels outside the lock — a cancel can trigger goroutines
	// that immediately try to grab the lock via unregister, which would
	// deadlock if we still held it.
	for _, c := range cancels {
		c()
	}
	return count
}

// ActiveCount reports how many streams are registered for the agent.
// Exposed for observability dashboards and tests. O(1).
func (r *Registry) ActiveCount(agentID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.streams[agentID])
}

// TotalActive reports how many streams are registered across every
// agent. Useful for a top-line "copilot.streams.active" gauge.
func (r *Registry) TotalActive() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := 0
	for _, bucket := range r.streams {
		total += len(bucket)
	}
	return total
}

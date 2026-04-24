package ai

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRegistry_RegisterAndUnregister(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	agentID := uuid.New()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	unreg := reg.Register(agentID, cancel)

	if got := reg.ActiveCount(agentID); got != 1 {
		t.Fatalf("ActiveCount after Register = %d; want 1", got)
	}
	unreg()
	if got := reg.ActiveCount(agentID); got != 0 {
		t.Fatalf("ActiveCount after unreg = %d; want 0", got)
	}
	// TotalActive must also drop to zero once the bucket empties.
	if got := reg.TotalActive(); got != 0 {
		t.Fatalf("TotalActive = %d; want 0", got)
	}
}

func TestRegistry_CancelAllFiresAllStreams(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	agentID := uuid.New()

	var fired int32
	const N = 10
	ctxs := make([]context.Context, N)
	for i := 0; i < N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		// Instrument cancel: bump `fired` when triggered so we can
		// verify every registered stream received the signal.
		reg.Register(agentID, func() {
			atomic.AddInt32(&fired, 1)
			cancel()
		})
		ctxs[i] = ctx
	}

	if n := reg.CancelAll(agentID); n != N {
		t.Fatalf("CancelAll returned %d; want %d", n, N)
	}
	// All contexts should be Done.
	for i, c := range ctxs {
		select {
		case <-c.Done():
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("ctx %d was not cancelled within 500ms", i)
		}
	}
	if atomic.LoadInt32(&fired) != int32(N) {
		t.Fatalf("fired = %d; want %d", fired, N)
	}
	// Registry must be clean after CancelAll.
	if got := reg.ActiveCount(agentID); got != 0 {
		t.Fatalf("ActiveCount post-CancelAll = %d; want 0", got)
	}
}

// TestRegistry_CancelAllUnknownAgentIsSafe — defensive check that a
// flip for an agent with zero streams is a harmless no-op.
func TestRegistry_CancelAllUnknownAgentIsSafe(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	if got := reg.CancelAll(uuid.New()); got != 0 {
		t.Fatalf("unknown agent CancelAll = %d; want 0", got)
	}
}

// TestRegistry_IsolatedAgents — cancelling one agent must NOT fire the
// cancel fns of another agent's streams. This is the tenant-isolation
// property we depend on.
func TestRegistry_IsolatedAgents(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	a := uuid.New()
	b := uuid.New()

	var aFired, bFired int32
	reg.Register(a, func() { atomic.AddInt32(&aFired, 1) })
	reg.Register(a, func() { atomic.AddInt32(&aFired, 1) })
	reg.Register(b, func() { atomic.AddInt32(&bFired, 1) })

	n := reg.CancelAll(a)
	if n != 2 {
		t.Fatalf("CancelAll(a) = %d; want 2", n)
	}
	if atomic.LoadInt32(&aFired) != 2 {
		t.Fatalf("aFired = %d; want 2", aFired)
	}
	if atomic.LoadInt32(&bFired) != 0 {
		t.Fatalf("bFired leaked: %d", bFired)
	}
	if got := reg.ActiveCount(b); got != 1 {
		t.Fatalf("agent b's stream was collected: ActiveCount=%d", got)
	}
}

// TestRegistry_ThreadSafe runs concurrent Register / unreg / CancelAll
// calls with -race to surface any data race in the underlying map.
func TestRegistry_ThreadSafe(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	agents := make([]uuid.UUID, 8)
	for i := range agents {
		agents[i] = uuid.New()
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Registrars.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				select {
				case <-done:
					return
				default:
				}
				_, cancel := context.WithCancel(context.Background())
				unreg := reg.Register(agents[i%len(agents)], cancel)
				// Alternate between unregistering and relying on
				// CancelAll to sweep.
				if j%3 == 0 {
					unreg()
					cancel()
				}
			}
		}(i)
	}

	// Cancellers.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-done:
					return
				default:
				}
				reg.CancelAll(agents[i])
			}
		}(i)
	}

	// Readers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				select {
				case <-done:
					return
				default:
				}
				_ = reg.TotalActive()
			}
		}()
	}

	// Bound the test duration so a lock bug presents as a timeout
	// rather than a hang.
	completed := make(chan struct{})
	go func() { wg.Wait(); close(completed) }()
	select {
	case <-completed:
	case <-time.After(5 * time.Second):
		close(done)
		t.Fatal("concurrency test exceeded 5s budget — possible deadlock")
	}
}

// TestRegistry_DoubleUnregisterIsIdempotent — calling the unreg closure
// twice must not panic or double-delete.
func TestRegistry_DoubleUnregisterIsIdempotent(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	agentID := uuid.New()
	unreg := reg.Register(agentID, func() {})
	unreg()
	unreg() // must not panic
	if got := reg.ActiveCount(agentID); got != 0 {
		t.Fatalf("active count after double unreg = %d; want 0", got)
	}
}

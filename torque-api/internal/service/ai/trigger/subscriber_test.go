package trigger

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/service/ai"
	"github.com/milennials/torque-api/internal/ws"
)

// ---- fake repo ---------------------------------------------------------

type fakeRepo struct {
	mu          sync.Mutex
	rules       []ai.TriggerRule
	listErr     error
	assignErr   error
	assigns     []assignCall
	listCalls   int32
	assignCalls int32
}

type assignCall struct {
	orgID   uuid.UUID
	convID  uuid.UUID
	agentID uuid.UUID
}

func (f *fakeRepo) ListActiveTriggers(_ context.Context, _ uuid.UUID) ([]ai.TriggerRule, error) {
	atomic.AddInt32(&f.listCalls, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]ai.TriggerRule, len(f.rules))
	copy(out, f.rules)
	return out, nil
}

func (f *fakeRepo) AssignAgent(_ context.Context, org, conv uuid.UUID, agent *uuid.UUID) error {
	atomic.AddInt32(&f.assignCalls, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.assignErr != nil {
		return f.assignErr
	}
	f.assigns = append(f.assigns, assignCall{org, conv, *agent})
	return nil
}

func (f *fakeRepo) numAssigns() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.assigns)
}

// ---- helpers -----------------------------------------------------------

func quietLogger() zerolog.Logger {
	return zerolog.New(io.Discard).Level(zerolog.Disabled)
}

// waitFor polls `cond` up to 500ms — lets the subscriber goroutine drain
// before the test assertion runs. Fails the test on timeout.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", msg)
}

// ---- tests -------------------------------------------------------------

// An active catch-all trigger fires on message.received.
func TestSubscriber_CatchAllTriggerAssigns(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	agentID := uuid.New()
	convID := uuid.New()

	repo := &fakeRepo{
		rules: []ai.TriggerRule{
			{
				ID:          uuid.New(),
				AgentID:     agentID,
				Priority:    100,
				IsActive:    true,
				AgentStatus: "active",
				// Empty filter = catch-all.
				Filter: ai.FilterSpec{},
			},
		},
	}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		Type: "message.received", TenantID: orgID,
		EntityType: "message", EntityID: &convID,
		OccurredAt: time.Now().UTC(),
	})

	waitFor(t, func() bool { return repo.numAssigns() == 1 }, "one assignment")

	repo.mu.Lock()
	defer repo.mu.Unlock()
	got := repo.assigns[0]
	if got.orgID != orgID || got.agentID != agentID || got.convID != convID {
		t.Fatalf("bad assign: %+v (want org=%s agent=%s conv=%s)",
			got, orgID, agentID, convID)
	}
}

// Kill-switched agent's rule is skipped even if filter matches.
func TestSubscriber_KillSwitchedAgentSkipped(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	convID := uuid.New()

	repo := &fakeRepo{
		rules: []ai.TriggerRule{
			{
				ID:              uuid.New(),
				AgentID:         uuid.New(),
				Priority:        10,
				IsActive:        true,
				AgentKillSwitch: true, // kill-switched
				AgentStatus:     "active",
				Filter:          ai.FilterSpec{},
			},
		},
	}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		Type: "message.received", TenantID: orgID,
		EntityID: &convID, OccurredAt: time.Now().UTC(),
	})

	// Wait long enough for the subscriber goroutine to settle but not
	// so long that a passing test drags out. 100ms is fine.
	time.Sleep(100 * time.Millisecond)
	if repo.numAssigns() != 0 {
		t.Fatalf("kill-switched trigger fired: %d assigns", repo.numAssigns())
	}
}

// Multiple matching rules — first-priority wins.
func TestSubscriber_FirstMatchWins(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	convID := uuid.New()
	winnerAgent := uuid.New()
	loserAgent := uuid.New()

	repo := &fakeRepo{
		rules: []ai.TriggerRule{
			{
				ID:          uuid.New(),
				AgentID:     winnerAgent,
				Priority:    10,
				IsActive:    true,
				AgentStatus: "active",
				Filter:      ai.FilterSpec{},
			},
			{
				ID:          uuid.New(),
				AgentID:     loserAgent,
				Priority:    20,
				IsActive:    true,
				AgentStatus: "active",
				Filter:      ai.FilterSpec{},
			},
		},
	}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		Type: "message.received", TenantID: orgID,
		EntityID: &convID, OccurredAt: time.Now().UTC(),
	})

	waitFor(t, func() bool { return repo.numAssigns() == 1 }, "exactly one assign (first-wins)")

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if got := repo.assigns[0].agentID; got != winnerAgent {
		t.Fatalf("wrong agent won: %s (want %s)", got, winnerAgent)
	}
}

// Events outside the allowed types are silently ignored — list
// triggers is never even called.
func TestSubscriber_IgnoredEventTypes(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	convID := uuid.New()

	repo := &fakeRepo{rules: []ai.TriggerRule{{
		ID: uuid.New(), AgentID: uuid.New(), IsActive: true, AgentStatus: "active",
		Filter: ai.FilterSpec{},
	}}}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		// A workflow/lead event is for the workflow subscriber, not us.
		Type: "lead.created", TenantID: orgID, EntityID: &convID,
		OccurredAt: time.Now().UTC(),
	})

	time.Sleep(100 * time.Millisecond)
	if got := atomic.LoadInt32(&repo.listCalls); got != 0 {
		t.Fatalf("list triggers was called %d times for an ignored event", got)
	}
}

// ListActiveTriggers error is logged and swallowed (no panic, no assign).
func TestSubscriber_RepoErrorSwallowed(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	convID := uuid.New()

	repo := &fakeRepo{listErr: errors.New("db dead")}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		Type: "message.received", TenantID: orgID,
		EntityID: &convID, OccurredAt: time.Now().UTC(),
	})

	time.Sleep(100 * time.Millisecond)
	if repo.numAssigns() != 0 {
		t.Fatalf("assigns fired despite repo error: %d", repo.numAssigns())
	}
}

// Shutdown unblocks the goroutine — no leak.
func TestSubscriber_ShutdownIsGraceful(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)

	shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer c()
	if err := sub.Shutdown(shCtx); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}
}

// Event with nil EntityID is ignored without calling the repo — we have
// no conversation to assign to.
func TestSubscriber_NilEntityIDIgnored(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{rules: []ai.TriggerRule{{
		ID: uuid.New(), AgentID: uuid.New(), IsActive: true, AgentStatus: "active",
		Filter: ai.FilterSpec{},
	}}}
	bus := event.NewBus(event.DropOldest)
	sub := New(bus, repo, quietLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Start(ctx)
	defer func() {
		shCtx, c := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer c()
		_ = sub.Shutdown(shCtx)
	}()

	bus.Publish(ws.Event{
		Type: "message.received", TenantID: uuid.New(),
		EntityID: nil, OccurredAt: time.Now().UTC(),
	})
	time.Sleep(80 * time.Millisecond)
	if got := atomic.LoadInt32(&repo.listCalls); got != 0 {
		t.Fatalf("list called for nil entity: %d", got)
	}
}

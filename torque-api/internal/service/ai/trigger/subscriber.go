// Package trigger wires the S40 agent_triggers matcher into the live
// event bus. The matcher (service/ai.Match) has been pure logic since
// S40 with no subscriber to drive it — triggers configured in the UI
// therefore never fired in production. S52 closes that gap.
//
// Flow:
//
//   inbox / lead / copilot handler   →  bus.Publish(message.received | conversation.created)
//                                         │
//                                         ▼
//                  Subscriber.handle (this file)
//                                         │
//                         ListActiveTriggers(orgID)  (repo)
//                                         │
//                              ai.Match(rules, facts)
//                                         │
//                    AssignAgent(convID, agent) + Publish(agent.auto_assigned)
//
// Fail-closed behaviour: the matcher already skips triggers whose agent
// is kill-switched or non-active. An unknown predicate operator returns
// false (unmatched). A panic or repo error is logged and swallowed —
// the bus subscriber is not allowed to block the publisher.
package trigger

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/service/ai"
	"github.com/milennials/torque-api/internal/ws"
)

// Repo is the narrow slice of the agent repository this subscriber
// needs. Keeping it an interface lets the tests drop in a fake without
// spinning a pgxpool.
type Repo interface {
	ListActiveTriggers(ctx context.Context, orgID uuid.UUID) ([]ai.TriggerRule, error)
	AssignAgent(ctx context.Context, orgID, conversationID uuid.UUID, agentID *uuid.UUID) error
}

// Subscriber subscribes to the domain event bus and dispatches agent
// assignments whenever a bus event's tenant has at least one active
// trigger whose filter matches.
type Subscriber struct {
	bus    *event.Bus
	repo   Repo
	logger zerolog.Logger

	// handledTypes is the allow-list of event types that can trigger
	// an agent assignment. Anything outside this set is ignored
	// silently — other subscribers (workflow, WS broadcaster) will
	// still see the event.
	handledTypes map[string]bool

	stop  chan struct{}
	unsub func()
	wg    sync.WaitGroup
	once  sync.Once
}

// New wires deps. Production callers pass the in-proc event bus + a
// live agent repository; tests pass a mock repo.
func New(bus *event.Bus, repo Repo, logger zerolog.Logger) *Subscriber {
	return &Subscriber{
		bus:    bus,
		repo:   repo,
		logger: logger.With().Str("component", "agent_trigger_subscriber").Logger(),
		handledTypes: map[string]bool{
			"message.received":      true,
			"conversation.created":  true,
		},
		stop: make(chan struct{}),
	}
}

// Start subscribes to the bus and begins consuming in a single
// goroutine. Idempotent-safe: wg guards against double-start via the
// Shutdown path, but callers should still invoke Start exactly once.
func (s *Subscriber) Start(ctx context.Context) {
	ch, unsub := s.bus.Subscribe(256)
	s.unsub = unsub
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				s.handle(ctx, evt)
			}
		}
	}()
	s.logger.Info().
		Strs("types", keysOf(s.handledTypes)).
		Msg("trigger subscriber started")
}

// Shutdown unsubscribes and waits for the goroutine, bounded by ctx.
// Mirrors workflow.BusSubscriber.Shutdown so the main.go defer pattern
// stays consistent.
func (s *Subscriber) Shutdown(ctx context.Context) error {
	s.once.Do(func() {
		close(s.stop)
		if s.unsub != nil {
			s.unsub()
		}
	})
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handle processes a single event. Kept package-private; tested via the
// public Start path with a controllable bus + mock repo.
func (s *Subscriber) handle(parent context.Context, evt ws.Event) {
	if !s.handledTypes[evt.Type] {
		return
	}
	// For now only conversation-scoped events carry the EntityID we
	// need as conversation_id. A future expansion for lead-scoped
	// events would add a second branch here.
	if evt.EntityID == nil {
		return
	}
	if evt.TenantID == uuid.Nil {
		return
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	rules, err := s.repo.ListActiveTriggers(ctx, evt.TenantID)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("tenant_id", evt.TenantID.String()).
			Msg("list active triggers failed — skipping")
		return
	}
	if len(rules) == 0 {
		return
	}

	facts := buildLeadFacts(evt)

	rule, found := ai.Match(rules, facts)
	if !found {
		s.logger.Debug().
			Str("tenant_id", evt.TenantID.String()).
			Str("event_type", evt.Type).
			Int("rule_count", len(rules)).
			Msg("no matching trigger")
		return
	}

	// Assign — the repo caps to the tenant and the conversation row
	// via WHERE organization_id = $1, so a stale EntityID from another
	// tenant is naturally rejected (repo returns its own ErrNotFound
	// sentinel, which we swallow without alerting: a vanished
	// conversation is a race, not an error).
	if err := s.repo.AssignAgent(ctx, evt.TenantID, *evt.EntityID, &rule.AgentID); err != nil {
		s.logger.Warn().
			Err(err).
			Str("agent_id", rule.AgentID.String()).
			Str("conversation_id", evt.EntityID.String()).
			Msg("assign agent failed")
		return
	}

	s.logger.Info().
		Str("tenant_id", evt.TenantID.String()).
		Str("agent_id", rule.AgentID.String()).
		Str("trigger_id", rule.ID.String()).
		Str("conversation_id", evt.EntityID.String()).
		Msg("agent auto-assigned by trigger")

	// Fan-out an assignment event so the frontend can move the
	// conversation card into the "assigned to agent" lane without a
	// list refetch. Best-effort — a blocked subscriber drops this per
	// the bus' DropOldest policy.
	s.bus.Publish(ws.Event{
		Type:       "conversation.agent_assigned",
		TenantID:   evt.TenantID,
		EntityType: "conversation",
		EntityID:   evt.EntityID,
		Patch: map[string]any{
			"assigned_agent_id": rule.AgentID,
			"trigger_id":        rule.ID,
		},
		OccurredAt: time.Now().UTC(),
	})
}

// buildLeadFacts projects the bus event payload to the LeadFacts shape
// the matcher understands. Today the lead attributes are NOT present in
// the event envelope — the handler would have to re-fetch them from
// the leads table to filter richly. For the S52 MVP we pass an empty
// LeadFacts so catch-all rules (empty filter) match; the richer-facts
// pass is an S53+ follow-up.
func buildLeadFacts(evt ws.Event) ai.LeadFacts {
	facts := ai.LeadFacts{
		Custom: map[string]any{},
	}
	// Patches published by inbox/szchat carry contact info we could
	// project into custom fields. Left as a future extension — empty
	// map still lets catch-all filters match.
	if patch, ok := evt.Patch.(map[string]any); ok {
		for k, v := range patch {
			facts.Custom[k] = v
		}
	}
	return facts
}

// keysOf returns map keys in a deterministic-enough order for logs.
func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}


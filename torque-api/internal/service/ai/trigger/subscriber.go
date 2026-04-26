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

// LeadResolver hydrates LeadFacts from a conversation_id. Optional —
// when nil, the subscriber falls back to the empty/Custom-only path
// (catch-all rules still match; predicates against Origin/Segment/UTMs
// silently miss).
//
// S63 (D074-k): wiring this resolver lets triggers filter on real lead
// attributes (origin/segment/UTMs/rating) instead of only the patch
// payload that the publisher happened to attach. Tags multi-value
// hydration is deferred — requires a join on `lead_tags` and is rarely
// the discriminator in practice.
type LeadResolver interface {
	LeadFactsByConversation(ctx context.Context, orgID, conversationID uuid.UUID) (LeadAttrs, error)
}

// LeadAttrs is the projection a LeadResolver returns. Pointers are nil
// when the column is NULL on the lead row. Resolver implementations
// should return ErrLeadNotFound (or any error) silently — the
// subscriber falls back to empty facts on lookup failure (race with
// soft-delete is not an alert-worthy event).
type LeadAttrs struct {
	Origin      *string
	Segment     *string
	UTMSource   *string
	UTMMedium   *string
	UTMCampaign *string
	Rating      *int16
}

// Subscriber subscribes to the domain event bus and dispatches agent
// assignments whenever a bus event's tenant has at least one active
// trigger whose filter matches.
type Subscriber struct {
	bus    *event.Bus
	repo   Repo
	leads  LeadResolver // optional; nil → catch-all-only matching
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

// WithLeads attaches an optional LeadResolver so handle() hydrates
// LeadFacts.{Origin,Segment,UTMs,Rating} from the leads row before
// running the matcher. Without it, only catch-all rules (empty filter)
// reliably match — D074-k gap.
func (s *Subscriber) WithLeads(resolver LeadResolver) *Subscriber {
	s.leads = resolver
	return s
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
	// S63 (D074-k): hydrate facts from the leads row when a resolver
	// is wired. Failure is logged at Debug and degrades to the empty
	// projection (catch-all rules still match). conversation_id IS the
	// EntityID for handled types — see early-return guard above.
	if s.leads != nil {
		attrs, err := s.leads.LeadFactsByConversation(ctx, evt.TenantID, *evt.EntityID)
		if err != nil {
			s.logger.Debug().
				Err(err).
				Str("conversation_id", evt.EntityID.String()).
				Msg("lead hydration miss; matching with empty facts")
		} else {
			if attrs.Origin != nil {
				facts.Origin = *attrs.Origin
			}
			if attrs.Segment != nil {
				facts.Segment = *attrs.Segment
			}
			if attrs.UTMSource != nil {
				facts.UTMSource = *attrs.UTMSource
			}
			if attrs.UTMMedium != nil {
				facts.UTMMedium = *attrs.UTMMedium
			}
			if attrs.UTMCampaign != nil {
				facts.UTMCampaign = *attrs.UTMCampaign
			}
			facts.Rating = attrs.Rating
		}
	}

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


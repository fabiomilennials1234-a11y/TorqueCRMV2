package workflow

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
	"github.com/milennials/torque-api/internal/ws"
)

// RunnerConfig tunes the polling loop.
type RunnerConfig struct {
	// PollInterval is the idle sleep between empty claims. Lower =
	// tighter latency on fresh runs at cost of DB chatter.
	PollInterval time.Duration
	// PollJitter prevents thundering-herd when multiple replicas poll.
	PollJitter time.Duration
}

// DefaultRunnerConfig returns sane values for a single-replica dev box.
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		PollInterval: 2 * time.Second,
		PollJitter:   500 * time.Millisecond,
	}
}

// Runner is the background goroutine that claims pending
// workflow_run rows and hands them to the Executor. One runner per
// process; scale horizontally by adding more replicas (SKIP LOCKED in
// ClaimPendingRun keeps them from fighting).
type Runner struct {
	cfg      RunnerConfig
	repo     *workflowrepo.Repository
	executor *Executor
	logger   zerolog.Logger

	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewRunner wires deps.
func NewRunner(cfg RunnerConfig, repo *workflowrepo.Repository, executor *Executor, logger zerolog.Logger) *Runner {
	if cfg.PollInterval <= 0 {
		cfg = DefaultRunnerConfig()
	}
	return &Runner{
		cfg:      cfg,
		repo:     repo,
		executor: executor,
		logger:   logger.With().Str("component", "workflow_runner").Logger(),
		stop:     make(chan struct{}),
	}
}

// Start launches the dispatcher goroutine.
func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loop(ctx)
	}()
}

// Shutdown signals stop + waits for the goroutine to exit, capped at
// ctx deadline.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.once.Do(func() { close(r.stop) })
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		default:
		}

		claimCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		run, err := r.repo.ClaimPendingRun(claimCtx)
		cancel()
		if errors.Is(err, workflowrepo.ErrNotFound) {
			r.sleep(ctx)
			continue
		}
		if err != nil {
			r.logger.Error().Err(err).Msg("claim pending run failed")
			r.sleep(ctx)
			continue
		}

		runCtx, runCancel := context.WithTimeout(ctx, 2*time.Minute)
		if err := r.executor.Run(runCtx, run); err != nil {
			r.logger.Error().Err(err).Str("run_id", run.ID.String()).Msg("executor returned error")
		}
		runCancel()
	}
}

func (r *Runner) sleep(ctx context.Context) {
	d := r.cfg.PollInterval
	if r.cfg.PollJitter > 0 {
		d += time.Duration(rand.Int63n(int64(r.cfg.PollJitter)))
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	case <-r.stop:
	}
}

// ---------- event bus subscriber ------------------------------------

// BusSubscriber listens on the tenant event bus and enqueues a
// workflow_run for every active workflow whose trigger matches the
// event kind. Today it handles `lead.created` → `lead_created`
// trigger. S45 will extend the map to cover
// `lead.stage_changed` + `message.received`.
type BusSubscriber struct {
	bus    *event.Bus
	repo   *workflowrepo.Repository
	logger zerolog.Logger

	stop   chan struct{}
	unsub  func()
	wg     sync.WaitGroup
	once   sync.Once
}

// NewBusSubscriber wires deps.
func NewBusSubscriber(bus *event.Bus, repo *workflowrepo.Repository, logger zerolog.Logger) *BusSubscriber {
	return &BusSubscriber{
		bus:    bus,
		repo:   repo,
		logger: logger.With().Str("component", "workflow_bus_subscriber").Logger(),
		stop:   make(chan struct{}),
	}
}

// eventTriggerMap pins which trigger enum value corresponds to which
// event bus type. Anything not in this map is ignored. S45 extended
// this to cover stage changes (F01/F12 pipe moves publish
// `lead.stage_changed`) and inbound messages (F04 inbox publishes
// `message.received`). The schedule trigger fires via a cron-like
// scheduler rather than the event bus, so it lives outside this map.
var eventTriggerMap = map[string]string{
	"lead.created":        "lead_created",
	"lead.stage_changed":  "lead_stage_changed",
	"message.received":    "message_inbound",
}

// Start subscribes and begins consuming.
func (s *BusSubscriber) Start(ctx context.Context) {
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
}

// Shutdown unsubscribes and waits.
func (s *BusSubscriber) Shutdown(ctx context.Context) error {
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

// handle fans out one event to N workflows (one enqueue each).
func (s *BusSubscriber) handle(ctx context.Context, evt ws.Event) {
	trigger, ok := eventTriggerMap[evt.Type]
	if !ok {
		return
	}
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	wfs, err := s.repo.ListActiveWorkflowsByTrigger(scanCtx, evt.TenantID, trigger)
	if err != nil {
		s.logger.Error().Err(err).Str("trigger", trigger).Msg("list active workflows failed")
		return
	}
	for _, wf := range wfs {
		enqueueCtx, ec := context.WithTimeout(ctx, 5*time.Second)
		_, enqErr := s.repo.EnqueueRun(enqueueCtx, workflowrepo.EnqueueRunInput{
			OrganizationID: evt.TenantID,
			WorkflowID:     wf.ID,
			LeadID:         evt.EntityID, // lead.created uses lead id as entity
			TriggerSource:  evt.Type,
		})
		ec()
		if enqErr != nil {
			s.logger.Warn().
				Err(enqErr).
				Str("workflow_id", wf.ID.String()).
				Msg("enqueue run on bus event failed")
		}
	}
}

// keep rand reachable in all builds (otherwise unused import warn).
var _ = rand.Int63n

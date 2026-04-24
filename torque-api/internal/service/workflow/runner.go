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
	// WatchdogInterval is how often the orphan reclaimer runs.
	WatchdogInterval time.Duration
	// StaleAfter is the running-row age threshold for orphan reclaim.
	StaleAfter time.Duration
}

// DefaultRunnerConfig returns sane values for a single-replica dev box.
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		PollInterval:     2 * time.Second,
		PollJitter:       500 * time.Millisecond,
		WatchdogInterval: 2 * time.Minute,
		StaleAfter:       10 * time.Minute,
	}
}

// Runner is the background goroutine that claims pending
// workflow_run rows and hands them to the Executor. One runner per
// process; scale horizontally by adding more replicas (SKIP LOCKED in
// ClaimPendingRun keeps them from fighting).
//
// Two goroutines share the runner:
//   - claim loop: picks the next eligible pending run and executes it.
//   - watchdog loop: reclaims `running` rows whose process died mid-run.
type Runner struct {
	cfg      RunnerConfig
	repo     *workflowrepo.Repository
	executor *Executor
	bus      *event.Bus
	logger   zerolog.Logger

	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewRunner wires deps. bus is optional; when non-nil, the watchdog
// publishes `workflow_run.orphaned` events so ops dashboards light up
// on stuck runs.
func NewRunner(cfg RunnerConfig, repo *workflowrepo.Repository, executor *Executor, logger zerolog.Logger) *Runner {
	return NewRunnerWithBus(cfg, repo, executor, nil, logger)
}

// NewRunnerWithBus is the full constructor.
func NewRunnerWithBus(cfg RunnerConfig, repo *workflowrepo.Repository, executor *Executor, bus *event.Bus, logger zerolog.Logger) *Runner {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.PollJitter < 0 {
		cfg.PollJitter = 0
	}
	if cfg.WatchdogInterval <= 0 {
		cfg.WatchdogInterval = 2 * time.Minute
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = 10 * time.Minute
	}
	return &Runner{
		cfg:      cfg,
		repo:     repo,
		executor: executor,
		bus:      bus,
		logger:   logger.With().Str("component", "workflow_runner").Logger(),
		stop:     make(chan struct{}),
	}
}

// Start launches both goroutines.
func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(2)
	go func() {
		defer r.wg.Done()
		r.loop(ctx)
	}()
	go func() {
		defer r.wg.Done()
		r.watchdog(ctx)
	}()
}

// Shutdown signals stop + waits for both goroutines to exit.
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

// watchdog runs every WatchdogInterval and reclaims orphaned runs.
// An "orphaned" run is one in `running` status whose started_at is
// older than StaleAfter — typically the process died between
// AppendRunStep and CompleteRunStep, or the runCtx timed out without
// a graceful MarkRunFailed.
func (r *Runner) watchdog(ctx context.Context) {
	// Small offset on first tick so a boot storm doesn't slam the DB.
	initial := r.cfg.WatchdogInterval / 2
	if initial > 30*time.Second {
		initial = 30 * time.Second
	}
	t := time.NewTimer(initial)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case <-t.C:
		}
		r.reclaimOnce(ctx)
		t.Reset(r.cfg.WatchdogInterval)
	}
}

func (r *Runner) reclaimOnce(ctx context.Context) {
	qCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	orphaned, err := r.repo.ReclaimOrphanedRuns(qCtx, r.cfg.StaleAfter)
	if err != nil {
		r.logger.Error().Err(err).Msg("watchdog reclaim failed")
		return
	}
	if len(orphaned) == 0 {
		return
	}
	r.logger.Warn().
		Int("reclaimed", len(orphaned)).
		Dur("stale_after", r.cfg.StaleAfter).
		Msg("workflow watchdog reclaimed orphaned runs")
	for _, o := range orphaned {
		// DLQ row so the trail shows WHY the run died.
		attempt := o.Attempts + 1
		if attempt < 1 {
			attempt = 1
		}
		_, _ = r.repo.InsertRunFailure(qCtx, workflowrepo.RunFailure{
			OrganizationID: o.OrganizationID,
			RunID:          o.ID,
			WorkflowID:     o.WorkflowID,
			StepID:         o.CurrentStepID,
			Attempt:        attempt,
			ErrorCode:      "ORPHANED",
			ErrorMessage:   "watchdog reclaimed run exceeding stale threshold",
		})
		if r.bus != nil {
			rid := o.ID
			r.bus.Publish(ws.Event{
				Type:       "workflow_run.orphaned",
				TenantID:   o.OrganizationID,
				EntityType: "workflow_run",
				EntityID:   &rid,
				OccurredAt: time.Now().UTC(),
			})
		}
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
// event kind.
type BusSubscriber struct {
	bus    *event.Bus
	repo   *workflowrepo.Repository
	logger zerolog.Logger

	stop  chan struct{}
	unsub func()
	wg    sync.WaitGroup
	once  sync.Once
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
// event bus type. `schedule` trigger is cron-driven and lives outside
// this map.
var eventTriggerMap = map[string]string{
	"lead.created":       "lead_created",
	"lead.stage_changed": "lead_stage_changed",
	"message.received":   "message_inbound",
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
			LeadID:         evt.EntityID,
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

// keep rand reachable in all builds.
var _ = rand.Int63n

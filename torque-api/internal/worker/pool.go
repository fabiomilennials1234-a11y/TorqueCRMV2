// Package worker runs async jobs claimed from the operations ledger.
//
// The pool polls the ledger with a backoff, claims a pending row with
// SKIP LOCKED (so many workers can run side-by-side without contention), runs
// the registered Handler for that kind, and records the outcome.
//
// This is intentionally simple — single-process, goroutine-based. When we
// move to multiple replicas, the DB's SKIP LOCKED already gives us coordination;
// no changes needed here. When we need cross-process scheduling (priority lanes
// etc.), swap in a proper queue (Riverqueue, Temporal, etc.) behind the same
// Handler interface.
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	operationrepo "github.com/milennials/torque-api/internal/repository/operation"
	"github.com/milennials/torque-api/internal/ws"
)

// HandlerFunc processes one claimed operation.
//
// It receives the full operation record. The function MAY call `progress(p)`
// as often as it likes; the returned result is persisted on success. A retryable
// error lets the pool decrement retry_remaining and re-enqueue the row.
type HandlerFunc func(ctx context.Context, op domain.Operation, progress func(float64)) (result any, err error)

// Handler is a (kind, function) pair registered with the pool.
type Handler struct {
	Kind      string
	Fn        HandlerFunc
	Retryable func(err error) bool // return true to burn a retry
}

// Config tunes the pool.
type Config struct {
	// Concurrency is the maximum number of jobs running at once.
	Concurrency int
	// PollInterval is the idle wait between empty-ledger polls. A job pickup
	// resets to zero wait so bursts drain quickly.
	PollInterval time.Duration
	// PollJitter is uniform noise added to PollInterval to spread replicas.
	PollJitter time.Duration
	// WorkerID tags claim rows for debugging; defaults to a uuid per process.
	WorkerID string
}

// Pool is the set of goroutines that claim and execute operations.
type Pool struct {
	cfg      Config
	logger   zerolog.Logger
	repo     *operationrepo.Repository
	bus      *event.Bus
	handlers map[string]Handler

	sem chan struct{}
	wg  sync.WaitGroup

	stop   chan struct{}
	once   sync.Once
}

// New returns a pool ready to Start. `handlers` must cover every `kind` that
// may be submitted to the ledger — an unknown kind is logged and retried.
func New(cfg Config, repo *operationrepo.Repository, bus *event.Bus, logger zerolog.Logger, handlers []Handler) *Pool {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = "worker-" + uuid.NewString()[:8]
	}
	m := make(map[string]Handler, len(handlers))
	for _, h := range handlers {
		if h.Kind == "" || h.Fn == nil {
			panic("worker: invalid handler registration")
		}
		m[h.Kind] = h
	}
	return &Pool{
		cfg:      cfg,
		logger:   logger.With().Str("component", "worker").Str("worker_id", cfg.WorkerID).Logger(),
		repo:     repo,
		bus:      bus,
		handlers: m,
		sem:      make(chan struct{}, cfg.Concurrency),
		stop:     make(chan struct{}),
	}
}

// Start launches the dispatcher. Returns immediately; Shutdown is the
// coordinated stop.
func (p *Pool) Start(ctx context.Context) {
	go p.dispatcher(ctx)
}

// Shutdown stops accepting new jobs and waits for running handlers to finish.
// Respects the caller's ctx as an upper bound — if it expires, in-flight jobs
// may be left RUNNING (a future process will see the stale worker_id and
// optionally reclaim).
func (p *Pool) Shutdown(ctx context.Context) error {
	p.once.Do(func() { close(p.stop) })

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) dispatcher(ctx context.Context) {
	kinds := make([]string, 0, len(p.handlers))
	for k := range p.handlers {
		kinds = append(kinds, k)
	}
	if len(kinds) == 0 {
		p.logger.Warn().Msg("no handlers registered; dispatcher idle")
		// Still respond to shutdown.
		<-p.stop
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		default:
		}

		// Reserve a slot before claiming — if we claim first and fail to
		// reserve, we would hold a RUNNING row with no worker.
		select {
		case p.sem <- struct{}{}:
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		}

		claimCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		op, err := p.repo.ClaimOne(claimCtx, kinds, p.cfg.WorkerID)
		cancel()
		if errors.Is(err, operationrepo.ErrNotFound) {
			<-p.sem // release; nothing to do
			p.sleepWithJitter(ctx)
			continue
		}
		if err != nil {
			<-p.sem
			p.logger.Error().Err(err).Msg("claim failed")
			p.sleepWithJitter(ctx)
			continue
		}

		p.wg.Add(1)
		go func(op domain.Operation) {
			defer p.wg.Done()
			defer func() { <-p.sem }()
			p.run(ctx, op)
		}(op)
	}
}

func (p *Pool) run(ctx context.Context, op domain.Operation) {
	handler, ok := p.handlers[op.Kind]
	if !ok {
		// Should not happen — dispatcher only claims known kinds. Fail safe.
		_ = p.repo.Fail(ctx, op.ID, map[string]string{"code": "UNKNOWN_KIND", "message": op.Kind}, false)
		p.publish(op, ws.TypeOperationFailed, 0, nil, "unknown kind")
		return
	}

	progress := func(v float64) {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		// Fire-and-forget DB write; ignore transient errors.
		_ = p.repo.UpdateProgress(ctx, op.ID, v)
		// Push an interim WS patch so the UI's gauge animates.
		pp := v
		p.bus.Publish(ws.Event{
			Type:       ws.TypeOperationUpdated,
			TenantID:   op.OrganizationID,
			EntityType: "operation",
			EntityID:   &op.ID,
			Patch: ws.OperationPatch{
				ID:       op.ID,
				Status:   string(domain.OperationRunning),
				Progress: &pp,
			},
			OccurredAt: time.Now().UTC(),
		})
	}

	result, err := handler.Fn(ctx, op, progress)
	if err == nil {
		if serr := p.repo.Succeed(ctx, op.ID, result); serr != nil {
			p.logger.Error().Err(serr).Str("op_id", op.ID.String()).Msg("mark succeeded failed")
			return
		}
		p.publish(op, ws.TypeOperationSucceeded, 1, result, "")
		return
	}

	retryable := false
	if handler.Retryable != nil {
		retryable = handler.Retryable(err)
	}
	payload := map[string]any{
		"code":      classifyError(err),
		"message":   err.Error(),
		"retryable": retryable,
	}
	if ferr := p.repo.Fail(ctx, op.ID, payload, retryable); ferr != nil {
		p.logger.Error().Err(ferr).Str("op_id", op.ID.String()).Msg("mark failed failed")
		return
	}
	if retryable && op.RetryRemaining > 0 {
		p.publish(op, ws.TypeOperationUpdated, 0, nil, "retry scheduled")
	} else {
		p.publish(op, ws.TypeOperationFailed, 0, nil, err.Error())
	}
}

func (p *Pool) publish(op domain.Operation, eventType string, progress float64, result any, errMsg string) {
	patch := ws.OperationPatch{
		ID:     op.ID,
		Status: string(statusForEvent(eventType)),
	}
	if progress > 0 {
		pp := progress
		patch.Progress = &pp
	}
	if result != nil {
		b, err := json.Marshal(result)
		if err == nil {
			patch.Result = json.RawMessage(b)
		}
	}
	if errMsg != "" {
		patch.Error = map[string]string{"message": errMsg}
	}
	p.bus.Publish(ws.Event{
		Type:       eventType,
		TenantID:   op.OrganizationID,
		EntityType: "operation",
		EntityID:   &op.ID,
		Patch:      patch,
		OccurredAt: time.Now().UTC(),
	})
}

func statusForEvent(t string) domain.OperationStatus {
	switch t {
	case ws.TypeOperationSucceeded:
		return domain.OperationSucceeded
	case ws.TypeOperationFailed:
		return domain.OperationFailed
	default:
		return domain.OperationRunning
	}
}

// classifyError returns a short code string for known failure modes. Falls
// back to INTERNAL when the error is opaque.
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "CANCELLED"
	case errors.Is(err, context.DeadlineExceeded):
		return "TIMEOUT"
	default:
		return "INTERNAL"
	}
}

func (p *Pool) sleepWithJitter(ctx context.Context) {
	d := p.cfg.PollInterval
	if p.cfg.PollJitter > 0 {
		j := time.Duration(rand.Int63n(int64(p.cfg.PollJitter)))
		d += j
	}
	if d < time.Millisecond {
		d = time.Millisecond
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	case <-p.stop:
	}
}

// BackoffFor returns an exponential backoff duration for a given attempt,
// capped at 5 minutes. Exposed for handler use when they want to short-circuit.
func BackoffFor(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := math.Min(float64(attempt*attempt)*500, float64(5*time.Minute/time.Millisecond))
	return time.Duration(base) * time.Millisecond
}

// math/rand has been auto-seeded per-call since Go 1.20, so no init is needed.
// If we pin to an older toolchain we must re-seed here.
var _ = rand.Int63n // keep math/rand import alive

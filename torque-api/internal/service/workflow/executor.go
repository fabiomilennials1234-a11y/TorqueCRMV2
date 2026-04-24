package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/milennials/torque-api/internal/event"
	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
	"github.com/milennials/torque-api/internal/ws"
)

// ErrStepNotFound fires when next_step_ids references an id not in the
// workflow — a data-integrity bug the executor refuses to mask.
var ErrStepNotFound = errors.New("workflow: step not found in workflow")

// MaxStepsPerRun bounds the DAG walk defensively. A run that wants
// more is almost certainly an unintended loop; we fail it out rather
// than spin the executor forever.
const MaxStepsPerRun = 100

// Executor advances a single workflow_run from start to terminal state.
// Stateless across runs; the same Executor instance handles every run
// in the runner's poll loop.
type Executor struct {
	repo       *workflowrepo.Repository
	dispatcher *Dispatcher
	bus        *event.Bus
	logger     zerolog.Logger
}

// NewExecutor wires dependencies. bus is optional — when nil, the
// executor still records runs but skips workflow.* bus events.
func NewExecutor(repo *workflowrepo.Repository, dispatcher *Dispatcher, logger zerolog.Logger) *Executor {
	return NewExecutorWithBus(repo, dispatcher, nil, logger)
}

// NewExecutorWithBus is the full constructor used in production so the
// executor can publish `workflow_run.*` events.
func NewExecutorWithBus(repo *workflowrepo.Repository, dispatcher *Dispatcher, bus *event.Bus, logger zerolog.Logger) *Executor {
	return &Executor{
		repo:       repo,
		dispatcher: dispatcher,
		bus:        bus,
		logger:     logger.With().Str("component", "workflow_executor").Logger(),
	}
}

// backoffFor returns the wait before the next retry attempt. Ladder:
// attempts=1 → 15s, =2 → 1m, =3 → 5m, =4 → 30m, else 2h (cap).
//
// Clamping: bad input (attempts <= 0) yields 15s; huge values cap.
func backoffFor(attempts int) time.Duration {
	switch {
	case attempts <= 1:
		return 15 * time.Second
	case attempts == 2:
		return time.Minute
	case attempts == 3:
		return 5 * time.Minute
	case attempts == 4:
		return 30 * time.Minute
	default:
		return 2 * time.Hour
	}
}

// Run walks the DAG for a claimed run. The run must already be in
// `running` status (ClaimPendingRun transitions it). On success /
// failure, Run flips the terminal status and stamps ended_at.
//
// Error vs. return: Run only returns an error when the executor
// itself hits a persistence problem. Handler failures do NOT bubble
// up — they're classified and either scheduled for retry or sent to
// the DLQ, and Run returns nil so the runner loop doesn't retry the
// same row out-of-band.
func (e *Executor) Run(ctx context.Context, run workflowrepo.Run) error {
	orgID := run.OrganizationID

	wf, err := e.repo.Get(ctx, orgID, run.WorkflowID)
	if err != nil {
		return e.failRun(ctx, run, nil, "WORKFLOW_GONE", err.Error(), nil)
	}
	if wf.Status != "active" || wf.EntryStepID == nil {
		return e.failRun(ctx, run, nil, "WORKFLOW_INACTIVE", "workflow archived or missing entry", nil)
	}

	prevOutputs := map[string]json.RawMessage{}
	if len(run.Input) > 0 {
		prevOutputs["__input"] = run.Input
	}
	if run.LeadID != nil {
		leadJSON, _ := json.Marshal(map[string]string{"id": run.LeadID.String()})
		prevOutputs["__lead"] = leadJSON
	}

	// Resume from current_step_id when the run was previously parked
	// by a wait handler (next_retry_at put it back in the queue).
	currentID := *wf.EntryStepID
	if run.CurrentStepID != nil {
		currentID = *run.CurrentStepID
	}
	visited := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if visited >= MaxStepsPerRun {
			return e.failRun(ctx, run, nil, "STEP_LIMIT",
				fmt.Sprintf("run exceeded %d steps", MaxStepsPerRun), nil)
		}
		visited++

		steps, err := e.repo.ListStepsByIDs(ctx, orgID, []uuid.UUID{currentID})
		if err != nil || len(steps) == 0 {
			return e.failRun(ctx, run, &currentID, "STEP_MISSING", currentID.String(), nil)
		}
		step := steps[0]

		_ = e.repo.SetRunCurrentStep(ctx, orgID, run.ID, &step.ID)

		rs, err := e.repo.AppendRunStep(ctx, orgID, run.ID, step.ID, step.Config)
		if err != nil {
			return fmt.Errorf("append run step: %w", err)
		}

		outcome, handlerErr := e.dispatcher.Dispatch(ctx, step.Kind, StepContext{
			OrgID:           orgID.String(),
			RunID:           run.ID.String(),
			StepID:          step.ID.String(),
			LeadID:          leadIDString(run.LeadID),
			Config:          step.Config,
			PreviousOutputs: prevOutputs,
		})

		// Wait/suspend is a first-class control-flow signal, not a
		// failure. The handler packed the resume-at into NextStepID
		// as `__suspend:<rfc3339>`. We advance the current_step_id
		// pointer to the NEXT step (so reclaim resumes there) and
		// park the run with next_retry_at = resume-at.
		if errors.Is(handlerErr, ErrSuspend) {
			_ = e.repo.CompleteRunStep(ctx, rs.ID, "succeeded", outcome.Output, nil)
			resumeAt, nextID, ok := parseSuspendMarker(outcome.NextStepID, step)
			if !ok {
				return e.failRun(ctx, run, &step.ID, "SUSPEND_BAD_MARKER",
					"wait handler returned malformed suspend marker", outcome.Output)
			}
			if err := e.repo.SuspendRun(ctx, orgID, run.ID, nextID, resumeAt); err != nil {
				return fmt.Errorf("suspend: %w", err)
			}
			e.publish(orgID, run.ID, "workflow_run.suspended", map[string]any{
				"resume_at": resumeAt.Format(time.RFC3339),
				"next_step": nextID.String(),
			})
			e.logger.Info().
				Str("run_id", run.ID.String()).
				Time("resume_at", resumeAt).
				Msg("workflow run suspended")
			return nil
		}

		if handlerErr != nil {
			errBytes, _ := json.Marshal(map[string]string{
				"code":    errorCode(handlerErr),
				"message": handlerErr.Error(),
				"step_id": step.ID.String(),
			})
			_ = e.repo.CompleteRunStep(ctx, rs.ID, "failed", outcome.Output, errBytes)
			return e.handleFailure(ctx, run, step.ID, handlerErr, outcome.Output)
		}

		if err := e.repo.CompleteRunStep(ctx, rs.ID, "succeeded", outcome.Output, nil); err != nil {
			return err
		}
		prevOutputs[step.ID.String()] = outcome.Output

		nextID, terminal := nextStep(step, outcome)
		if terminal {
			break
		}
		currentID = nextID
	}

	resultBytes, _ := json.Marshal(map[string]any{
		"steps_visited": visited,
		"completed_at":  time.Now().UTC().Format(time.RFC3339),
	})
	_ = e.repo.SetRunCurrentStep(ctx, orgID, run.ID, nil)
	if err := e.repo.MarkRunSucceeded(ctx, orgID, run.ID, resultBytes); err != nil {
		return err
	}
	e.publish(orgID, run.ID, "workflow_run.succeeded", nil)
	e.logger.Info().
		Str("run_id", run.ID.String()).
		Int("steps_visited", visited).
		Msg("workflow run succeeded")
	return nil
}

// handleFailure classifies the handler error and either schedules a
// retry or lands the run in the DLQ. Both paths always insert a
// workflow_run_failures row so operators see the full trail.
func (e *Executor) handleFailure(ctx context.Context, run workflowrepo.Run, stepID uuid.UUID, handlerErr error, snapshot []byte) error {
	orgID := run.OrganizationID
	code := errorCode(handlerErr)
	attempts := run.Attempts + 1

	// Max attempts default (when schema default didn't fire on a
	// grandfathered row). 3 matches the migration CHECK default.
	max := run.MaxAttempts
	if max <= 0 {
		max = 3
	}

	// Always log to DLQ trail, even on a transient attempt.
	_, _ = e.repo.InsertRunFailure(ctx, workflowrepo.RunFailure{
		OrganizationID: orgID,
		RunID:          run.ID,
		WorkflowID:     run.WorkflowID,
		StepID:         &stepID,
		Attempt:        attempts,
		ErrorCode:      code,
		ErrorMessage:   handlerErr.Error(),
		Snapshot:       snapshot,
	})

	// Terminal if non-retryable OR retries exhausted.
	terminal := errors.Is(handlerErr, ErrNonRetryable) || attempts >= max
	if terminal {
		return e.failRun(ctx, run, &stepID, code, handlerErr.Error(), snapshot)
	}

	// Transient → schedule retry with backoff.
	nextAt := time.Now().UTC().Add(backoffFor(attempts))
	if err := e.repo.ScheduleRetry(ctx, orgID, run.ID, attempts, nextAt); err != nil {
		return e.failRun(ctx, run, &stepID, "SCHEDULE_RETRY_FAILED",
			err.Error(), snapshot)
	}
	e.publish(orgID, run.ID, "workflow_run.retry_scheduled", map[string]any{
		"attempt":      attempts,
		"max_attempts": max,
		"next_retry":   nextAt.Format(time.RFC3339),
		"error_code":   code,
	})
	e.logger.Warn().
		Str("run_id", run.ID.String()).
		Int("attempt", attempts).
		Int("max", max).
		Time("next_retry", nextAt).
		Str("error_code", code).
		Err(handlerErr).
		Msg("workflow retry scheduled")
	return nil
}

// failRun flips the run to failed with a structured error payload,
// emits a bus event, and always returns nil so the runner moves on.
func (e *Executor) failRun(ctx context.Context, run workflowrepo.Run, stepID *uuid.UUID, code, message string, snapshot []byte) error {
	orgID := run.OrganizationID
	payload, _ := json.Marshal(map[string]any{
		"code":    code,
		"message": message,
	})
	_ = e.repo.SetRunCurrentStep(ctx, orgID, run.ID, nil)
	if err := e.repo.MarkRunFailed(ctx, orgID, run.ID, payload); err != nil {
		e.logger.Error().Err(err).Str("run_id", run.ID.String()).Msg("mark failed failed")
	}
	// Ensure DLQ row exists (handleFailure writes one on handler
	// failures; direct failRun calls from the executor structural
	// paths — WORKFLOW_GONE etc — still need a trail).
	attempts := run.Attempts + 1
	if attempts < 1 {
		attempts = 1
	}
	_, _ = e.repo.InsertRunFailure(ctx, workflowrepo.RunFailure{
		OrganizationID: orgID,
		RunID:          run.ID,
		WorkflowID:     run.WorkflowID,
		StepID:         stepID,
		Attempt:        attempts,
		ErrorCode:      code,
		ErrorMessage:   message,
		Snapshot:       snapshot,
	})
	e.publish(orgID, run.ID, "workflow_run.failed", map[string]any{
		"code":    code,
		"message": message,
	})
	e.logger.Warn().
		Str("run_id", run.ID.String()).
		Str("code", code).
		Msg("workflow run failed")
	return nil
}

// parseSuspendMarker extracts the resumeAt + next step id from a
// WaitAction NextStepID ("__suspend:<rfc3339>"). The NEXT step comes
// from the wait step's next_step_ids[0] (waits always have exactly
// one outgoing edge).
func parseSuspendMarker(marker string, step workflowrepo.Step) (time.Time, uuid.UUID, bool) {
	const prefix = "__suspend:"
	if !strings.HasPrefix(marker, prefix) {
		return time.Time{}, uuid.Nil, false
	}
	when, err := time.Parse(time.RFC3339, strings.TrimPrefix(marker, prefix))
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	if len(step.NextStepIDs) == 0 {
		return time.Time{}, uuid.Nil, false
	}
	return when, step.NextStepIDs[0], true
}

// errorCode extracts a short machine-readable classifier from a
// handler error. Matches the sentinels declared in dispatcher.go.
func errorCode(err error) string {
	switch {
	case errors.Is(err, ErrTransient):
		return "TRANSIENT"
	case errors.Is(err, ErrNonRetryable):
		return "NON_RETRYABLE"
	case errors.Is(err, ErrActionUnknown):
		return "ACTION_UNKNOWN"
	default:
		return "HANDLER_FAILED"
	}
}

// publish is a thin wrapper that no-ops when bus is nil (tests).
func (e *Executor) publish(orgID, runID uuid.UUID, eventType string, patch map[string]any) {
	if e.bus == nil {
		return
	}
	rid := runID
	e.bus.Publish(ws.Event{
		Type:       eventType,
		TenantID:   orgID,
		EntityType: "workflow_run",
		EntityID:   &rid,
		Patch:      patch,
		OccurredAt: time.Now().UTC(),
	})
}

// nextStep returns (nextID, terminal). terminal=true when the step has
// no outgoing edges (end of chain or branch with missing falsy edge).
func nextStep(step workflowrepo.Step, outcome StepOutcome) (uuid.UUID, bool) {
	switch outcome.NextStepID {
	case "__branch:true":
		if len(step.NextStepIDs) > 0 {
			return step.NextStepIDs[0], false
		}
		return uuid.Nil, true
	case "__branch:false":
		if len(step.NextStepIDs) > 1 {
			return step.NextStepIDs[1], false
		}
		return uuid.Nil, true
	default:
		if len(step.NextStepIDs) > 0 {
			return step.NextStepIDs[0], false
		}
		return uuid.Nil, true
	}
}

func leadIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

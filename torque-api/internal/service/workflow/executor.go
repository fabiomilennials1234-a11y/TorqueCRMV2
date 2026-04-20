package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	workflowrepo "github.com/milennials/torque-api/internal/repository/workflow"
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
	logger     zerolog.Logger
}

// NewExecutor wires dependencies.
func NewExecutor(repo *workflowrepo.Repository, dispatcher *Dispatcher, logger zerolog.Logger) *Executor {
	return &Executor{
		repo:       repo,
		dispatcher: dispatcher,
		logger:     logger.With().Str("component", "workflow_executor").Logger(),
	}
}

// Run walks the DAG for a claimed run. The run must already be in
// `running` status (ClaimPendingRun transitions it). On success /
// failure, Run flips the terminal status and stamps ended_at.
//
// Error vs. return: Run only returns an error when the executor
// itself hits a persistence problem. Handler failures do NOT bubble
// up — they're recorded as a failed step + failed run, and Run
// returns nil so the runner loop doesn't retry the same row.
func (e *Executor) Run(ctx context.Context, run workflowrepo.Run) error {
	orgID := run.OrganizationID

	// Hydrate the workflow + entry step. The workflow must still be
	// active and must still have an entry — defensively re-check so
	// a concurrent archive doesn't leave a zombie run in progress.
	wf, err := e.repo.Get(ctx, orgID, run.WorkflowID)
	if err != nil {
		return e.fail(ctx, run, map[string]string{"code": "WORKFLOW_GONE", "message": err.Error()})
	}
	if wf.Status != "active" || wf.EntryStepID == nil {
		return e.fail(ctx, run, map[string]string{"code": "WORKFLOW_INACTIVE"})
	}

	prevOutputs := map[string]json.RawMessage{}
	// Seed `__input` so branch expressions can read `input.*`.
	if len(run.Input) > 0 {
		prevOutputs["__input"] = run.Input
	}
	// If the run targets a lead, seed a minimal __lead fact bag.
	// Richer hydration (name, stage, custom_fields) lands when the
	// real lead repo wires in S45. For S44 only the id is known.
	if run.LeadID != nil {
		leadJSON, _ := json.Marshal(map[string]string{"id": run.LeadID.String()})
		prevOutputs["__lead"] = leadJSON
	}

	currentID := *wf.EntryStepID
	visited := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if visited >= MaxStepsPerRun {
			return e.fail(ctx, run, map[string]string{
				"code":    "STEP_LIMIT",
				"message": fmt.Sprintf("run exceeded %d steps", MaxStepsPerRun),
			})
		}
		visited++

		steps, err := e.repo.ListStepsByIDs(ctx, orgID, []uuid.UUID{currentID})
		if err != nil || len(steps) == 0 {
			return e.fail(ctx, run, map[string]string{"code": "STEP_MISSING", "message": currentID.String()})
		}
		step := steps[0]

		// Point the run at the current step for external observers.
		_ = e.repo.SetRunCurrentStep(ctx, orgID, run.ID, &step.ID)

		// Insert a run step row in `running` state.
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

		if handlerErr != nil {
			errBytes, _ := json.Marshal(map[string]string{"code": "HANDLER_FAILED", "message": handlerErr.Error()})
			if err := e.repo.CompleteRunStep(ctx, rs.ID, "failed", nil, errBytes); err != nil {
				return err
			}
			return e.fail(ctx, run, map[string]string{
				"code":    "HANDLER_FAILED",
				"message": handlerErr.Error(),
				"step_id": step.ID.String(),
			})
		}

		// Success trace for this step.
		if err := e.repo.CompleteRunStep(ctx, rs.ID, "succeeded", outcome.Output, nil); err != nil {
			return err
		}
		prevOutputs[step.ID.String()] = outcome.Output

		// Resolve next step. Branch handlers return a synthetic
		// marker ("__branch:true" / "__branch:false") which we map
		// to next_step_ids[0] (true) or [1] (false). Straight
		// actions walk next_step_ids[0].
		nextID, terminal := nextStep(step, outcome)
		if terminal {
			break
		}
		currentID = nextID
	}

	// Run completed successfully.
	resultBytes, _ := json.Marshal(map[string]any{
		"steps_visited": visited,
		"completed_at":  time.Now().UTC().Format(time.RFC3339),
	})
	_ = e.repo.SetRunCurrentStep(ctx, orgID, run.ID, nil)
	if err := e.repo.MarkRunSucceeded(ctx, orgID, run.ID, resultBytes); err != nil {
		return err
	}
	e.logger.Info().
		Str("run_id", run.ID.String()).
		Int("steps_visited", visited).
		Msg("workflow run succeeded")
	return nil
}

// fail wraps MarkRunFailed + log. Always returns nil so the runner
// loop treats the run as handled.
func (e *Executor) fail(ctx context.Context, run workflowrepo.Run, payload map[string]string) error {
	raw, _ := json.Marshal(payload)
	_ = e.repo.SetRunCurrentStep(ctx, run.OrganizationID, run.ID, nil)
	if err := e.repo.MarkRunFailed(ctx, run.OrganizationID, run.ID, raw); err != nil {
		e.logger.Error().Err(err).Str("run_id", run.ID.String()).Msg("mark failed failed")
	}
	e.logger.Warn().
		Str("run_id", run.ID.String()).
		Interface("failure", payload).
		Msg("workflow run failed")
	return nil
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

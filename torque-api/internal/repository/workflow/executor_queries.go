// Executor-side read/write surface for F07 workflow runs (S44).
//
// Split from workflow.go so the builder/CRUD paths don't drag in the
// run-advancement queries — and so reviewers can read the executor
// data access in one file.

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RunStep is the per-step trace row. Status transitions:
// pending → running → succeeded | failed. Populated by the executor
// as it walks the DAG.
type RunStep struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RunID          uuid.UUID
	StepID         uuid.UUID
	Status         string
	Input          []byte
	Output         []byte
	ErrorPayload   []byte
	StartedAt      *time.Time
	EndedAt        *time.Time
	CreatedAt      time.Time
}

// ClaimPendingRun atomically claims one pending workflow_run using
// FOR UPDATE SKIP LOCKED so multiple executor replicas won't fight
// over the same row. Flips status → running and stamps started_at.
// Returns ErrNotFound when the queue is empty.
func (r *Repository) ClaimPendingRun(ctx context.Context) (Run, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Run{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var run Run
	err = tx.QueryRow(ctx,
		`SELECT id, organization_id, workflow_id, lead_id, trigger_source,
		        status::text, current_step_id, input, result, error_payload,
		        started_at, ended_at, created_at
		   FROM workflow_runs
		  WHERE status = 'pending'
		  ORDER BY created_at
		  FOR UPDATE SKIP LOCKED
		  LIMIT 1`,
	).Scan(
		&run.ID, &run.OrganizationID, &run.WorkflowID, &run.LeadID, &run.TriggerSource,
		&run.Status, &run.CurrentStepID, &run.Input, &run.Result, &run.ErrorPayload,
		&run.StartedAt, &run.EndedAt, &run.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("claim run: %w", err)
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx,
		`UPDATE workflow_runs SET status = 'running', started_at = $2
		  WHERE id = $1`,
		run.ID, now,
	); err != nil {
		return Run{}, fmt.Errorf("mark running: %w", err)
	}
	run.Status = "running"
	run.StartedAt = &now
	return run, tx.Commit(ctx)
}

// SetRunCurrentStep updates the current_step_id pointer so external
// observers can see where a long-running run is parked.
func (r *Repository) SetRunCurrentStep(ctx context.Context, orgID, runID uuid.UUID, stepID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET current_step_id = $3 WHERE id = $1 AND organization_id = $2`,
		runID, orgID, stepID,
	)
	return err
}

// MarkRunSucceeded flips status=succeeded, stamps ended_at, writes result.
func (r *Repository) MarkRunSucceeded(ctx context.Context, orgID, runID uuid.UUID, result json.RawMessage) error {
	payload := result
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'succeeded', result = $3::jsonb, ended_at = now()
		  WHERE id = $1 AND organization_id = $2`,
		runID, orgID, string(payload),
	)
	return err
}

// MarkRunFailed flips status=failed, stamps ended_at, writes error.
func (r *Repository) MarkRunFailed(ctx context.Context, orgID, runID uuid.UUID, errPayload json.RawMessage) error {
	payload := errPayload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs SET status = 'failed', error_payload = $3::jsonb, ended_at = now()
		  WHERE id = $1 AND organization_id = $2`,
		runID, orgID, string(payload),
	)
	return err
}

// AppendRunStep inserts a workflow_run_step row and stamps it running.
// Returns the fresh row so the executor can mark it complete later.
func (r *Repository) AppendRunStep(
	ctx context.Context, orgID, runID, stepID uuid.UUID, input json.RawMessage,
) (RunStep, error) {
	payload := input
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	var s RunStep
	err := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_run_steps (
		   organization_id, run_id, step_id, status, input, started_at
		 ) VALUES ($1, $2, $3, 'running', $4::jsonb, now())
		 RETURNING id, status::text, started_at, created_at`,
		orgID, runID, stepID, string(payload),
	).Scan(&s.ID, &s.Status, &s.StartedAt, &s.CreatedAt)
	if err != nil {
		return RunStep{}, fmt.Errorf("append run step: %w", err)
	}
	s.OrganizationID = orgID
	s.RunID = runID
	s.StepID = stepID
	s.Input = payload
	return s, nil
}

// CompleteRunStep transitions a step to its terminal state.
func (r *Repository) CompleteRunStep(
	ctx context.Context, id uuid.UUID, status string, output, errPayload json.RawMessage,
) error {
	if status != "succeeded" && status != "failed" && status != "cancelled" {
		return fmt.Errorf("invalid terminal status: %s", status)
	}
	var out, errp any
	if len(output) > 0 {
		out = []byte(output)
	}
	if len(errPayload) > 0 {
		errp = []byte(errPayload)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE workflow_run_steps
		    SET status = $2::workflow_run_status,
		        output = $3,
		        error_payload = $4,
		        ended_at = now()
		  WHERE id = $1`,
		id, status, out, errp,
	)
	return err
}

// ListRunSteps returns the per-step trace (ordered by created_at ASC)
// for a run. Used by the executions UI + tests.
func (r *Repository) ListRunSteps(ctx context.Context, orgID, runID uuid.UUID) ([]RunStep, error) {
	const q = `
		SELECT id, organization_id, run_id, step_id, status::text,
		       input, output, error_payload, started_at, ended_at, created_at
		  FROM workflow_run_steps
		 WHERE organization_id = $1 AND run_id = $2
		 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, runID)
	if err != nil {
		return nil, fmt.Errorf("list run steps: %w", err)
	}
	defer rows.Close()
	out := make([]RunStep, 0, 8)
	for rows.Next() {
		var s RunStep
		if err := rows.Scan(
			&s.ID, &s.OrganizationID, &s.RunID, &s.StepID, &s.Status,
			&s.Input, &s.Output, &s.ErrorPayload, &s.StartedAt, &s.EndedAt, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListStepsByIDs fetches a batch of steps by id, scoped to tenant.
// Used by the executor to resolve next_step_ids in bulk.
func (r *Repository) ListStepsByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]Step, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `
		SELECT id, workflow_id, kind::text, name, config, next_step_ids,
		       position_x, position_y
		  FROM workflow_steps
		 WHERE organization_id = $1 AND id = ANY($2)
	`
	rows, err := r.pool.Query(ctx, q, orgID, ids)
	if err != nil {
		return nil, fmt.Errorf("list steps by ids: %w", err)
	}
	defer rows.Close()
	out := make([]Step, 0, len(ids))
	for rows.Next() {
		var s Step
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Kind, &s.Name, &s.Config, &s.NextStepIDs,
			&s.PositionX, &s.PositionY,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListActiveWorkflowsByTrigger scans every active workflow whose
// trigger matches `kind` and is tenant-scoped to `orgID`. Used by
// the event-bus subscriber (e.g. on `lead.created`) to decide which
// workflows to enqueue.
func (r *Repository) ListActiveWorkflowsByTrigger(ctx context.Context, orgID uuid.UUID, kind string) ([]Workflow, error) {
	const q = `
		SELECT id, organization_id, name, description, trigger::text,
		       trigger_config, status::text, entry_step_id,
		       created_by, created_at, updated_at
		  FROM workflows
		 WHERE organization_id = $1
		   AND status = 'active'
		   AND trigger = $2::workflow_trigger
		   AND entry_step_id IS NOT NULL
	`
	rows, err := r.pool.Query(ctx, q, orgID, kind)
	if err != nil {
		return nil, fmt.Errorf("list active workflows: %w", err)
	}
	defer rows.Close()
	out := make([]Workflow, 0, 4)
	for rows.Next() {
		var w Workflow
		if err := rows.Scan(
			&w.ID, &w.OrganizationID, &w.Name, &w.Description, &w.Trigger,
			&w.TriggerConfig, &w.Status, &w.EntryStepID,
			&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

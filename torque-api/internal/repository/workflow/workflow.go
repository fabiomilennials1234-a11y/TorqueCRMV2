// Package workflow is the pgx-backed repository for F07 workflows + steps +
// runs. The runtime (step executor) is a separate worker kind that reads
// pending runs via ClaimRun and advances them; this package ships only the
// synchronous surface consumed by the builder + admin dashboards.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("workflow not found")
	ErrInvalidState = errors.New("illegal workflow state transition")
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	Trigger        string
	TriggerConfig  []byte
	Status         string
	EntryStepID    *uuid.UUID
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Step struct {
	ID             uuid.UUID
	WorkflowID     uuid.UUID
	Kind           string
	Name           string
	Config         []byte
	NextStepIDs    []uuid.UUID
	PositionX      *int
	PositionY      *int
}

type Run struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	WorkflowID     uuid.UUID
	LeadID         *uuid.UUID
	TriggerSource  string
	Status         string
	CurrentStepID  *uuid.UUID
	Input          []byte
	Result         []byte
	ErrorPayload   []byte
	StartedAt      *time.Time
	EndedAt        *time.Time
	CreatedAt      time.Time
	// Retry/DLQ bookkeeping (migration 0028). Attempts counts handler
	// executions; MaxAttempts caps retries before DLQ; NextRetryAt is
	// the scheduled reclaim time (also used by wait-step suspension).
	Attempts     int
	MaxAttempts  int
	NextRetryAt  *time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- workflows ----------------------------------------------------

type CreateInput struct {
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	Trigger        string
	TriggerConfig  json.RawMessage
	CreatedBy      *uuid.UUID
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (Workflow, error) {
	if in.Name == "" || in.Trigger == "" {
		return Workflow{}, errors.New("name and trigger are required")
	}
	cfg := in.TriggerConfig
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	const q = `
		INSERT INTO workflows (organization_id, name, description, trigger, trigger_config, created_by)
		VALUES ($1, $2, $3, $4::workflow_trigger, $5, $6)
		RETURNING id, status::text, created_at, updated_at
	`
	var w Workflow
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.Name, in.Description, in.Trigger, cfg, in.CreatedBy,
	).Scan(&w.ID, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return Workflow{}, fmt.Errorf("create workflow: %w", err)
	}
	w.OrganizationID = in.OrganizationID
	w.Name = in.Name
	w.Description = in.Description
	w.Trigger = in.Trigger
	w.TriggerConfig = cfg
	w.CreatedBy = in.CreatedBy
	return w, nil
}

func (r *Repository) List(ctx context.Context, orgID uuid.UUID) ([]Workflow, error) {
	const q = `
		SELECT id, organization_id, name, description, trigger::text, trigger_config,
		       status::text, entry_step_id, created_by, created_at, updated_at
		  FROM workflows
		 WHERE organization_id = $1
		 ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()
	out := make([]Workflow, 0, 4)
	for rows.Next() {
		var w Workflow
		if err := rows.Scan(
			&w.ID, &w.OrganizationID, &w.Name, &w.Description, &w.Trigger, &w.TriggerConfig,
			&w.Status, &w.EntryStepID, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, orgID, id uuid.UUID) (Workflow, error) {
	const q = `
		SELECT id, organization_id, name, description, trigger::text, trigger_config,
		       status::text, entry_step_id, created_by, created_at, updated_at
		  FROM workflows
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var w Workflow
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&w.ID, &w.OrganizationID, &w.Name, &w.Description, &w.Trigger, &w.TriggerConfig,
		&w.Status, &w.EntryStepID, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workflow{}, ErrNotFound
	}
	if err != nil {
		return Workflow{}, fmt.Errorf("get workflow: %w", err)
	}
	return w, nil
}

// SetStatus transitions draft → active → paused/archived. Active requires
// an entry_step_id; the publish endpoint enforces that at the handler level.
func (r *Repository) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE workflows SET status = $3::workflow_status
		  WHERE id = $1 AND organization_id = $2`,
		id, orgID, status,
	)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- steps --------------------------------------------------------

type UpsertStepInput struct {
	OrganizationID uuid.UUID
	WorkflowID     uuid.UUID
	ID             *uuid.UUID // nil → create; non-nil → update-by-id (still scoped by org+workflow)
	Kind           string
	Name           string
	Config         json.RawMessage
	NextStepIDs    []uuid.UUID
	PositionX      *int
	PositionY      *int
}

func (r *Repository) UpsertStep(ctx context.Context, in UpsertStepInput) (Step, error) {
	if in.Name == "" || in.Kind == "" {
		return Step{}, errors.New("name and kind are required")
	}
	cfg := in.Config
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	next := in.NextStepIDs
	if next == nil {
		next = []uuid.UUID{}
	}

	// Ownership: workflow must belong to org.
	var ownedBy uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT organization_id FROM workflows WHERE id = $1 LIMIT 1`,
		in.WorkflowID,
	).Scan(&ownedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Step{}, ErrNotFound
		}
		return Step{}, fmt.Errorf("workflow ownership: %w", err)
	}
	if ownedBy != in.OrganizationID {
		return Step{}, ErrNotFound
	}

	var s Step
	if in.ID == nil {
		const q = `
			INSERT INTO workflow_steps (organization_id, workflow_id, kind, name, config, next_step_ids, position_x, position_y)
			VALUES ($1, $2, $3::workflow_step_kind, $4, $5, $6, $7, $8)
			RETURNING id
		`
		if err := r.pool.QueryRow(ctx, q,
			in.OrganizationID, in.WorkflowID, in.Kind, in.Name, cfg, next, in.PositionX, in.PositionY,
		).Scan(&s.ID); err != nil {
			return Step{}, fmt.Errorf("create step: %w", err)
		}
	} else {
		ct, err := r.pool.Exec(ctx,
			`UPDATE workflow_steps
			    SET kind = $3::workflow_step_kind, name = $4, config = $5,
			        next_step_ids = $6, position_x = $7, position_y = $8
			  WHERE id = $1 AND workflow_id = $2 AND organization_id = $9`,
			*in.ID, in.WorkflowID, in.Kind, in.Name, cfg, next, in.PositionX, in.PositionY, in.OrganizationID,
		)
		if err != nil {
			return Step{}, fmt.Errorf("update step: %w", err)
		}
		if ct.RowsAffected() == 0 {
			return Step{}, ErrNotFound
		}
		s.ID = *in.ID
	}
	s.WorkflowID = in.WorkflowID
	s.Kind = in.Kind
	s.Name = in.Name
	s.Config = cfg
	s.NextStepIDs = next
	s.PositionX = in.PositionX
	s.PositionY = in.PositionY
	return s, nil
}

func (r *Repository) ListSteps(ctx context.Context, orgID, workflowID uuid.UUID) ([]Step, error) {
	const q = `
		SELECT id, workflow_id, kind::text, name, config, next_step_ids, position_x, position_y
		  FROM workflow_steps
		 WHERE organization_id = $1 AND workflow_id = $2
		 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, workflowID)
	if err != nil {
		return nil, fmt.Errorf("list steps: %w", err)
	}
	defer rows.Close()
	out := make([]Step, 0, 8)
	for rows.Next() {
		var s Step
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Kind, &s.Name, &s.Config, &s.NextStepIDs, &s.PositionX, &s.PositionY,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteStep(ctx context.Context, orgID, workflowID, stepID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM workflow_steps
		  WHERE id = $1 AND workflow_id = $2 AND organization_id = $3`,
		stepID, workflowID, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete step: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetEntry updates workflows.entry_step_id after validating the step lives
// in the same workflow.
func (r *Repository) SetEntry(ctx context.Context, orgID, workflowID, stepID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE workflows SET entry_step_id = $3
		  WHERE id = $1 AND organization_id = $2
		    AND EXISTS (
		      SELECT 1 FROM workflow_steps
		       WHERE id = $3 AND workflow_id = $1 AND organization_id = $2
		    )`,
		workflowID, orgID, stepID,
	)
	if err != nil {
		return fmt.Errorf("set entry: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- runs ---------------------------------------------------------

type EnqueueRunInput struct {
	OrganizationID uuid.UUID
	WorkflowID     uuid.UUID
	LeadID         *uuid.UUID
	TriggeredBy    *uuid.UUID
	TriggerSource  string
	Input          json.RawMessage
}

// EnqueueRun creates a pending run. The runtime worker (kind
// `workflow.execute`) claims and advances it. Returns ErrNotFound if the
// workflow is outside the tenant or not active.
func (r *Repository) EnqueueRun(ctx context.Context, in EnqueueRunInput) (Run, error) {
	if in.TriggerSource == "" {
		in.TriggerSource = "manual"
	}
	input := in.Input
	if len(input) == 0 {
		input = []byte(`{}`)
	}

	// Only active workflows can run.
	var status, entryStepID sql_nullable
	if err := r.pool.QueryRow(ctx,
		`SELECT status::text, COALESCE(entry_step_id::text, '')
		   FROM workflows
		  WHERE id = $1 AND organization_id = $2 LIMIT 1`,
		in.WorkflowID, in.OrganizationID,
	).Scan(&status, &entryStepID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Run{}, ErrNotFound
		}
		return Run{}, fmt.Errorf("validate workflow: %w", err)
	}
	if status.String != "active" || entryStepID.String == "" {
		return Run{}, ErrInvalidState
	}

	var run Run
	err := r.pool.QueryRow(ctx,
		`INSERT INTO workflow_runs (
		   organization_id, workflow_id, lead_id, triggered_by, trigger_source, input
		 ) VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, status::text, created_at`,
		in.OrganizationID, in.WorkflowID, in.LeadID, in.TriggeredBy, in.TriggerSource, input,
	).Scan(&run.ID, &run.Status, &run.CreatedAt)
	if err != nil {
		return Run{}, fmt.Errorf("enqueue run: %w", err)
	}
	run.OrganizationID = in.OrganizationID
	run.WorkflowID = in.WorkflowID
	run.LeadID = in.LeadID
	run.TriggerSource = in.TriggerSource
	run.Input = input
	return run, nil
}

// ListRuns returns the most recent runs for a workflow.
func (r *Repository) ListRuns(ctx context.Context, orgID, workflowID uuid.UUID, limit int) ([]Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
		SELECT id, organization_id, workflow_id, lead_id, trigger_source,
		       status::text, current_step_id, input, result, error_payload,
		       started_at, ended_at, created_at,
		       attempts, max_attempts, next_retry_at
		  FROM workflow_runs
		 WHERE organization_id = $1 AND workflow_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, orgID, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()
	out := make([]Run, 0, limit)
	for rows.Next() {
		var rn Run
		if err := rows.Scan(
			&rn.ID, &rn.OrganizationID, &rn.WorkflowID, &rn.LeadID, &rn.TriggerSource,
			&rn.Status, &rn.CurrentStepID, &rn.Input, &rn.Result, &rn.ErrorPayload,
			&rn.StartedAt, &rn.EndedAt, &rn.CreatedAt,
			&rn.Attempts, &rn.MaxAttempts, &rn.NextRetryAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rn)
	}
	return out, rows.Err()
}

// CancelRun transitions pending|running → cancelled.
func (r *Repository) CancelRun(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE workflow_runs
		    SET status = 'cancelled', ended_at = now()
		  WHERE id = $1 AND organization_id = $2
		    AND status IN ('pending','running')`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("cancel run: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// sql_nullable is a tiny type that scans NULL safely to an empty string.
// Used inline so we don't pull database/sql just for sql.NullString.
type sql_nullable struct {
	String string
}

// Scan implements the pgx interface for text-like columns.
func (s *sql_nullable) Scan(src any) error {
	if src == nil {
		s.String = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		s.String = v
	case []byte:
		s.String = string(v)
	default:
		return fmt.Errorf("unexpected scan type: %T", src)
	}
	return nil
}

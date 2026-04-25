// Triggers CRUD for F06 Copilot auto-assignment (S40).
//
// agent_triggers rows live in the tenant's scope and point at a single
// agent. The matcher reads a projection of these rows (see ListActiveTriggers)
// that already joins agents to surface kill_switch + status — the
// matcher itself is pure and does no IO.

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/milennials/torque-api/internal/service/ai"
)

// AgentTrigger is the full row as returned by the editor endpoints.
type AgentTrigger struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	AgentID        uuid.UUID
	Name           string
	Description    *string
	Priority       int
	Filter         ai.FilterSpec
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateTriggerInput struct {
	OrganizationID uuid.UUID
	AgentID        uuid.UUID
	Name           string
	Description    *string
	Priority       int
	Filter         ai.FilterSpec
	CreatedBy      *uuid.UUID
}

// CreateTrigger inserts a new trigger. Validates basic shape; the
// filter_json is written as-is after round-tripping through json.Marshal
// so the stored shape is canonicalised.
func (r *Repository) CreateTrigger(ctx context.Context, in CreateTriggerInput) (AgentTrigger, error) {
	name := strings.TrimSpace(in.Name)
	if len(name) < 2 || len(name) > 120 {
		return AgentTrigger{}, errors.New("name must be 2-120 chars")
	}
	if in.Priority < 1 || in.Priority > 10000 {
		in.Priority = 100
	}
	// Ownership guard: agent must belong to the caller's tenant.
	var ownedBy uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT organization_id FROM agents WHERE id = $1 LIMIT 1`,
		in.AgentID,
	).Scan(&ownedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentTrigger{}, ErrNotFound
		}
		return AgentTrigger{}, fmt.Errorf("agent ownership: %w", err)
	}
	if ownedBy != in.OrganizationID {
		return AgentTrigger{}, ErrNotFound
	}

	filterJSON, err := json.Marshal(in.Filter)
	if err != nil {
		return AgentTrigger{}, fmt.Errorf("marshal filter: %w", err)
	}

	const q = `
		INSERT INTO agent_triggers (
		  organization_id, agent_id, name, description, priority, filter_json, created_by
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7)
		RETURNING id, created_at, updated_at
	`
	var t AgentTrigger
	if err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.AgentID, name, in.Description, in.Priority,
		string(filterJSON), in.CreatedBy,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return AgentTrigger{}, fmt.Errorf("create trigger: %w", err)
	}
	t.OrganizationID = in.OrganizationID
	t.AgentID = in.AgentID
	t.Name = name
	t.Description = in.Description
	t.Priority = in.Priority
	t.Filter = in.Filter
	t.IsActive = true
	return t, nil
}

// ListTriggersByAgent returns every trigger for an agent, ordered by
// priority ASC then created_at ASC — the same ordering the matcher
// consumes, but with active + inactive both included (editor needs
// to show disabled rules too).
func (r *Repository) ListTriggersByAgent(ctx context.Context, orgID, agentID uuid.UUID) ([]AgentTrigger, error) {
	const q = `
		SELECT id, organization_id, agent_id, name, description,
		       priority, filter_json, is_active, created_at, updated_at
		  FROM agent_triggers
		 WHERE organization_id = $1 AND agent_id = $2
		 ORDER BY priority, created_at
	`
	rows, err := r.pool.Query(ctx, q, orgID, agentID)
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	defer rows.Close()
	out := make([]AgentTrigger, 0, 4)
	for rows.Next() {
		var t AgentTrigger
		var raw []byte
		if err := rows.Scan(&t.ID, &t.OrganizationID, &t.AgentID, &t.Name, &t.Description,
			&t.Priority, &raw, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &t.Filter)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTriggerInput is the partial patch shape for UpdateTrigger.
type UpdateTriggerInput struct {
	Name        *string
	Description *string
	Priority    *int
	Filter      *ai.FilterSpec
	IsActive    *bool
}

func (r *Repository) UpdateTrigger(ctx context.Context, orgID, id uuid.UUID, in UpdateTriggerInput) (AgentTrigger, error) {
	sets := make([]string, 0, 5)
	args := []any{orgID, id}
	idx := 3
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if len(n) < 2 || len(n) > 120 {
			return AgentTrigger{}, errors.New("name must be 2-120 chars")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, n)
		idx++
	}
	if in.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *in.Description)
		idx++
	}
	if in.Priority != nil {
		p := *in.Priority
		if p < 1 || p > 10000 {
			return AgentTrigger{}, errors.New("priority must be between 1 and 10000")
		}
		sets = append(sets, fmt.Sprintf("priority = $%d", idx))
		args = append(args, p)
		idx++
	}
	if in.Filter != nil {
		raw, err := json.Marshal(*in.Filter)
		if err != nil {
			return AgentTrigger{}, fmt.Errorf("marshal filter: %w", err)
		}
		sets = append(sets, fmt.Sprintf("filter_json = $%d::jsonb", idx))
		args = append(args, string(raw))
		idx++
	}
	if in.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *in.IsActive)
		// idx not bumped — IsActive is the last optional field.
	}
	if len(sets) == 0 {
		return r.GetTrigger(ctx, orgID, id)
	}
	q := fmt.Sprintf(
		`UPDATE agent_triggers SET %s WHERE organization_id = $1 AND id = $2`,
		strings.Join(sets, ", "),
	)
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return AgentTrigger{}, fmt.Errorf("update trigger: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return AgentTrigger{}, ErrNotFound
	}
	return r.GetTrigger(ctx, orgID, id)
}

func (r *Repository) GetTrigger(ctx context.Context, orgID, id uuid.UUID) (AgentTrigger, error) {
	const q = `
		SELECT id, organization_id, agent_id, name, description,
		       priority, filter_json, is_active, created_at, updated_at
		  FROM agent_triggers
		 WHERE organization_id = $1 AND id = $2
		 LIMIT 1
	`
	var t AgentTrigger
	var raw []byte
	err := r.pool.QueryRow(ctx, q, orgID, id).Scan(
		&t.ID, &t.OrganizationID, &t.AgentID, &t.Name, &t.Description,
		&t.Priority, &raw, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentTrigger{}, ErrNotFound
	}
	if err != nil {
		return AgentTrigger{}, fmt.Errorf("get trigger: %w", err)
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &t.Filter)
	}
	return t, nil
}

func (r *Repository) DeleteTrigger(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM agent_triggers WHERE organization_id = $1 AND id = $2`,
		orgID, id,
	)
	if err != nil {
		return fmt.Errorf("delete trigger: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListActiveTriggers returns every active trigger for the tenant,
// joined with agents to carry kill_switch + status (so the matcher can
// enforce the two runtime guards without another query). Ordered by
// priority ASC, created_at ASC — the canonical matcher order.
func (r *Repository) ListActiveTriggers(ctx context.Context, orgID uuid.UUID) ([]ai.TriggerRule, error) {
	const q = `
		SELECT t.id, t.agent_id, t.priority, t.is_active,
		       t.filter_json,
		       a.kill_switch, a.status::text
		  FROM agent_triggers t
		  JOIN agents a ON a.id = t.agent_id
		 WHERE t.organization_id = $1
		   AND t.is_active = true
		 ORDER BY t.priority, t.created_at
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active triggers: %w", err)
	}
	defer rows.Close()
	out := make([]ai.TriggerRule, 0, 4)
	for rows.Next() {
		var r ai.TriggerRule
		var raw []byte
		if err := rows.Scan(&r.ID, &r.AgentID, &r.Priority, &r.IsActive,
			&raw, &r.AgentKillSwitch, &r.AgentStatus); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &r.Filter)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AssignAgent sets conversations.assigned_agent_id for a conversation.
// No-op (rows 0) if the row doesn't exist. Used by the matcher's
// downstream action (worker `conversation.assign_agent`) and by any
// future explicit-assign endpoint.
func (r *Repository) AssignAgent(ctx context.Context, orgID, conversationID uuid.UUID, agentID *uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE conversations SET assigned_agent_id = $3
		  WHERE organization_id = $1 AND id = $2`,
		orgID, conversationID, agentID,
	)
	if err != nil {
		return fmt.Errorf("assign agent: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

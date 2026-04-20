// Package analytics computes tenant metrics directly against the domain
// tables. F09 — first pass is read-only queries over leads / pipe_entries /
// messages / tasks / proposals / campaign_recipients. No materialized views
// yet — Postgres handles the volumes we'll see pre-S30 without pre-aggregation.
// Materialized views land when a query here shows up in pg_stat_statements.
//
// Every query is tenant-scoped and accepts an explicit time window
// [since, until). Callers set the window per metric; there is no server-side
// default to prevent accidental full-table scans.
package analytics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Window is the [since, until) range used by every query. If until.IsZero()
// it defaults to now(); since must be non-zero.
type Window struct {
	Since time.Time
	Until time.Time
}

// ErrInvalidWindow is returned when since is zero or until <= since.
var ErrInvalidWindow = errors.New("invalid analytics window")

func (w Window) normalize() (time.Time, time.Time, error) {
	if w.Since.IsZero() {
		return time.Time{}, time.Time{}, ErrInvalidWindow
	}
	until := w.Until
	if until.IsZero() {
		until = time.Now().UTC()
	}
	if !until.After(w.Since) {
		return time.Time{}, time.Time{}, ErrInvalidWindow
	}
	return w.Since.UTC(), until.UTC(), nil
}

// -------- leads -------------------------------------------------------

type LeadsSummary struct {
	Total       int64 `json:"total"`
	InWindow    int64 `json:"in_window"`
	Assigned    int64 `json:"assigned"`
	Unassigned  int64 `json:"unassigned"`
}

// LeadsSummary returns counts of active (non-deleted) leads. `in_window`
// counts leads created between Window.Since and Window.Until.
func (r *Repository) LeadsSummary(ctx context.Context, orgID uuid.UUID, w Window) (LeadsSummary, error) {
	since, until, err := w.normalize()
	if err != nil {
		return LeadsSummary{}, err
	}
	const q = `
		SELECT
		  count(*)                                                            AS total,
		  count(*) FILTER (WHERE created_at >= $2 AND created_at < $3)        AS in_window,
		  count(*) FILTER (WHERE responsible_id IS NOT NULL)                  AS assigned,
		  count(*) FILTER (WHERE responsible_id IS NULL)                      AS unassigned
		  FROM leads
		 WHERE organization_id = $1 AND deleted_at IS NULL
	`
	var s LeadsSummary
	err = r.pool.QueryRow(ctx, q, orgID, since, until).Scan(&s.Total, &s.InWindow, &s.Assigned, &s.Unassigned)
	if err != nil {
		return LeadsSummary{}, fmt.Errorf("leads summary: %w", err)
	}
	return s, nil
}

// -------- pipe throughput --------------------------------------------

type StageVolume struct {
	PipeID  uuid.UUID `json:"pipe_id"`
	StageID uuid.UUID `json:"stage_id"`
	Count   int64     `json:"count"`
}

// StageVolumeByPipe returns active entries per stage for a given pipe.
// "Active" = left_at IS NULL. Useful for Kanban column counts.
func (r *Repository) StageVolumeByPipe(ctx context.Context, orgID, pipeID uuid.UUID) ([]StageVolume, error) {
	const q = `
		SELECT pipe_id, stage_id, count(*)
		  FROM pipe_entries
		 WHERE organization_id = $1 AND pipe_id = $2 AND left_at IS NULL
		 GROUP BY pipe_id, stage_id
	`
	rows, err := r.pool.Query(ctx, q, orgID, pipeID)
	if err != nil {
		return nil, fmt.Errorf("stage volume: %w", err)
	}
	defer rows.Close()
	out := make([]StageVolume, 0, 8)
	for rows.Next() {
		var s StageVolume
		if err := rows.Scan(&s.PipeID, &s.StageID, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// -------- messages ----------------------------------------------------

type MessagesSummary struct {
	Inbound  int64 `json:"inbound"`
	Outbound int64 `json:"outbound"`
}

// MessagesSummary counts inbound vs outbound messages in the window.
func (r *Repository) MessagesSummary(ctx context.Context, orgID uuid.UUID, w Window) (MessagesSummary, error) {
	since, until, err := w.normalize()
	if err != nil {
		return MessagesSummary{}, err
	}
	const q = `
		SELECT
		  count(*) FILTER (WHERE direction = 'inbound')  AS inbound,
		  count(*) FILTER (WHERE direction = 'outbound') AS outbound
		  FROM messages
		 WHERE organization_id = $1
		   AND occurred_at >= $2 AND occurred_at < $3
	`
	var s MessagesSummary
	if err := r.pool.QueryRow(ctx, q, orgID, since, until).Scan(&s.Inbound, &s.Outbound); err != nil {
		return MessagesSummary{}, fmt.Errorf("messages summary: %w", err)
	}
	return s, nil
}

// -------- tasks -------------------------------------------------------

type TasksSummary struct {
	Pending    int64 `json:"pending"`
	InProgress int64 `json:"in_progress"`
	DoneInWin  int64 `json:"done_in_window"`
	MissedInWin int64 `json:"missed_in_window"`
	Overdue    int64 `json:"overdue"`
}

// TasksSummary returns the task-state snapshot (point-in-time) plus the
// done/missed counts inside the window.
func (r *Repository) TasksSummary(ctx context.Context, orgID uuid.UUID, w Window) (TasksSummary, error) {
	since, until, err := w.normalize()
	if err != nil {
		return TasksSummary{}, err
	}
	const q = `
		SELECT
		  count(*) FILTER (WHERE status = 'pending')                                   AS pending,
		  count(*) FILTER (WHERE status = 'in_progress')                               AS in_progress,
		  count(*) FILTER (WHERE status = 'done' AND completed_at >= $2 AND completed_at < $3)
		                                                                               AS done_in_window,
		  count(*) FILTER (WHERE status = 'missed' AND updated_at >= $2 AND updated_at < $3)
		                                                                               AS missed_in_window,
		  count(*) FILTER (WHERE status IN ('pending','in_progress') AND due_at < now())
		                                                                               AS overdue
		  FROM tasks
		 WHERE organization_id = $1
	`
	var s TasksSummary
	if err := r.pool.QueryRow(ctx, q, orgID, since, until).
		Scan(&s.Pending, &s.InProgress, &s.DoneInWin, &s.MissedInWin, &s.Overdue); err != nil {
		return TasksSummary{}, fmt.Errorf("tasks summary: %w", err)
	}
	return s, nil
}

// -------- proposals ---------------------------------------------------

type ProposalsSummary struct {
	Sent       int64 `json:"sent_in_window"`
	Viewed     int64 `json:"viewed_in_window"`
	Accepted   int64 `json:"accepted_in_window"`
	Rejected   int64 `json:"rejected_in_window"`
	WonAmount  int64 `json:"won_amount_cents"`
}

func (r *Repository) ProposalsSummary(ctx context.Context, orgID uuid.UUID, w Window) (ProposalsSummary, error) {
	since, until, err := w.normalize()
	if err != nil {
		return ProposalsSummary{}, err
	}
	const q = `
		SELECT
		  count(*) FILTER (WHERE sent_at        >= $2 AND sent_at        < $3)  AS sent_in_window,
		  count(*) FILTER (WHERE first_viewed_at>= $2 AND first_viewed_at< $3)  AS viewed_in_window,
		  count(*) FILTER (WHERE accepted_at    >= $2 AND accepted_at    < $3)  AS accepted_in_window,
		  count(*) FILTER (WHERE rejected_at    >= $2 AND rejected_at    < $3)  AS rejected_in_window,
		  COALESCE(sum(amount_cents) FILTER (WHERE accepted_at >= $2 AND accepted_at < $3), 0)
		                                                                         AS won_amount_cents
		  FROM pipe_proposals
		 WHERE organization_id = $1
	`
	var s ProposalsSummary
	if err := r.pool.QueryRow(ctx, q, orgID, since, until).
		Scan(&s.Sent, &s.Viewed, &s.Accepted, &s.Rejected, &s.WonAmount); err != nil {
		return ProposalsSummary{}, fmt.Errorf("proposals summary: %w", err)
	}
	return s, nil
}

// -------- leaderboard -------------------------------------------------

type MemberStats struct {
	MemberID         uuid.UUID `json:"member_id"`
	LeadsAssigned    int64     `json:"leads_assigned"`
	TasksCompleted   int64     `json:"tasks_completed_in_window"`
	ProposalsAccepted int64     `json:"proposals_accepted_in_window"`
}

// Leaderboard joins per-member assignments and completions in the window.
// Returns rows ONLY for members with at least one non-zero counter.
func (r *Repository) Leaderboard(ctx context.Context, orgID uuid.UUID, w Window) ([]MemberStats, error) {
	since, until, err := w.normalize()
	if err != nil {
		return nil, err
	}
	const q = `
		WITH tm AS (
		  SELECT id FROM team_members WHERE organization_id = $1 AND is_active
		),
		la AS (
		  SELECT responsible_id AS member_id, count(*) AS n
		    FROM leads
		   WHERE organization_id = $1 AND deleted_at IS NULL AND responsible_id IS NOT NULL
		   GROUP BY responsible_id
		),
		tc AS (
		  SELECT completed_by AS member_id, count(*) AS n
		    FROM tasks
		   WHERE organization_id = $1 AND status = 'done'
		     AND completed_at >= $2 AND completed_at < $3
		     AND completed_by IS NOT NULL
		   GROUP BY completed_by
		),
		pa AS (
		  SELECT accepted_by_member_id AS member_id, count(*) AS n
		    FROM pipe_proposals
		   WHERE organization_id = $1 AND accepted_at >= $2 AND accepted_at < $3
		     AND accepted_by_member_id IS NOT NULL
		   GROUP BY accepted_by_member_id
		)
		SELECT tm.id,
		       COALESCE(la.n, 0),
		       COALESCE(tc.n, 0),
		       COALESCE(pa.n, 0)
		  FROM tm
		  LEFT JOIN la ON la.member_id = tm.id
		  LEFT JOIN tc ON tc.member_id = tm.id
		  LEFT JOIN pa ON pa.member_id = tm.id
		 WHERE COALESCE(la.n, 0) + COALESCE(tc.n, 0) + COALESCE(pa.n, 0) > 0
		 ORDER BY COALESCE(pa.n, 0) DESC, COALESCE(tc.n, 0) DESC
	`
	rows, err := r.pool.Query(ctx, q, orgID, since, until)
	if err != nil {
		return nil, fmt.Errorf("leaderboard: %w", err)
	}
	defer rows.Close()
	out := make([]MemberStats, 0, 8)
	for rows.Next() {
		var m MemberStats
		if err := rows.Scan(&m.MemberID, &m.LeadsAssigned, &m.TasksCompleted, &m.ProposalsAccepted); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Package performance persists F09 Performance data (S47).
//
// Reads aggregate live metrics (ranking, goal progress) from source
// tables (proposals, leads) rather than materialising counters —
// avoids drift between the source-of-truth and a cached projection.
// Writes land in goals/commissions/awards created in migration 0022.
package performance

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

var ErrNotFound = errors.New("performance: not found")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- goals -------------------------------------------------------

type Goal struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	MemberID       *uuid.UUID
	Metric         string
	Target         int64
	PeriodStart    time.Time
	PeriodEnd      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateGoalInput struct {
	OrganizationID uuid.UUID
	MemberID       *uuid.UUID
	Metric         string
	Target         int64
	PeriodStart    time.Time
	PeriodEnd      time.Time
	CreatedBy      *uuid.UUID
}

func (r *Repository) CreateGoal(ctx context.Context, in CreateGoalInput) (Goal, error) {
	if in.Target <= 0 {
		return Goal{}, errors.New("target must be positive")
	}
	if !in.PeriodEnd.After(in.PeriodStart) {
		return Goal{}, errors.New("period_end must be after period_start")
	}
	const q = `
		INSERT INTO goals (organization_id, member_id, metric, target, period_start, period_end, created_by)
		VALUES ($1, $2, $3::goal_metric, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	var g Goal
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.MemberID, in.Metric, in.Target,
		in.PeriodStart, in.PeriodEnd, in.CreatedBy,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return Goal{}, fmt.Errorf("create goal: %w", err)
	}
	g.OrganizationID = in.OrganizationID
	g.MemberID = in.MemberID
	g.Metric = in.Metric
	g.Target = in.Target
	g.PeriodStart = in.PeriodStart
	g.PeriodEnd = in.PeriodEnd
	return g, nil
}

func (r *Repository) ListGoals(ctx context.Context, orgID uuid.UUID) ([]Goal, error) {
	const q = `
		SELECT id, organization_id, member_id, metric::text, target,
		       period_start, period_end, created_at, updated_at
		  FROM goals
		 WHERE organization_id = $1
		 ORDER BY period_start DESC
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()
	out := make([]Goal, 0, 8)
	for rows.Next() {
		var g Goal
		if err := rows.Scan(
			&g.ID, &g.OrganizationID, &g.MemberID, &g.Metric, &g.Target,
			&g.PeriodStart, &g.PeriodEnd, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// -------- commissions -------------------------------------------------

type Commission struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProposalID     *uuid.UUID
	MemberID       uuid.UUID
	Percentage     float64
	AmountCents    int64
	Currency       string
	Status         string
	EarnedAt       time.Time
	ApprovedAt     *time.Time
	PaidAt         *time.Time
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r *Repository) ListCommissions(ctx context.Context, orgID uuid.UUID, memberID *uuid.UUID) ([]Commission, error) {
	q := `
		SELECT id, organization_id, proposal_id, member_id, percentage, amount_cents,
		       currency, status::text, earned_at, approved_at, paid_at, notes,
		       created_at, updated_at
		  FROM commissions
		 WHERE organization_id = $1
	`
	args := []any{orgID}
	if memberID != nil {
		q += " AND member_id = $2"
		args = append(args, *memberID)
	}
	q += " ORDER BY earned_at DESC LIMIT 200"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list commissions: %w", err)
	}
	defer rows.Close()
	out := make([]Commission, 0, 16)
	for rows.Next() {
		var c Commission
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.ProposalID, &c.MemberID, &c.Percentage,
			&c.AmountCents, &c.Currency, &c.Status, &c.EarnedAt,
			&c.ApprovedAt, &c.PaidAt, &c.Notes, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetCommissionStatus advances the lifecycle. Admin-only endpoint.
func (r *Repository) SetCommissionStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	if status != "approved" && status != "paid" && status != "cancelled" && status != "pending" {
		return fmt.Errorf("invalid status: %s", status)
	}
	// Stamp approved_at / paid_at when transitioning to those states.
	stamp := ""
	switch status {
	case "approved":
		stamp = ", approved_at = COALESCE(approved_at, now())"
	case "paid":
		stamp = ", paid_at = COALESCE(paid_at, now())"
	}
	q := fmt.Sprintf(
		`UPDATE commissions SET status = $3::commission_status%s
		  WHERE organization_id = $1 AND id = $2`,
		stamp,
	)
	ct, err := r.pool.Exec(ctx, q, orgID, id, status)
	if err != nil {
		return fmt.Errorf("set commission status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- awards ------------------------------------------------------

type Award struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Title          string
	Description    *string
	CriteriaJSON   []byte
	WinnersJSON    []byte
	AwardedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateAwardInput struct {
	OrganizationID uuid.UUID
	Title          string
	Description    *string
	Criteria       json.RawMessage
	Winners        json.RawMessage
	AwardedAt      *time.Time
	CreatedBy      *uuid.UUID
}

func (r *Repository) CreateAward(ctx context.Context, in CreateAwardInput) (Award, error) {
	crit := in.Criteria
	if len(crit) == 0 {
		crit = []byte(`{}`)
	}
	winners := in.Winners
	if len(winners) == 0 {
		winners = []byte(`[]`)
	}
	const q = `
		INSERT INTO awards (organization_id, title, description, criteria_json, winners_json, awarded_at, created_by)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7)
		RETURNING id, created_at, updated_at
	`
	var a Award
	err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.Title, in.Description,
		string(crit), string(winners), in.AwardedAt, in.CreatedBy,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Award{}, fmt.Errorf("create award: %w", err)
	}
	a.OrganizationID = in.OrganizationID
	a.Title = in.Title
	a.Description = in.Description
	a.CriteriaJSON = crit
	a.WinnersJSON = winners
	a.AwardedAt = in.AwardedAt
	return a, nil
}

func (r *Repository) ListAwards(ctx context.Context, orgID uuid.UUID) ([]Award, error) {
	const q = `
		SELECT id, organization_id, title, description, criteria_json, winners_json,
		       awarded_at, created_at, updated_at
		  FROM awards
		 WHERE organization_id = $1
		 ORDER BY awarded_at DESC NULLS LAST, created_at DESC
		 LIMIT 100
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list awards: %w", err)
	}
	defer rows.Close()
	out := make([]Award, 0, 8)
	for rows.Next() {
		var a Award
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.Title, &a.Description,
			&a.CriteriaJSON, &a.WinnersJSON, &a.AwardedAt,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// -------- ranking (live aggregation) ---------------------------------

type RankingEntry struct {
	MemberID   uuid.UUID
	MemberName string
	DealsWon   int64
	Revenue    int64
}

// Ranking aggregates proposals WON per member in the window. Uses
// proposals.status = 'won' + value_cents as the revenue source. The
// join on team_members filters to active members only so deactivated
// sellers don't pollute the leaderboard.
//
// Falls back to empty rows when proposals table / status enum differs
// from expectation — the live DB path validates at runtime.
func (r *Repository) Ranking(ctx context.Context, orgID uuid.UUID, since, until time.Time) ([]RankingEntry, error) {
	const q = `
		SELECT tm.id, COALESCE(u.full_name, u.email, '?'),
		       COUNT(p.id) FILTER (WHERE p.status = 'won') AS deals_won,
		       COALESCE(SUM(p.value_cents) FILTER (WHERE p.status = 'won'), 0) AS revenue
		  FROM team_members tm
		  JOIN users u ON u.id = tm.user_id
		  LEFT JOIN proposals p
		    ON p.organization_id = tm.organization_id
		   AND p.responsible_id = tm.id
		   AND p.updated_at >= $2 AND p.updated_at < $3
		 WHERE tm.organization_id = $1
		   AND tm.is_active = true
		 GROUP BY tm.id, u.full_name, u.email
		 ORDER BY revenue DESC, deals_won DESC
	`
	rows, err := r.pool.Query(ctx, q, orgID, since, until)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ranking: %w", err)
	}
	defer rows.Close()
	out := make([]RankingEntry, 0, 16)
	for rows.Next() {
		var e RankingEntry
		if err := rows.Scan(&e.MemberID, &e.MemberName, &e.DealsWon, &e.Revenue); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

package pipe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/milennials/torque-api/internal/domain"
)

// ErrInvalidKind — pipe kind is not one of the persisted ENUM values.
var ErrInvalidKind = errors.New("pipe kind must be one of: whatsapp, confirmation, proposal, custom")

// ErrNameTaken — another pipe in the same tenant already uses this name.
var ErrNameTaken = errors.New("pipe name already exists in this organization")

// ErrStageRangePosition — delete blocked because the stage has active entries.
var ErrStageHasEntries = errors.New("stage still has active entries; move or archive them first")

var allowedKinds = map[string]bool{
	"whatsapp": true, "confirmation": true, "proposal": true, "custom": true,
}

// CreatePipeInput is the write-side shape for CreatePipe.
type CreatePipeInput struct {
	Kind      string
	Name      string
	IsDefault bool
	Position  int
}

// CreatePipe inserts a new pipe for the tenant. Only 'custom' pipes are
// typically created at runtime; the structural trio (whatsapp, confirmation,
// proposal) is seeded by bootstrap, but the endpoint allows them too so an
// org that was created without them can add them later.
func (r *Repository) CreatePipe(ctx context.Context, orgID uuid.UUID, in CreatePipeInput) (domain.Pipe, error) {
	trimmed := strings.TrimSpace(in.Name)
	if len(trimmed) < 1 || len(trimmed) > 120 {
		return domain.Pipe{}, errors.New("name must be between 1 and 120 chars")
	}
	if !allowedKinds[in.Kind] {
		return domain.Pipe{}, ErrInvalidKind
	}

	const q = `
		INSERT INTO pipes (organization_id, kind, name, is_default, position)
		VALUES ($1, $2::pipe_kind, $3, $4, $5)
		RETURNING id, created_at, updated_at, is_archived
	`
	var p domain.Pipe
	err := r.pool.QueryRow(ctx, q, orgID, in.Kind, trimmed, in.IsDefault, in.Position).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.IsArchived)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Pipe{}, ErrNameTaken
		}
		return domain.Pipe{}, fmt.Errorf("create pipe: %w", err)
	}
	p.OrganizationID = orgID
	p.Kind = in.Kind
	p.Name = trimmed
	p.IsDefault = in.IsDefault
	p.Position = in.Position
	return p, nil
}

// UpdatePipeInput is the patch shape for UpdatePipe.
type UpdatePipeInput struct {
	Name      *string
	IsDefault *bool
	Position  *int
}

// UpdatePipe applies a partial patch. Kind is immutable — callers who need to
// switch pipe behaviour should archive + create new.
func (r *Repository) UpdatePipe(ctx context.Context, orgID, id uuid.UUID, in UpdatePipeInput) (domain.Pipe, error) {
	sets := make([]string, 0, 3)
	args := []any{id, orgID}
	idx := 3
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if len(trimmed) < 1 || len(trimmed) > 120 {
			return domain.Pipe{}, errors.New("name must be between 1 and 120 chars")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, trimmed)
		idx++
	}
	if in.IsDefault != nil {
		sets = append(sets, fmt.Sprintf("is_default = $%d", idx))
		args = append(args, *in.IsDefault)
		idx++
	}
	if in.Position != nil {
		sets = append(sets, fmt.Sprintf("position = $%d", idx))
		args = append(args, *in.Position)
		// idx not bumped — Position is the last optional field on UpdatePipe.
	}
	if len(sets) == 0 {
		return r.GetPipe(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE pipes SET %s WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Pipe{}, ErrNameTaken
		}
		return domain.Pipe{}, fmt.Errorf("update pipe: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.Pipe{}, ErrNotFound
	}
	return r.GetPipe(ctx, orgID, id)
}

// GetPipe loads a single pipe regardless of archive state.
func (r *Repository) GetPipe(ctx context.Context, orgID, id uuid.UUID) (domain.Pipe, error) {
	const q = `
		SELECT id, organization_id, kind::text, name, is_default, is_archived, position,
		       created_at, updated_at
		  FROM pipes
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var p domain.Pipe
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&p.ID, &p.OrganizationID, &p.Kind, &p.Name, &p.IsDefault, &p.IsArchived, &p.Position,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Pipe{}, ErrNotFound
	}
	if err != nil {
		return domain.Pipe{}, fmt.Errorf("get pipe: %w", err)
	}
	return p, nil
}

// ArchivePipe flips is_archived=true. Active entries are preserved (frozen in
// their last stage) so history remains inspectable; the pipe simply stops
// appearing in list results.
func (r *Repository) ArchivePipe(ctx context.Context, orgID, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE pipes SET is_archived = true
		  WHERE id = $1 AND organization_id = $2 AND is_archived = false`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("archive pipe: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- stages -----------------------------------------------------

// CreateStageInput carries the shape for CreateStage.
type CreateStageInput struct {
	Name            string
	ColorToken      *string
	Position        int
	IsFinalPositive bool
	IsFinalNegative bool
}

// CreateStage inserts a new stage. `uq_pipe_stages_pipe_position` is
// DEFERRABLE INITIALLY DEFERRED so callers can reorder stages in a tx.
func (r *Repository) CreateStage(ctx context.Context, orgID, pipeID uuid.UUID, in CreateStageInput) (domain.PipeStage, error) {
	trimmed := strings.TrimSpace(in.Name)
	if len(trimmed) < 1 || len(trimmed) > 80 {
		return domain.PipeStage{}, errors.New("name must be between 1 and 80 chars")
	}
	if in.IsFinalPositive && in.IsFinalNegative {
		return domain.PipeStage{}, errors.New("stage cannot be both final-positive and final-negative")
	}
	// Ownership check — pipe must belong to tenant.
	var ok bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pipes WHERE id=$1 AND organization_id=$2)`,
		pipeID, orgID,
	).Scan(&ok); err != nil {
		return domain.PipeStage{}, fmt.Errorf("pipe ownership: %w", err)
	}
	if !ok {
		return domain.PipeStage{}, ErrNotFound
	}
	const q = `
		INSERT INTO pipe_stages
		  (organization_id, pipe_id, name, color_token, position, is_final_positive, is_final_negative)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	var s domain.PipeStage
	err := r.pool.QueryRow(ctx, q,
		orgID, pipeID, trimmed, in.ColorToken, in.Position, in.IsFinalPositive, in.IsFinalNegative,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return domain.PipeStage{}, fmt.Errorf("create stage: %w", err)
	}
	s.OrganizationID = orgID
	s.PipeID = pipeID
	s.Name = trimmed
	s.ColorToken = in.ColorToken
	s.Position = in.Position
	s.IsFinalPositive = in.IsFinalPositive
	s.IsFinalNegative = in.IsFinalNegative
	return s, nil
}

// UpdateStageInput is the patch shape.
type UpdateStageInput struct {
	Name            *string
	ColorToken      *string
	Position        *int
	IsFinalPositive *bool
	IsFinalNegative *bool
}

// UpdateStage applies a partial patch.
func (r *Repository) UpdateStage(ctx context.Context, orgID, id uuid.UUID, in UpdateStageInput) (domain.PipeStage, error) {
	sets := make([]string, 0, 5)
	args := []any{id, orgID}
	idx := 3
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if len(trimmed) < 1 || len(trimmed) > 80 {
			return domain.PipeStage{}, errors.New("name must be between 1 and 80 chars")
		}
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, trimmed)
		idx++
	}
	if in.ColorToken != nil {
		sets = append(sets, fmt.Sprintf("color_token = $%d", idx))
		args = append(args, *in.ColorToken)
		idx++
	}
	if in.Position != nil {
		sets = append(sets, fmt.Sprintf("position = $%d", idx))
		args = append(args, *in.Position)
		idx++
	}
	if in.IsFinalPositive != nil {
		sets = append(sets, fmt.Sprintf("is_final_positive = $%d", idx))
		args = append(args, *in.IsFinalPositive)
		idx++
	}
	if in.IsFinalNegative != nil {
		sets = append(sets, fmt.Sprintf("is_final_negative = $%d", idx))
		args = append(args, *in.IsFinalNegative)
		// idx not bumped — IsFinalNegative is the last optional UpdateStage field.
	}
	if len(sets) == 0 {
		return r.getStage(ctx, orgID, id)
	}
	q := fmt.Sprintf(`UPDATE pipe_stages SET %s WHERE id = $1 AND organization_id = $2`,
		strings.Join(sets, ", "))
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return domain.PipeStage{}, fmt.Errorf("update stage: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.PipeStage{}, ErrNotFound
	}
	return r.getStage(ctx, orgID, id)
}

func (r *Repository) getStage(ctx context.Context, orgID, id uuid.UUID) (domain.PipeStage, error) {
	const q = `
		SELECT id, organization_id, pipe_id, name, color_token, position,
		       is_final_positive, is_final_negative, created_at, updated_at
		  FROM pipe_stages
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var s domain.PipeStage
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&s.ID, &s.OrganizationID, &s.PipeID, &s.Name, &s.ColorToken, &s.Position,
		&s.IsFinalPositive, &s.IsFinalNegative, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PipeStage{}, ErrNotFound
	}
	if err != nil {
		return domain.PipeStage{}, fmt.Errorf("get stage: %w", err)
	}
	return s, nil
}

// DeleteStage removes an empty stage. If the stage has any active entries
// the delete is refused (ErrStageHasEntries) — callers must migrate those
// entries to another stage first.
func (r *Repository) DeleteStage(ctx context.Context, orgID, id uuid.UUID) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete stage: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var active int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM pipe_entries
		  WHERE stage_id = $1 AND organization_id = $2 AND left_at IS NULL`,
		id, orgID,
	).Scan(&active); err != nil {
		return fmt.Errorf("count active entries: %w", err)
	}
	if active > 0 {
		return ErrStageHasEntries
	}
	ct, err := tx.Exec(ctx,
		`DELETE FROM pipe_stages WHERE id = $1 AND organization_id = $2`,
		id, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete stage: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Package refresh persists opaque refresh tokens with rotation + reuse
// detection. Raw token values never touch disk; only sha256 hashes do.
package refresh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the presented token hash is absent.
var ErrNotFound = errors.New("refresh token not found")

// ErrReused is returned when a token is presented after having been consumed.
// The caller MUST revoke the entire rotation chain and force a fresh login.
var ErrReused = errors.New("refresh token reused")

// ErrExpired is returned when the token is past its expires_at.
var ErrExpired = errors.New("refresh token expired")

// ErrRevoked is returned when the token was administratively revoked
// (user logout, sibling reuse, or master action).
var ErrRevoked = errors.New("refresh token revoked")

// Token is the in-memory representation of a refresh row.
type Token struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	TeamMemberID   uuid.UUID
	TokenHash      string
	ReplacedBy     *uuid.UUID
	UsedAt         *time.Time
	RevokedAt      *time.Time
	RevokedReason  *string
	IssuedAt       time.Time
	ExpiresAt      time.Time
}

// Fingerprint bundles audit metadata captured at issue time.
type Fingerprint struct {
	UserAgent string
	ClientIP  net.IP // may be nil when behind a proxy that did not forward
}

// Repository is a pgx-backed store for refresh_tokens.
type Repository struct {
	pool *pgxpool.Pool
	now  func() time.Time // seam for deterministic tests
}

// New binds the repository to a pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, now: time.Now}
}

// Issue persists a new refresh row and returns it.
// `tokenHash` is the sha256 hex of the raw token already computed by the caller.
func (r *Repository) Issue(ctx context.Context, row Token, fp Fingerprint) (Token, error) {
	if row.TokenHash == "" {
		return Token{}, errors.New("token_hash required")
	}
	if row.ExpiresAt.IsZero() || !row.ExpiresAt.After(r.now()) {
		return Token{}, errors.New("expires_at must be in the future")
	}
	const q = `
		INSERT INTO refresh_tokens (
		  user_id, organization_id, team_member_id, token_hash,
		  user_agent, client_ip, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, issued_at
	`
	var ip any
	if fp.ClientIP != nil {
		ip = fp.ClientIP.String()
	}
	var userAgent any
	if fp.UserAgent != "" {
		userAgent = truncate(fp.UserAgent, 512)
	}
	err := r.pool.QueryRow(ctx, q,
		row.UserID, row.OrganizationID, row.TeamMemberID, row.TokenHash,
		userAgent, ip, row.ExpiresAt,
	).Scan(&row.ID, &row.IssuedAt)
	if err != nil {
		return Token{}, fmt.Errorf("insert refresh token: %w", err)
	}
	return row, nil
}

// Lookup returns the token row with the given hash.
// Returns ErrNotFound, ErrExpired, or ErrRevoked on well-known states.
// ErrReused is NOT returned here — see Consume, which is the one-shot gate.
func (r *Repository) Lookup(ctx context.Context, tokenHash string) (Token, error) {
	const q = `
		SELECT id, user_id, organization_id, team_member_id, token_hash,
		       replaced_by, used_at, revoked_at, revoked_reason,
		       issued_at, expires_at
		  FROM refresh_tokens
		 WHERE token_hash = $1
		 LIMIT 1
	`
	var t Token
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.OrganizationID, &t.TeamMemberID, &t.TokenHash,
		&t.ReplacedBy, &t.UsedAt, &t.RevokedAt, &t.RevokedReason,
		&t.IssuedAt, &t.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, ErrNotFound
	}
	if err != nil {
		return Token{}, fmt.Errorf("lookup refresh token: %w", err)
	}
	if t.RevokedAt != nil {
		return t, ErrRevoked
	}
	if !t.ExpiresAt.After(r.now()) {
		return t, ErrExpired
	}
	return t, nil
}

// Rotate consumes `oldID` and inserts a new row pointing back at it via
// replaced_by. Atomic via a single transaction.
//
// If the caller detects reuse beforehand (used_at set on lookup), call
// RevokeChain instead — Rotate does NOT detect reuse by itself.
func (r *Repository) Rotate(ctx context.Context, oldID uuid.UUID, next Token, fp Fingerprint) (Token, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Token{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// One-shot guard via conditional UPDATE: if used_at is already set, no row
	// is affected — caller treats that as reuse.
	ct, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = now() WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL`,
		oldID,
	)
	if err != nil {
		return Token{}, fmt.Errorf("mark used: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return Token{}, ErrReused
	}

	var ip any
	if fp.ClientIP != nil {
		ip = fp.ClientIP.String()
	}
	var userAgent any
	if fp.UserAgent != "" {
		userAgent = truncate(fp.UserAgent, 512)
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO refresh_tokens (
			user_id, organization_id, team_member_id, token_hash,
			user_agent, client_ip, expires_at
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, issued_at`,
		next.UserID, next.OrganizationID, next.TeamMemberID, next.TokenHash,
		userAgent, ip, next.ExpiresAt,
	).Scan(&next.ID, &next.IssuedAt)
	if err != nil {
		return Token{}, fmt.Errorf("insert next: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET replaced_by = $1 WHERE id = $2`,
		next.ID, oldID,
	); err != nil {
		return Token{}, fmt.Errorf("link replaced_by: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Token{}, fmt.Errorf("commit rotation: %w", err)
	}
	return next, nil
}

// RevokeChain walks the rotation chain starting at `startID` in both directions
// and marks every unrevoked row as revoked with `reason`.
// Called on reuse detection; also called by logout for the single-row case.
func (r *Repository) RevokeChain(ctx context.Context, startID uuid.UUID, reason string) error {
	// Walk with a recursive CTE: from startID, follow replaced_by forward and
	// predecessors backward. Bounded by the chain depth (usually 1-2 hops).
	const q = `
		WITH RECURSIVE lineage(id) AS (
		  SELECT id FROM refresh_tokens WHERE id = $1
		  UNION
		  SELECT r.id
		    FROM refresh_tokens r
		    JOIN lineage l ON l.id = r.replaced_by OR r.id = (
		      SELECT replaced_by FROM refresh_tokens WHERE id = l.id
		    )
		)
		UPDATE refresh_tokens
		   SET revoked_at = now(),
		       revoked_reason = $2
		 WHERE id IN (SELECT id FROM lineage)
		   AND revoked_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, q, startID, truncate(reason, 80)); err != nil {
		return fmt.Errorf("revoke chain: %w", err)
	}
	return nil
}

// RevokeByUser marks every unused, unrevoked token for a user as revoked.
// Used on force-logout-all flows (password change, suspicious activity).
func (r *Repository) RevokeByUser(ctx context.Context, userID uuid.UUID, reason string) error {
	const q = `
		UPDATE refresh_tokens
		   SET revoked_at = now(), revoked_reason = $2
		 WHERE user_id = $1
		   AND revoked_at IS NULL
		   AND used_at IS NULL
	`
	if _, err := r.pool.Exec(ctx, q, userID, truncate(reason, 80)); err != nil {
		return fmt.Errorf("revoke by user: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

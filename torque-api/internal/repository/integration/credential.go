// Package integration persists encrypted third-party credentials
// per (organization, provider). Every row's access + refresh tokens are
// sealed with AES-256-GCM (internal/service/crypto) before insert and
// decrypted only on Get. List returns metadata (no plaintext) for the
// settings UI; MarkSuccess/MarkError drive the operator-visible health
// signal on each integration.
//
// Multi-tenancy invariant: every statement filters by organization_id.
// No call path can read a row without the org scope — the UNIQUE
// (organization_id, provider) carries the integrity guarantee; the
// repository carries the access guarantee.
package integration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/service/crypto"
)

// ErrNotFound is the sentinel for a missing (org, provider) row.
var ErrNotFound = errors.New("integration credential: not found")

// Supported providers (mirrors the DB CHECK). Exported for handler-level
// validation so an unknown provider fails at the edge, not in the DB.
const (
	ProviderGoogle  = "google"
	ProviderTinyERP = "tinyerp"
	ProviderMeta    = "meta"
)

// IsValidProvider reports whether p matches a provider accepted by the
// DB CHECK constraint. Kept in the repository package so the invariant
// lives next to the schema it mirrors.
func IsValidProvider(p string) bool {
	switch p {
	case ProviderGoogle, ProviderTinyERP, ProviderMeta:
		return true
	}
	return false
}

// Credential is the full decrypted row. AccessToken / RefreshToken are
// plaintext; callers MUST NOT log either.
type Credential struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	Provider          string
	AccessToken       string
	RefreshToken      string
	TokenType         string
	ExpiresAt         *time.Time
	Scopes            []string
	ExternalAccountID *string
	LastSuccessAt     *time.Time
	LastErrorText     *string
	LastErrorAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CredentialMeta is the no-secret projection used by List — safe to
// serialize into an HTTP response.
type CredentialMeta struct {
	Provider          string
	ExternalAccountID *string
	Scopes            []string
	ExpiresAt         *time.Time
	LastSuccessAt     *time.Time
	LastErrorText     *string
	LastErrorAt       *time.Time
}

// UpsertInput is the write shape. Tokens are plaintext; the repository
// seals them before INSERT.
type UpsertInput struct {
	AccessToken       string
	RefreshToken      string
	TokenType         string
	ExpiresAt         *time.Time
	Scopes            []string
	ExternalAccountID *string
}

// Store is the repository facade.
type Store struct {
	pool   *pgxpool.Pool
	cipher *crypto.Cipher
}

// NewStore binds a pool + cipher.
func NewStore(pool *pgxpool.Pool, cipher *crypto.Cipher) *Store {
	return &Store{pool: pool, cipher: cipher}
}

// Upsert inserts or updates the (org, provider) row. An empty RefreshToken
// is stored as NULL (Google's refresh token is only returned on the first
// consent; subsequent refreshes preserve the stored value by passing the
// old refresh through the input — repository has no way to know the
// caller's intent and treats empty as "clear"). Callers must read-modify-
// write when they want to retain the previous refresh on an access-only
// rotation.
func (s *Store) Upsert(ctx context.Context, orgID uuid.UUID, provider string, in UpsertInput) error {
	if !IsValidProvider(provider) {
		return fmt.Errorf("integration: unknown provider %q", provider)
	}
	if in.AccessToken == "" {
		return errors.New("integration: access_token required")
	}
	tokenType := in.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	accessEnc, err := s.cipher.Encrypt([]byte(in.AccessToken))
	if err != nil {
		return fmt.Errorf("encrypt access: %w", err)
	}
	var refreshEnc []byte
	if in.RefreshToken != "" {
		refreshEnc, err = s.cipher.Encrypt([]byte(in.RefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt refresh: %w", err)
		}
	}

	const q = `
		INSERT INTO integration_credentials (
		  organization_id, provider,
		  access_token_encrypted, refresh_token_encrypted,
		  token_type, expires_at, scopes, external_account_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (organization_id, provider) DO UPDATE SET
		  access_token_encrypted  = EXCLUDED.access_token_encrypted,
		  refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
		  token_type              = EXCLUDED.token_type,
		  expires_at              = EXCLUDED.expires_at,
		  scopes                  = EXCLUDED.scopes,
		  external_account_id     = EXCLUDED.external_account_id,
		  -- reset error-health on a successful auth rotation
		  last_error_text         = NULL,
		  last_error_at           = NULL
	`
	if _, err := s.pool.Exec(ctx, q,
		orgID, provider,
		accessEnc, refreshEnc,
		tokenType, in.ExpiresAt, in.Scopes, in.ExternalAccountID,
	); err != nil {
		return fmt.Errorf("upsert integration credential: %w", err)
	}
	return nil
}

// Get fetches and decrypts the row. Returns ErrNotFound when no row exists.
func (s *Store) Get(ctx context.Context, orgID uuid.UUID, provider string) (*Credential, error) {
	if !IsValidProvider(provider) {
		return nil, fmt.Errorf("integration: unknown provider %q", provider)
	}
	const q = `
		SELECT id, organization_id, provider,
		       access_token_encrypted, refresh_token_encrypted,
		       token_type, expires_at, scopes, external_account_id,
		       last_success_at, last_error_text, last_error_at,
		       created_at, updated_at
		  FROM integration_credentials
		 WHERE organization_id = $1 AND provider = $2
		 LIMIT 1
	`
	var (
		c             Credential
		accessEnc     []byte
		refreshEnc    []byte
	)
	err := s.pool.QueryRow(ctx, q, orgID, provider).Scan(
		&c.ID, &c.OrganizationID, &c.Provider,
		&accessEnc, &refreshEnc,
		&c.TokenType, &c.ExpiresAt, &c.Scopes, &c.ExternalAccountID,
		&c.LastSuccessAt, &c.LastErrorText, &c.LastErrorAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get integration credential: %w", err)
	}

	accessPT, err := s.cipher.Decrypt(accessEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt access: %w", err)
	}
	c.AccessToken = string(accessPT)

	if len(refreshEnc) > 0 {
		refreshPT, err := s.cipher.Decrypt(refreshEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypt refresh: %w", err)
		}
		c.RefreshToken = string(refreshPT)
	}
	return &c, nil
}

// List returns metadata for every provider in an org (no plaintext). The
// Integrations page reads this to render connection status.
func (s *Store) List(ctx context.Context, orgID uuid.UUID) ([]CredentialMeta, error) {
	const q = `
		SELECT provider, external_account_id, scopes, expires_at,
		       last_success_at, last_error_text, last_error_at
		  FROM integration_credentials
		 WHERE organization_id = $1
		 ORDER BY provider
	`
	rows, err := s.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list integration credentials: %w", err)
	}
	defer rows.Close()
	out := make([]CredentialMeta, 0, 4)
	for rows.Next() {
		var m CredentialMeta
		if err := rows.Scan(
			&m.Provider, &m.ExternalAccountID, &m.Scopes, &m.ExpiresAt,
			&m.LastSuccessAt, &m.LastErrorText, &m.LastErrorAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkSuccess stamps last_success_at = now() and clears any prior error.
func (s *Store) MarkSuccess(ctx context.Context, orgID uuid.UUID, provider string) error {
	if !IsValidProvider(provider) {
		return fmt.Errorf("integration: unknown provider %q", provider)
	}
	const q = `
		UPDATE integration_credentials
		   SET last_success_at = now(),
		       last_error_text = NULL,
		       last_error_at   = NULL
		 WHERE organization_id = $1 AND provider = $2
	`
	ct, err := s.pool.Exec(ctx, q, orgID, provider)
	if err != nil {
		return fmt.Errorf("mark integration success: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkError records a provider error. Text is truncated to 2000 chars to
// match the DB CHECK.
func (s *Store) MarkError(ctx context.Context, orgID uuid.UUID, provider, errText string) error {
	if !IsValidProvider(provider) {
		return fmt.Errorf("integration: unknown provider %q", provider)
	}
	if len(errText) > 2000 {
		errText = errText[:2000]
	}
	const q = `
		UPDATE integration_credentials
		   SET last_error_text = $3,
		       last_error_at   = now()
		 WHERE organization_id = $1 AND provider = $2
	`
	ct, err := s.pool.Exec(ctx, q, orgID, provider, errText)
	if err != nil {
		return fmt.Errorf("mark integration error: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the row entirely. Used by "Disconnect".
func (s *Store) Delete(ctx context.Context, orgID uuid.UUID, provider string) error {
	if !IsValidProvider(provider) {
		return fmt.Errorf("integration: unknown provider %q", provider)
	}
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM integration_credentials WHERE organization_id = $1 AND provider = $2`,
		orgID, provider,
	)
	if err != nil {
		return fmt.Errorf("delete integration credential: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

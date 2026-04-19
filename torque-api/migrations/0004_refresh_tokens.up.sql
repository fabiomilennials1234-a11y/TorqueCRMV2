-- =====================================================================
-- 0004_refresh_tokens.up.sql
-- Refresh-token storage for JWT session rotation with reuse detection.
--
-- Design:
--   * Opaque tokens (random 256-bit), never JWTs. The binary never trusts
--     the client value — only the hash is stored, verified with constant-time
--     compare at refresh time.
--   * Rotation: every /auth/refresh mints a new token and marks the prior
--     row as used. A reused token flags a compromise — we revoke the whole
--     chain via `replaced_by` walk.
--   * Per-org binding: a refresh token carries `organization_id` so a user
--     cannot pivot tenants via refresh. Switching orgs requires full login.
-- =====================================================================

CREATE TABLE refresh_tokens (
  id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id            uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  organization_id    uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  team_member_id     uuid        NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,

  -- sha256(token) hex. Never store the raw token.
  token_hash         text        NOT NULL UNIQUE CHECK (char_length(token_hash) = 64),

  -- Rotation chain. Lets us revoke a whole lineage when reuse is detected.
  replaced_by        uuid        NULL REFERENCES refresh_tokens(id) ON DELETE SET NULL,

  -- Set when /auth/refresh consumes this row (one-shot). Reuse after this
  -- is set is the compromise signal.
  used_at            timestamptz NULL,

  -- Revocation (manual logout, reuse detected on a sibling, admin kill).
  revoked_at         timestamptz NULL,
  revoked_reason     text        NULL CHECK (revoked_reason IS NULL OR char_length(revoked_reason) <= 80),

  -- Fingerprint. Not load-bearing for auth decisions, but audit-friendly.
  user_agent         text        NULL CHECK (user_agent IS NULL OR char_length(user_agent) <= 512),
  client_ip          inet        NULL,

  issued_at          timestamptz NOT NULL DEFAULT now(),
  expires_at         timestamptz NOT NULL,

  CONSTRAINT ck_refresh_tokens_expires_future CHECK (expires_at > issued_at)
);

-- Hot lookup path: verifying a presented token.
-- token_hash is UNIQUE already, so this is just a reminder that it is the seek key.

-- Session listing / revoke-by-user flows.
CREATE INDEX idx_refresh_tokens_user_active
  ON refresh_tokens (user_id, issued_at DESC)
  WHERE used_at IS NULL AND revoked_at IS NULL;

-- Janitor: expire old rows. Partial index keeps it tight.
CREATE INDEX idx_refresh_tokens_expires_at
  ON refresh_tokens (expires_at)
  WHERE used_at IS NULL AND revoked_at IS NULL;

-- Org/member audit walk.
CREATE INDEX idx_refresh_tokens_org_member
  ON refresh_tokens (organization_id, team_member_id);

COMMENT ON TABLE refresh_tokens IS
  'Opaque refresh tokens (hashed). One-shot rotation with reuse detection; org-bound.';
COMMENT ON COLUMN refresh_tokens.token_hash IS
  'sha256 hex of the raw token. Raw value is never persisted.';
COMMENT ON COLUMN refresh_tokens.replaced_by IS
  'Next token in the rotation chain; NULL for head. Used to revoke a lineage on reuse.';
COMMENT ON COLUMN refresh_tokens.used_at IS
  'Set on first consumption. Presenting the token again after this flags compromise.';

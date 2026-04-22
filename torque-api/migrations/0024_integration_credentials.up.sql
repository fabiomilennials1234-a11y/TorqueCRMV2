-- =====================================================================
-- 0024_integration_credentials.up.sql
-- S49 / Fase F.1 — Google Calendar real + TinyERP foundation.
--
-- Stores per-(organization, provider) OAuth tokens and API keys, encrypted
-- at rest with AES-256-GCM. The cipher key comes from the process env
-- (INTEGRATION_ENCRYPTION_KEY, base64 32 bytes) — never committed, never
-- logged. Plaintext never leaves the repository boundary.
--
-- 'meta' is pre-declared in the provider CHECK so the S50 Meta Ads wiring
-- does not require a schema churn; callers just start using it.
-- =====================================================================

CREATE TABLE integration_credentials (
  id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id         uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider                text        NOT NULL CHECK (provider IN ('google','tinyerp','meta')),
  access_token_encrypted  bytea       NOT NULL,
  refresh_token_encrypted bytea       NULL,
  token_type              text        NOT NULL DEFAULT 'Bearer',
  expires_at              timestamptz NULL,
  scopes                  text[]      NULL,
  external_account_id     text        NULL,
  last_success_at         timestamptz NULL,
  last_error_text         text        NULL
                                      CHECK (last_error_text IS NULL
                                             OR char_length(last_error_text) <= 2000),
  last_error_at           timestamptz NULL,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_integration_credentials_org_provider
    UNIQUE (organization_id, provider)
);

CREATE INDEX idx_integration_credentials_org_last_success
  ON integration_credentials (organization_id, last_success_at DESC)
  WHERE last_success_at IS NOT NULL;

CREATE TRIGGER trg_integration_credentials_updated_at
  BEFORE UPDATE ON integration_credentials
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE integration_credentials IS
  'S49 — encrypted OAuth tokens + API keys per (org, provider). AES-GCM key in INTEGRATION_ENCRYPTION_KEY env. S50 will use meta provider.';

-- ---------- permission catalog seeds ----------------------------------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('integrations.view',   false, false, true,  'Ver status das integrações conectadas.'),
  ('integrations.manage', true,  false, false, 'Conectar, desconectar e usar integrações externas (admin).')
ON CONFLICT (feature_key) DO NOTHING;

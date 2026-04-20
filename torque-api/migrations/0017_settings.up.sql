-- =====================================================================
-- 0017_settings.up.sql
-- S25 — F15 Configurações (consolidation).
--
-- Adds org-level fields the UI has been showing as static content:
--   - display_name (vs slug/name which is commercial identity)
--   - timezone (IANA)
--   - cnpj / legal name
-- Adds `webhook_endpoints` for F15 webhooks tab + per-member
-- `notification_preferences` for the notifications tab.
-- =====================================================================

ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS timezone   text NULL CHECK (timezone IS NULL OR char_length(timezone) <= 64),
  ADD COLUMN IF NOT EXISTS cnpj       text NULL CHECK (cnpj IS NULL OR char_length(cnpj) <= 20),
  ADD COLUMN IF NOT EXISTS legal_name text NULL CHECK (legal_name IS NULL OR char_length(legal_name) <= 200);

-- Webhook destinations. Each row is an HTTPS endpoint the org wants to
-- receive event notifications on; event_types is an array of wildcard
-- patterns (e.g. 'lead.*', 'pipe_entry.moved').
CREATE TABLE webhook_endpoints (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  url              text        NOT NULL CHECK (url ~* '^https://'),
  description      text        NULL CHECK (description IS NULL OR char_length(description) <= 500),
  event_types      text[]      NOT NULL DEFAULT ARRAY[]::text[]
                               CHECK (array_length(event_types, 1) IS NULL OR array_length(event_types, 1) <= 100),
  secret           text        NOT NULL CHECK (char_length(secret) BETWEEN 16 AND 200),
  is_active        bool        NOT NULL DEFAULT true,
  last_success_at  timestamptz NULL,
  last_failure_at  timestamptz NULL,
  last_error       text        NULL,
  created_by       uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_webhook_endpoints_org_url UNIQUE (organization_id, url)
);

CREATE INDEX idx_webhook_endpoints_org_active
  ON webhook_endpoints (organization_id)
  WHERE is_active = true;

CREATE TRIGGER trg_webhook_endpoints_updated_at
  BEFORE UPDATE ON webhook_endpoints
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE webhook_endpoints IS 'F15 — per-tenant outbound webhook destinations. Secret is for HMAC signature.';

-- Notification prefs per member + channel. Canonical channels: email,
-- in_app, push. Opt-out writes enabled=false; absent row = enabled=true
-- by default (resolved client-side).
CREATE TABLE notification_preferences (
  team_member_id  uuid        NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
  channel         text        NOT NULL CHECK (channel IN ('email', 'in_app', 'push')),
  topic           text        NOT NULL CHECK (char_length(topic) BETWEEN 1 AND 80),
  enabled         bool        NOT NULL DEFAULT true,
  updated_at      timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (team_member_id, channel, topic)
);

CREATE TRIGGER trg_notification_preferences_updated_at
  BEFORE UPDATE ON notification_preferences
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE notification_preferences IS 'F15 — per-member channel+topic opt-in. Absent row = default true.';

-- Permission keys for F15 settings tabs beyond what members/products seeded.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('organization.view',   false, false, true, 'Ver perfil da organização.'),
  ('organization.manage', true,  false, true, 'Editar perfil da organização (admin).'),
  ('webhooks.view',       false, false, true, 'Ver destinos de webhook.'),
  ('webhooks.manage',     true,  false, true, 'Criar, editar, desativar webhooks (admin).'),
  ('notifications.manage', false, false, true, 'Editar preferências próprias de notificação.')
ON CONFLICT (feature_key) DO NOTHING;

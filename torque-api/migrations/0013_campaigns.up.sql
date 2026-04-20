-- =====================================================================
-- 0013_campaigns.up.sql
-- F08 — Campanhas (bulk outbound messaging).
--
-- Model:
--   campaigns                — a named campaign with a template + audience query.
--   campaign_recipients      — per-lead materialization of the audience, with
--                              individual delivery status (queued/sent/failed/
--                              opted_out/skipped). Rows are created at
--                              campaign-launch time by the worker; the table
--                              is the ledger that "this lead got this
--                              campaign once" (anti-duplicate + opt-out
--                              enforcement).
-- =====================================================================

CREATE TYPE campaign_status AS ENUM (
  'draft',
  'scheduled',
  'running',
  'paused',
  'completed',
  'cancelled'
);

CREATE TABLE campaigns (
  id                 uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id    uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name               text             NOT NULL CHECK (char_length(name) BETWEEN 2 AND 160),
  description        text             NULL CHECK (description IS NULL OR char_length(description) <= 2000),
  channel_id         uuid             NULL REFERENCES channels(id) ON DELETE SET NULL,
  template_body      text             NOT NULL CHECK (char_length(template_body) BETWEEN 1 AND 16000),
  audience_query     jsonb            NOT NULL DEFAULT '{}'::jsonb,
  status             campaign_status  NOT NULL DEFAULT 'draft',
  scheduled_at       timestamptz      NULL,
  started_at        timestamptz      NULL,
  ended_at          timestamptz      NULL,
  stats_queued      int              NOT NULL DEFAULT 0 CHECK (stats_queued >= 0),
  stats_sent        int              NOT NULL DEFAULT 0 CHECK (stats_sent >= 0),
  stats_failed      int              NOT NULL DEFAULT 0 CHECK (stats_failed >= 0),
  stats_skipped     int              NOT NULL DEFAULT 0 CHECK (stats_skipped >= 0),
  created_by         uuid             NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at         timestamptz      NOT NULL DEFAULT now(),
  updated_at         timestamptz      NOT NULL DEFAULT now(),

  CONSTRAINT uq_campaigns_org_name UNIQUE (organization_id, name)
);

CREATE INDEX idx_campaigns_org_status
  ON campaigns (organization_id, status)
  WHERE status IN ('scheduled', 'running');
CREATE INDEX idx_campaigns_org_scheduled
  ON campaigns (organization_id, scheduled_at)
  WHERE status = 'scheduled' AND scheduled_at IS NOT NULL;

CREATE TRIGGER trg_campaigns_updated_at
  BEFORE UPDATE ON campaigns
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE campaign_recipient_status AS ENUM (
  'queued',
  'sent',
  'failed',
  'opted_out',
  'skipped'
);

CREATE TABLE campaign_recipients (
  id                uuid                        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  campaign_id       uuid                        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  lead_id           uuid                        NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  status            campaign_recipient_status   NOT NULL DEFAULT 'queued',
  message_id        uuid                        NULL REFERENCES messages(id) ON DELETE SET NULL,
  error_payload     jsonb                       NULL,
  sent_at           timestamptz                 NULL,
  failed_at         timestamptz                 NULL,
  created_at        timestamptz                 NOT NULL DEFAULT now(),
  updated_at        timestamptz                 NOT NULL DEFAULT now(),

  -- A lead may be targeted by the same campaign at most once. Enforces the
  -- anti-duplicate invariant without application-level locking.
  CONSTRAINT uq_campaign_recipients_campaign_lead UNIQUE (campaign_id, lead_id)
);

CREATE INDEX idx_campaign_recipients_campaign_status
  ON campaign_recipients (campaign_id, status);
CREATE INDEX idx_campaign_recipients_org_lead
  ON campaign_recipients (organization_id, lead_id);

CREATE TRIGGER trg_campaign_recipients_updated_at
  BEFORE UPDATE ON campaign_recipients
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE campaigns IS 'F08 — named outbound campaign with template + audience query.';
COMMENT ON TABLE campaign_recipients IS 'F08 — per-lead ledger of delivery status; dedup by (campaign_id, lead_id).';

-- Seed permission keys — campaigns are admin-only by default; views can
-- open up later via member overrides.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('campaigns.view',    false, false, true, 'Ver campanhas.'),
  ('campaigns.manage',  true,  false, true, 'Criar, editar, agendar campanhas (admin).'),
  ('campaigns.launch',  true,  false, true, 'Iniciar envio de campanha (admin).')
ON CONFLICT (feature_key) DO NOTHING;

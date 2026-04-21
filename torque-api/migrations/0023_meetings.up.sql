-- =====================================================================
-- 0023_meetings.up.sql
-- S48 / Fase E.3 — F13 Agenda interna.
--
-- Intencional: meetings internas só. Google Calendar sync vem em S49
-- (real OAuth + token storage + push/pull). Neste schema o `external_id`
-- fica pronto como ponto de extensão para quando o gcal adapter chegar
-- — qualquer row com external_id não-nulo é considerada espelhada do
-- provedor externo e mutations são idempotentes por (external_id, tenant).
-- =====================================================================

CREATE TYPE meeting_status AS ENUM ('scheduled', 'completed', 'cancelled', 'no_show');

CREATE TABLE meetings (
  id                uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  title             text             NOT NULL CHECK (char_length(title) BETWEEN 2 AND 200),
  description       text             NULL CHECK (description IS NULL OR char_length(description) <= 4000),
  starts_at         timestamptz      NOT NULL,
  ends_at           timestamptz      NOT NULL,
  location          text             NULL CHECK (location IS NULL OR char_length(location) <= 400),
  lead_id           uuid             NULL REFERENCES leads(id) ON DELETE SET NULL,
  owner_member_id   uuid             NULL REFERENCES team_members(id) ON DELETE SET NULL,
  status            meeting_status   NOT NULL DEFAULT 'scheduled',
  external_provider text             NULL CHECK (external_provider IS NULL OR external_provider IN ('gcal','outlook')),
  external_id       text             NULL,
  created_at        timestamptz      NOT NULL DEFAULT now(),
  updated_at        timestamptz      NOT NULL DEFAULT now(),

  CONSTRAINT chk_meeting_window CHECK (ends_at > starts_at),
  CONSTRAINT uq_meetings_external UNIQUE (external_provider, external_id) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX idx_meetings_org_starts
  ON meetings (organization_id, starts_at);

CREATE INDEX idx_meetings_org_owner
  ON meetings (organization_id, owner_member_id)
  WHERE owner_member_id IS NOT NULL;

CREATE INDEX idx_meetings_org_lead
  ON meetings (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE TRIGGER trg_meetings_updated_at
  BEFORE UPDATE ON meetings
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('meetings.view',   false, false, true, 'Ver reuniões.'),
  ('meetings.manage', false, false, true, 'Criar e editar reuniões (todos os membros podem agendar com seus próprios leads).')
ON CONFLICT (feature_key) DO NOTHING;

COMMENT ON TABLE meetings IS 'S48 — agenda interna com ponto de extensão para Google Calendar em S49.';

-- =====================================================================
-- 0002_team_members_leads_pipes.up.sql
-- Core tenant-scoped tables. Every table here carries organization_id
-- and is covered by a (organization_id, ...) index.
-- =====================================================================

-- ---------- team_members (user ↔ org ↔ role) --------------------------

CREATE TYPE team_member_role AS ENUM ('admin', 'membro');
-- Note: 'master' is NOT a team role; it lives in users_master (migration 0001).

CREATE TABLE team_members (
  id                uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id           uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role              team_member_role NOT NULL DEFAULT 'membro',
  display_name      text             NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 120),
  avatar_url        text             NULL,
  is_active         bool             NOT NULL DEFAULT true,
  invited_by        uuid             NULL REFERENCES team_members(id) ON DELETE SET NULL,
  invited_at        timestamptz      NULL,
  joined_at         timestamptz      NOT NULL DEFAULT now(),
  deactivated_at    timestamptz      NULL,
  metadata          jsonb            NOT NULL DEFAULT '{}'::jsonb,
  created_at        timestamptz      NOT NULL DEFAULT now(),
  updated_at        timestamptz      NOT NULL DEFAULT now(),

  CONSTRAINT uq_team_members_org_user UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_team_members_org_active
  ON team_members (organization_id, is_active) WHERE is_active = true;

CREATE INDEX idx_team_members_user
  ON team_members (user_id);

CREATE TRIGGER trg_team_members_updated_at
  BEFORE UPDATE ON team_members
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE team_members IS 'Membership: binds user to organization with a role. (organization_id, user_id) is unique.';

-- ---------- member_feature_permissions (per-member overrides) ---------

CREATE TABLE member_feature_permissions (
  team_member_id   uuid        NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
  feature_key      text        NOT NULL REFERENCES feature_permissions(feature_key) ON DELETE CASCADE,
  value            bool        NOT NULL,
  updated_by       uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  updated_at       timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (team_member_id, feature_key)
);

CREATE TRIGGER trg_member_feature_permissions_updated_at
  BEFORE UPDATE ON member_feature_permissions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE member_feature_permissions IS 'Per-member override of feature_permissions.default_value.';

-- ---------- org_quotas (delta model per Multi-tenancy.md) -------------

CREATE TABLE org_quotas (
  organization_id    uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  resource_key       text        NOT NULL CHECK (resource_key ~ '^[a-z][a-z0-9_]{1,40}$'),
  plan_base          int         NOT NULL DEFAULT 0 CHECK (plan_base >= 0),
  purchased_addons   int         NOT NULL DEFAULT 0 CHECK (purchased_addons >= 0),
  admin_adjustment   int         NOT NULL DEFAULT 0,          -- may be negative
  current_usage      int         NOT NULL DEFAULT 0 CHECK (current_usage >= 0),
  updated_at         timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (organization_id, resource_key)
);

CREATE TRIGGER trg_org_quotas_updated_at
  BEFORE UPDATE ON org_quotas
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE org_quotas IS 'Quota per tenant per resource. effective_limit = plan_base + purchased_addons + admin_adjustment.';

-- ---------- tags ------------------------------------------------------

CREATE TABLE tags (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name              text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  color_token       text        NULL,                         -- e.g. 'stage-5'
  created_at        timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_tags_org_name_ci UNIQUE (organization_id, name)
);

CREATE INDEX idx_tags_org ON tags (organization_id);

COMMENT ON TABLE tags IS 'Free-form labels per tenant. N:N with leads via lead_tags.';

-- ---------- leads -----------------------------------------------------

CREATE TABLE leads (
  id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id     uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  external_id         text        NULL,                        -- dedupe key from upstream
  name                text        NOT NULL CHECK (char_length(trim(name)) >= 2),
  company             text        NULL CHECK (char_length(company) <= 120),
  phone               text        NULL CHECK (phone IS NULL OR phone ~ '^\+[1-9][0-9]{6,14}$'),  -- E.164
  email               citext      NULL CHECK (email IS NULL OR char_length(email) <= 254),
  position            text        NULL CHECK (position IS NULL OR char_length(position) <= 120),
  cnpj_cpf            text        NULL CHECK (cnpj_cpf IS NULL OR char_length(cnpj_cpf) <= 20),
  responsible_id      uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  sdr_id              uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  closer_id           uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  rating              smallint    NULL CHECK (rating BETWEEN 1 AND 5),
  qualification_score smallint    NULL CHECK (qualification_score BETWEEN 0 AND 100),
  segment             text        NULL,
  origin              text        NULL,
  utm_source          text        NULL,
  utm_medium          text        NULL,
  utm_campaign        text        NULL,
  utm_term            text        NULL,
  utm_content         text        NULL,
  custom_fields       jsonb       NOT NULL DEFAULT '{}'::jsonb,
  first_response_at   timestamptz NULL,
  last_interaction_at timestamptz NULL,
  deleted_at          timestamptz NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT chk_leads_phone_or_email CHECK (phone IS NOT NULL OR email IS NOT NULL),
  CONSTRAINT uq_leads_org_external_id UNIQUE (organization_id, external_id) DEFERRABLE INITIALLY IMMEDIATE
);

-- Tenant-first compound indexes.
CREATE INDEX idx_leads_org_updated_at
  ON leads (organization_id, updated_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_leads_org_responsible
  ON leads (organization_id, responsible_id)
  WHERE deleted_at IS NULL AND responsible_id IS NOT NULL;

CREATE INDEX idx_leads_org_email
  ON leads (organization_id, email)
  WHERE deleted_at IS NULL AND email IS NOT NULL;

CREATE INDEX idx_leads_org_phone
  ON leads (organization_id, phone)
  WHERE deleted_at IS NULL AND phone IS NOT NULL;

CREATE TRIGGER trg_leads_updated_at
  BEFORE UPDATE ON leads
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE leads IS 'Central domain entity. Every write filters by organization_id first.';

-- ---------- lead_tags (N:N) -------------------------------------------

CREATE TABLE lead_tags (
  lead_id          uuid NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  tag_id           uuid NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
  organization_id  uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_at       timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (lead_id, tag_id)
);

CREATE INDEX idx_lead_tags_org_tag ON lead_tags (organization_id, tag_id);

-- ---------- pipes + pipe_stages + pipe_entries ------------------------

CREATE TYPE pipe_kind AS ENUM ('whatsapp', 'confirmation', 'proposal', 'custom');

CREATE TABLE pipes (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  kind              pipe_kind   NOT NULL,
  name              text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
  is_default        bool        NOT NULL DEFAULT false,
  is_archived       bool        NOT NULL DEFAULT false,
  position          int         NOT NULL DEFAULT 0,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_pipes_org_name UNIQUE (organization_id, name)
);

CREATE INDEX idx_pipes_org ON pipes (organization_id) WHERE is_archived = false;

CREATE TRIGGER trg_pipes_updated_at
  BEFORE UPDATE ON pipes
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE pipe_stages (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  pipe_id           uuid        NOT NULL REFERENCES pipes(id) ON DELETE CASCADE,
  name              text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
  color_token       text        NULL,                         -- e.g. 'stage-3'
  position          int         NOT NULL,
  is_final_positive bool        NOT NULL DEFAULT false,
  is_final_negative bool        NOT NULL DEFAULT false,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT chk_pipe_stages_final_xor CHECK (NOT (is_final_positive AND is_final_negative)),
  CONSTRAINT uq_pipe_stages_pipe_position UNIQUE (pipe_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_pipe_stages_org_pipe ON pipe_stages (organization_id, pipe_id, position);

CREATE TRIGGER trg_pipe_stages_updated_at
  BEFORE UPDATE ON pipe_stages
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE pipe_entries (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  pipe_id           uuid        NOT NULL REFERENCES pipes(id) ON DELETE CASCADE,
  stage_id          uuid        NOT NULL REFERENCES pipe_stages(id) ON DELETE RESTRICT,
  lead_id           uuid        NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  entered_stage_at  timestamptz NOT NULL DEFAULT now(),
  left_at           timestamptz NULL,                         -- non-null when finalized
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipe_entries_org_lead ON pipe_entries (organization_id, lead_id);
CREATE INDEX idx_pipe_entries_org_pipe_stage
  ON pipe_entries (organization_id, pipe_id, stage_id)
  WHERE left_at IS NULL;

CREATE TRIGGER trg_pipe_entries_updated_at
  BEFORE UPDATE ON pipe_entries
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE pipe_entries IS 'A lead is in one stage per pipe at a time. left_at IS NULL = currently active.';

-- ---------- lead_history (audit per lead) -----------------------------

CREATE TABLE lead_history (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  lead_id          uuid        NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  actor_id         uuid        NULL REFERENCES team_members(id) ON DELETE SET NULL,
  action           text        NOT NULL,                      -- 'created' | 'updated' | 'assigned' | ...
  before           jsonb       NULL,
  after            jsonb       NULL,
  created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_lead_history_org_lead
  ON lead_history (organization_id, lead_id, created_at DESC);

-- ---------- audit_log (org-wide) --------------------------------------

CREATE TABLE audit_log (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid        NULL REFERENCES organizations(id) ON DELETE SET NULL,
  target_org_id    uuid        NULL REFERENCES organizations(id) ON DELETE SET NULL,
  actor_type       text        NOT NULL CHECK (actor_type IN ('admin', 'membro', 'master', 'system')),
  actor_user_id    uuid        NULL REFERENCES users(id) ON DELETE SET NULL,
  action           text        NOT NULL CHECK (char_length(action) <= 80),
  entity_type      text        NULL,
  entity_id        uuid        NULL,
  payload          jsonb       NOT NULL DEFAULT '{}'::jsonb,
  request_id       text        NULL,
  created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_org_created
  ON audit_log (organization_id, created_at DESC);

CREATE INDEX idx_audit_log_target_org_created
  ON audit_log (target_org_id, created_at DESC)
  WHERE target_org_id IS NOT NULL;

COMMENT ON TABLE audit_log IS 'Append-only. Master impersonation writes actor_type=master + target_org_id.';

-- ---------- seed default feature_permissions for leads ----------------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('leads.view',    false, false, true,  'Ver leads da organização.'),
  ('leads.create',  false, false, true,  'Criar leads.'),
  ('leads.update',  false, false, true,  'Editar leads.'),
  ('leads.delete',  true,  false, true,  'Deletar leads (admin).'),
  ('leads.assign',  false, false, true,  'Atribuir responsável/SDR/closer.'),
  ('tags.manage',   true,  false, true,  'Criar/remover tags (admin).'),
  ('pipes.manage',  true,  false, true,  'Gerenciar funis e stages (admin).'),
  ('billing.manage', true, true,  true,  'Gerenciar plano e faturamento (admin; bloqueado em trial).')
ON CONFLICT (feature_key) DO NOTHING;

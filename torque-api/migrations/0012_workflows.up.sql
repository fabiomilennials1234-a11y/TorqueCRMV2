-- =====================================================================
-- 0012_workflows.up.sql
-- F07 — Workflow Builder.
--
-- Model:
--   workflows       — named automation per tenant (trigger + status).
--   workflow_steps  — ordered DAG nodes. Each step has `kind`
--                     (send_message, wait, create_task, branch, etc.) and
--                     typed `config` jsonb. `next_step_ids` is text[] of
--                     uuids to support branching (multiple outgoing edges).
--   workflow_runs   — one execution of the workflow against a lead.
--   workflow_run_steps — per-step execution trace (status + payload).
-- =====================================================================

CREATE TYPE workflow_trigger AS ENUM (
  'manual',
  'lead_created',
  'lead_stage_changed',
  'message_inbound',
  'schedule'
);

CREATE TYPE workflow_status AS ENUM ('draft', 'active', 'paused', 'archived');

CREATE TABLE workflows (
  id                uuid              PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid              NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name              text              NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  description       text              NULL CHECK (description IS NULL OR char_length(description) <= 2000),
  trigger           workflow_trigger  NOT NULL,
  trigger_config    jsonb             NOT NULL DEFAULT '{}'::jsonb,
  status            workflow_status   NOT NULL DEFAULT 'draft',
  entry_step_id     uuid              NULL,
  created_by        uuid              NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at        timestamptz       NOT NULL DEFAULT now(),
  updated_at        timestamptz       NOT NULL DEFAULT now(),

  CONSTRAINT uq_workflows_org_name UNIQUE (organization_id, name)
);

CREATE INDEX idx_workflows_org_status ON workflows (organization_id, status);

CREATE TRIGGER trg_workflows_updated_at
  BEFORE UPDATE ON workflows
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE workflow_step_kind AS ENUM (
  'send_message',
  'wait',
  'create_task',
  'branch',
  'update_lead',
  'call_agent',
  'http'
);

CREATE TABLE workflow_steps (
  id                uuid                PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  workflow_id       uuid                NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  kind              workflow_step_kind  NOT NULL,
  name              text                NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
  config            jsonb               NOT NULL DEFAULT '{}'::jsonb,
  next_step_ids     uuid[]              NOT NULL DEFAULT '{}',
  position_x        int                 NULL,
  position_y        int                 NULL,
  created_at        timestamptz         NOT NULL DEFAULT now(),
  updated_at        timestamptz         NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_steps_org_workflow
  ON workflow_steps (organization_id, workflow_id);

CREATE TRIGGER trg_workflow_steps_updated_at
  BEFORE UPDATE ON workflow_steps
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The entry_step_id FK closes the cycle between workflows and its first step.
ALTER TABLE workflows
  ADD CONSTRAINT fk_workflows_entry_step
  FOREIGN KEY (entry_step_id) REFERENCES workflow_steps(id) ON DELETE SET NULL;

CREATE TYPE workflow_run_status AS ENUM (
  'pending',
  'running',
  'succeeded',
  'failed',
  'cancelled'
);

CREATE TABLE workflow_runs (
  id                uuid                  PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                  NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  workflow_id       uuid                  NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  lead_id           uuid                  NULL REFERENCES leads(id) ON DELETE SET NULL,
  triggered_by      uuid                  NULL REFERENCES team_members(id) ON DELETE SET NULL,
  trigger_source    text                  NOT NULL CHECK (char_length(trigger_source) <= 80),
  status            workflow_run_status   NOT NULL DEFAULT 'pending',
  current_step_id   uuid                  NULL REFERENCES workflow_steps(id) ON DELETE SET NULL,
  input             jsonb                 NOT NULL DEFAULT '{}'::jsonb,
  result            jsonb                 NULL,
  error_payload     jsonb                 NULL,
  started_at        timestamptz           NULL,
  ended_at          timestamptz           NULL,
  created_at        timestamptz           NOT NULL DEFAULT now(),
  updated_at        timestamptz           NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_runs_org_status
  ON workflow_runs (organization_id, status)
  WHERE status IN ('pending','running');
CREATE INDEX idx_workflow_runs_org_workflow_created
  ON workflow_runs (organization_id, workflow_id, created_at DESC);
CREATE INDEX idx_workflow_runs_org_lead
  ON workflow_runs (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE TRIGGER trg_workflow_runs_updated_at
  BEFORE UPDATE ON workflow_runs
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE workflow_run_steps (
  id                uuid                PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  run_id            uuid                NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  step_id           uuid                NOT NULL REFERENCES workflow_steps(id) ON DELETE RESTRICT,
  status            workflow_run_status NOT NULL DEFAULT 'pending',
  input             jsonb               NOT NULL DEFAULT '{}'::jsonb,
  output            jsonb               NULL,
  error_payload     jsonb               NULL,
  started_at        timestamptz         NULL,
  ended_at          timestamptz         NULL,
  created_at        timestamptz         NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_run_steps_run_created
  ON workflow_run_steps (run_id, created_at);

COMMENT ON TABLE workflows IS 'F07 — named automation per tenant.';
COMMENT ON TABLE workflow_steps IS 'F07 — DAG node. next_step_ids enables branching.';
COMMENT ON TABLE workflow_runs IS 'F07 — one execution against a lead.';
COMMENT ON TABLE workflow_run_steps IS 'F07 — per-step trace of a run.';

-- Seed admin-only permissions for workflow management.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('workflows.view',    false, false, true, 'Ver workflows.'),
  ('workflows.manage',  true,  false, true, 'Criar, editar, publicar workflows (admin).'),
  ('workflows.run',     true,  false, true, 'Disparar execucao manual de workflow (admin).')
ON CONFLICT (feature_key) DO NOTHING;

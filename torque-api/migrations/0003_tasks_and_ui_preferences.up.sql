-- =====================================================================
-- 0003_tasks_and_ui_preferences.up.sql
-- Extensão do Sistema Base: entidade Task + preferência de modo de UI.
-- Absorve e substitui a entidade legada 'followup' (nunca implementada).
-- Decidido em ADR-007. Schema operacional em
-- `Torque-dir-new/05 - Sistema Base/UI Modes - Vendedor e Gerente.md` §1.
-- =====================================================================

-- ---------- Enums ----------------------------------------------------

CREATE TYPE task_kind AS ENUM (
  'followup',
  'call',
  'qualification',
  'send_proposal',
  'confirm_meeting',
  'objection',
  'generic'
);

CREATE TYPE task_priority AS ENUM ('low', 'normal', 'high', 'urgent');

CREATE TYPE task_status AS ENUM (
  'pending',
  'in_progress',
  'done',
  'cancelled',
  'missed'
);

CREATE TYPE task_origin AS ENUM (
  'manual',
  'workflow',
  'agent',
  'rule',
  'system'
);

CREATE TYPE ui_mode AS ENUM ('manager', 'salesperson');

-- ---------- users: preferência de modo -------------------------------

ALTER TABLE users
  ADD COLUMN ui_mode_preference ui_mode NOT NULL DEFAULT 'manager';

COMMENT ON COLUMN users.ui_mode_preference IS
  'Per-user UI mode preference. Default ''manager'' at DB; backend overrides to ''salesperson'' for members on signup/invite.';

-- ---------- tabela tasks ---------------------------------------------

CREATE TABLE tasks (
  id                uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid           NOT NULL
                                   REFERENCES organizations(id) ON DELETE CASCADE,
  lead_id           uuid           NULL
                                   REFERENCES leads(id) ON DELETE SET NULL,
  assigned_to       uuid           NOT NULL
                                   REFERENCES team_members(id) ON DELETE RESTRICT,
  created_by        uuid           NULL
                                   REFERENCES team_members(id) ON DELETE SET NULL,
  kind              task_kind      NOT NULL DEFAULT 'generic',
  title             text           NOT NULL CHECK (char_length(title) BETWEEN 3 AND 200),
  description       text           NULL      CHECK (description IS NULL OR char_length(description) <= 2000),
  priority          task_priority  NOT NULL DEFAULT 'normal',
  status            task_status    NOT NULL DEFAULT 'pending',
  in_queue          bool           NOT NULL DEFAULT false,
  queue_position    int            NULL CHECK (queue_position IS NULL OR queue_position >= 1),
  due_at            timestamptz    NULL,
  started_at        timestamptz    NULL,
  completed_at      timestamptz    NULL,
  completed_by      uuid           NULL
                                   REFERENCES team_members(id) ON DELETE SET NULL,
  cancelled_at      timestamptz    NULL,
  cancelled_reason  text           NULL,
  missed_reason     text           NULL,
  origin            task_origin    NOT NULL DEFAULT 'manual',
  context           jsonb          NULL,
  result_note       text           NULL      CHECK (result_note IS NULL OR char_length(result_note) <= 4000),
  created_at        timestamptz    NOT NULL DEFAULT now(),
  updated_at        timestamptz    NOT NULL DEFAULT now(),

  -- Integridade de estado --------------------------------------------
  CONSTRAINT tasks_queue_consistency CHECK (
    (in_queue = true  AND status = 'pending' AND queue_position IS NOT NULL)
    OR
    (in_queue = false AND queue_position IS NULL)
  ),
  CONSTRAINT tasks_in_progress_requires_started CHECK (
    status <> 'in_progress' OR started_at IS NOT NULL
  ),
  CONSTRAINT tasks_done_requires_completion CHECK (
    status <> 'done'
    OR (completed_at IS NOT NULL AND completed_by IS NOT NULL)
  ),
  CONSTRAINT tasks_cancelled_requires_reason CHECK (
    status <> 'cancelled'
    OR (cancelled_at IS NOT NULL AND cancelled_reason IS NOT NULL)
  )
);

-- ---------- Índices --------------------------------------------------

-- Multi-tenancy: tudo começa por organization_id.
CREATE INDEX idx_tasks_org_assignee_status
  ON tasks (organization_id, assigned_to, status);

CREATE INDEX idx_tasks_org_lead
  ON tasks (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE INDEX idx_tasks_org_created_at
  ON tasks (organization_id, created_at DESC);

-- Fila priorizada: ordenar por queue_position em pending+in_queue.
CREATE INDEX idx_tasks_queue
  ON tasks (organization_id, assigned_to, queue_position)
  WHERE status = 'pending' AND in_queue = true;

-- SLA cron: encontrar pending com due_at vencido.
CREATE INDEX idx_tasks_due_pending
  ON tasks (organization_id, due_at)
  WHERE status = 'pending' AND due_at IS NOT NULL;

-- ---------- Constraints parciais únicas ------------------------------

-- Uma única task in_progress por (org, assignee).
CREATE UNIQUE INDEX uq_tasks_one_in_progress_per_assignee
  ON tasks (organization_id, assigned_to)
  WHERE status = 'in_progress';

-- queue_position único dentro da fila de cada assignee.
CREATE UNIQUE INDEX uq_tasks_queue_position_per_assignee
  ON tasks (organization_id, assigned_to, queue_position)
  WHERE status = 'pending' AND in_queue = true;

-- ---------- Trigger updated_at ---------------------------------------

CREATE TRIGGER trg_tasks_updated_at
  BEFORE UPDATE ON tasks
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();

-- ---------- Comentários ----------------------------------------------

COMMENT ON TABLE  tasks IS 'Unidade de trabalho atribuível; unifica follow-ups e demais tasks (ADR-007).';
COMMENT ON COLUMN tasks.in_queue IS 'True = aparece em "A fazer"; false = em "Em aberto" (backlog).';
COMMENT ON COLUMN tasks.queue_position IS 'Posição 1..N dentro da fila do assignee.';
COMMENT ON COLUMN tasks.origin IS 'Quem criou: manual (humano), workflow, agent (IA), rule (pipe rule), system.';

-- ---------- Seed de feature_permissions para tasks (ADR-007 §5) ------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('tasks.view_own',           false, false, true,  'Ver próprias tasks.'),
  ('tasks.view_team',          true,  false, true,  'Ver tasks do time (admin).'),
  ('tasks.create',             false, false, true,  'Criar task para si.'),
  ('tasks.create_for_others',  true,  false, true,  'Criar task para outro membro (admin).'),
  ('tasks.update',             false, false, true,  'Editar task própria.'),
  ('tasks.complete',           false, false, true,  'Concluir task própria.'),
  ('tasks.cancel',             false, false, true,  'Cancelar task criada por si.'),
  ('tasks.reassign',           true,  false, true,  'Reatribuir task a outro membro (admin).'),
  ('tasks.bulk_assign',        true,  false, true,  'Atribuir tasks em lote (admin).'),
  ('tasks.reopen',             true,  false, true,  'Reabrir task missed/cancelled.')
ON CONFLICT (feature_key) DO NOTHING;

-- =====================================================================
-- 0010_rbac_seeds.up.sql
-- Remediation: seed feature_permissions for every resource surface shipped
-- in S11-S15. Without these rows, RequireFeature has nothing to cascade
-- against.
--
-- Semantics (ADR: Modelo de Permissoes):
--   is_admin_only = true    → only admins pass (members denied)
--   master_only   = true    → admins cannot bypass either
--   default_value = true    → if no member_feature_permissions override,
--                             the caller passes the cascade.
-- =====================================================================

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  -- Proposals (F03) — money flow. Read is member-wide; mutations need a role gate.
  ('proposals.view',       false, false, true,  'Ver propostas.'),
  ('proposals.manage',     true,  false, true,  'Criar, editar (rascunho) e enviar propostas (admin).'),
  ('proposals.accept',     true,  false, true,  'Marcar proposta como aceita (admin).'),
  ('proposals.reject',     true,  false, true,  'Marcar proposta como rejeitada (admin).'),

  -- Inbox (F04) — send surfaces the tenant externally.
  ('inbox.view',           false, false, true,  'Ver conversas.'),
  ('inbox.reply',          false, false, true,  'Responder conversas atribuidas a si.'),
  ('inbox.manage',         true,  false, true,  'Atribuir, arquivar, resolver conversas (admin).'),

  -- Tasks (F05) — member can manage own tasks; admin manages team.
  ('tasks.view_own',       false, false, true,  'Ver proprias tarefas.'),
  ('tasks.view_team',      true,  false, true,  'Ver tarefas de todo o time (admin).'),
  ('tasks.create_for_self',false, false, true,  'Criar tarefas para si.'),
  ('tasks.create_for_others', true, false, true,'Criar tarefas para outros membros (admin).'),
  ('tasks.complete',       false, false, true,  'Completar tarefa in_progress.'),
  ('tasks.cancel',         false, false, true,  'Cancelar propria tarefa.'),

  -- Copilot (F06) — kill switch is tenant-wide blast radius, admin-only.
  ('copilot.view',         false, false, true,  'Ver agentes e sessoes.'),
  ('copilot.manage',       true,  false, true,  'Criar, editar, ativar, desativar agentes (admin).'),
  ('copilot.kill_switch',  true,  false, true,  'Acionar ou desarmar kill-switch do copilot (admin).'),
  ('knowledge.view',       false, false, true,  'Ver colecoes e fontes de conhecimento.'),
  ('knowledge.manage',     true,  false, true,  'Criar colecoes e enfileirar fontes para ingest (admin).')
ON CONFLICT (feature_key) DO NOTHING;

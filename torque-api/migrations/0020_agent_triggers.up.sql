-- =====================================================================
-- 0020_agent_triggers.up.sql
-- S40 / Fase C.4 — F06 Copilot triggers + auto-assignment.
--
-- Design:
--   * agent_triggers evaluates a tenant-scoped filter DSL against a lead
--     (or conversation's lead) and, when it matches, assigns the agent to
--     the conversation. Priority is a simple integer — lower values win
--     (1 = highest priority). Ties broken by created_at ASC so the
--     first-configured trigger wins.
--
--   * filter_json is a JSON document with the shape:
--       { "all": [ { "field": "origin", "op": "eq",    "value": "meta-ads" }, ... ],
--         "any": [ { "field": "tags",   "op": "in",    "value": ["vip","hot"] } ] }
--     Matching semantics: every predicate in `all` must pass AND at least
--     one predicate in `any` must pass (if `any` is non-empty). Empty
--     both blocks = matches everything (a "default" catch-all trigger).
--
--   * is_active flags the trigger. We soft-disable rather than delete so
--     the audit trail keeps history of which rules were in force when a
--     past assignment happened.
--
--   * Matcher honors kill_switch AND agent.status:
--       agent.status != 'active' → skipped
--       agent.kill_switch = true → skipped
--
--   * conversations.assigned_agent_id: new FK, mirrors assigned_to (human)
--     but points at agents. Both columns coexist — human takeover is
--     distinct from agent autopilot. Worker `conversation.assign_agent`
--     sets this; the inbox UI reads it to render the "agent attending"
--     badge.
--
-- Permission keys: triggers.view (all members) / triggers.manage (admin).
-- =====================================================================

-- ---------- conversations — add assigned_agent_id --------------------

ALTER TABLE conversations
  ADD COLUMN assigned_agent_id uuid NULL
    REFERENCES agents(id) ON DELETE SET NULL;

CREATE INDEX idx_conversations_org_assigned_agent
  ON conversations (organization_id, assigned_agent_id)
  WHERE assigned_agent_id IS NOT NULL;

COMMENT ON COLUMN conversations.assigned_agent_id IS
  'S40 — optional Copilot agent attending the conversation. NULL = no agent (human-only or waiting for trigger).';

-- ---------- agent_triggers -------------------------------------------

CREATE TABLE agent_triggers (
  id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  agent_id         uuid         NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  name             text         NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  description      text         NULL CHECK (description IS NULL OR char_length(description) <= 1000),
  -- Lower = higher priority. int4 keeps room for reordering without
  -- having to re-number every sibling when slotting a new rule.
  priority         int          NOT NULL DEFAULT 100 CHECK (priority BETWEEN 1 AND 10000),
  filter_json      jsonb        NOT NULL DEFAULT '{}'::jsonb,
  is_active        bool         NOT NULL DEFAULT true,
  created_by       uuid         NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at       timestamptz  NOT NULL DEFAULT now(),
  updated_at       timestamptz  NOT NULL DEFAULT now()
);

-- Ordered scan for the matcher: priority first, then deterministic tie-break.
CREATE INDEX idx_agent_triggers_org_active_priority
  ON agent_triggers (organization_id, priority, created_at)
  WHERE is_active = true;

-- Lookups from the agent editor (list-by-agent).
CREATE INDEX idx_agent_triggers_org_agent
  ON agent_triggers (organization_id, agent_id);

CREATE TRIGGER trg_agent_triggers_updated_at
  BEFORE UPDATE ON agent_triggers
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE agent_triggers IS
  'S40 — filter-based auto-assignment rules for F06 Copilot agents. Worker conversation.assign_agent evaluates these in priority order.';

-- ---------- permission seeds -----------------------------------------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('triggers.view',   false, false, true, 'Ver gatilhos de ativação automática de agents.'),
  ('triggers.manage', true,  false, true, 'Criar, editar e remover gatilhos de ativação (admin).')
ON CONFLICT (feature_key) DO NOTHING;

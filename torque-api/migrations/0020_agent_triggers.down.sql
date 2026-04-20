-- =====================================================================
-- 0020_agent_triggers.down.sql — revert S40 triggers + assigned_agent_id.
-- =====================================================================

DELETE FROM feature_permissions
 WHERE feature_key IN ('triggers.view', 'triggers.manage');

DROP INDEX IF EXISTS idx_agent_triggers_org_agent;
DROP INDEX IF EXISTS idx_agent_triggers_org_active_priority;
DROP TABLE IF EXISTS agent_triggers;

DROP INDEX IF EXISTS idx_conversations_org_assigned_agent;
ALTER TABLE conversations
  DROP COLUMN IF EXISTS assigned_agent_id;

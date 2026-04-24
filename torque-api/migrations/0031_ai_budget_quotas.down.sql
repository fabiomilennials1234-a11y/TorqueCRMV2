-- =====================================================================
-- 0031_ai_budget_quotas.down.sql
-- Removes the ai_tokens + tts_seconds rows from plan_quotas + org_quotas.
-- Safe to run repeatedly: DELETE WHERE resource_key IN (...) is idempotent.
-- =====================================================================

DELETE FROM org_quotas
 WHERE resource_key IN ('ai_tokens', 'tts_seconds');

DELETE FROM plan_quotas
 WHERE resource_key IN ('ai_tokens', 'tts_seconds');

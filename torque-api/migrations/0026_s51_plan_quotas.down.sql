-- =====================================================================
-- 0026_s51_plan_quotas.down.sql
-- Revert S51 schema additions.
-- org_quotas rows seeded by this migration are NOT removed automatically
-- (we don't know which rows pre-existed). If a full rollback is needed,
-- `TRUNCATE org_quotas CASCADE` separately.
-- =====================================================================

DELETE FROM feature_permissions WHERE feature_key = 'quotas.view';
DROP TABLE IF EXISTS plan_quotas;

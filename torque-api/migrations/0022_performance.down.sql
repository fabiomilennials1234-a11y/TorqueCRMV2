-- =====================================================================
-- 0022_performance.down.sql — revert S47 performance tables.
-- =====================================================================

DELETE FROM feature_permissions
 WHERE feature_key IN ('performance.view', 'performance.manage');

DROP TABLE IF EXISTS awards;
DROP TABLE IF EXISTS commissions;
DROP TYPE IF EXISTS commission_status;
DROP TABLE IF EXISTS goals;
DROP TYPE IF EXISTS goal_metric;

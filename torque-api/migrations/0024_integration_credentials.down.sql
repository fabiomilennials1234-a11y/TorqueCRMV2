-- =====================================================================
-- 0024_integration_credentials.down.sql — revert S49 credentials.
-- =====================================================================

DELETE FROM feature_permissions
 WHERE feature_key IN ('integrations.view', 'integrations.manage');

DROP TABLE IF EXISTS integration_credentials;

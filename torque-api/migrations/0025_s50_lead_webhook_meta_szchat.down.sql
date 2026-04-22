-- =====================================================================
-- 0025_s50_lead_webhook_meta_szchat.down.sql
-- Revert S50 schema additions.
-- =====================================================================

ALTER TABLE integration_credentials
  DROP CONSTRAINT IF EXISTS integration_credentials_provider_check;

ALTER TABLE integration_credentials
  ADD CONSTRAINT integration_credentials_provider_check
  CHECK (provider IN ('google','tinyerp','meta'));

DROP TABLE IF EXISTS meta_insights_cache;
DROP TABLE IF EXISTS lead_webhook_events;

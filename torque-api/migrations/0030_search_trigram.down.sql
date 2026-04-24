-- =====================================================================
-- 0030_search_trigram.down.sql
-- Reverse of 0030.
--
-- We do NOT drop the pg_trgm extension itself: other tenants / future
-- migrations may already depend on it and DROP EXTENSION would cascade
-- the operator class which is shared. Leaving it in place is cheap
-- (CREATE EXTENSION is idempotent for re-UP) and safer.
-- =====================================================================

DROP INDEX IF EXISTS idx_messages_body_trgm;
DROP INDEX IF EXISTS idx_leads_phone_trgm;
DROP INDEX IF EXISTS idx_leads_email_lower_trgm;
DROP INDEX IF EXISTS idx_leads_name_lower_trgm;

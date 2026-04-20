-- =====================================================================
-- 0006_pipe_confirmations.down.sql
-- =====================================================================

DROP TRIGGER IF EXISTS trg_pipe_confirmations_updated_at ON pipe_confirmations;
DROP INDEX IF EXISTS idx_pipe_confirmations_org_overdue;
DROP INDEX IF EXISTS idx_pipe_confirmations_org_lead;
DROP INDEX IF EXISTS idx_pipe_confirmations_org_meeting;
DROP TABLE IF EXISTS pipe_confirmations;

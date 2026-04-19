-- =====================================================================
-- 0005_operations.down.sql
-- =====================================================================

DROP TRIGGER IF EXISTS trg_operations_updated_at ON operations;
DROP INDEX IF EXISTS idx_operations_org_status;
DROP INDEX IF EXISTS idx_operations_kind_pending;
DROP INDEX IF EXISTS idx_operations_org_created;
DROP TABLE IF EXISTS operations;
DROP TYPE IF EXISTS operation_status;

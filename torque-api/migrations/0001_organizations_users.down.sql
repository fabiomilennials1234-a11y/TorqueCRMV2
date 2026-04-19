-- =====================================================================
-- 0001_organizations_users.down.sql
-- =====================================================================

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TABLE IF EXISTS users_master;
DROP TABLE IF EXISTS users;

DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;
DROP INDEX IF EXISTS idx_organizations_deleted_at;
DROP TABLE IF EXISTS organizations;
DROP TYPE  IF EXISTS org_payment_status;

DROP TRIGGER IF EXISTS trg_plans_updated_at ON plans;
DROP TABLE IF EXISTS plans;

DROP TABLE IF EXISTS feature_permissions;

DROP FUNCTION IF EXISTS set_updated_at();

-- Extensions are left in place; they may be used by other modules.

-- =====================================================================
-- 0014_products_and_members_perms.down.sql
-- Reverse S21 schema. Permission rows are best-effort deleted; if a
-- member override still references them the migrator will refuse (FK
-- cascade is NOT wanted here — we do not want to silently revoke overrides).
-- =====================================================================

DROP TABLE IF EXISTS products;

DELETE FROM feature_permissions
 WHERE feature_key IN ('members.view','members.manage','products.view','products.manage');

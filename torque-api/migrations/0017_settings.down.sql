-- Reverse S25.
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS webhook_endpoints;

ALTER TABLE organizations
  DROP COLUMN IF EXISTS legal_name,
  DROP COLUMN IF EXISTS cnpj,
  DROP COLUMN IF EXISTS timezone;

DELETE FROM feature_permissions
 WHERE feature_key IN ('organization.view','organization.manage',
                       'webhooks.view','webhooks.manage','notifications.manage');

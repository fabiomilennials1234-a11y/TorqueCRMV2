-- Reverse S35.
DROP TABLE IF EXISTS message_templates;
DELETE FROM feature_permissions WHERE feature_key IN ('templates.view', 'templates.manage');

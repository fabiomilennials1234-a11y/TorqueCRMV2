-- =====================================================================
-- 0023_meetings.down.sql — revert S48 meetings table.
-- =====================================================================

DELETE FROM feature_permissions
 WHERE feature_key IN ('meetings.view', 'meetings.manage');

DROP TABLE IF EXISTS meetings;
DROP TYPE IF EXISTS meeting_status;

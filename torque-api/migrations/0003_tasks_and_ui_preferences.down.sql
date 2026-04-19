-- =====================================================================
-- 0003_tasks_and_ui_preferences.down.sql
-- =====================================================================

DROP TRIGGER IF EXISTS trg_tasks_updated_at ON tasks;
DROP INDEX IF EXISTS uq_tasks_queue_position_per_assignee;
DROP INDEX IF EXISTS uq_tasks_one_in_progress_per_assignee;
DROP INDEX IF EXISTS idx_tasks_due_pending;
DROP INDEX IF EXISTS idx_tasks_queue;
DROP INDEX IF EXISTS idx_tasks_org_created_at;
DROP INDEX IF EXISTS idx_tasks_org_lead;
DROP INDEX IF EXISTS idx_tasks_org_assignee_status;
DROP TABLE IF EXISTS tasks;

ALTER TABLE users DROP COLUMN IF EXISTS ui_mode_preference;

DROP TYPE IF EXISTS ui_mode;
DROP TYPE IF EXISTS task_origin;
DROP TYPE IF EXISTS task_status;
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_kind;

DELETE FROM feature_permissions WHERE feature_key LIKE 'tasks.%';

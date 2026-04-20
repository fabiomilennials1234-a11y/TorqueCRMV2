-- Rollback: restore stricter tasks constraint, drop proposal audit columns.
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_cancelled_requires_timestamp;
ALTER TABLE tasks ADD CONSTRAINT tasks_cancelled_requires_reason
  CHECK (status <> 'cancelled' OR (cancelled_at IS NOT NULL AND cancelled_reason IS NOT NULL));

ALTER TABLE pipe_proposals
  DROP COLUMN IF EXISTS rejected_by_member_id,
  DROP COLUMN IF EXISTS accepted_by_member_id,
  DROP COLUMN IF EXISTS sent_by_member_id;

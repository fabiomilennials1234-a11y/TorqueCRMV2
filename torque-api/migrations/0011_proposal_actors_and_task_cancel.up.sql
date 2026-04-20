-- =====================================================================
-- 0011_proposal_actors_and_task_cancel.up.sql
-- Remediation:
--   (1) Audit columns on pipe_proposals — persist WHO performed each
--       lifecycle transition. Forensic requirement for money flow.
--   (2) Relax the tasks CHECK that required cancelled_reason to be
--       non-null — product decision: reason is optional, the handler
--       enforces presence when it's required by UX. DB should not 500.
-- =====================================================================

-- --- (1) proposal audit columns -------------------------------------
ALTER TABLE pipe_proposals
  ADD COLUMN IF NOT EXISTS sent_by_member_id      uuid NULL REFERENCES team_members(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS accepted_by_member_id  uuid NULL REFERENCES team_members(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS rejected_by_member_id  uuid NULL REFERENCES team_members(id) ON DELETE SET NULL;

COMMENT ON COLUMN pipe_proposals.sent_by_member_id IS
  'Team member who moved the proposal from draft → sent.';
COMMENT ON COLUMN pipe_proposals.accepted_by_member_id IS
  'Team member who accepted the proposal on the tenant''s behalf.';
COMMENT ON COLUMN pipe_proposals.rejected_by_member_id IS
  'Team member who recorded rejection. May differ from the recipient party.';

-- --- (2) relax the cancelled_reason NOT NULL check -----------------
-- Original constraint (migration 0003):
--   CHECK (status <> 'cancelled' OR (cancelled_at IS NOT NULL AND cancelled_reason IS NOT NULL))
-- We still require cancelled_at to be set on cancel, but allow reason to be NULL.
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_cancelled_requires_reason;
ALTER TABLE tasks ADD CONSTRAINT tasks_cancelled_requires_timestamp
  CHECK (status <> 'cancelled' OR cancelled_at IS NOT NULL);

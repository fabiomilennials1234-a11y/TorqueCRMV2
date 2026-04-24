-- Reverse 0028_workflow_retry.
DROP INDEX IF EXISTS idx_workflow_run_failures_org_run;
DROP TABLE IF EXISTS workflow_run_failures;

DROP INDEX IF EXISTS idx_workflow_runs_retry_scheduled;
ALTER TABLE workflow_runs
  DROP COLUMN IF EXISTS next_retry_at,
  DROP COLUMN IF EXISTS max_attempts,
  DROP COLUMN IF EXISTS attempts;

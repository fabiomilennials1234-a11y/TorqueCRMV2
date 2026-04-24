-- =====================================================================
-- 0028_workflow_retry.up.sql
-- S52 Fase G — Workflow engine real (retry + DLQ + watchdog).
--
-- Adds durable retry state and a dead-letter queue so the executor can
-- treat transient failures (Evolution 5xx, HTTP timeouts, rate-limits)
-- as "try again with backoff" instead of "fail the run forever".
--
-- Columns added to workflow_runs:
--   attempts        — how many times the current step has been tried.
--   max_attempts    — cap per run (default 3, hard ceiling 10).
--   next_retry_at   — when runner.ClaimPendingRun may re-pick the row.
--
-- New table workflow_run_failures:
--   One row per *attempt* that failed. Final attempts (attempts =
--   max_attempts) mark the run as failed and land in the DLQ. Earlier
--   attempts also land here so operators can see the retry trail.
-- =====================================================================

ALTER TABLE workflow_runs
  ADD COLUMN attempts        int          NOT NULL DEFAULT 0
    CHECK (attempts >= 0),
  ADD COLUMN max_attempts    int          NOT NULL DEFAULT 3
    CHECK (max_attempts >= 1 AND max_attempts <= 10),
  ADD COLUMN next_retry_at   timestamptz  NULL;

-- Partial index drives the runner's retry-aware claim query: we only
-- scan pending rows that have a scheduled retry, which on a tenant with
-- thousands of completed runs is a ~100x reduction in planner work.
CREATE INDEX idx_workflow_runs_retry_scheduled
  ON workflow_runs (organization_id, next_retry_at)
  WHERE status = 'pending' AND next_retry_at IS NOT NULL;

-- Dead-letter queue + retry trail. One row per failed attempt.
CREATE TABLE workflow_run_failures (
  id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  run_id           uuid         NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  workflow_id      uuid         NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  step_id          uuid         NULL REFERENCES workflow_steps(id) ON DELETE SET NULL,
  attempt          int          NOT NULL CHECK (attempt >= 1),
  error_code       text         NOT NULL CHECK (char_length(error_code) BETWEEN 1 AND 80),
  error_message    text         NOT NULL CHECK (char_length(error_message) BETWEEN 1 AND 2000),
  snapshot_json    jsonb        NOT NULL DEFAULT '{}'::jsonb,
  created_at       timestamptz  NOT NULL DEFAULT now()
);

-- Fetch failures per run, newest attempt first — drives the admin
-- dashboard + any future DLQ replay tooling.
CREATE INDEX idx_workflow_run_failures_org_run
  ON workflow_run_failures (organization_id, run_id, attempt DESC);

COMMENT ON TABLE workflow_run_failures
  IS 'S52 — per-attempt failure trail. Final attempt doubles as DLQ entry.';
COMMENT ON COLUMN workflow_runs.attempts
  IS 'S52 — executor retry counter; resets per workflow_run.';
COMMENT ON COLUMN workflow_runs.max_attempts
  IS 'S52 — retry cap. Default 3, hard ceiling 10 (CHECK).';
COMMENT ON COLUMN workflow_runs.next_retry_at
  IS 'S52 — runner reclaim timestamp; NULL means ready immediately.';

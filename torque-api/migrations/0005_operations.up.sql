-- =====================================================================
-- 0005_operations.up.sql
-- Async-job ledger for the 202-Accepted pattern (ADR-006).
--
-- Design:
--   * One row per long-running job. Handlers write PENDING, workers flip
--     to RUNNING then SUCCEEDED/FAILED; the caller polls GET /operations/:id
--     and simultaneously subscribes to WS `operation.updated` patches.
--   * Progress is a float in [0,1] so the UI has smooth fill without the
--     job needing to know totals upfront. Nullable for steps that do not
--     report progress.
--   * Result and error_payload are small jsonb envelopes. Large artifacts
--     belong in object storage; the DB keeps references, not payloads.
-- =====================================================================

CREATE TYPE operation_status AS ENUM (
  'pending',
  'running',
  'succeeded',
  'failed',
  'cancelled'
);

CREATE TABLE operations (
  id                uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  -- Who submitted. NULL for system-scheduled jobs (seeding, cron).
  actor_user_id     uuid             NULL REFERENCES users(id) ON DELETE SET NULL,
  actor_type        text             NOT NULL DEFAULT 'system'
                                     CHECK (actor_type IN ('admin','membro','master','system')),

  kind              text             NOT NULL CHECK (char_length(kind) BETWEEN 1 AND 80),
  status            operation_status NOT NULL DEFAULT 'pending',

  -- Worker fingerprinting for debugging. Set by the worker on pickup.
  worker_id         text             NULL,

  -- Inputs snapshot. Keep the shape small — big blobs go to object storage.
  input             jsonb            NOT NULL DEFAULT '{}'::jsonb,
  -- Final result envelope. NULL until the job completes.
  result            jsonb            NULL,
  -- Structured error (code + message + retryable). NULL unless status='failed'.
  error_payload     jsonb            NULL,

  progress          numeric(5,4)     NULL CHECK (progress IS NULL OR (progress >= 0 AND progress <= 1)),

  -- Retry budget. The worker decrements on retryable failures and flips
  -- to 'failed' once exhausted. Zero means no retries.
  retry_remaining   int              NOT NULL DEFAULT 0 CHECK (retry_remaining >= 0),

  scheduled_at      timestamptz      NOT NULL DEFAULT now(),
  started_at        timestamptz      NULL,
  ended_at          timestamptz      NULL,
  expires_at        timestamptz      NULL,

  created_at        timestamptz      NOT NULL DEFAULT now(),
  updated_at        timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX idx_operations_org_created
  ON operations (organization_id, created_at DESC);

-- Worker polling: pull the oldest pending row per kind.
CREATE INDEX idx_operations_kind_pending
  ON operations (kind, scheduled_at)
  WHERE status = 'pending';

-- UI list: active ops per tenant.
CREATE INDEX idx_operations_org_status
  ON operations (organization_id, status)
  WHERE status IN ('pending','running');

CREATE TRIGGER trg_operations_updated_at
  BEFORE UPDATE ON operations
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE operations IS
  'Async-job ledger (ADR-006). 202-Accepted handlers insert pending; workers claim, advance, finish.';
COMMENT ON COLUMN operations.progress IS
  'Float [0,1]; NULL when the job does not report fine-grained progress.';
COMMENT ON COLUMN operations.retry_remaining IS
  'Retries left on retryable failure. Worker decrements; on zero, status becomes failed.';

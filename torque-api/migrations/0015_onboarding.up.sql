-- =====================================================================
-- 0015_onboarding.up.sql
-- S23 — F13 Onboarding Wizard + Gate.
--
-- The wizard is a per-user, per-org state machine with 6 steps. Progress
-- is stored in one row per (user, org) keyed to team_members so we never
-- orphan state when a user leaves the tenant.
--
-- `steps_completed` is a text[] of step keys (declared below). Order is
-- canonical — when a new step is inserted in a future sprint we add it
-- here and old tenants see it as pending without needing a migration.
-- =====================================================================

CREATE TABLE onboarding_status (
  id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  team_member_id   uuid         NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
  current_step     text         NOT NULL DEFAULT 'welcome'
                                CHECK (char_length(current_step) BETWEEN 1 AND 40),
  steps_completed  text[]       NOT NULL DEFAULT ARRAY[]::text[],
  dismissed        bool         NOT NULL DEFAULT false,
  completed_at     timestamptz  NULL,
  created_at       timestamptz  NOT NULL DEFAULT now(),
  updated_at       timestamptz  NOT NULL DEFAULT now(),

  CONSTRAINT uq_onboarding_member UNIQUE (team_member_id)
);

CREATE INDEX idx_onboarding_org_pending
  ON onboarding_status (organization_id)
  WHERE completed_at IS NULL AND dismissed = false;

CREATE TRIGGER trg_onboarding_updated_at
  BEFORE UPDATE ON onboarding_status
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE onboarding_status IS 'F13 — per-member wizard state. completed_at IS NULL = gate still active.';

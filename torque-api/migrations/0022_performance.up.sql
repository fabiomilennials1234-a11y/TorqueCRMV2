-- =====================================================================
-- 0022_performance.up.sql
-- S47 / Fase E.2 — F09 Performance: goals, commissions, awards.
--
-- Design:
--   * goals — per-member or per-org target for a metric (deals_won,
--     revenue_cents, leads_contacted). Period is a closed-open range
--     (start inclusive, end exclusive) so monthly/quarterly cadence
--     reads naturally against a tz-aware timestamp column.
--   * commissions — per-deal attribution to a member with percentage
--     and computed amount_cents. Status tracks pending → approved →
--     paid lifecycle without forcing a transaction when a proposal
--     closes (the commission row lands in pending and the admin
--     approves in bulk monthly).
--   * awards — named recognition issued to a winner. Criteria stored
--     as JSON so tenants can encode "top 3 by revenue this quarter"
--     without schema changes; winners_json snapshots the result at
--     award time for audit.
--
--   Permission keys: `performance.view` (default true) +
--   `performance.manage` (admin only — create/edit goals + approve
--   commissions + issue awards).
-- =====================================================================

CREATE TYPE goal_metric AS ENUM (
  'deals_won',
  'revenue_cents',
  'leads_contacted',
  'response_time_ms',
  'first_response_minutes'
);

CREATE TABLE goals (
  id                uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  member_id         uuid         NULL REFERENCES team_members(id) ON DELETE CASCADE,
  metric            goal_metric  NOT NULL,
  target            bigint       NOT NULL CHECK (target > 0),
  period_start      timestamptz  NOT NULL,
  period_end        timestamptz  NOT NULL,
  -- Current progress is NOT stored here — aggregated on demand from
  -- deals/conversations/messages. This avoids a drift hazard between
  -- the source event tables and a materialized counter.
  created_by        uuid         NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at        timestamptz  NOT NULL DEFAULT now(),
  updated_at        timestamptz  NOT NULL DEFAULT now(),

  CONSTRAINT chk_goals_period CHECK (period_end > period_start)
);

CREATE INDEX idx_goals_org_period
  ON goals (organization_id, period_start DESC, period_end DESC);

CREATE INDEX idx_goals_org_member
  ON goals (organization_id, member_id)
  WHERE member_id IS NOT NULL;

CREATE TRIGGER trg_goals_updated_at
  BEFORE UPDATE ON goals
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE commission_status AS ENUM ('pending', 'approved', 'paid', 'cancelled');

CREATE TABLE commissions (
  id                uuid               PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid               NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  proposal_id       uuid               NULL REFERENCES proposals(id) ON DELETE SET NULL,
  member_id         uuid               NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
  -- percentage stored as numeric to support 0.005 decimals; bounds
  -- 0..100 so a config bug can't pay negative or >100% commission.
  percentage        numeric(5,2)       NOT NULL CHECK (percentage >= 0 AND percentage <= 100),
  amount_cents      bigint             NOT NULL CHECK (amount_cents >= 0),
  currency          text               NOT NULL DEFAULT 'BRL' CHECK (char_length(currency) = 3),
  status            commission_status  NOT NULL DEFAULT 'pending',
  earned_at         timestamptz        NOT NULL DEFAULT now(),
  approved_at       timestamptz        NULL,
  paid_at           timestamptz        NULL,
  notes             text               NULL CHECK (notes IS NULL OR char_length(notes) <= 2000),
  created_at        timestamptz        NOT NULL DEFAULT now(),
  updated_at        timestamptz        NOT NULL DEFAULT now()
);

CREATE INDEX idx_commissions_org_member_earned
  ON commissions (organization_id, member_id, earned_at DESC);

CREATE INDEX idx_commissions_org_status
  ON commissions (organization_id, status)
  WHERE status IN ('pending','approved');

CREATE TRIGGER trg_commissions_updated_at
  BEFORE UPDATE ON commissions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE awards (
  id                uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  title             text         NOT NULL CHECK (char_length(title) BETWEEN 2 AND 120),
  description       text         NULL CHECK (description IS NULL OR char_length(description) <= 2000),
  criteria_json     jsonb        NOT NULL DEFAULT '{}'::jsonb,
  winners_json      jsonb        NOT NULL DEFAULT '[]'::jsonb,
  awarded_at        timestamptz  NULL,
  created_by        uuid         NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at        timestamptz  NOT NULL DEFAULT now(),
  updated_at        timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX idx_awards_org_awarded
  ON awards (organization_id, awarded_at DESC)
  WHERE awarded_at IS NOT NULL;

CREATE TRIGGER trg_awards_updated_at
  BEFORE UPDATE ON awards
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('performance.view',   false, false, true, 'Ver ranking, metas, comissões e premiações.'),
  ('performance.manage', true,  false, true, 'Criar metas, aprovar comissões e emitir premiações (admin).')
ON CONFLICT (feature_key) DO NOTHING;

COMMENT ON TABLE goals IS 'S47 — per-member or org-level targets with period window. Progress is aggregated on demand.';
COMMENT ON TABLE commissions IS 'S47 — per-deal attribution + lifecycle pending→approved→paid.';
COMMENT ON TABLE awards IS 'S47 — named recognition with JSON criteria + snapshot of winners.';

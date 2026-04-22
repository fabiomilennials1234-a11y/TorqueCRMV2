-- =====================================================================
-- 0026_s51_plan_quotas.up.sql
-- Sprint S51 / Fase G.1 — Asaas real + quota enforcement runtime.
--
-- Two concerns:
--
--  1. `plan_quotas` — catalog that answers "what's the plan_base for
--     resource X on plan Y?". S24 introduced the `plans` table and S02
--     introduced `org_quotas` with the delta model, but nothing tied
--     them together — plan_base had to be computed elsewhere. We carve
--     the table now so the subscription-activated event path can seed
--     `org_quotas` from `plan_quotas` atomically.
--
--     Seeds mirror the pricing tiers (free/growth/enterprise) already
--     inserted in 0001. Resource keys chosen for quota enforcement
--     are the ones whose unbounded growth would actually break the
--     operator's economics: leads, team_members, workflows, agents.
--     Not every object type needs a quota — Torque stays permissive
--     by default and reaches for quotas only where it hurts.
--
--  2. `org_quotas` seeds for existing tenants. A tenant provisioned
--     before S51 has no row in org_quotas; the middleware would treat
--     the missing row as `effective_limit = 0` and 402 every call.
--     We INSERT ... ON CONFLICT for every (existing org, resource_key)
--     pair with the plan_base of their current subscription (live or
--     free fallback when no subscription row exists yet).
-- =====================================================================

-- ---------- 1. plan_quotas ------------------------------------------

CREATE TABLE plan_quotas (
  plan_id       text NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
  resource_key  text NOT NULL CHECK (resource_key ~ '^[a-z][a-z0-9_]{1,40}$'),
  plan_base     int  NOT NULL CHECK (plan_base >= 0),
  created_at    timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (plan_id, resource_key)
);

COMMENT ON TABLE plan_quotas IS
  'S51 — plan_base per (plan, resource). Seeds org_quotas on subscription.activated.';

-- Seed the three tiers. Conservative on free; unlimited on enterprise
-- (a 100_000 ceiling that's effectively "no limit" for any realistic
-- tenant but still a finite number so the delta formula stays sane).
INSERT INTO plan_quotas (plan_id, resource_key, plan_base) VALUES
  -- Free: small-team trial.
  ('free',       'leads',         100),
  ('free',       'team_members',  2),
  ('free',       'workflows',     2),
  ('free',       'agents',        1),
  -- Growth: the sweet spot for Torque's ICP.
  ('growth',     'leads',         5000),
  ('growth',     'team_members',  10),
  ('growth',     'workflows',     25),
  ('growth',     'agents',        5),
  -- Enterprise: effectively unlimited (six-figure ceiling so the
  -- delta math + int32 stay well-defined).
  ('enterprise', 'leads',         100000),
  ('enterprise', 'team_members',  500),
  ('enterprise', 'workflows',     1000),
  ('enterprise', 'agents',        100)
ON CONFLICT (plan_id, resource_key) DO NOTHING;

-- ---------- 2. backfill org_quotas for existing tenants -------------
--
-- Every org that has a subscription row (live or terminal) gets its
-- plan_base from that plan. Orgs without any subscription yet default
-- to free tier so the middleware doesn't trip on migration day.
--
-- resource_key list mirrors the seeds above — kept inline to avoid a
-- second roundtrip.

WITH resource_keys(resource_key) AS (
  VALUES ('leads'), ('team_members'), ('workflows'), ('agents')
),
org_plan AS (
  -- Prefer a live (pending|active|past_due) subscription's plan. Fall
  -- back to the most recent terminal subscription so a just-cancelled
  -- org doesn't regress to free on migration. No subscription at all
  -- → free.
  SELECT
    o.id AS organization_id,
    COALESCE(
      (SELECT plan_id FROM subscriptions s
        WHERE s.organization_id = o.id
          AND s.status IN ('pending','active','past_due')
        LIMIT 1),
      (SELECT plan_id FROM subscriptions s
        WHERE s.organization_id = o.id
        ORDER BY s.created_at DESC
        LIMIT 1),
      'free'
    ) AS plan_id
  FROM organizations o
)
INSERT INTO org_quotas (organization_id, resource_key, plan_base, current_usage)
SELECT
  op.organization_id,
  rk.resource_key,
  COALESCE(pq.plan_base, 0) AS plan_base,
  0 AS current_usage
FROM org_plan op
CROSS JOIN resource_keys rk
LEFT JOIN plan_quotas pq
  ON pq.plan_id = op.plan_id
 AND pq.resource_key = rk.resource_key
ON CONFLICT (organization_id, resource_key) DO NOTHING;

-- ---------- 3. permission catalog seeds (quota observability) ------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('quotas.view', false, false, true, 'Ver uso e limite de cada recurso.')
ON CONFLICT (feature_key) DO NOTHING;

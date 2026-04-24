-- =====================================================================
-- 0031_ai_budget_quotas.up.sql
-- Sprint S52 / AI Guard-rails — budget cap for LLM tokens + TTS seconds.
--
-- Motivation:
--   S37-S42 shipped Copilot adapters (OpenRouter, Gemini embeddings,
--   ElevenLabs TTS) but the runtime had no economic guard-rail: a
--   runaway prompt loop or a compromised tenant key could burn through
--   provider credits without any 402 gate. S52 closes the loop by
--   charging every LLM + TTS call against a per-tenant meter that
--   reuses the `org_quotas` delta model introduced in S02/S51.
--
-- Two new resource keys:
--   * ai_tokens   — total tokens (input + output) per billing period.
--                   The playground handler increments by `input_tokens +
--                   output_tokens` returned by the provider on the
--                   terminal streaming frame. Middleware gates admission
--                   on current_usage < effective_limit (no pre-increment
--                   — tokens are variable-cost, known only post-stream).
--   * tts_seconds — synthesized audio seconds per billing period.
--                   Handlers increment by estimated duration (chars / 15)
--                   post-synthesis. Same admission semantics.
--
-- Ceilings by tier mirror S51's leads/workflows/agents pattern:
--   free        — 50k tokens / 300s TTS   (enough for a handful of demos)
--   growth      — 500k / 3600s            (one hour of TTS; ~a week of
--                                          active Playground tinkering)
--   enterprise  — 5M / 36000s             (effectively uncapped for any
--                                          real tenant; finite number so
--                                          the delta math stays sane).
--
-- Both resources use the regex CHECK (`^[a-z][a-z0-9_]{1,40}$`) already
-- installed on plan_quotas + org_quotas; no schema change needed.
-- =====================================================================

-- ---------- 1. plan_quotas catalog -----------------------------------

INSERT INTO plan_quotas (plan_id, resource_key, plan_base) VALUES
  -- Free: trial ceiling.
  ('free',       'ai_tokens',    50000),
  ('free',       'tts_seconds',    300),
  -- Growth: the sweet spot.
  ('growth',     'ai_tokens',   500000),
  ('growth',     'tts_seconds',   3600),
  -- Enterprise: effectively uncapped.
  ('enterprise', 'ai_tokens',  5000000),
  ('enterprise', 'tts_seconds',  36000)
ON CONFLICT (plan_id, resource_key) DO NOTHING;

-- ---------- 2. backfill org_quotas for existing tenants --------------
--
-- Same pattern as 0026: every org picks up a row per new resource
-- seeded from its current plan. Orgs without any subscription default
-- to 'free' so the middleware doesn't 402 on day 0.

WITH new_resources(resource_key) AS (
  VALUES ('ai_tokens'), ('tts_seconds')
),
org_plan AS (
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
  nr.resource_key,
  COALESCE(pq.plan_base, 0) AS plan_base,
  0 AS current_usage
FROM org_plan op
CROSS JOIN new_resources nr
LEFT JOIN plan_quotas pq
  ON pq.plan_id = op.plan_id
 AND pq.resource_key = nr.resource_key
ON CONFLICT (organization_id, resource_key) DO NOTHING;

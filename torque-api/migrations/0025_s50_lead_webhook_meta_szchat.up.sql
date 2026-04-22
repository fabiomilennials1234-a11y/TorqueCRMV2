-- =====================================================================
-- 0025_s50_lead_webhook_meta_szchat.up.sql
-- Sprint S50 / Fase F.2 — Meta Ads Insights + Lead Webhook + SZ.Chat.
--
-- Three concerns collapsed into one migration because they all deliver
-- together and share a boot cycle:
--
--  1. lead_webhook_events — dedup log for the public POST /webhooks/lead
--     endpoint. Same shape + rationale as billing_events (0016): UNIQUE
--     (organization_id, external_id) absorbs retries without mutating
--     the lead twice. organization_id is NOT a FK target for logging
--     here — the webhook resolves the org from the signing secret and
--     we want the row to survive a tenant delete for forensic purposes.
--
--  2. meta_insights_cache — 15 minute TTL cache for Meta Graph API
--     Ads Insights responses. (org, account_id, date_range) is unique;
--     hitting the same window within TTL short-circuits to the cached
--     payload. Why a table instead of Redis: we already pay for pgx
--     + pool; reads are single-row by composite key; TTL is enforced
--     at query time (WHERE fetched_at > now() - interval '15 min').
--
--  3. integration_credentials.provider CHECK extension — add 'szchat'
--     alongside google/tinyerp/meta so the SZ.Chat MessagingProvider
--     can persist its per-tenant API key through the same encrypted
--     credential store.
-- =====================================================================

-- ---------- 1. lead webhook dedup log ---------------------------------

CREATE TABLE lead_webhook_events (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  external_id       text        NOT NULL
                                  CHECK (char_length(external_id) BETWEEN 1 AND 200),
  source            text        NOT NULL DEFAULT 'webhook'
                                  CHECK (source IN ('webhook','meta','szchat','manual')),
  lead_id           uuid        NULL REFERENCES leads(id) ON DELETE SET NULL,
  raw_payload       jsonb       NOT NULL,
  signature_ok      boolean     NOT NULL DEFAULT true,
  received_at       timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_lead_webhook_events_org_external
    UNIQUE (organization_id, external_id)
);

CREATE INDEX idx_lead_webhook_events_org_received
  ON lead_webhook_events (organization_id, received_at DESC);

COMMENT ON TABLE lead_webhook_events IS
  'S50 — public /webhooks/lead dedup log. UNIQUE(org,external_id) absorbs provider retries.';

-- ---------- 2. Meta Ads insights cache --------------------------------

CREATE TABLE meta_insights_cache (
  id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  account_id        text        NOT NULL
                                  CHECK (char_length(account_id) BETWEEN 1 AND 80),
  date_range        text        NOT NULL
                                  CHECK (char_length(date_range) BETWEEN 1 AND 40),
  payload           jsonb       NOT NULL,
  fetched_at        timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_meta_insights_org_account_range
    UNIQUE (organization_id, account_id, date_range)
);

CREATE INDEX idx_meta_insights_org_fetched
  ON meta_insights_cache (organization_id, fetched_at DESC);

COMMENT ON TABLE meta_insights_cache IS
  'S50 — Meta Graph API Ads Insights cache (TTL 15min enforced at query time).';

-- ---------- 3. extend providers CHECK ---------------------------------

-- Drop + re-add with the wider set. 'meta' already exists from 0024; we
-- add 'szchat' so the SZ.Chat MessagingProvider can park its tenant
-- API key through the same encrypted store.

ALTER TABLE integration_credentials
  DROP CONSTRAINT IF EXISTS integration_credentials_provider_check;

ALTER TABLE integration_credentials
  ADD CONSTRAINT integration_credentials_provider_check
  CHECK (provider IN ('google','tinyerp','meta','szchat'));

-- ---------- 4. permission catalog seed --------------------------------

-- integrations.view + integrations.manage already seeded by S49. No
-- new permissions needed for S50 — Meta/SZ.Chat/Sync operations reuse
-- the existing integrations.manage (admin-only) scope.

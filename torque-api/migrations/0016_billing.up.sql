-- =====================================================================
-- 0016_billing.up.sql
-- S24 — F14 Checkout + PIX + Provisioning (data plane).
--
-- The money flow: user picks a plan → we create a `subscription` row in
-- state 'pending' → provider issues a PIX charge → user pays → provider
-- webhook flips state to 'active'. Plans already exist since 0001.
--
-- This sprint ships the data model + adapter interface + webhook trail.
-- The live Asaas provider requires credentials + a human-in-the-loop
-- dual review before going to prod; a mock provider satisfies dev/test.
-- =====================================================================

CREATE TYPE subscription_status AS ENUM (
  'pending',        -- charge issued, waiting for payment
  'active',         -- paid and within the billing cycle
  'past_due',       -- provider reports delinquency
  'cancelled',      -- tenant or admin cancelled
  'expired'         -- cycle ended without renewal
);

CREATE TYPE billing_provider AS ENUM ('mock', 'asaas');

CREATE TABLE subscriptions (
  id                   uuid                PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id      uuid                NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  plan_id              text                NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
  status               subscription_status NOT NULL DEFAULT 'pending',
  provider             billing_provider    NOT NULL DEFAULT 'mock',
  provider_customer_id text                NULL,
  provider_charge_id   text                NULL,
  amount_cents         bigint              NOT NULL CHECK (amount_cents >= 0 AND amount_cents <= 1e14),
  currency             text                NOT NULL DEFAULT 'BRL' CHECK (char_length(currency) = 3),
  pix_qr_code          text                NULL,
  pix_qr_code_image    text                NULL,
  pix_expires_at       timestamptz         NULL,
  current_period_start timestamptz         NULL,
  current_period_end   timestamptz         NULL,
  cancelled_at         timestamptz         NULL,
  created_by           uuid                NULL REFERENCES team_members(id) ON DELETE SET NULL,
  created_at           timestamptz         NOT NULL DEFAULT now(),
  updated_at           timestamptz         NOT NULL DEFAULT now()
);

-- Single active subscription per tenant. Hard invariant — a second
-- concurrent active sub would split billing in unpredictable ways.
CREATE UNIQUE INDEX uq_subscriptions_one_active_per_org
  ON subscriptions (organization_id)
  WHERE status IN ('pending', 'active', 'past_due');

CREATE INDEX idx_subscriptions_provider_charge
  ON subscriptions (provider, provider_charge_id)
  WHERE provider_charge_id IS NOT NULL;

CREATE TRIGGER trg_subscriptions_updated_at
  BEFORE UPDATE ON subscriptions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE subscriptions IS 'F14 — one live subscription per tenant; state transitions driven by provider webhook.';

-- Append-only webhook ledger. Dedup on (provider, provider_event_id) so
-- the same webhook replay is idempotent.
CREATE TABLE billing_events (
  id                  uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id     uuid             NULL REFERENCES organizations(id) ON DELETE SET NULL,
  subscription_id     uuid             NULL REFERENCES subscriptions(id) ON DELETE SET NULL,
  provider            billing_provider NOT NULL,
  provider_event_id   text             NOT NULL,
  event_type          text             NOT NULL CHECK (char_length(event_type) BETWEEN 1 AND 80),
  raw_payload         jsonb            NOT NULL,
  processed_at        timestamptz      NOT NULL DEFAULT now(),

  CONSTRAINT uq_billing_events_provider_event UNIQUE (provider, provider_event_id)
);

CREATE INDEX idx_billing_events_subscription
  ON billing_events (subscription_id)
  WHERE subscription_id IS NOT NULL;

COMMENT ON TABLE billing_events IS 'F14 — provider webhook ledger; append-only, dedup by (provider, provider_event_id).';

-- Seed permission keys — billing mutations are admin-only + master_only
-- for the most destructive ones. billing.manage already exists since
-- migration 0001 (admin + master_only). We add finer-grained keys here.
INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('billing.view',    false, false, true,  'Ver plano atual e histórico de cobrança.'),
  ('billing.checkout', true, false, true,  'Iniciar checkout e gerar cobrança PIX (admin).'),
  ('billing.cancel',  true,  false, true,  'Cancelar assinatura (admin).')
ON CONFLICT (feature_key) DO NOTHING;

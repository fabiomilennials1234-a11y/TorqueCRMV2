-- =====================================================================
-- 0007_pipe_proposals.up.sql
-- F03 — Pipe de Propostas (documento comercial + ciclo de vida).
--
-- 1:1 com pipe_entries for proposal-kind pipes. Money is integer cents +
-- currency code (CLAUDE.md — "Money: Integer cents + currency code"). A
-- proposal moves through states:
--   draft → sent → viewed → {accepted | rejected | expired}
-- The timestamps are additive — we keep every transition point so analytics
-- can compute lead time, time-to-view, etc.
-- =====================================================================

CREATE TYPE proposal_status AS ENUM (
  'draft',
  'sent',
  'viewed',
  'accepted',
  'rejected',
  'expired'
);

CREATE TABLE pipe_proposals (
  pipe_entry_id    uuid             PRIMARY KEY REFERENCES pipe_entries(id) ON DELETE CASCADE,
  organization_id  uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  lead_id          uuid             NOT NULL REFERENCES leads(id) ON DELETE CASCADE,

  title            text             NOT NULL CHECK (char_length(title) BETWEEN 2 AND 200),
  amount_cents     bigint           NOT NULL CHECK (amount_cents >= 0),
  currency         text             NOT NULL DEFAULT 'BRL' CHECK (currency ~ '^[A-Z]{3}$'),

  -- S3-compatible object key for the PDF. The pre-signed URL is minted per
  -- read; the key is stable.
  attachment_key   text             NULL CHECK (attachment_key IS NULL OR char_length(attachment_key) BETWEEN 1 AND 500),
  attachment_size  bigint           NULL CHECK (attachment_size IS NULL OR attachment_size >= 0),

  status           proposal_status  NOT NULL DEFAULT 'draft',

  sent_at          timestamptz      NULL,
  first_viewed_at  timestamptz      NULL,
  accepted_at      timestamptz      NULL,
  rejected_at      timestamptz      NULL,
  rejection_reason text             NULL CHECK (rejection_reason IS NULL OR char_length(rejection_reason) <= 500),
  expires_at       timestamptz      NULL,

  notes            text             NULL CHECK (notes IS NULL OR char_length(notes) <= 4000),

  created_at       timestamptz      NOT NULL DEFAULT now(),
  updated_at       timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipe_proposals_org_status
  ON pipe_proposals (organization_id, status)
  WHERE status IN ('sent', 'viewed');

CREATE INDEX idx_pipe_proposals_org_lead
  ON pipe_proposals (organization_id, lead_id);

CREATE INDEX idx_pipe_proposals_org_expires
  ON pipe_proposals (organization_id, expires_at)
  WHERE status IN ('sent', 'viewed') AND expires_at IS NOT NULL;

CREATE TRIGGER trg_pipe_proposals_updated_at
  BEFORE UPDATE ON pipe_proposals
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE pipe_proposals IS
  'F03 — commercial proposal per pipe_entry (1:1). Money as integer cents + currency.';

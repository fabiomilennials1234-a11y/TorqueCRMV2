-- =====================================================================
-- 0006_pipe_confirmations.up.sql
-- F02 — Pipe de Confirmação (agendamento de reuniões).
--
-- The confirmation pipe extends the pipe_entries model with meeting
-- metadata. A pipe_entry in a confirmation-kind pipe may have exactly
-- ONE associated confirmation row; the relationship is 1:1 and keyed
-- by pipe_entry_id.
--
-- Design notes:
--   * Kept in a separate table (not extra columns on pipe_entries) so the
--     generic entries table stays narrow. N rows of whatsapp entries pay
--     no cost for fields that only apply to confirmation.
--   * meeting_at is in UTC; no_show reconciles when the meeting time has
--     passed and the meeting did not occur.
-- =====================================================================

CREATE TABLE pipe_confirmations (
  pipe_entry_id   uuid        PRIMARY KEY REFERENCES pipe_entries(id) ON DELETE CASCADE,
  organization_id uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  lead_id         uuid        NOT NULL REFERENCES leads(id) ON DELETE CASCADE,

  meeting_at      timestamptz NOT NULL,
  meeting_channel text        NULL CHECK (meeting_channel IS NULL OR char_length(meeting_channel) <= 40),
  meeting_notes   text        NULL CHECK (meeting_notes IS NULL OR char_length(meeting_notes) <= 2000),

  confirmed_at    timestamptz NULL, -- set when the lead confirms attendance
  no_show         bool        NOT NULL DEFAULT false,
  no_show_reason  text        NULL CHECK (no_show_reason IS NULL OR char_length(no_show_reason) <= 200),

  reminder_sent_at timestamptz NULL,

  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipe_confirmations_org_meeting
  ON pipe_confirmations (organization_id, meeting_at);

CREATE INDEX idx_pipe_confirmations_org_lead
  ON pipe_confirmations (organization_id, lead_id);

-- Janitor-friendly: find overdue unconfirmed meetings.
CREATE INDEX idx_pipe_confirmations_org_overdue
  ON pipe_confirmations (organization_id, meeting_at)
  WHERE confirmed_at IS NULL AND no_show = false;

CREATE TRIGGER trg_pipe_confirmations_updated_at
  BEFORE UPDATE ON pipe_confirmations
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE pipe_confirmations IS
  'F02 meeting metadata — 1:1 with pipe_entries for confirmation-kind pipes.';

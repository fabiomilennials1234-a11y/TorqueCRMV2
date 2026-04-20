-- =====================================================================
-- 0008_inbox.up.sql
-- F04 — Inbox multi-canal (WhatsApp, Instagram DM, Messenger, Email, etc).
--
-- Shape:
--   channels          — per-tenant connection config (provider, auth state).
--   conversations     — one row per (tenant, channel, external_thread_id).
--   messages          — one row per individual inbound/outbound message.
--
-- Conversations are bound 1:N to a lead (NULL until association). Messages
-- keep their provider-supplied external_id so delivery callbacks dedup
-- cleanly.
-- =====================================================================

CREATE TYPE channel_kind AS ENUM (
  'whatsapp',
  'instagram',
  'messenger',
  'email',
  'sz_chat',
  'other'
);

CREATE TYPE channel_status AS ENUM (
  'connected',
  'connecting',
  'disconnected',
  'error'
);

CREATE TABLE channels (
  id                uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid           NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  kind              channel_kind   NOT NULL,
  name              text           NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
  external_id       text           NULL,                              -- provider identifier
  status            channel_status NOT NULL DEFAULT 'connecting',
  last_error        text           NULL CHECK (last_error IS NULL OR char_length(last_error) <= 500),
  metadata          jsonb          NOT NULL DEFAULT '{}'::jsonb,     -- provider-specific
  last_seen_at      timestamptz    NULL,
  created_at        timestamptz    NOT NULL DEFAULT now(),
  updated_at        timestamptz    NOT NULL DEFAULT now(),

  CONSTRAINT uq_channels_org_kind_external UNIQUE (organization_id, kind, external_id)
);

CREATE INDEX idx_channels_org_status ON channels (organization_id, status);

CREATE TRIGGER trg_channels_updated_at
  BEFORE UPDATE ON channels
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------- conversations ---------------------------------------------

CREATE TYPE conversation_state AS ENUM (
  'open',
  'pending',
  'resolved',
  'archived'
);

CREATE TABLE conversations (
  id                  uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id     uuid             NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  channel_id          uuid             NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
  lead_id             uuid             NULL REFERENCES leads(id) ON DELETE SET NULL,
  external_thread_id  text             NOT NULL,                     -- chat id on provider
  contact_name        text             NULL CHECK (contact_name IS NULL OR char_length(contact_name) <= 200),
  contact_handle      text             NULL CHECK (contact_handle IS NULL OR char_length(contact_handle) <= 200),
  state               conversation_state NOT NULL DEFAULT 'open',
  assigned_to         uuid             NULL REFERENCES team_members(id) ON DELETE SET NULL,
  unread_count        int              NOT NULL DEFAULT 0 CHECK (unread_count >= 0),
  last_message_at     timestamptz      NULL,
  last_message_preview text            NULL CHECK (last_message_preview IS NULL OR char_length(last_message_preview) <= 400),
  metadata            jsonb            NOT NULL DEFAULT '{}'::jsonb,
  created_at          timestamptz      NOT NULL DEFAULT now(),
  updated_at          timestamptz      NOT NULL DEFAULT now(),

  CONSTRAINT uq_conversations_channel_thread UNIQUE (channel_id, external_thread_id)
);

CREATE INDEX idx_conversations_org_last_msg
  ON conversations (organization_id, last_message_at DESC);
CREATE INDEX idx_conversations_org_state
  ON conversations (organization_id, state);
CREATE INDEX idx_conversations_org_assignee
  ON conversations (organization_id, assigned_to)
  WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_conversations_org_lead
  ON conversations (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE TRIGGER trg_conversations_updated_at
  BEFORE UPDATE ON conversations
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------- messages --------------------------------------------------

CREATE TYPE message_direction AS ENUM ('inbound', 'outbound');

CREATE TYPE message_kind AS ENUM (
  'text',
  'image',
  'audio',
  'video',
  'document',
  'sticker',
  'system'
);

CREATE TYPE message_status AS ENUM (
  'queued',
  'sent',
  'delivered',
  'read',
  'failed'
);

CREATE TABLE messages (
  id                uuid              PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid              NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  conversation_id   uuid              NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  external_id       text              NULL,
  direction         message_direction NOT NULL,
  kind              message_kind      NOT NULL DEFAULT 'text',
  body              text              NULL CHECK (body IS NULL OR char_length(body) <= 16000),
  media_url         text              NULL CHECK (media_url IS NULL OR char_length(media_url) <= 2000),
  media_mime        text              NULL CHECK (media_mime IS NULL OR char_length(media_mime) <= 120),
  sent_by_member_id uuid              NULL REFERENCES team_members(id) ON DELETE SET NULL,
  status            message_status    NOT NULL DEFAULT 'queued',
  occurred_at       timestamptz       NOT NULL DEFAULT now(),
  delivered_at      timestamptz       NULL,
  read_at           timestamptz       NULL,
  error_payload     jsonb             NULL,
  metadata          jsonb             NOT NULL DEFAULT '{}'::jsonb,
  created_at        timestamptz       NOT NULL DEFAULT now(),
  updated_at        timestamptz       NOT NULL DEFAULT now(),

  -- Dedup on provider callbacks; external_id may repeat for different
  -- conversations (same provider message-id scheme per chat).
  CONSTRAINT uq_messages_conversation_external UNIQUE (conversation_id, external_id) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX idx_messages_conversation_occurred
  ON messages (conversation_id, occurred_at);
CREATE INDEX idx_messages_org_occurred
  ON messages (organization_id, occurred_at DESC);

CREATE TRIGGER trg_messages_updated_at
  BEFORE UPDATE ON messages
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE channels IS 'F04 — per-tenant inbox connections (provider, auth state).';
COMMENT ON TABLE conversations IS 'F04 — one row per (channel, external_thread_id); may bind to a lead.';
COMMENT ON TABLE messages IS 'F04 — inbound + outbound messages; dedup by (conversation_id, external_id).';

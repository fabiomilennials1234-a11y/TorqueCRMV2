DROP TRIGGER IF EXISTS trg_messages_updated_at ON messages;
DROP INDEX IF EXISTS idx_messages_org_occurred;
DROP INDEX IF EXISTS idx_messages_conversation_occurred;
DROP TABLE IF EXISTS messages;
DROP TYPE IF EXISTS message_status;
DROP TYPE IF EXISTS message_kind;
DROP TYPE IF EXISTS message_direction;

DROP TRIGGER IF EXISTS trg_conversations_updated_at ON conversations;
DROP INDEX IF EXISTS idx_conversations_org_lead;
DROP INDEX IF EXISTS idx_conversations_org_assignee;
DROP INDEX IF EXISTS idx_conversations_org_state;
DROP INDEX IF EXISTS idx_conversations_org_last_msg;
DROP TABLE IF EXISTS conversations;
DROP TYPE IF EXISTS conversation_state;

DROP TRIGGER IF EXISTS trg_channels_updated_at ON channels;
DROP INDEX IF EXISTS idx_channels_org_status;
DROP TABLE IF EXISTS channels;
DROP TYPE IF EXISTS channel_status;
DROP TYPE IF EXISTS channel_kind;

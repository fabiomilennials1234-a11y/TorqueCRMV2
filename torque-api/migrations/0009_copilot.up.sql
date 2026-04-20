-- =====================================================================
-- 0009_copilot.up.sql
-- F06 — Copilot backend: agents + knowledge base collections + embeddings.
--
-- Design:
--   agents             — per-tenant configuration of a named assistant
--                        (model, system prompt, tool allowlist, kill switch).
--   knowledge_sources  — ingestable documents tied to a collection.
--   knowledge_chunks   — extractive chunks with metadata for RAG retrieval.
--   agent_sessions     — conversation threads with the agent.
--   agent_messages     — message history inside a session.
--
-- Embeddings are stored as `text` for forward-compatibility: when pgvector
-- is provisioned (migration 0010+), we add an `embedding vector(...)` column
-- and backfill. Keeping the provisional format small avoids re-writing the
-- ingest pipeline twice.
-- =====================================================================

-- ---------- agents ----------------------------------------------------

CREATE TYPE agent_status AS ENUM ('draft', 'active', 'disabled');

CREATE TABLE agents (
  id                 uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id    uuid         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name               text         NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  description        text         NULL CHECK (description IS NULL OR char_length(description) <= 2000),
  system_prompt      text         NOT NULL CHECK (char_length(system_prompt) BETWEEN 10 AND 16000),
  model              text         NOT NULL CHECK (char_length(model) BETWEEN 2 AND 80),
  temperature        numeric(3,2) NOT NULL DEFAULT 0.3 CHECK (temperature >= 0 AND temperature <= 2),
  max_output_tokens  int          NOT NULL DEFAULT 1024 CHECK (max_output_tokens BETWEEN 16 AND 8192),
  tools_allowlist    text[]       NOT NULL DEFAULT '{}',
  -- kill_switch: setting true halts every in-flight session for this agent.
  kill_switch        bool         NOT NULL DEFAULT false,
  status             agent_status NOT NULL DEFAULT 'draft',
  metadata           jsonb        NOT NULL DEFAULT '{}'::jsonb,
  created_at         timestamptz  NOT NULL DEFAULT now(),
  updated_at         timestamptz  NOT NULL DEFAULT now(),

  CONSTRAINT uq_agents_org_name UNIQUE (organization_id, name)
);

CREATE INDEX idx_agents_org_status ON agents (organization_id, status);

CREATE TRIGGER trg_agents_updated_at
  BEFORE UPDATE ON agents
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------- knowledge base -------------------------------------------

CREATE TABLE knowledge_collections (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name             text        NOT NULL CHECK (char_length(name) BETWEEN 2 AND 120),
  description      text        NULL CHECK (description IS NULL OR char_length(description) <= 2000),
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_collections_org_name UNIQUE (organization_id, name)
);

CREATE TRIGGER trg_knowledge_collections_updated_at
  BEFORE UPDATE ON knowledge_collections
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE knowledge_source_status AS ENUM (
  'queued', 'ingesting', 'ready', 'failed'
);

CREATE TABLE knowledge_sources (
  id                uuid                    PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                    NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  collection_id     uuid                    NOT NULL REFERENCES knowledge_collections(id) ON DELETE CASCADE,
  kind              text                    NOT NULL CHECK (kind IN ('pdf','url','text','markdown','docx')),
  title             text                    NOT NULL CHECK (char_length(title) BETWEEN 1 AND 300),
  uri               text                    NULL CHECK (uri IS NULL OR char_length(uri) <= 2000),
  status            knowledge_source_status NOT NULL DEFAULT 'queued',
  error             text                    NULL CHECK (error IS NULL OR char_length(error) <= 2000),
  content_hash      text                    NULL,
  metadata          jsonb                   NOT NULL DEFAULT '{}'::jsonb,
  ingested_at       timestamptz             NULL,
  created_at        timestamptz             NOT NULL DEFAULT now(),
  updated_at        timestamptz             NOT NULL DEFAULT now()
);

CREATE INDEX idx_knowledge_sources_org_collection
  ON knowledge_sources (organization_id, collection_id);
CREATE INDEX idx_knowledge_sources_org_status
  ON knowledge_sources (organization_id, status)
  WHERE status IN ('queued','ingesting');

CREATE TRIGGER trg_knowledge_sources_updated_at
  BEFORE UPDATE ON knowledge_sources
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE knowledge_chunks (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id  uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  source_id        uuid        NOT NULL REFERENCES knowledge_sources(id) ON DELETE CASCADE,
  ord              int         NOT NULL,
  content          text        NOT NULL CHECK (char_length(content) <= 16000),
  tokens_estimate  int         NULL,
  -- Placeholder: stored as text until pgvector lands. The ingest worker
  -- writes a JSON array (e.g. "[0.123,-0.456,...]"); retrieval post-filters
  -- via cosine similarity in the app tier until the vector column is live.
  embedding_text   text        NULL,
  metadata         jsonb       NOT NULL DEFAULT '{}'::jsonb,
  created_at       timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT uq_knowledge_chunks_source_ord UNIQUE (source_id, ord)
);

CREATE INDEX idx_knowledge_chunks_org_source
  ON knowledge_chunks (organization_id, source_id);

-- ---------- agent sessions + messages --------------------------------

CREATE TYPE agent_session_state AS ENUM ('open', 'ended', 'escalated');

CREATE TABLE agent_sessions (
  id                uuid                 PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                 NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  agent_id          uuid                 NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  lead_id           uuid                 NULL REFERENCES leads(id) ON DELETE SET NULL,
  conversation_id   uuid                 NULL REFERENCES conversations(id) ON DELETE SET NULL,
  state             agent_session_state  NOT NULL DEFAULT 'open',
  started_at        timestamptz          NOT NULL DEFAULT now(),
  ended_at          timestamptz          NULL,
  metadata          jsonb                NOT NULL DEFAULT '{}'::jsonb,
  created_at        timestamptz          NOT NULL DEFAULT now(),
  updated_at        timestamptz          NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_sessions_org_state
  ON agent_sessions (organization_id, state);
CREATE INDEX idx_agent_sessions_org_lead
  ON agent_sessions (organization_id, lead_id)
  WHERE lead_id IS NOT NULL;

CREATE TRIGGER trg_agent_sessions_updated_at
  BEFORE UPDATE ON agent_sessions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TYPE agent_message_role AS ENUM ('system', 'user', 'assistant', 'tool');

CREATE TABLE agent_messages (
  id                uuid                PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id   uuid                NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  session_id        uuid                NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
  role              agent_message_role  NOT NULL,
  content           text                NOT NULL CHECK (char_length(content) <= 32000),
  tool_name         text                NULL CHECK (tool_name IS NULL OR char_length(tool_name) <= 120),
  tool_payload      jsonb               NULL,
  tokens_input      int                 NULL,
  tokens_output     int                 NULL,
  latency_ms        int                 NULL CHECK (latency_ms IS NULL OR latency_ms >= 0),
  occurred_at       timestamptz         NOT NULL DEFAULT now(),
  created_at        timestamptz         NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_messages_session_occurred
  ON agent_messages (session_id, occurred_at);
CREATE INDEX idx_agent_messages_org_occurred
  ON agent_messages (organization_id, occurred_at DESC);

COMMENT ON TABLE agents IS 'F06 — Copilot agent config with kill_switch + tool allowlist.';
COMMENT ON TABLE knowledge_collections IS 'F06 — Named collection (folder) of knowledge sources.';
COMMENT ON TABLE knowledge_sources IS 'F06 — Ingestable docs; ingest worker flips status queued → ingesting → ready|failed.';
COMMENT ON TABLE knowledge_chunks IS 'F06 — Retrievable chunk; embedding stored as text until pgvector migration.';
COMMENT ON TABLE agent_sessions IS 'F06 — Conversation thread with a copilot agent.';
COMMENT ON TABLE agent_messages IS 'F06 — Per-turn message incl. tool calls + usage counters for billing + observability.';

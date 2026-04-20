DROP INDEX IF EXISTS idx_agent_messages_org_occurred;
DROP INDEX IF EXISTS idx_agent_messages_session_occurred;
DROP TABLE IF EXISTS agent_messages;
DROP TYPE IF EXISTS agent_message_role;

DROP TRIGGER IF EXISTS trg_agent_sessions_updated_at ON agent_sessions;
DROP INDEX IF EXISTS idx_agent_sessions_org_lead;
DROP INDEX IF EXISTS idx_agent_sessions_org_state;
DROP TABLE IF EXISTS agent_sessions;
DROP TYPE IF EXISTS agent_session_state;

DROP INDEX IF EXISTS idx_knowledge_chunks_org_source;
DROP TABLE IF EXISTS knowledge_chunks;

DROP TRIGGER IF EXISTS trg_knowledge_sources_updated_at ON knowledge_sources;
DROP INDEX IF EXISTS idx_knowledge_sources_org_status;
DROP INDEX IF EXISTS idx_knowledge_sources_org_collection;
DROP TABLE IF EXISTS knowledge_sources;
DROP TYPE IF EXISTS knowledge_source_status;

DROP TRIGGER IF EXISTS trg_knowledge_collections_updated_at ON knowledge_collections;
DROP TABLE IF EXISTS knowledge_collections;

DROP TRIGGER IF EXISTS trg_agents_updated_at ON agents;
DROP INDEX IF EXISTS idx_agents_org_status;
DROP TABLE IF EXISTS agents;
DROP TYPE IF EXISTS agent_status;

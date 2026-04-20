-- =====================================================================
-- 0019_copilot_rag.down.sql — revert S39 RAG additions.
-- Drops in reverse order of creation. Extension is left installed since
-- other tenants may have started storing vectors on siblings — re-running
-- CREATE EXTENSION is idempotent and a left-over extension is harmless.
-- =====================================================================

DELETE FROM feature_permissions
 WHERE feature_key IN ('knowledge.view', 'knowledge.manage');

DROP INDEX IF EXISTS idx_agents_org_knowledge_collection;

ALTER TABLE agents
  DROP COLUMN IF EXISTS knowledge_collection_id;

DROP INDEX IF EXISTS idx_knowledge_chunks_embedding_hnsw;

ALTER TABLE knowledge_chunks
  DROP COLUMN IF EXISTS embedding;

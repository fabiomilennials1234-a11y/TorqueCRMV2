-- =====================================================================
-- 0019_copilot_rag.up.sql
-- S39 / Fase C.3 — F06 Copilot RAG: pgvector + retrieval.
--
-- Design:
--   * CREATE EXTENSION vector — pgvector provides the `vector` type +
--     cosine distance operator `<=>`. The Docker dev image switches to
--     pgvector/pgvector:pg15 which ships the binaries.
--   * knowledge_chunks gains an `embedding vector(768)` column. Gemini
--     `text-embedding-004` emits 768 floats at task_type=RETRIEVAL_DOCUMENT
--     by default; other providers (Voyage, Cohere) are compatible at this
--     dim. Text placeholder column (embedding_text) stays for migration
--     purposes and will be dropped when the vector backfill is green.
--   * HNSW index over `embedding` using `vector_cosine_ops`: builds with
--     modest memory (m=16, ef_construction=64) and returns top-K in
--     milliseconds. vector_cosine_ops matches the `<=>` operator used by
--     SimilaritySearch.
--   * agents.knowledge_collection_id FK → optional per-agent RAG binding.
--     NULL means "no retrieval" — the playground skips the similarity
--     call entirely in that case.
--   * Permission seeds: knowledge.view (read-only on collections + chunks)
--     and knowledge.manage (admin — create/ingest). Standard pattern from
--     0014, adjusted for this domain.
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS vector;

-- ---------- embedding column on knowledge_chunks ----------------------

ALTER TABLE knowledge_chunks
  ADD COLUMN embedding vector(768) NULL;

COMMENT ON COLUMN knowledge_chunks.embedding IS
  'S39 — 768-d dense embedding (Gemini text-embedding-004 default). NULL until ingest worker backfills. Use <=> operator for cosine distance.';

-- HNSW offers the best recall/latency trade-off at our scale (<1M chunks
-- per tenant). Build it only over non-NULL rows to avoid index bloat
-- during the ingest backfill window.
CREATE INDEX idx_knowledge_chunks_embedding_hnsw
  ON knowledge_chunks
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64)
  WHERE embedding IS NOT NULL;

-- ---------- agent ↔ knowledge binding ---------------------------------
--
-- Only one collection per agent in S39. A richer M:N model (agent bound
-- to several collections with weights) is a refinement pending real
-- usage data.

ALTER TABLE agents
  ADD COLUMN knowledge_collection_id uuid NULL
    REFERENCES knowledge_collections(id) ON DELETE SET NULL;

CREATE INDEX idx_agents_org_knowledge_collection
  ON agents (organization_id, knowledge_collection_id)
  WHERE knowledge_collection_id IS NOT NULL;

-- ---------- permission seeds ------------------------------------------

INSERT INTO feature_permissions (feature_key, is_admin_only, master_only, default_value, description) VALUES
  ('knowledge.view',   false, false, true,  'Ver coleções de conhecimento e fontes ingeridas.'),
  ('knowledge.manage', true,  false, true,  'Criar coleções, ingerir fontes e vincular a agents (admin).')
ON CONFLICT (feature_key) DO NOTHING;

COMMENT ON COLUMN agents.knowledge_collection_id IS
  'S39 — optional RAG binding. When set, playground/production injects topK chunks as system context.';

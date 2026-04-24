-- =====================================================================
-- 0029_cleanup_knowledge_meetings.down.sql
-- Reverse of 0029.
--
-- Note: embedding_text cannot be reconstructed (the data was thrown
-- away on UP). Re-adding a nullable text column restores the shape of
-- the schema so downstream migrations that referenced it keep applying,
-- but values will be NULL. This matches the invariant "DOWN restores
-- structure, not data lost intentionally on UP".
-- =====================================================================

-- ---------- (1) restore knowledge_chunks.embedding_text shape ---------

ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_text text NULL;

-- ---------- (2) restore global UNIQUE on meetings --------------------

ALTER TABLE meetings DROP CONSTRAINT IF EXISTS uq_meetings_org_external;

ALTER TABLE meetings
  ADD CONSTRAINT uq_meetings_external
  UNIQUE (external_provider, external_id)
  DEFERRABLE INITIALLY IMMEDIATE;

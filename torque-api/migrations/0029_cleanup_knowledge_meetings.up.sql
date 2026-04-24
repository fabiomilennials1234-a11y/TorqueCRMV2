-- =====================================================================
-- 0029_cleanup_knowledge_meetings.up.sql
-- S52 / Fase G — DBA cleanup audit 2026-04-24.
--
-- Two unrelated fixes batched because they are both small, low-risk,
-- and flagged in the same audit pass:
--
-- 1) knowledge_chunks.embedding_text — legacy column carried over from
--    0009_copilot (pre-pgvector). After S39 the `embedding vector(768)`
--    column introduced by 0019 is the source of truth for retrieval and
--    the text placeholder is dead weight (never read by services, never
--    written outside the original seed). Drop it now that the vector
--    backfill has been green in prod for > 1 sprint.
--
-- 2) meetings.uq_meetings_external — the UNIQUE (external_provider,
--    external_id) constraint from 0023 is global (not tenant-scoped).
--    Google Calendar event IDs can legitimately appear in two tenants
--    at once (same event, two organizations invited) and outlook id
--    collisions are possible across unrelated workspaces. Migrating
--    to UNIQUE (organization_id, external_provider, external_id)
--    preserves "one row per external event per tenant" (the real
--    invariant) without false cross-tenant conflicts.
--
-- Idempotency: DROP COLUMN IF EXISTS + DROP CONSTRAINT IF EXISTS make
-- this replayable in case it runs after a partial rollback.
-- =====================================================================

-- ---------- (1) knowledge_chunks legacy column ------------------------

ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS embedding_text;

-- ---------- (2) meetings UNIQUE scoped to tenant ----------------------

ALTER TABLE meetings DROP CONSTRAINT IF EXISTS uq_meetings_external;

ALTER TABLE meetings
  ADD CONSTRAINT uq_meetings_org_external
  UNIQUE (organization_id, external_provider, external_id)
  DEFERRABLE INITIALLY IMMEDIATE;

COMMENT ON CONSTRAINT uq_meetings_org_external ON meetings IS
  'S52 — per-tenant uniqueness of external calendar event. Replaces 0023 uq_meetings_external which was global and could collide across tenants for shared provider event IDs.';

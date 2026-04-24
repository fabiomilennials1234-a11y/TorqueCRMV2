-- =====================================================================
-- 0030_search_trigram.up.sql
-- S52 / Fase G — DBA search indexes audit 2026-04-24.
--
-- Adds pg_trgm GIN indexes to accelerate the substring search paths
-- already exposed by the application (lead search on name/email/phone,
-- messages full-text lookup reserved for S53+). At current volume the
-- planner falls back to a seq scan which is ~100-300 ms in prod tenants
-- with >10k leads and becomes unacceptable around 50k. Trigram GIN
-- turns the same query into a low-double-digit-millisecond response
-- when the search term is >= 3 characters (pg_trgm minimum).
--
-- Expression choice:
--   * leads.name and leads.email — repository lowercases the term
--     client-side and the WHERE clause uses `LOWER(name) LIKE $N` /
--     `LOWER(COALESCE(email::text,'')) LIKE $N`. The expression GIN
--     indexes mirror that shape so the planner actually picks them.
--     Plain `gin (name gin_trgm_ops)` would NOT match a `LOWER(name)`
--     predicate — validated against
--     torque-api/internal/repository/lead/lead.go L86-L91.
--   * leads.phone — compared without LOWER (phone is E.164, already
--     canonical). A raw trigram index is correct here.
--   * messages.body — raw text trigram index. No repository path uses
--     it yet, but conversation-history search is on the S53 backlog
--     and shipping the index now avoids a concurrent CREATE INDEX
--     sprint under load later.
--
-- Partial WHERE clauses match the production query shape:
--   leads: deleted_at IS NULL
--   messages: body NOT NULL AND non-empty
-- so the indexes stay small and hot-cache friendly.
--
-- Repositórios já usam LIKE sobre o termo preparado em lead/list e,
-- futuramente, messages/list. O planner escolherá idx_*_trgm
-- automaticamente quando o termo tem >= 3 caracteres. Sem mudança no
-- código Go necessária.
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------- leads: name / email / phone -------------------------------

CREATE INDEX IF NOT EXISTS idx_leads_name_lower_trgm
  ON leads USING gin (LOWER(name) gin_trgm_ops)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_leads_email_lower_trgm
  ON leads USING gin (LOWER(email::text) gin_trgm_ops)
  WHERE deleted_at IS NULL AND email IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_leads_phone_trgm
  ON leads USING gin (phone gin_trgm_ops)
  WHERE deleted_at IS NULL AND phone IS NOT NULL;

-- ---------- messages: body (reserved for conversation history search)

CREATE INDEX IF NOT EXISTS idx_messages_body_trgm
  ON messages USING gin (body gin_trgm_ops)
  WHERE body IS NOT NULL AND char_length(body) > 0;

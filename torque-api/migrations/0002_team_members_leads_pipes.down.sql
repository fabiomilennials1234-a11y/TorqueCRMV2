-- =====================================================================
-- 0002_team_members_leads_pipes.down.sql
-- =====================================================================

DROP INDEX IF EXISTS idx_audit_log_target_org_created;
DROP INDEX IF EXISTS idx_audit_log_org_created;
DROP TABLE IF EXISTS audit_log;

DROP INDEX IF EXISTS idx_lead_history_org_lead;
DROP TABLE IF EXISTS lead_history;

DROP INDEX IF EXISTS idx_pipe_entries_org_pipe_stage;
DROP INDEX IF EXISTS idx_pipe_entries_org_lead;
DROP TRIGGER IF EXISTS trg_pipe_entries_updated_at ON pipe_entries;
DROP TABLE IF EXISTS pipe_entries;

DROP INDEX IF EXISTS idx_pipe_stages_org_pipe;
DROP TRIGGER IF EXISTS trg_pipe_stages_updated_at ON pipe_stages;
DROP TABLE IF EXISTS pipe_stages;

DROP INDEX IF EXISTS idx_pipes_org;
DROP TRIGGER IF EXISTS trg_pipes_updated_at ON pipes;
DROP TABLE IF EXISTS pipes;
DROP TYPE  IF EXISTS pipe_kind;

DROP INDEX IF EXISTS idx_lead_tags_org_tag;
DROP TABLE IF EXISTS lead_tags;

DROP INDEX IF EXISTS idx_leads_org_phone;
DROP INDEX IF EXISTS idx_leads_org_email;
DROP INDEX IF EXISTS idx_leads_org_responsible;
DROP INDEX IF EXISTS idx_leads_org_updated_at;
DROP TRIGGER IF EXISTS trg_leads_updated_at ON leads;
DROP TABLE IF EXISTS leads;

DROP TABLE IF EXISTS tags;

DROP TRIGGER IF EXISTS trg_org_quotas_updated_at ON org_quotas;
DROP TABLE IF EXISTS org_quotas;

DROP TRIGGER IF EXISTS trg_member_feature_permissions_updated_at ON member_feature_permissions;
DROP TABLE IF EXISTS member_feature_permissions;

DROP INDEX IF EXISTS idx_team_members_user;
DROP INDEX IF EXISTS idx_team_members_org_active;
DROP TRIGGER IF EXISTS trg_team_members_updated_at ON team_members;
DROP TABLE IF EXISTS team_members;
DROP TYPE  IF EXISTS team_member_role;

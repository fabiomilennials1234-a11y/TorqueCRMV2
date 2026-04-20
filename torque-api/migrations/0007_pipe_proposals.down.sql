DROP TRIGGER IF EXISTS trg_pipe_proposals_updated_at ON pipe_proposals;
DROP INDEX IF EXISTS idx_pipe_proposals_org_expires;
DROP INDEX IF EXISTS idx_pipe_proposals_org_lead;
DROP INDEX IF EXISTS idx_pipe_proposals_org_status;
DROP TABLE IF EXISTS pipe_proposals;
DROP TYPE IF EXISTS proposal_status;

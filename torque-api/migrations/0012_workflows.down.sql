DELETE FROM feature_permissions
 WHERE feature_key IN ('workflows.view','workflows.manage','workflows.run');

DROP INDEX IF EXISTS idx_workflow_run_steps_run_created;
DROP TABLE IF EXISTS workflow_run_steps;

DROP TRIGGER IF EXISTS trg_workflow_runs_updated_at ON workflow_runs;
DROP INDEX IF EXISTS idx_workflow_runs_org_lead;
DROP INDEX IF EXISTS idx_workflow_runs_org_workflow_created;
DROP INDEX IF EXISTS idx_workflow_runs_org_status;
DROP TABLE IF EXISTS workflow_runs;
DROP TYPE IF EXISTS workflow_run_status;

ALTER TABLE workflows DROP CONSTRAINT IF EXISTS fk_workflows_entry_step;

DROP TRIGGER IF EXISTS trg_workflow_steps_updated_at ON workflow_steps;
DROP INDEX IF EXISTS idx_workflow_steps_org_workflow;
DROP TABLE IF EXISTS workflow_steps;
DROP TYPE IF EXISTS workflow_step_kind;

DROP TRIGGER IF EXISTS trg_workflows_updated_at ON workflows;
DROP INDEX IF EXISTS idx_workflows_org_status;
DROP TABLE IF EXISTS workflows;
DROP TYPE IF EXISTS workflow_status;
DROP TYPE IF EXISTS workflow_trigger;

-- Reverse multi-tenancy migration

-- Drop unique constraints
ALTER TABLE workflows DROP CONSTRAINT IF EXISTS workflows_org_workflow_id_version_key;
ALTER TABLE workflows ADD CONSTRAINT workflows_workflow_id_version_key UNIQUE(workflow_id, version);

ALTER TABLE rules DROP CONSTRAINT IF EXISTS rules_org_rule_id_key;
ALTER TABLE rules ADD CONSTRAINT rules_rule_id_key UNIQUE(rule_id);

-- Remove organization_id from api_keys
DROP INDEX IF EXISTS idx_api_keys_organization;
ALTER TABLE api_keys DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from workflow_schedules
DROP INDEX IF EXISTS idx_schedules_organization;
ALTER TABLE workflow_schedules DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from audit_log
DROP INDEX IF EXISTS idx_audit_org_timestamp;
DROP INDEX IF EXISTS idx_audit_organization;
ALTER TABLE audit_log DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from context_cache
DROP INDEX IF EXISTS idx_context_cache_organization;
ALTER TABLE context_cache DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from approval_requests
DROP INDEX IF EXISTS idx_approvals_org_status;
DROP INDEX IF EXISTS idx_approvals_organization;
ALTER TABLE approval_requests DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from events
DROP INDEX IF EXISTS idx_events_org_type;
DROP INDEX IF EXISTS idx_events_organization;
ALTER TABLE events DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from rules
DROP INDEX IF EXISTS idx_rules_org_enabled;
DROP INDEX IF EXISTS idx_rules_organization;
ALTER TABLE rules DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from step_executions
DROP INDEX IF EXISTS idx_step_executions_organization;
ALTER TABLE step_executions DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from workflow_executions
DROP INDEX IF EXISTS idx_executions_org_status;
DROP INDEX IF EXISTS idx_executions_organization;
ALTER TABLE workflow_executions DROP COLUMN IF EXISTS organization_id;

-- Remove organization_id from workflows
DROP INDEX IF EXISTS idx_workflows_org_workflow_id;
DROP INDEX IF EXISTS idx_workflows_organization;
ALTER TABLE workflows DROP COLUMN IF EXISTS organization_id;

-- Drop organization_users table
DROP TABLE IF EXISTS organization_users;

-- Drop organizations table
DROP TABLE IF EXISTS organizations;

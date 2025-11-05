-- Drop triggers
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
DROP TRIGGER IF EXISTS update_rules_updated_at ON rules;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS context_cache;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS step_executions;
DROP TABLE IF EXISTS workflow_executions;
DROP TABLE IF EXISTS workflows;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";

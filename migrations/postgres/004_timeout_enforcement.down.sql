-- Remove timeout enforcement indexes
DROP INDEX IF EXISTS idx_executions_timeout;
DROP INDEX IF EXISTS idx_executions_timeout_status;

-- Remove timeout enforcement columns
ALTER TABLE workflow_executions
DROP COLUMN IF EXISTS timeout_at,
DROP COLUMN IF EXISTS timeout_duration;

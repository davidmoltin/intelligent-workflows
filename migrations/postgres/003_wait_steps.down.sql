-- Remove wait steps support
DROP INDEX IF EXISTS idx_executions_wait_state;
DROP INDEX IF EXISTS idx_executions_waiting;

ALTER TABLE workflow_executions
DROP COLUMN IF EXISTS wait_state,
DROP COLUMN IF EXISTS current_step_id;

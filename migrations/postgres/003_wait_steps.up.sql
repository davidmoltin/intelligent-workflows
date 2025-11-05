-- Add columns to support wait steps and workflow resumption
ALTER TABLE workflow_executions
ADD COLUMN current_step_id VARCHAR(255),
ADD COLUMN wait_state JSONB;

-- Add index for querying waiting executions
CREATE INDEX idx_executions_waiting ON workflow_executions(status)
WHERE status = 'waiting';

-- Add index for wait state to find executions waiting for specific events
CREATE INDEX idx_executions_wait_state ON workflow_executions USING GIN(wait_state)
WHERE wait_state IS NOT NULL;

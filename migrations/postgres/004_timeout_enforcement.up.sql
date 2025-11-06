-- Add timeout enforcement columns to workflow_executions table
ALTER TABLE workflow_executions
ADD COLUMN timeout_at TIMESTAMP,
ADD COLUMN timeout_duration INTEGER; -- Duration in seconds

-- Add index for finding timed-out executions
CREATE INDEX idx_executions_timeout ON workflow_executions(timeout_at)
WHERE timeout_at IS NOT NULL AND status IN ('running', 'waiting');

-- Add index for timeout queries
CREATE INDEX idx_executions_timeout_status ON workflow_executions(status, timeout_at)
WHERE timeout_at IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN workflow_executions.timeout_at IS 'Absolute timestamp when this execution should timeout';
COMMENT ON COLUMN workflow_executions.timeout_duration IS 'Timeout duration in seconds (for reference)';

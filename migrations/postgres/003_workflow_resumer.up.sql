-- Add workflow resumer columns to workflow_executions table
ALTER TABLE workflow_executions
    ADD COLUMN paused_at TIMESTAMP,
    ADD COLUMN paused_reason VARCHAR(255),
    ADD COLUMN paused_step_id UUID,
    ADD COLUMN next_step_id UUID,
    ADD COLUMN resume_data JSONB,
    ADD COLUMN resume_count INTEGER DEFAULT 0,
    ADD COLUMN last_resumed_at TIMESTAMP;

-- Add indexes for paused execution queries
CREATE INDEX idx_executions_paused_at ON workflow_executions(paused_at) WHERE paused_at IS NOT NULL;
CREATE INDEX idx_executions_resume_count ON workflow_executions(resume_count);

-- Update status column comment to include paused status
COMMENT ON COLUMN workflow_executions.status IS 'Execution status: pending, running, completed, failed, blocked, cancelled, paused';

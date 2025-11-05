-- Remove workflow resumer columns from workflow_executions table
ALTER TABLE workflow_executions
    DROP COLUMN IF EXISTS last_resumed_at,
    DROP COLUMN IF EXISTS resume_count,
    DROP COLUMN IF EXISTS resume_data,
    DROP COLUMN IF EXISTS next_step_id,
    DROP COLUMN IF EXISTS paused_step_id,
    DROP COLUMN IF EXISTS paused_reason,
    DROP COLUMN IF EXISTS paused_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_executions_paused_at;
DROP INDEX IF EXISTS idx_executions_resume_count;

-- Restore status column comment
COMMENT ON COLUMN workflow_executions.status IS 'Execution status: pending, running, completed, failed, blocked, cancelled';

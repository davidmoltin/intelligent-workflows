-- Create workflow_schedules table for cron-based scheduling
CREATE TABLE workflow_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMP,
    next_trigger_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Validate cron expression format (basic validation)
    CONSTRAINT valid_cron CHECK (cron_expression ~ '^[@0-9*/,-]+(\s+[@0-9*/,-]+){4,5}$')
);

-- Index for efficiently querying due schedules
CREATE INDEX idx_schedules_next_trigger ON workflow_schedules(next_trigger_at) WHERE enabled = true;

-- Index for querying schedules by workflow
CREATE INDEX idx_schedules_workflow ON workflow_schedules(workflow_id);

-- Index for querying enabled schedules
CREATE INDEX idx_schedules_enabled ON workflow_schedules(enabled) WHERE enabled = true;

-- Comment on table
COMMENT ON TABLE workflow_schedules IS 'Stores cron-based schedules for triggering workflows automatically';

-- Comments on key columns
COMMENT ON COLUMN workflow_schedules.cron_expression IS 'Cron expression defining when the workflow should run (5 or 6 fields)';
COMMENT ON COLUMN workflow_schedules.timezone IS 'Timezone for schedule evaluation (e.g., UTC, America/New_York)';
COMMENT ON COLUMN workflow_schedules.next_trigger_at IS 'Cached next execution time for efficient querying';

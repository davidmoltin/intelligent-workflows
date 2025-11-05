-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflows table
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    definition JSONB NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    tags TEXT[],
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflows_enabled ON workflows(enabled);
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_workflows_workflow_id ON workflows(workflow_id);

-- Workflow executions
CREATE TABLE workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    execution_id VARCHAR(255) UNIQUE NOT NULL,
    trigger_event VARCHAR(255),
    trigger_payload JSONB,
    context JSONB,
    status VARCHAR(50) NOT NULL, -- pending, running, completed, failed, blocked, cancelled
    result VARCHAR(50), -- allowed, blocked, executed, failed
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX idx_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX idx_executions_status ON workflow_executions(status);
CREATE INDEX idx_executions_trigger ON workflow_executions(trigger_event);
CREATE INDEX idx_executions_started_at ON workflow_executions(started_at DESC);

-- Step executions (for tracing)
CREATE TABLE step_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE CASCADE,
    step_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(50),
    status VARCHAR(50) NOT NULL,
    input JSONB,
    output JSONB,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT
);

CREATE INDEX idx_step_executions ON step_executions(execution_id);
CREATE INDEX idx_step_executions_status ON step_executions(status);

-- Rules (reusable conditions)
CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- condition, validation, enrichment
    definition JSONB NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rules_enabled ON rules(enabled);
CREATE INDEX idx_rules_type ON rules(rule_type);

-- Events registry
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    source VARCHAR(255), -- which microservice
    payload JSONB NOT NULL,
    triggered_workflows TEXT[],
    received_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_received ON events(received_at DESC);
CREATE INDEX idx_events_processed ON events(processed_at);

-- Approval requests
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id VARCHAR(255) UNIQUE NOT NULL,
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE CASCADE,
    entity_type VARCHAR(100), -- order, quote, product, etc
    entity_id VARCHAR(255),
    requester_id UUID,
    approver_role VARCHAR(100),
    approver_id UUID,
    status VARCHAR(50) NOT NULL, -- pending, approved, rejected, expired
    reason TEXT,
    decision_reason TEXT,
    requested_at TIMESTAMP DEFAULT NOW(),
    decided_at TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE INDEX idx_approvals_status ON approval_requests(status);
CREATE INDEX idx_approvals_entity ON approval_requests(entity_type, entity_id);
CREATE INDEX idx_approvals_execution ON approval_requests(execution_id);
CREATE INDEX idx_approvals_approver ON approval_requests(approver_id);

-- Context cache (for enrichment data)
CREATE TABLE context_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key VARCHAR(255) UNIQUE NOT NULL,
    entity_type VARCHAR(100),
    entity_id VARCHAR(255),
    data JSONB NOT NULL,
    cached_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX idx_context_cache_entity ON context_cache(entity_type, entity_id);
CREATE INDEX idx_context_cache_expires ON context_cache(expires_at);

-- Audit log
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100),
    entity_id UUID,
    action VARCHAR(100),
    actor_id UUID,
    actor_type VARCHAR(50), -- user, ai_agent, system
    changes JSONB,
    timestamp TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_actor ON audit_log(actor_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rules_updated_at BEFORE UPDATE ON rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

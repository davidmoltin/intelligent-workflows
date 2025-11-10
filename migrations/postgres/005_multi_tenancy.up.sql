-- Organizations table (core multi-tenancy entity)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_is_active ON organizations(is_active);

-- Organization users (many-to-many with role per org)
CREATE TABLE organization_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    joined_at TIMESTAMP DEFAULT NOW(),
    invited_by UUID REFERENCES users(id),
    UNIQUE(organization_id, user_id)
);

CREATE INDEX idx_organization_users_org ON organization_users(organization_id);
CREATE INDEX idx_organization_users_user ON organization_users(user_id);
CREATE INDEX idx_organization_users_role ON organization_users(role_id);
CREATE INDEX idx_organization_users_active ON organization_users(organization_id, user_id, is_active);

-- Add organization_id to workflows table
ALTER TABLE workflows ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_workflows_organization ON workflows(organization_id);
CREATE INDEX idx_workflows_org_workflow_id ON workflows(organization_id, workflow_id);

-- Add organization_id to workflow_executions table
ALTER TABLE workflow_executions ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_executions_organization ON workflow_executions(organization_id);
CREATE INDEX idx_executions_org_status ON workflow_executions(organization_id, status);

-- Add organization_id to step_executions table
ALTER TABLE step_executions ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_step_executions_organization ON step_executions(organization_id);

-- Add organization_id to rules table
ALTER TABLE rules ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_rules_organization ON rules(organization_id);
CREATE INDEX idx_rules_org_enabled ON rules(organization_id, enabled);

-- Add organization_id to events table
ALTER TABLE events ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_events_organization ON events(organization_id);
CREATE INDEX idx_events_org_type ON events(organization_id, event_type);

-- Add organization_id to approval_requests table
ALTER TABLE approval_requests ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_approvals_organization ON approval_requests(organization_id);
CREATE INDEX idx_approvals_org_status ON approval_requests(organization_id, status);

-- Add organization_id to context_cache table
ALTER TABLE context_cache ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_context_cache_organization ON context_cache(organization_id);

-- Add organization_id to audit_log table
ALTER TABLE audit_log ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
CREATE INDEX idx_audit_organization ON audit_log(organization_id);
CREATE INDEX idx_audit_org_timestamp ON audit_log(organization_id, timestamp DESC);

-- Add organization_id to workflow_schedules table (if exists)
ALTER TABLE workflow_schedules ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_schedules_organization ON workflow_schedules(organization_id);

-- Add organization_id to api_keys table (API keys are org-scoped)
ALTER TABLE api_keys ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX idx_api_keys_organization ON api_keys(organization_id);

-- Trigger for organizations updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create default organization and migrate existing data
DO $$
DECLARE
    default_org_id UUID;
    first_user_id UUID;
    admin_role_id UUID;
BEGIN
    -- Get first user (if any) to be the creator
    SELECT id INTO first_user_id FROM users ORDER BY created_at LIMIT 1;

    -- Create default organization
    INSERT INTO organizations (name, slug, description, is_active, created_by)
    VALUES (
        'Default Organization',
        'default',
        'Default organization for migrated data',
        true,
        first_user_id
    )
    RETURNING id INTO default_org_id;

    -- Get admin role
    SELECT id INTO admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;

    -- Assign all existing users to default organization
    INSERT INTO organization_users (organization_id, user_id, role_id, joined_at, invited_by)
    SELECT
        default_org_id,
        u.id,
        COALESCE(
            (SELECT role_id FROM user_roles WHERE user_id = u.id LIMIT 1),
            admin_role_id
        ),
        u.created_at,
        first_user_id
    FROM users u
    ON CONFLICT (organization_id, user_id) DO NOTHING;

    -- Migrate existing workflows to default organization
    UPDATE workflows SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing workflow_executions to default organization
    UPDATE workflow_executions SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing step_executions to default organization
    UPDATE step_executions SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing rules to default organization
    UPDATE rules SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing events to default organization
    UPDATE events SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing approval_requests to default organization
    UPDATE approval_requests SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing context_cache to default organization
    UPDATE context_cache SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing audit_log to default organization
    UPDATE audit_log SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing workflow_schedules to default organization (if any)
    UPDATE workflow_schedules SET organization_id = default_org_id WHERE organization_id IS NULL;

    -- Migrate existing api_keys to default organization
    UPDATE api_keys SET organization_id = default_org_id WHERE organization_id IS NULL;

    RAISE NOTICE 'Multi-tenancy migration completed. Default organization ID: %', default_org_id;
END $$;

-- Make organization_id NOT NULL after migration
ALTER TABLE workflows ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE workflow_executions ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE step_executions ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE rules ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE events ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE approval_requests ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE context_cache ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE workflow_schedules ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE api_keys ALTER COLUMN organization_id SET NOT NULL;
-- Note: audit_log organization_id remains nullable for system-level audits

-- Add unique constraint for workflow_id scoped to organization
ALTER TABLE workflows DROP CONSTRAINT IF EXISTS workflows_workflow_id_version_key;
ALTER TABLE workflows ADD CONSTRAINT workflows_org_workflow_id_version_key
    UNIQUE(organization_id, workflow_id, version);

-- Add unique constraint for rule_id scoped to organization
ALTER TABLE rules DROP CONSTRAINT IF EXISTS rules_rule_id_key;
ALTER TABLE rules ADD CONSTRAINT rules_org_rule_id_key
    UNIQUE(organization_id, rule_id);

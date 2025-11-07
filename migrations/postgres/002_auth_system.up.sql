-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_roles_name ON roles(name);

-- Permissions table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(100) NOT NULL, -- workflow, execution, approval, event, etc.
    action VARCHAR(50) NOT NULL, -- create, read, update, delete, execute
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_permissions_resource ON permissions(resource);
CREATE INDEX idx_permissions_action ON permissions(action);
CREATE UNIQUE INDEX idx_permissions_resource_action ON permissions(resource, action);

-- User roles (many-to-many)
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);

-- Role permissions (many-to-many)
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

-- API Keys table (for agent authentication)
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    key_prefix VARCHAR(20) NOT NULL, -- First few chars for identification
    name VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    scopes TEXT[], -- Array of allowed scopes/permissions
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX idx_api_keys_expires ON api_keys(expires_at);

-- Refresh tokens table (for JWT refresh)
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- Rate limit tracking table
CREATE TABLE rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL, -- user_id, api_key, or IP address
    identifier_type VARCHAR(50) NOT NULL, -- user, api_key, ip
    endpoint VARCHAR(255) NOT NULL,
    request_count INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT NOW(),
    last_request_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rate_limits_identifier ON rate_limits(identifier, endpoint);
CREATE INDEX idx_rate_limits_window ON rate_limits(window_start);

-- Login attempts tracking (for security)
CREATE TABLE login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    success BOOLEAN NOT NULL,
    attempted_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_username ON login_attempts(username);
CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address);
CREATE INDEX idx_login_attempts_attempted ON login_attempts(attempted_at DESC);

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default roles (idempotent)
INSERT INTO roles (name, description) VALUES
    ('admin', 'Administrator with full system access'),
    ('workflow_manager', 'Can manage workflows and view executions'),
    ('workflow_viewer', 'Can view workflows and executions'),
    ('approver', 'Can approve/reject approval requests'),
    ('agent', 'AI agent with limited API access')
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    updated_at = NOW();

-- Insert default permissions (idempotent)
INSERT INTO permissions (name, resource, action, description) VALUES
    -- Workflow permissions
    ('workflow:create', 'workflow', 'create', 'Create new workflows'),
    ('workflow:read', 'workflow', 'read', 'View workflows'),
    ('workflow:update', 'workflow', 'update', 'Update workflows'),
    ('workflow:delete', 'workflow', 'delete', 'Delete workflows'),
    ('workflow:execute', 'workflow', 'execute', 'Execute workflows'),

    -- Execution permissions
    ('execution:read', 'execution', 'read', 'View workflow executions'),
    ('execution:cancel', 'execution', 'cancel', 'Cancel running executions'),
    ('execution:pause', 'execution', 'pause', 'Pause running executions'),
    ('execution:resume', 'execution', 'resume', 'Resume paused executions'),

    -- Approval permissions
    ('approval:read', 'approval', 'read', 'View approval requests'),
    ('approval:approve', 'approval', 'approve', 'Approve requests'),
    ('approval:reject', 'approval', 'reject', 'Reject requests'),

    -- Event permissions
    ('event:create', 'event', 'create', 'Create events'),
    ('event:read', 'event', 'read', 'View events'),

    -- User management permissions
    ('user:create', 'user', 'create', 'Create users'),
    ('user:read', 'user', 'read', 'View users'),
    ('user:update', 'user', 'update', 'Update users'),
    ('user:delete', 'user', 'delete', 'Delete users'),

    -- Role management permissions (granular)
    ('role:create', 'role', 'create', 'Create roles'),
    ('role:read', 'role', 'read', 'View roles'),
    ('role:update', 'role', 'update', 'Update roles'),
    ('role:delete', 'role', 'delete', 'Delete roles'),
    ('role:assign', 'role', 'assign', 'Assign roles to users')
ON CONFLICT (name) DO UPDATE
SET resource = EXCLUDED.resource,
    action = EXCLUDED.action,
    description = EXCLUDED.description;

-- Assign permissions to roles (idempotent)
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Workflow Manager permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'workflow_manager'
AND p.name IN (
    'workflow:create', 'workflow:read', 'workflow:update', 'workflow:delete', 'workflow:execute',
    'execution:read', 'execution:cancel', 'execution:pause', 'execution:resume',
    'event:read', 'approval:read'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Workflow Viewer permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'workflow_viewer'
AND p.name IN ('workflow:read', 'execution:read', 'event:read', 'approval:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Approver permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'approver'
AND p.name IN ('approval:read', 'approval:approve', 'approval:reject', 'execution:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Agent permissions (limited API access)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'agent'
AND p.name IN ('workflow:execute', 'event:create', 'event:read', 'execution:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

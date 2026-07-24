-- Phase 12: Centralized IAM Schema
-- Single source of truth for all identity and access management

-- ============================================================================
-- STEP 1: Create IAM Schema
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS iam;

-- ============================================================================
-- STEP 2: Core IAM Tables
-- ============================================================================

-- Roles table (tenant-scoped or global)
CREATE TABLE IF NOT EXISTS iam.roles (
    role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- 'uisce_org_uuid' for global roles
    role_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_global_admin BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    created_by UUID,
    UNIQUE(tenant_id, role_name)
);

CREATE INDEX idx_iam_roles_tenant ON iam.roles(tenant_id);
CREATE INDEX idx_iam_roles_global_admin ON iam.roles(is_global_admin) WHERE is_global_admin = true;

-- Permissions table (actions on resources)
CREATE TABLE IF NOT EXISTS iam.permissions (
    permission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(255) NOT NULL, -- e.g., 'read', 'write', 'delete'
    resource VARCHAR(255) NOT NULL, -- e.g., 'orders', 'dashboard:123', 'business_objects'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(action, resource)
);

CREATE INDEX idx_iam_permissions_resource ON iam.permissions(resource);

-- Role-Permission mappings
CREATE TABLE IF NOT EXISTS iam.role_permissions (
    role_id UUID REFERENCES iam.roles(role_id) ON DELETE CASCADE,
    permission_id UUID REFERENCES iam.permissions(permission_id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ DEFAULT now(),
    granted_by UUID,
    PRIMARY KEY(role_id, permission_id)
);

CREATE INDEX idx_iam_role_permissions_role ON iam.role_permissions(role_id);
CREATE INDEX idx_iam_role_permissions_permission ON iam.role_permissions(permission_id);

-- User-Role assignments
CREATE TABLE IF NOT EXISTS iam.user_roles (
    user_id UUID NOT NULL,
    role_id UUID REFERENCES iam.roles(role_id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT now(),
    assigned_by UUID,
    expires_at TIMESTAMPTZ, -- Optional: for temporary role assignments
    PRIMARY KEY(user_id, role_id)
);

CREATE INDEX idx_iam_user_roles_user ON iam.user_roles(user_id);
CREATE INDEX idx_iam_user_roles_role ON iam.user_roles(role_id);
CREATE INDEX idx_iam_user_roles_expires ON iam.user_roles(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- STEP 3: Audit Tables for Security Events
-- ============================================================================

-- Security event log (for audit and analytics)
CREATE TABLE IF NOT EXISTS iam.security_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL, -- 'role_created', 'role_updated', 'permission_granted', etc.
    entity_type VARCHAR(50) NOT NULL, -- 'role', 'permission', 'user_role'
    entity_id UUID NOT NULL,
    tenant_id UUID,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by UUID,
    processed BOOLEAN DEFAULT false
);

CREATE INDEX idx_iam_security_events_type ON iam.security_events(event_type);
CREATE INDEX idx_iam_security_events_created ON iam.security_events(created_at);
CREATE INDEX idx_iam_security_events_processed ON iam.security_events(processed) WHERE processed = false;

-- Sync status tracking (which systems have processed each event)
CREATE TABLE IF NOT EXISTS iam.sync_status (
    event_id UUID REFERENCES iam.security_events(event_id) ON DELETE CASCADE,
    system VARCHAR(50) NOT NULL, -- 'postgresql', 'hasura', 'superset', 'starrocks'
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'success', 'failed')),
    error_message TEXT,
    synced_at TIMESTAMPTZ,
    retry_count INTEGER DEFAULT 0,
    PRIMARY KEY(event_id, system)
);

CREATE INDEX idx_iam_sync_status_status ON iam.sync_status(status, system);

-- ============================================================================
-- STEP 4: Helper Functions
-- ============================================================================

-- Function to create a security event (called by application code, not triggers)
CREATE OR REPLACE FUNCTION iam.create_security_event(
    p_event_type VARCHAR,
    p_entity_type VARCHAR,
    p_entity_id UUID,
    p_tenant_id UUID,
    p_payload JSONB,
    p_created_by UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
BEGIN
    INSERT INTO iam.security_events (event_type, entity_type, entity_id, tenant_id, payload, created_by)
    VALUES (p_event_type, p_entity_type, p_entity_id, p_tenant_id, p_payload, p_created_by)
    RETURNING event_id INTO v_event_id;
    
    -- Initialize sync status for all systems
    INSERT INTO iam.sync_status (event_id, system, status)
    VALUES 
        (v_event_id, 'postgresql', 'pending'),
        (v_event_id, 'hasura', 'pending'),
        (v_event_id, 'superset', 'pending'),
        (v_event_id, 'starrocks', 'pending');
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get all roles for a user (including inherited permissions)
CREATE OR REPLACE FUNCTION iam.get_user_permissions(p_user_id UUID)
RETURNS TABLE (
    role_name VARCHAR,
    action VARCHAR,
    resource VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.role_name,
        p.action,
        p.resource
    FROM iam.user_roles ur
    JOIN iam.roles r ON ur.role_id = r.role_id
    JOIN iam.role_permissions rp ON r.role_id = rp.role_id
    JOIN iam.permissions p ON rp.permission_id = p.permission_id
    WHERE ur.user_id = p_user_id
      AND (ur.expires_at IS NULL OR ur.expires_at > now());
END;
$$ LANGUAGE plpgsql;

-- Function to check if user has specific permission
CREATE OR REPLACE FUNCTION iam.user_has_permission(
    p_user_id UUID,
    p_action VARCHAR,
    p_resource VARCHAR
)
RETURNS BOOLEAN AS $$
DECLARE
    v_has_permission BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1
        FROM iam.get_user_permissions(p_user_id)
        WHERE action = p_action AND resource = p_resource
    ) INTO v_has_permission;
    
    RETURN v_has_permission;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 5: Seed Data for Uisce Global Admin
-- ============================================================================

-- Insert Uisce organization tenant (if not exists)
INSERT INTO tenants (id, name, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'uisce',
    'Uisce Organization'
) ON CONFLICT (id) DO NOTHING;

-- Create global admin role
INSERT INTO iam.roles (role_id, tenant_id, role_name, description, is_global_admin)
VALUES (
    '99999999-9999-9999-9999-999999999999',
    '00000000-0000-0000-0000-000000000000',
    'global_admin',
    'Uisce organization global administrators with full access to all tenants',
    true
) ON CONFLICT (tenant_id, role_name) DO NOTHING;

-- Create standard user role template
INSERT INTO iam.roles (role_id, tenant_id, role_name, description, is_global_admin)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000000',
    'tenant_user',
    'Standard tenant user with read/write access to their tenant data',
    false
) ON CONFLICT (tenant_id, role_name) DO NOTHING;

-- Create common permissions
INSERT INTO iam.permissions (action, resource, description) VALUES
    ('read', 'business_objects', 'Read business objects'),
    ('write', 'business_objects', 'Create/update business objects'),
    ('delete', 'business_objects', 'Delete business objects'),
    ('read', 'processes', 'Read processes'),
    ('write', 'processes', 'Create/update processes'),
    ('execute', 'processes', 'Execute processes'),
    ('read', 'dashboards', 'View dashboards'),
    ('write', 'dashboards', 'Create/edit dashboards'),
    ('admin', '*', 'Full administrative access')
ON CONFLICT (action, resource) DO NOTHING;

-- Grant all permissions to global admin
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT 
    '99999999-9999-9999-9999-999999999999',
    permission_id
FROM iam.permissions
ON CONFLICT DO NOTHING;

-- Grant read/write permissions to tenant_user role
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT 
    '11111111-1111-1111-1111-111111111111',
    permission_id
FROM iam.permissions
WHERE action IN ('read', 'write') AND resource != '*'
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VERIFICATION
-- ============================================================================

-- Verify IAM schema
SELECT 'IAM Schema Created' as status;
SELECT COUNT(*) as role_count FROM iam.roles;
SELECT COUNT(*) as permission_count FROM iam.permissions;
SELECT COUNT(*) as role_permission_count FROM iam.role_permissions;

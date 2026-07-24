-- ============================================================================
-- Advanced RBAC & Permissions System
-- Fortune 500 Enterprise-Grade Security
-- ============================================================================

-- ============================================================================
-- 1. CORE RBAC TABLES
-- ============================================================================

-- Roles Table (if not exists)
CREATE TABLE IF NOT EXISTS bp_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    role_key VARCHAR(100) NOT NULL,
    role_name VARCHAR(200) NOT NULL,
    description TEXT,
    role_type VARCHAR(50) NOT NULL, -- 'system' | 'custom'
    role_level VARCHAR(50) NOT NULL, -- 'viewer' | 'editor' | 'admin' | 'super_admin'
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, datasource_id, role_key)
);

CREATE INDEX idx_bp_roles_tenant_datasource ON bp_roles(tenant_id, datasource_id);
CREATE INDEX idx_bp_roles_level ON bp_roles(role_level);
CREATE INDEX idx_bp_roles_active ON bp_roles(is_active);

-- Permissions Table
CREATE TABLE IF NOT EXISTS bp_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    permission_key VARCHAR(100) NOT NULL,
    permission_name VARCHAR(200) NOT NULL,
    description TEXT,
    resource_type VARCHAR(100) NOT NULL, -- 'process' | 'step' | 'field' | 'document' | 'report'
    action VARCHAR(50) NOT NULL, -- 'read' | 'create' | 'update' | 'delete' | 'execute' | 'approve'
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, permission_key)
);

CREATE INDEX idx_bp_permissions_tenant ON bp_permissions(tenant_id);
CREATE INDEX idx_bp_permissions_resource ON bp_permissions(resource_type);
CREATE INDEX idx_bp_permissions_action ON bp_permissions(action);

-- Role-Permission Mapping
CREATE TABLE IF NOT EXISTS bp_role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES bp_roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES bp_permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID,
    UNIQUE(role_id, permission_id)
);

CREATE INDEX idx_bp_role_permissions_role ON bp_role_permissions(role_id);
CREATE INDEX idx_bp_role_permissions_permission ON bp_role_permissions(permission_id);

-- User-Role Assignment
CREATE TABLE IF NOT EXISTS bp_user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL REFERENCES bp_roles(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    scope_type VARCHAR(50), -- 'global' | 'process' | 'step' | 'team'
    scope_id UUID, -- ID of process/step/team if scoped
    assigned_by UUID,
    assigned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ, -- Optional expiration
    is_active BOOLEAN DEFAULT true,
    UNIQUE(user_id, role_id, tenant_id, datasource_id, scope_type, scope_id)
);

CREATE INDEX idx_bp_user_roles_user ON bp_user_roles(user_id);
CREATE INDEX idx_bp_user_roles_role ON bp_user_roles(role_id);
CREATE INDEX idx_bp_user_roles_tenant_datasource ON bp_user_roles(tenant_id, datasource_id);
CREATE INDEX idx_bp_user_roles_scope ON bp_user_roles(scope_type, scope_id);
CREATE INDEX idx_bp_user_roles_active ON bp_user_roles(is_active);

-- ============================================================================
-- 2. FIELD-LEVEL PERMISSIONS
-- ============================================================================

-- Field-Level Access Control
CREATE TABLE IF NOT EXISTS bp_field_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    role_id UUID NOT NULL REFERENCES bp_roles(id) ON DELETE CASCADE,
    resource_type VARCHAR(100) NOT NULL, -- 'process' | 'step' | 'form' | 'document'
    resource_id UUID, -- NULL for all resources of type
    field_name VARCHAR(200) NOT NULL,
    permission_level VARCHAR(50) NOT NULL, -- 'none' | 'read' | 'write' | 'mask'
    field_condition JSONB, -- Optional conditions: {"when": {"status": "draft"}}
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bp_field_permissions_tenant_datasource ON bp_field_permissions(tenant_id, datasource_id);
CREATE INDEX idx_bp_field_permissions_role ON bp_field_permissions(role_id);
CREATE INDEX idx_bp_field_permissions_resource ON bp_field_permissions(resource_type, resource_id);
CREATE INDEX idx_bp_field_permissions_field ON bp_field_permissions(field_name);

-- Field Masking Rules (PII/Sensitive Data)
CREATE TABLE IF NOT EXISTS bp_field_masking_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    field_name VARCHAR(200) NOT NULL,
    masking_type VARCHAR(50) NOT NULL, -- 'full' | 'partial' | 'hash' | 'tokenize'
    masking_pattern VARCHAR(200), -- e.g., 'XXX-XX-####' for SSN
    unmasked_roles UUID[], -- Array of role IDs that can see unmasked
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, datasource_id, resource_type, field_name)
);

CREATE INDEX idx_bp_field_masking_tenant_datasource ON bp_field_masking_rules(tenant_id, datasource_id);
CREATE INDEX idx_bp_field_masking_resource ON bp_field_masking_rules(resource_type);
CREATE INDEX idx_bp_field_masking_active ON bp_field_masking_rules(is_active);

-- ============================================================================
-- 3. APPROVAL DELEGATION
-- ============================================================================

-- Delegation Rules
CREATE TABLE IF NOT EXISTS bp_approval_delegations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    delegator_user_id UUID NOT NULL, -- User delegating authority
    delegate_user_id UUID NOT NULL, -- User receiving authority
    delegation_type VARCHAR(50) NOT NULL, -- 'full' | 'partial' | 'backup'
    resource_type VARCHAR(100), -- NULL for all, or specific: 'process' | 'approval'
    resource_id UUID, -- NULL for all resources, or specific process/approval ID
    scope_conditions JSONB, -- {"process_types": ["expense"], "amount_limit": 10000}
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ, -- NULL for indefinite
    reason TEXT,
    is_active BOOLEAN DEFAULT true,
    requires_notification BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bp_approval_delegations_delegator ON bp_approval_delegations(delegator_user_id);
CREATE INDEX idx_bp_approval_delegations_delegate ON bp_approval_delegations(delegate_user_id);
CREATE INDEX idx_bp_approval_delegations_tenant_datasource ON bp_approval_delegations(tenant_id, datasource_id);
CREATE INDEX idx_bp_approval_delegations_dates ON bp_approval_delegations(start_date, end_date);
CREATE INDEX idx_bp_approval_delegations_active ON bp_approval_delegations(is_active);

-- Delegation History/Audit
CREATE TABLE IF NOT EXISTS bp_delegation_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delegation_id UUID NOT NULL REFERENCES bp_approval_delegations(id) ON DELETE CASCADE,
    delegate_user_id UUID NOT NULL,
    action_type VARCHAR(100) NOT NULL, -- 'approval' | 'rejection' | 'reassignment'
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID NOT NULL,
    action_details JSONB,
    action_taken_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bp_delegation_usage_delegation ON bp_delegation_usage_log(delegation_id);
CREATE INDEX idx_bp_delegation_usage_delegate ON bp_delegation_usage_log(delegate_user_id);
CREATE INDEX idx_bp_delegation_usage_resource ON bp_delegation_usage_log(resource_type, resource_id);

-- ============================================================================
-- 4. TEAM ACCESS & SHARED OWNERSHIP
-- ============================================================================

-- Teams
CREATE TABLE IF NOT EXISTS bp_teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    team_key VARCHAR(100) NOT NULL,
    team_name VARCHAR(200) NOT NULL,
    description TEXT,
    team_type VARCHAR(50) NOT NULL, -- 'functional' | 'project' | 'cross_functional'
    parent_team_id UUID REFERENCES bp_teams(id) ON DELETE SET NULL,
    manager_user_id UUID,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, datasource_id, team_key)
);

CREATE INDEX idx_bp_teams_tenant_datasource ON bp_teams(tenant_id, datasource_id);
CREATE INDEX idx_bp_teams_parent ON bp_teams(parent_team_id);
CREATE INDEX idx_bp_teams_manager ON bp_teams(manager_user_id);
CREATE INDEX idx_bp_teams_active ON bp_teams(is_active);

-- Team Members
CREATE TABLE IF NOT EXISTS bp_team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES bp_teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role_in_team VARCHAR(50) NOT NULL, -- 'member' | 'lead' | 'admin'
    joined_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    UNIQUE(team_id, user_id)
);

CREATE INDEX idx_bp_team_members_team ON bp_team_members(team_id);
CREATE INDEX idx_bp_team_members_user ON bp_team_members(user_id);
CREATE INDEX idx_bp_team_members_active ON bp_team_members(is_active);

-- Team Permissions (Resource Access)
CREATE TABLE IF NOT EXISTS bp_team_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES bp_teams(id) ON DELETE CASCADE,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID NOT NULL,
    permission_level VARCHAR(50) NOT NULL, -- 'read' | 'write' | 'admin'
    granted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID,
    UNIQUE(team_id, resource_type, resource_id)
);

CREATE INDEX idx_bp_team_permissions_team ON bp_team_permissions(team_id);
CREATE INDEX idx_bp_team_permissions_resource ON bp_team_permissions(resource_type, resource_id);

-- ============================================================================
-- 5. PERMISSION AUDIT LOG
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_permission_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    action_type VARCHAR(100) NOT NULL, -- 'permission_grant' | 'permission_revoke' | 'role_assign' | 'delegation_create'
    subject_type VARCHAR(50) NOT NULL, -- 'user' | 'role' | 'team'
    subject_id UUID NOT NULL,
    object_type VARCHAR(50) NOT NULL, -- 'permission' | 'role' | 'resource'
    object_id UUID NOT NULL,
    performed_by UUID NOT NULL,
    before_state JSONB,
    after_state JSONB,
    reason TEXT,
    ip_address INET,
    user_agent TEXT,
    performed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bp_permission_audit_tenant_datasource ON bp_permission_audit_log(tenant_id, datasource_id);
CREATE INDEX idx_bp_permission_audit_action ON bp_permission_audit_log(action_type);
CREATE INDEX idx_bp_permission_audit_subject ON bp_permission_audit_log(subject_type, subject_id);
CREATE INDEX idx_bp_permission_audit_object ON bp_permission_audit_log(object_type, object_id);
CREATE INDEX idx_bp_permission_audit_performed_by ON bp_permission_audit_log(performed_by);
CREATE INDEX idx_bp_permission_audit_time ON bp_permission_audit_log(performed_at);

-- ============================================================================
-- 6. AUTO-UPDATE TRIGGERS
-- ============================================================================

-- Auto-update bp_roles.updated_at
CREATE OR REPLACE FUNCTION bp_update_roles_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER bp_roles_update_timestamp
BEFORE UPDATE ON bp_roles
FOR EACH ROW EXECUTE FUNCTION bp_update_roles_timestamp();

-- Auto-update bp_field_permissions.updated_at
CREATE TRIGGER bp_field_permissions_update_timestamp
BEFORE UPDATE ON bp_field_permissions
FOR EACH ROW EXECUTE FUNCTION bp_update_roles_timestamp();

-- Auto-update bp_approval_delegations.updated_at
CREATE TRIGGER bp_approval_delegations_update_timestamp
BEFORE UPDATE ON bp_approval_delegations
FOR EACH ROW EXECUTE FUNCTION bp_update_roles_timestamp();

-- Auto-update bp_teams.updated_at
CREATE TRIGGER bp_teams_update_timestamp
BEFORE UPDATE ON bp_teams
FOR EACH ROW EXECUTE FUNCTION bp_update_roles_timestamp();

-- ============================================================================
-- 7. HELPER FUNCTIONS
-- ============================================================================

-- Check if user has permission
CREATE OR REPLACE FUNCTION bp_user_has_permission(
    p_user_id UUID,
    p_tenant_id UUID,
    p_datasource_id UUID,
    p_permission_key VARCHAR
)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM bp_user_roles ur
        JOIN bp_role_permissions rp ON ur.role_id = rp.role_id
        JOIN bp_permissions p ON rp.permission_id = p.id
        WHERE ur.user_id = p_user_id
          AND ur.tenant_id = p_tenant_id
          AND ur.datasource_id = p_datasource_id
          AND p.permission_key = p_permission_key
          AND ur.is_active = true
          AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
    ) INTO has_perm;
    
    RETURN has_perm;
END;
$$ LANGUAGE plpgsql;

-- Get user's effective permissions for a resource
CREATE OR REPLACE FUNCTION bp_get_user_resource_permissions(
    p_user_id UUID,
    p_tenant_id UUID,
    p_datasource_id UUID,
    p_resource_type VARCHAR,
    p_resource_id UUID
)
RETURNS TABLE(permission_key VARCHAR, action VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT p.permission_key, p.action
    FROM bp_user_roles ur
    JOIN bp_role_permissions rp ON ur.role_id = rp.role_id
    JOIN bp_permissions p ON rp.permission_id = p.id
    WHERE ur.user_id = p_user_id
      AND ur.tenant_id = p_tenant_id
      AND ur.datasource_id = p_datasource_id
      AND p.resource_type = p_resource_type
      AND ur.is_active = true
      AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP);
END;
$$ LANGUAGE plpgsql;

-- Check if user can act on behalf of another (delegation)
CREATE OR REPLACE FUNCTION bp_get_active_delegation(
    p_delegator_user_id UUID,
    p_delegate_user_id UUID,
    p_tenant_id UUID,
    p_datasource_id UUID,
    p_resource_type VARCHAR DEFAULT NULL,
    p_resource_id UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    delegation_id UUID;
BEGIN
    SELECT id INTO delegation_id
    FROM bp_approval_delegations
    WHERE delegator_user_id = p_delegator_user_id
      AND delegate_user_id = p_delegate_user_id
      AND tenant_id = p_tenant_id
      AND datasource_id = p_datasource_id
      AND is_active = true
      AND start_date <= CURRENT_TIMESTAMP
      AND (end_date IS NULL OR end_date > CURRENT_TIMESTAMP)
      AND (resource_type IS NULL OR resource_type = p_resource_type)
      AND (resource_id IS NULL OR resource_id = p_resource_id)
    ORDER BY start_date DESC
    LIMIT 1;
    
    RETURN delegation_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 8. COMMENTS
-- ============================================================================

COMMENT ON TABLE bp_roles IS 'Role definitions with hierarchical levels (viewer/editor/admin)';
COMMENT ON TABLE bp_permissions IS 'Granular permission definitions for resources and actions';
COMMENT ON TABLE bp_role_permissions IS 'Maps permissions to roles';
COMMENT ON TABLE bp_user_roles IS 'Assigns roles to users with optional scope and expiration';
COMMENT ON TABLE bp_field_permissions IS 'Field-level access control for sensitive data';
COMMENT ON TABLE bp_field_masking_rules IS 'PII/sensitive field masking rules';
COMMENT ON TABLE bp_approval_delegations IS 'Approval authority delegation (vacation, backup)';
COMMENT ON TABLE bp_delegation_usage_log IS 'Audit trail for delegated actions';
COMMENT ON TABLE bp_teams IS 'Team/group definitions for shared access';
COMMENT ON TABLE bp_team_members IS 'Team membership roster';
COMMENT ON TABLE bp_team_permissions IS 'Team-level resource access grants';
COMMENT ON TABLE bp_permission_audit_log IS 'Complete audit trail for all permission changes';

COMMENT ON FUNCTION bp_user_has_permission IS 'Check if user has a specific permission';
COMMENT ON FUNCTION bp_get_user_resource_permissions IS 'Get all permissions user has on a resource';
COMMENT ON FUNCTION bp_get_active_delegation IS 'Find active delegation between two users for a resource';

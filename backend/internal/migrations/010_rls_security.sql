-- Phase 11.2: Row-Level Security (RLS) Implementation
-- This migration creates the database roles and RLS policies for multi-tenant security

-- ============================================================================
-- STEP 1: Create Database Roles
-- ============================================================================

-- Role for standard tenant users (cannot bypass RLS)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tenant_user_role') THEN
        CREATE ROLE tenant_user_role NOLOGIN;
    END IF;
END
$$;

-- Role for global admins (Uisce organization) - can bypass RLS
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'global_admin_role') THEN
        CREATE ROLE global_admin_role NOLOGIN BYPASSRLS;
    END IF;
END
$$;

-- Grant base database permissions
GRANT CONNECT ON DATABASE alpha TO tenant_user_role;
GRANT CONNECT ON DATABASE alpha TO global_admin_role;

GRANT USAGE ON SCHEMA public TO tenant_user_role;
GRANT USAGE ON SCHEMA public TO global_admin_role;

-- ============================================================================
-- STEP 2: Grant Table Permissions
-- ============================================================================

-- Grant SELECT, INSERT, UPDATE, DELETE to tenant_user_role on all tenant-scoped tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO tenant_user_role;

-- Grant ALL permissions to global_admin_role
GRANT ALL ON ALL TABLES IN SCHEMA public TO global_admin_role;

-- Grant sequence permissions (for auto-increment IDs)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO tenant_user_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO global_admin_role;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tenant_user_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT ALL ON TABLES TO global_admin_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT USAGE, SELECT ON SEQUENCES TO tenant_user_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT ALL ON SEQUENCES TO global_admin_role;

-- ============================================================================
-- STEP 3: Enable RLS on Tenant-Scoped Tables
-- ============================================================================

-- Core catalog tables
ALTER TABLE catalog_node ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_edge ENABLE ROW LEVEL SECURITY;

-- Business process tables
ALTER TABLE business_objects ENABLE ROW LEVEL SECURITY;
ALTER TABLE processes ENABLE ROW LEVEL SECURITY;
ALTER TABLE process_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE validation_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_handlers ENABLE ROW LEVEL SECURITY;
ALTER TABLE step_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE rule_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE process_execution_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE process_designer_permissions ENABLE ROW LEVEL SECURITY;

-- Workflow configuration tables
ALTER TABLE process_step_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE validation_operators ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_events ENABLE ROW LEVEL SECURITY;

-- Page and pipeline tables
ALTER TABLE page_layouts ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipelines ENABLE ROW LEVEL SECURITY;

-- Household tables
ALTER TABLE households ENABLE ROW LEVEL SECURITY;
ALTER TABLE household_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE household_semantic_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE household_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE household_report_logs ENABLE ROW LEVEL SECURITY;

-- User and tenant management
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_datasources ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_connections ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- STEP 4: Create RLS Policies for Tenant Isolation
-- ============================================================================

-- Policy template: tenant_user_role can only see rows matching their tenant_id
-- The tenant_id is set via session variable: app.current_tenant_id

-- Catalog tables
CREATE POLICY tenant_isolation_policy ON catalog_node
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON catalog_edge
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Business process tables
CREATE POLICY tenant_isolation_policy ON business_objects
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON processes
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON process_versions
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON validation_rules
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON event_handlers
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON step_templates
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON rule_templates
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON process_execution_log
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON process_designer_permissions
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Workflow configuration tables (allow NULL tenant_id for system-wide configs)
CREATE POLICY tenant_isolation_policy ON process_step_types
    FOR ALL
    TO tenant_user_role
    USING (tenant_id IS NULL OR tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON validation_operators
    FOR ALL
    TO tenant_user_role
    USING (tenant_id IS NULL OR tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON workflow_events
    FOR ALL
    TO tenant_user_role
    USING (tenant_id IS NULL OR tenant_id::text = current_setting('app.current_tenant_id', true));

-- Page and pipeline tables
CREATE POLICY tenant_isolation_policy ON page_layouts
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON pipelines
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Household tables
CREATE POLICY tenant_isolation_policy ON households
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON household_members
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON household_semantic_mappings
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON household_reports
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON household_report_logs
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Users table (users can only see users in their tenant)
CREATE POLICY tenant_isolation_policy ON users
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- Tenant datasources and connections
CREATE POLICY tenant_isolation_policy ON tenant_datasources
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON tenant_connections
    FOR ALL
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================================
-- STEP 5: Create Policies for Global Admins
-- ============================================================================

-- Global admins have BYPASSRLS attribute, so they automatically bypass all policies
-- However, we create explicit policies for clarity and auditability

CREATE POLICY global_admin_access_policy ON catalog_node
    FOR ALL
    TO global_admin_role
    USING (true);

CREATE POLICY global_admin_access_policy ON catalog_edge
    FOR ALL
    TO global_admin_role
    USING (true);

CREATE POLICY global_admin_access_policy ON business_objects
    FOR ALL
    TO global_admin_role
    USING (true);

CREATE POLICY global_admin_access_policy ON processes
    FOR ALL
    TO global_admin_role
    USING (true);

CREATE POLICY global_admin_access_policy ON users
    FOR ALL
    TO global_admin_role
    USING (true);

-- Note: Due to BYPASSRLS, these policies are informational
-- The global_admin_role will bypass RLS regardless

-- ============================================================================
-- STEP 6: Create Security Audit Log Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS security_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_id UUID,
    tenant_id UUID,
    is_global_admin BOOLEAN DEFAULT false,
    action TEXT NOT NULL,
    resource TEXT,
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    session_id TEXT,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_security_audit_log_timestamp ON security_audit_log(timestamp DESC);
CREATE INDEX idx_security_audit_log_user ON security_audit_log(user_id);
CREATE INDEX idx_security_audit_log_tenant ON security_audit_log(tenant_id);
CREATE INDEX idx_security_audit_log_global_admin ON security_audit_log(is_global_admin) WHERE is_global_admin = true;

-- Global admins can see all audit logs, tenant users can only see their tenant's logs
ALTER TABLE security_audit_log ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON security_audit_log
    FOR SELECT
    TO tenant_user_role
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

CREATE POLICY global_admin_access_policy ON security_audit_log
    FOR ALL
    TO global_admin_role
    USING (true);

-- ============================================================================
-- STEP 7: Create Helper Functions
-- ============================================================================

-- Function to get current tenant from session variable
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true)::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if current user is global admin
CREATE OR REPLACE FUNCTION is_global_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN current_setting('app.is_global_admin', true)::BOOLEAN;
EXCEPTION
    WHEN OTHERS THEN
        RETURN false;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- To verify RLS is enabled:
-- SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname = 'public' AND rowsecurity = true;

-- To verify policies exist:
-- SELECT schemaname, tablename, policyname, roles, cmd FROM pg_policies WHERE schemaname = 'public';

-- To test as tenant user (run as superuser):
-- SET ROLE tenant_user_role;
-- SET app.current_tenant_id = '<some-tenant-uuid>';
-- SELECT * FROM business_objects; -- Should only see rows for that tenant
-- RESET ROLE;

-- To test as global admin:
-- SET ROLE global_admin_role;
-- SELECT * FROM business_objects; -- Should see ALL rows
-- RESET ROLE;

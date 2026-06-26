-- Multi-Tenant Authentication Schema Enhancement
-- Adds tenant_scope support for single/multi/all tenant access patterns

-- 1. Add tenant_scope columns to users table
ALTER TABLE users 
  ADD COLUMN IF NOT EXISTS tenant_scope VARCHAR(20) DEFAULT 'single' CHECK (tenant_scope IN ('single', 'multi', 'all')),
  ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS org_id UUID;

-- Create index for tenant lookups
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_scope ON users(tenant_scope);

-- 2. Create tenant_assignments table for multi-tenant ops
CREATE TABLE IF NOT EXISTS tenant_assignments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  access_level VARCHAR(50) DEFAULT 'read',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  created_by UUID REFERENCES users(id),
  UNIQUE(user_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_assignments_user ON tenant_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_assignments_tenant ON tenant_assignments(tenant_id);

-- Add update trigger
DROP TRIGGER IF EXISTS update_tenant_assignments_updated_at ON tenant_assignments;
CREATE TRIGGER update_tenant_assignments_updated_at
  BEFORE UPDATE ON tenant_assignments
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- 3. Update auth_audit_log to track tenant context
ALTER TABLE auth_audit_log
  ADD COLUMN IF NOT EXISTS tenant_id UUID,
  ADD COLUMN IF NOT EXISTS tenant_scope VARCHAR(20);

CREATE INDEX IF NOT EXISTS idx_audit_tenant ON auth_audit_log(tenant_id);

-- 4. Update existing admin user to global ops
UPDATE users 
SET 
  tenant_scope = 'all',
  role = 'global_ops',
  permissions = jsonb_build_array('read:tenants', 'read:metrics', 'manage:incidents', 'manage:platform', 'admin')
WHERE email = 'admin@semlayer.com';

-- 5. Helper function to get user's accessible tenants
CREATE OR REPLACE FUNCTION get_user_tenants(p_user_id UUID)
RETURNS TABLE(tenant_id UUID, tenant_name VARCHAR, access_level VARCHAR) AS $$
BEGIN
  -- Get tenant scope
  DECLARE
    v_tenant_scope VARCHAR(20);
    v_single_tenant_id UUID;
  BEGIN
    SELECT u.tenant_scope, u.tenant_id 
    INTO v_tenant_scope, v_single_tenant_id
    FROM users u 
    WHERE u.id = p_user_id;
    
    IF v_tenant_scope = 'single' THEN
      -- Return only their assigned tenant
      RETURN QUERY
      SELECT t.id, t.name, 'owner'::VARCHAR
      FROM tenants t
      WHERE t.id = v_single_tenant_id;
      
    ELSIF v_tenant_scope = 'multi' THEN
      -- Return assigned tenants from tenant_assignments
      RETURN QUERY
      SELECT t.id, t.name, ta.access_level
      FROM tenant_assignments ta
      JOIN tenants t ON t.id = ta.tenant_id
      WHERE ta.user_id = p_user_id;
      
    ELSIF v_tenant_scope = 'all' THEN
      -- Return all tenants
      RETURN QUERY
      SELECT t.id, t.name, 'admin'::VARCHAR
      FROM tenants t;
      
    END IF;
  END;
END;
$$ LANGUAGE plpgsql;

-- 6. Add comments for documentation
COMMENT ON COLUMN users.tenant_scope IS 'Tenant access pattern: single (bound to one), multi (specific list), all (global ops)';
COMMENT ON COLUMN users.tenant_id IS 'For single-tenant users: their assigned tenant';
COMMENT ON COLUMN users.org_id IS 'Optional organizational grouping for multi-org deployments';
COMMENT ON TABLE tenant_assignments IS 'Multi-tenant user access mappings';
COMMENT ON FUNCTION get_user_tenants IS 'Returns list of tenants accessible by a user based on their tenant_scope';

-- 7. Sample data for testing (comment out in production)
-- Create test tenants if they don't exist
INSERT INTO tenants (id, name, status, created_at)
VALUES 
  ('11111111-1111-1111-1111-111111111111'::UUID, 'Tenant A', 'active', NOW()),
  ('22222222-2222-2222-2222-222222222222'::UUID, 'Tenant B', 'active', NOW()),
  ('33333333-3333-3333-3333-333333333333'::UUID, 'Tenant C', 'active', NOW())
ON CONFLICT (id) DO NOTHING;

-- Create test users for each tenant type
-- Single-tenant user
INSERT INTO users (username, email, password_hash, name, role, tenant_scope, tenant_id, is_active, status)
VALUES (
  'alice',
  'alice@tenant-a.com',
  '$2a$10$7tGk5tDQKmmnQ7AKzOjlWufdFNgueXG.q4zRKPr8uZEWb4uoDeNhe', -- Admin123!
  'Alice (Tenant A User)',
  'user',
  'single',
  '11111111-1111-1111-1111-111111111111'::UUID,
  true,
  'active'
) ON CONFLICT (username) DO NOTHING;

-- Multi-tenant ops user
INSERT INTO users (username, email, password_hash, name, role, tenant_scope, is_active, status)
VALUES (
  'ops_regional',
  'ops@region.com',
  '$2a$10$7tGk5tDQKmmnQ7AKzOjlWufdFNgueXG.q4zRKPr8uZEWb4uoDeNhe', -- Admin123!
  'Regional Ops',
  'ops',
  'multi',
  true,
  'active'
) ON CONFLICT (username) DO NOTHING;

-- Add tenant assignments for multi-tenant user
INSERT INTO tenant_assignments (user_id, tenant_id, access_level)
SELECT u.id, '11111111-1111-1111-1111-111111111111'::UUID, 'read'
FROM users u WHERE u.username = 'ops_regional'
ON CONFLICT DO NOTHING;

INSERT INTO tenant_assignments (user_id, tenant_id, access_level)
SELECT u.id, '22222222-2222-2222-2222-222222222222'::UUID, 'read'
FROM users u WHERE u.username = 'ops_regional'
ON CONFLICT DO NOTHING;

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON tenant_assignments TO postgres;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO postgres;

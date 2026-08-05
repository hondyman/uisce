-- ============================================================================
-- RBAC Instance-Level Migration
-- Roles are now scoped per tenant_instance (product within a tenant), not per
-- datasource. Core/template roles live in the Gold Copy tenant and are
-- inherited by child tenants via role_key matching.
-- Gold Copy is identified by tenants.gold_copy = true.
-- ============================================================================

-- 1. tenants.gold_copy already exists (from 20260214_add_region_to_tenants.up.sql)
--    Verify column exists:
DO $$
BEGIN
  ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gold_copy BOOLEAN NOT NULL DEFAULT FALSE;
END $$;

-- 2. bp_roles: replace datasource_id with tenant_instance_id, make role_key globally unique, add is_template
ALTER TABLE bp_roles ADD COLUMN IF NOT EXISTS tenant_instance_id UUID;
ALTER TABLE bp_roles ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT FALSE;

-- Mark existing roles as non-template (custom tenant roles)
UPDATE bp_roles SET is_template = false WHERE is_template IS NULL;

-- Make role_key globally unique (drop old composite unique constraint)
-- First, drop the old index and constraint if they exist
DROP INDEX IF EXISTS idx_bp_roles_tenant_datasource;
ALTER TABLE bp_roles DROP CONSTRAINT IF EXISTS bp_roles_tenant_id_datasource_id_role_key_key;

-- Add global unique constraint on role_key
ALTER TABLE bp_roles ADD CONSTRAINT bp_roles_role_key_unique UNIQUE (role_key);

-- 3. bp_user_roles: replace datasource_id with tenant_instance_id, add role_key denormalized
ALTER TABLE bp_user_roles ADD COLUMN IF NOT EXISTS tenant_instance_id UUID;
ALTER TABLE bp_user_roles ADD COLUMN IF NOT EXISTS role_key TEXT;

-- Populate role_key from bp_roles lookup
UPDATE bp_user_roles ur
SET role_key = r.role_key
FROM bp_roles r
WHERE ur.role_id = r.id;

-- Drop old datasource_id column
ALTER TABLE bp_user_roles DROP COLUMN IF EXISTS datasource_id;

-- Fix unique constraint to use role_key instead of datasource_id
-- Replace standard unique constraint with partial indexes to handle NULL tenant_instance_id properly
-- (Postgres UNIQUE treats multiple NULLs as violations, but we want NULL = "all instances")
ALTER TABLE bp_user_roles DROP CONSTRAINT IF EXISTS bp_user_roles_user_id_role_id_tenant_id_datasource_id_scope_type_scope_id_key;
ALTER TABLE bp_user_roles DROP CONSTRAINT IF EXISTS bp_user_roles_user_role_tenant_instance_unique;

-- Partial unique indexes: one for tenant-instance-scoped, one for tenant-wide (NULL instance)
CREATE UNIQUE INDEX idx_bp_user_roles_unique_instance ON bp_user_roles (user_id, tenant_id, role_key, tenant_instance_id) WHERE tenant_instance_id IS NOT NULL;
CREATE UNIQUE INDEX idx_bp_user_roles_unique_global ON bp_user_roles (user_id, tenant_id, role_key) WHERE tenant_instance_id IS NULL;

-- Drop old indexes and create new ones
DROP INDEX IF EXISTS idx_bp_user_roles_tenant_datasource;
CREATE INDEX IF NOT EXISTS idx_bp_user_roles_user ON bp_user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_roles_role ON bp_user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_roles_tenant_instance ON bp_user_roles(tenant_id, tenant_instance_id);

-- 4. bp_field_permissions: replace datasource_id with tenant_instance_id
ALTER TABLE bp_field_permissions ADD COLUMN IF NOT EXISTS tenant_instance_id UUID;
ALTER TABLE bp_field_permissions DROP COLUMN IF EXISTS datasource_id;
CREATE INDEX IF NOT EXISTS idx_bp_field_permissions_tenant_instance ON bp_field_permissions(tenant_id, tenant_instance_id);

-- 5. bp_role_pages: ensure page_id column exists and create template role tracking
-- (bp_role_pages already has role_id FK to bp_roles; is_template on bp_roles filters gold copy roles)

-- 6. Seed: ensure the Gold Copy tenant has is_template=true for its roles
UPDATE bp_roles SET is_template = true
WHERE tenant_id = (SELECT id FROM tenants WHERE gold_copy = true);

-- 7. Future-proofing for Enterprise AD / Group sync
-- Tables added here so schema is ready when an AD client arrives;
-- Go code does NOT yet query these tables (Phase 3 work).

CREATE TABLE IF NOT EXISTS bp_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    group_key TEXT NOT NULL,
    name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'local',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, group_key)
);

CREATE TABLE IF NOT EXISTS bp_user_groups (
    user_id UUID NOT NULL,
    group_id UUID NOT NULL REFERENCES bp_groups(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS bp_group_roles (
    group_id UUID NOT NULL REFERENCES bp_groups(id) ON DELETE CASCADE,
    role_key TEXT NOT NULL REFERENCES bp_roles(role_key) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    PRIMARY KEY (group_id, role_key)
);

CREATE INDEX IF NOT EXISTS idx_bp_groups_tenant ON bp_groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_groups_user ON bp_user_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_groups_tenant ON bp_user_groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_group_roles_group ON bp_group_roles(group_id);
CREATE INDEX IF NOT EXISTS idx_bp_group_roles_tenant ON bp_group_roles(tenant_id);

-- ============================================================================
-- Rollback (if needed):
-- ALTER TABLE bp_roles DROP COLUMN IF EXISTS tenant_instance_id;
-- ALTER TABLE bp_roles DROP COLUMN IF EXISTS is_template;
-- ALTER TABLE bp_roles ADD COLUMN datasource_id UUID;
-- ALTER TABLE bp_roles ADD CONSTRAINT bp_roles_tenant_id_datasource_id_role_key_key UNIQUE (tenant_id, datasource_id, role_key);
-- ALTER TABLE bp_user_roles ADD COLUMN datasource_id UUID;
-- UPDATE bp_user_roles ur SET datasource_id = r.datasource_id FROM bp_roles r WHERE ur.role_id = r.id;
-- ALTER TABLE bp_user_roles DROP COLUMN IF EXISTS tenant_instance_id;
-- ALTER TABLE bp_user_roles DROP COLUMN IF EXISTS role_key;
-- DROP INDEX IF EXISTS idx_bp_user_roles_unique_instance;
-- DROP INDEX IF EXISTS idx_bp_user_roles_unique_global;
-- ALTER TABLE bp_user_roles ADD CONSTRAINT bp_user_roles_user_id_role_id_tenant_id_datasource_id_scope_type_scope_id_key
--   UNIQUE (user_id, role_id, tenant_id, datasource_id, scope_type, scope_id);
-- ALTER TABLE bp_field_permissions ADD COLUMN datasource_id UUID;
-- ALTER TABLE bp_field_permissions DROP COLUMN IF EXISTS tenant_instance_id;
-- DROP TABLE IF EXISTS bp_group_roles;
-- DROP TABLE IF EXISTS bp_user_groups;
-- DROP TABLE IF EXISTS bp_groups;
-- ============================================================================

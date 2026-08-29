-- +goose Up
-- Migration: Gold-copy role inheritance
-- Purpose: let roles authored in the gold-copy tenant be inherited (and locally
-- overridden/extended) by every other tenant, and give the Studio entitlement
-- system (security.security_profiles) the tenant-override tables its handlers
-- (tenant_studio_handler.go) already expect but were never created for this
-- environment.
--
-- Note: this file targets the *deployed* bp_roles/security_profiles shape
-- (tenant_id NOT NULL, is_template boolean, tenant_instance_id) rather than
-- the aspirational schema in migrations/misc/rbac_field_level_permissions.sql,
-- which was never applied here.

-- ---------------------------------------------------------------------------
-- 1. bp_roles: self-referential lineage + link to a Studio security profile.
-- `is_template` already exists and is the live gold-copy marker (a template
-- role lives under the gold-copy tenant's own tenant_id, same pattern as
-- business_objects.is_core under the gold-copy tenant).
-- ---------------------------------------------------------------------------
ALTER TABLE bp_roles ADD COLUMN IF NOT EXISTS parent_role_id UUID REFERENCES bp_roles(id) ON DELETE SET NULL;
ALTER TABLE bp_roles ADD COLUMN IF NOT EXISTS security_profile_id UUID REFERENCES security.security_profiles(profile_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bp_roles_parent ON bp_roles(parent_role_id);
CREATE INDEX IF NOT EXISTS idx_bp_roles_is_template ON bp_roles(is_template);

-- ---------------------------------------------------------------------------
-- 2. studio schema: tenant_studio_handler.go (registered in api.go) already
-- serves /v1/tenant/{policies,entitlements}, but the backing tables were
-- never created in this environment, so those endpoints 500 today. Create
-- them per 20260828_001_tenant_studio_tables.up.sql, with tenant_id nullable
-- from the start so a gold-copy-tenant caller can author a global baseline
-- row (tenant_id NULL) that every other tenant inherits, mirroring
-- security.security_profiles (tenant_id NULL = System-Level Gold Copy Blueprint).
-- ---------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS studio;

CREATE TABLE IF NOT EXISTS studio.tenant_abac_policies (
    policy_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID,                                 -- NULL = gold-copy global baseline
    target_profile_key VARCHAR(100) NOT NULL,
    name               VARCHAR(255) NOT NULL,
    description        TEXT,
    effect             TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    priority           INT NOT NULL DEFAULT 100,
    enabled            BOOLEAN NOT NULL DEFAULT true,
    action_attribute   VARCHAR(255) NOT NULL,
    condition_dsl      TEXT,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_abac_policies_tenant
    ON studio.tenant_abac_policies (tenant_id, target_profile_key, action_attribute)
    WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_abac_policies_global
    ON studio.tenant_abac_policies (target_profile_key, action_attribute)
    WHERE tenant_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_abac_policies_tenant_profile ON studio.tenant_abac_policies (tenant_id, target_profile_key);
CREATE INDEX IF NOT EXISTS idx_tenant_abac_policies_enabled ON studio.tenant_abac_policies (enabled) WHERE enabled = true;

DO $$ BEGIN
    CREATE TYPE studio.entitlement_type AS ENUM ('MENU_PAGE', 'WORKFLOW_STEP', 'PUBLIC_API');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE studio.override_state AS ENUM ('INHERIT_BASELINE', 'EXPLICIT_ALLOW', 'FORCE_DENY');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS studio.tenant_component_entitlements (
    entitlement_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID,                                 -- NULL = gold-copy global baseline
    target_profile_key VARCHAR(100) NOT NULL,
    entitlement_type   studio.entitlement_type NOT NULL,
    node_path          VARCHAR(500) NOT NULL,
    override_state     studio.override_state NOT NULL DEFAULT 'INHERIT_BASELINE',
    condition_dsl      TEXT,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_entitlements_tenant
    ON studio.tenant_component_entitlements (tenant_id, target_profile_key, entitlement_type, node_path)
    WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_entitlements_global
    ON studio.tenant_component_entitlements (target_profile_key, entitlement_type, node_path)
    WHERE tenant_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_entitlements_tenant_profile ON studio.tenant_component_entitlements (tenant_id, target_profile_key);
CREATE INDEX IF NOT EXISTS idx_tenant_entitlements_type ON studio.tenant_component_entitlements (entitlement_type);

-- +goose Down
DROP TABLE IF EXISTS studio.tenant_component_entitlements;
DROP TABLE IF EXISTS studio.tenant_abac_policies;
DROP TYPE IF EXISTS studio.override_state;
DROP TYPE IF EXISTS studio.entitlement_type;
DROP SCHEMA IF EXISTS studio;

DROP INDEX IF EXISTS idx_bp_roles_is_template;
DROP INDEX IF EXISTS idx_bp_roles_parent;
ALTER TABLE bp_roles DROP COLUMN IF EXISTS security_profile_id;
ALTER TABLE bp_roles DROP COLUMN IF EXISTS parent_role_id;

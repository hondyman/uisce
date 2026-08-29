-- +goose Up
-- Migration: Tenant Studio - ABAC Policies and Component Entitlements
-- Created: 2026-08-28
-- Purpose: Support Phase D self-service studio for tenant security profiles

CREATE SCHEMA IF NOT EXISTS studio;

-- ---------------------------------------------------------------------------
-- tenant_abac_policies: ABAC policies scoped to a specific security profile
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS studio.tenant_abac_policies (
    policy_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    target_profile_key VARCHAR(100) NOT NULL,           -- e.g., 'northwind_sales_rep'
    name               VARCHAR(255) NOT NULL,
    description        TEXT,
    effect             TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    priority           INT NOT NULL DEFAULT 100,        -- Higher = evaluated first
    enabled            BOOLEAN NOT NULL DEFAULT true,
    action_attribute   VARCHAR(255) NOT NULL,           -- e.g., 'menu:admin:users', 'workflow:edit', 'api:read'
    condition_dsl      TEXT,                            -- Optional DSL condition
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_profile_action UNIQUE (tenant_id, target_profile_key, action_attribute)
);

CREATE INDEX idx_tenant_abac_policies_tenant_profile ON studio.tenant_abac_policies (tenant_id, target_profile_key);
CREATE INDEX idx_tenant_abac_policies_enabled ON studio.tenant_abac_policies (enabled) WHERE enabled = true;

-- ---------------------------------------------------------------------------
-- tenant_component_entitlements: Fine-grained component access control
-- ---------------------------------------------------------------------------
CREATE TYPE studio.entitlement_type AS ENUM ('MENU_PAGE', 'WORKFLOW_STEP', 'PUBLIC_API');
CREATE TYPE studio.override_state AS ENUM ('INHERIT_BASELINE', 'EXPLICIT_ALLOW', 'FORCE_DENY');

CREATE TABLE IF NOT EXISTS studio.tenant_component_entitlements (
    entitlement_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    target_profile_key VARCHAR(100) NOT NULL,
    entitlement_type  studio.entitlement_type NOT NULL,
    node_path         VARCHAR(500) NOT NULL,           -- e.g., '/admin/users', '/workflow/onboarding/step-1'
    override_state    studio.override_state NOT NULL DEFAULT 'INHERIT_BASELINE',
    condition_dsl     TEXT,                            -- Optional DSL condition for conditional access
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_profile_type_node UNIQUE (tenant_id, target_profile_key, entitlement_type, node_path)
);

CREATE INDEX idx_tenant_entitlements_tenant_profile ON studio.tenant_component_entitlements (tenant_id, target_profile_key);
CREATE INDEX idx_tenant_entitlements_type ON studio.tenant_component_entitlements (entitlement_type);

-- +goose Down
DROP TABLE IF EXISTS studio.tenant_component_entitlements;
DROP TABLE IF EXISTS studio.tenant_abac_policies;
DROP SCHEMA IF EXISTS studio;
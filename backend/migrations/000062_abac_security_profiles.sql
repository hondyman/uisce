-- +goose Up
-- Migration: Add ABAC Security Profiles and Identity Mapping with Core/Custom overlays
-- Created: 2026-06-29

CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS security.identity_profile_mappings (
    mapping_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL, -- Core isolation fence
    idp_group_claim   VARCHAR(255) NOT NULL, -- e.g., 'GG-Uisce-Compliance'
    functional_role   VARCHAR(100) NOT NULL, -- e.g., 'compliance_officer'
    clearance_level   VARCHAR(50) NOT NULL,  -- e.g., 'L3'
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_group UNIQUE (tenant_id, idp_group_claim)
);

CREATE INDEX IF NOT EXISTS idx_idp_mappings ON security.identity_profile_mappings(idp_group_claim, tenant_id);

CREATE TABLE IF NOT EXISTS security.security_profiles (
    profile_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID, -- NULL indicates System-Level Gold Copy Blueprint
    profile_key       VARCHAR(50) NOT NULL, -- e.g., 'trader', 'accountant', 'cio'
    profile_name      VARCHAR(100) NOT NULL, -- e.g., 'Global Copy - Trader'
    parent_profile_id UUID REFERENCES security.security_profiles(profile_id), -- For custom variants
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_profile_key UNIQUE (tenant_id, profile_key)
);

CREATE INDEX IF NOT EXISTS idx_profile_resolution ON security.security_profiles (profile_key, tenant_id);

-- ---------------------------------------------------------------------------
-- Make abac_policies resilient to prior schema variations.
-- Some environments already have a public.abac_policies table (created by
-- root migrations) without a datasource_id column and with a NOT NULL
-- tenant_id. Others have no table at all. This block ensures the table and
-- the required nullable columns exist before we seed Gold Copy policies.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'abac_policies'
    ) THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'abac_policies' AND column_name = 'datasource_id'
        ) THEN
            ALTER TABLE abac_policies ADD COLUMN datasource_id UUID;
        END IF;
        ALTER TABLE abac_policies ALTER COLUMN tenant_id DROP NOT NULL;
        ALTER TABLE abac_policies ALTER COLUMN datasource_id DROP NOT NULL;
    ELSE
        CREATE TABLE IF NOT EXISTS abac_policies (
            id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            tenant_id         UUID,
            datasource_id     UUID,
            name              TEXT NOT NULL,
            description       TEXT,
            subject_rules     JSONB NOT NULL DEFAULT '{}'::jsonb,
            action_rules      JSONB NOT NULL DEFAULT '{}'::jsonb,
            resource_rules    JSONB NOT NULL DEFAULT '{}'::jsonb,
            environment_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
            effect            TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
            priority          INT DEFAULT 100,
            enabled           BOOLEAN DEFAULT true,
            created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
        CREATE INDEX IF NOT EXISTS idx_abac_policies_tenant ON abac_policies (tenant_id);
    END IF;
END $$;

-- Seed Northwind Gold Copy profiles
INSERT INTO security.security_profiles (profile_id, tenant_id, profile_key, profile_name)
VALUES
    (gen_random_uuid(), NULL, 'northwind_sales_rep', 'Gold Copy - Sales Representative'),
    (gen_random_uuid(), NULL, 'northwind_inventory_manager', 'Gold Copy - Inventory Specialist'),
    (gen_random_uuid(), NULL, 'northwind_billing_specialist', 'Gold Copy - Billing Specialist'),
    (gen_random_uuid(), NULL, 'northwind_executive', 'Gold Copy - Commerce Executive')
ON CONFLICT (tenant_id, profile_key) DO NOTHING;

-- Seeds for Gold Copy global ABAC policies (using abac_policies table)
-- 1. Sales Rep Baseline Read/Write Orders/Customers
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Sales Representative Operations',
    'Allows read, create, update on order and customer resources',
    'allow',
    100,
    true,
    '{"roles": ["northwind_sales_rep"]}'::jsonb,
    '{"actions": ["read", "create", "update"]}'::jsonb,
    '{"resources": ["order", "customer"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- 2. Deny sales reps from modifying product/supplier catalog data
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Block Sales Rep Inventory Modifications',
    'Explicitly deny create, update, delete actions on product and supplier resources for Sales Reps',
    'deny',
    90,
    true,
    '{"roles": ["northwind_sales_rep"]}'::jsonb,
    '{"actions": ["create", "update", "delete"]}'::jsonb,
    '{"resources": ["product", "supplier"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- 3. Inventory Specialist: Manage products & suppliers
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Inventory Manager Catalog Operations',
    'Allows read, create, update on product and supplier resources',
    'allow',
    100,
    true,
    '{"roles": ["northwind_inventory_manager"]}'::jsonb,
    '{"actions": ["read", "create", "update"]}'::jsonb,
    '{"resources": ["product", "supplier"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- 4. Block Inventory Access to Order Book
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Block Inventory Access to Order Book',
    'Explicitly deny all actions on orders for Inventory Specialists',
    'deny',
    90,
    true,
    '{"roles": ["northwind_inventory_manager"]}'::jsonb,
    '{"actions": ["read", "create", "update", "delete"]}'::jsonb,
    '{"resources": ["order"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM abac_policies WHERE tenant_id IS NULL AND name LIKE 'Global Baseline - %';
DROP TABLE IF EXISTS security.security_profiles;
DROP TABLE IF EXISTS security.identity_profile_mappings;

-- Restore NOT NULL constraints only if the table and columns still exist.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'abac_policies'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'abac_policies' AND column_name = 'tenant_id'
        ) THEN
            ALTER TABLE abac_policies ALTER COLUMN tenant_id SET NOT NULL;
        END IF;
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = 'abac_policies' AND column_name = 'datasource_id'
        ) THEN
            ALTER TABLE abac_policies ALTER COLUMN datasource_id SET NOT NULL;
        END IF;
    END IF;
END $$;

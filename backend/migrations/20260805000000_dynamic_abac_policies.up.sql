-- +goose Up
-- Dynamic ABAC Policies: Phase 1 - Database Schema
-- Enables attribute-based access control for 500k+ resource scenarios

-- 1. Add attributes JSONB column to users table for storing user attributes
ALTER TABLE users ADD COLUMN IF NOT EXISTS attributes JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 2. Create dynamic policies table (keyed by role_key for Gold Copy inheritance)
CREATE TABLE IF NOT EXISTS bp_dynamic_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    role_key TEXT NOT NULL,                  -- e.g., 'portfolio.owner' (attaches to RBAC role via role_key)
    resource_type TEXT NOT NULL,              -- e.g., 'portfolios' (the table/resource type)
    user_attribute TEXT NOT NULL,             -- e.g., 'assigned_portfolio_id' (key from user's attributes JSONB)
    resource_attribute TEXT NOT NULL,         -- e.g., 'id' (column on resource table to match against)
    action TEXT NOT NULL DEFAULT 'read',       -- 'read', 'write', 'delete'
    description TEXT,                          -- Admin-facing description of this policy
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, role_key, resource_type, user_attribute, resource_attribute, action)
);

-- Indexes for efficient policy lookup
CREATE INDEX IF NOT EXISTS idx_bp_dynamic_policies_tenant_role ON bp_dynamic_policies(tenant_id, role_key);
CREATE INDEX IF NOT EXISTS idx_bp_dynamic_policies_resource ON bp_dynamic_policies(resource_type, action);
CREATE INDEX IF NOT EXISTS idx_bp_dynamic_policies_active ON bp_dynamic_policies(is_active) WHERE is_active = true;

-- 3. Create user attributes table for explicit attribute definitions (optional, for cases where JSONB isn't enough)
CREATE TABLE IF NOT EXISTS bp_user_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    attribute_key TEXT NOT NULL,              -- e.g., 'assigned_portfolio_id'
    attribute_value TEXT NOT NULL,            -- e.g., 'port_999'
    source TEXT DEFAULT 'manual',             -- 'manual', 'ad_sync', 'scim', 'rule'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, attribute_key)
);

CREATE INDEX IF NOT EXISTS idx_bp_user_attributes_user ON bp_user_attributes(user_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_attributes_tenant ON bp_user_attributes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_user_attributes_key ON bp_user_attributes(attribute_key, attribute_value);

-- +goose Down
DROP TABLE IF EXISTS bp_user_attributes;
DROP TABLE IF EXISTS bp_dynamic_policies;
ALTER TABLE users DROP COLUMN IF EXISTS attributes;

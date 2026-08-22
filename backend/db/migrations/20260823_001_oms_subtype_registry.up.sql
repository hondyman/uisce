-- 20260823_001_oms_subtype_registry.up.sql
-- Subtype registry across Trading, Alternatives, and Master Domains

CREATE SCHEMA IF NOT EXISTS oms;
CREATE SCHEMA IF NOT EXISTS altinv;
CREATE SCHEMA IF NOT EXISTS master;
CREATE SCHEMA IF NOT EXISTS cash_flow;

CREATE TABLE IF NOT EXISTS oms.subtype_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    root_object TEXT NOT NULL,         -- 'account', 'position', 'security', 'trade_order'
    subtype_code TEXT NOT NULL,        -- 'institutional', 'retail_wealth', etc.
    display_name TEXT NOT NULL,
    parent_subtype_code TEXT,
    field_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_root_subtype UNIQUE (tenant_id, root_object, subtype_code)
);

CREATE INDEX IF NOT EXISTS idx_subtype_registry_lookup
ON oms.subtype_registry (tenant_id, root_object, is_active);

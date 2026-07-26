-- Uisce Dynamic Layout Engine Schema
-- Rule 1 Config-Before-Code & Rule 7 Security Mandate

BEGIN;

CREATE SCHEMA IF NOT EXISTS platform;

-- 1. Page Registry Table (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS platform.page_registry (
    page_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    page_key VARCHAR(100) NOT NULL,            -- e.g., 'security_master_grid', 'mdm_steward_studio'
    page_name VARCHAR(150) NOT NULL,
    bo_id UUID NOT NULL,                        -- Links page directly to Business Object catalog_node
    layout_type VARCHAR(50) DEFAULT 'GRID',     -- GRID, FORM, SPLIT_MDM_STUDIO, DETAIL
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, page_key)
);

-- 2. Optional Field Display Overrides (Order, Column Widths, Hidden States)
CREATE TABLE IF NOT EXISTS platform.page_field_config (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    page_id UUID NOT NULL REFERENCES platform.page_registry(page_id) ON DELETE CASCADE,
    field_name VARCHAR(100) NOT NULL,
    display_order INT DEFAULT 10,
    column_width INT DEFAULT 150,
    is_hidden BOOLEAN DEFAULT FALSE,
    component_hint VARCHAR(50) DEFAULT 'AUTO', -- AUTO, BADGE, CURRENCY, DATE_PICKER, DROPDOWN
    UNIQUE(tenant_id, page_id, field_name)
);

-- Indices for Rule 7 Multi-Tenant Isolation
CREATE INDEX IF NOT EXISTS idx_page_reg_tenant ON platform.page_registry(tenant_id, page_key);
CREATE INDEX IF NOT EXISTS idx_page_field_tenant ON platform.page_field_config(tenant_id, page_id);

COMMIT;

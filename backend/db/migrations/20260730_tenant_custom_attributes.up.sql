-- Migration: Tenant Custom Attributes & Upgrade Exception Queue Schema
-- Date: 2026-07-30
-- Purpose: Support overlay architecture for upgrade-safe tenant custom fields and 3-way merge upgrade exceptions.

CREATE TABLE IF NOT EXISTS public.tenant_custom_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    bo_id VARCHAR(128) NOT NULL,
    attribute_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    data_type VARCHAR(50) NOT NULL, -- STRING, NUMBER, BOOLEAN, DATE, JSON
    jsonb_path VARCHAR(255) NOT NULL, -- e.g. 'config->custom->charge_code'
    is_filterable BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    CONSTRAINT uk_tenant_custom_attr UNIQUE (tenant_id, bo_id, attribute_name)
);

CREATE INDEX IF NOT EXISTS idx_custom_attrs_tenant_bo ON public.tenant_custom_attributes(tenant_id, bo_id);

CREATE TABLE IF NOT EXISTS public.upgrade_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    layout_key VARCHAR(100) NOT NULL,
    ancestor_version VARCHAR(32) NOT NULL,
    target_version VARCHAR(32) NOT NULL,
    property_path VARCHAR(255) NOT NULL,
    conflict_reason TEXT NOT NULL,
    ancestor_value JSONB,
    modified_value JSONB,
    target_value JSONB,
    status VARCHAR(32) DEFAULT 'PENDING_REVIEW', -- PENDING_REVIEW, RESOLVED, DISCARDED
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_upgrade_exceptions_tenant ON public.upgrade_exceptions(tenant_id, status);

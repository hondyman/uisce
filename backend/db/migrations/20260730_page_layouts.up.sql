-- Migration: Page Layouts Metadata Schema
-- Date: 2026-07-30
-- Purpose: Store versioned declarative page layout specifications driven by Business Objects.

CREATE TABLE IF NOT EXISTS public.page_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    key VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    domain VARCHAR(50) NOT NULL,
    target_bo_id VARCHAR(128) NOT NULL,
    is_default BOOLEAN DEFAULT false,
    layout_spec JSONB NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_by VARCHAR(255) NOT NULL DEFAULT 'system',
    CONSTRAINT uk_page_layouts_tenant_key_version UNIQUE (tenant_id, key, version)
);

CREATE INDEX IF NOT EXISTS idx_page_layouts_lookup ON public.page_layouts(tenant_id, target_bo_id, is_default);

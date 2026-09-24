-- Migration: 20260827_001_page_designer_registry.up.sql
-- Description: Creates page_registry table for declarative metapage and dynamic application studio blueprints

CREATE TABLE IF NOT EXISTS public.page_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    page_key VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    layout_spec JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_gold_copy BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_page_key UNIQUE (tenant_id, page_key)
);

CREATE INDEX IF NOT EXISTS idx_page_registry_tenant_lookup 
ON public.page_registry(tenant_id, page_key) WHERE is_active = TRUE;

-- Migration: Create page_layouts table for Workday-style page designer
-- Version: 20251231

CREATE TABLE IF NOT EXISTS page_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    primary_bo TEXT NOT NULL,  -- Business object key, e.g., 'customer'
    layout_type TEXT NOT NULL DEFAULT 'form',  -- 'form', 'list', 'detail'
    layout_json JSONB NOT NULL DEFAULT '{}',  -- Sections, fields, configuration
    pipeline_id UUID,  -- Optional link to Uisce Flow pipeline for validation/workflow
    is_active BOOLEAN DEFAULT true,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

-- Index for fast lookups by tenant and BO
CREATE INDEX IF NOT EXISTS idx_page_layouts_tenant_bo ON page_layouts(tenant_id, primary_bo);

-- Pipelines table (if not exists) for saving Uisce Flow definitions
CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    business_object TEXT,  -- Target BO for this pipeline
    pipeline_json JSONB NOT NULL DEFAULT '{}',  -- Nodes and edges from ReactFlow
    is_active BOOLEAN DEFAULT true,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines(tenant_id);

COMMENT ON TABLE page_layouts IS 'Stores Workday-style page layouts for business objects';
COMMENT ON TABLE pipelines IS 'Stores Uisce Flow pipeline definitions';

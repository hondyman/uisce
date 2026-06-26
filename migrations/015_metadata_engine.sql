-- 1. Object Definitions (The Blueprints)
CREATE TABLE object_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL, -- e.g., "Private Equity Fund"
    slug TEXT NOT NULL, -- e.g., "private_equity_fund"
    version INTEGER NOT NULL DEFAULT 1,
    description TEXT,
    
    -- The Blueprint: Defines the expected structure (JSON Schema 2020-12)
    json_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- The Default Layout: Defines how it looks (UI Schema)
    ui_schema JSONB DEFAULT '{}'::jsonb,
    
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tenant_id, slug, version)
);

CREATE INDEX idx_obj_def_tenant ON object_definitions(tenant_id);
CREATE INDEX idx_obj_def_slug ON object_definitions(slug);

-- 2. Dynamic Entities (The Instances)
CREATE TABLE dynamic_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    definition_id UUID NOT NULL REFERENCES object_definitions(id),
    tenant_id UUID NOT NULL,
    
    -- The core "bag of attributes"
    attributes JSONB NOT NULL, 
    
    -- Metadata about the entity state
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dyn_ent_tenant ON dynamic_entities(tenant_id);
CREATE INDEX idx_dyn_ent_def ON dynamic_entities(definition_id);

-- GIN Index for fast JSONB searching (contains, exists)
CREATE INDEX idx_dynamic_attributes_gin ON dynamic_entities USING GIN (attributes);

-- 3. View Definitions (The Layouts)
-- Distinct from object_definitions because one object can have multiple views (e.g. "Trader View", "Compliance View")
CREATE TABLE view_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    slug TEXT NOT NULL, -- e.g., "hedge_fund_summary_v2"
    version INTEGER NOT NULL DEFAULT 1,
    title TEXT,
    
    -- The Layout: Grid system, widgets, component references
    layout_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Role-based access control for this view (optional, can be handled by app logic)
    allowed_roles TEXT[], 
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tenant_id, slug, version)
);

CREATE INDEX idx_view_def_tenant ON view_definitions(tenant_id);
CREATE INDEX idx_view_def_slug ON view_definitions(slug);

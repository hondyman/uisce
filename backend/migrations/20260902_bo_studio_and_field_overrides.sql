-- 20260902_bo_studio_and_field_overrides.sql
-- Single-Screen BO Studio Enhancements, Scope Fencing & Atomic Persistence Support

-- 1. Ensure public.business_objects exists
CREATE TABLE IF NOT EXISTS public.business_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    model_id UUID,
    bo_key VARCHAR(100) NOT NULL,
    bo_name VARCHAR(255) NOT NULL,
    description TEXT,
    bo_type VARCHAR(50) DEFAULT 'ENTITY',
    classification_node_id UUID,
    business_key_node_id UUID,
    semantic_id_node_id UUID,
    grain_node_id UUID,
    status VARCHAR(50) DEFAULT 'DRAFT',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_bo_key UNIQUE (tenant_id, bo_key)
);

-- 2. Ensure public.business_object_bindings exists
CREATE TABLE IF NOT EXISTS public.business_object_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    backend_id UUID NOT NULL,
    driving_node_id UUID,
    is_default BOOLEAN DEFAULT FALSE,
    temporal_override VARCHAR(50) DEFAULT 'NONE',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_bo_backend UNIQUE (tenant_id, bo_id, backend_id)
);

-- 3. Ensure public.business_object_fields exists with eligibility tracking and override metadata
CREATE TABLE IF NOT EXISTS public.business_object_fields (
    field_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    term_node_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    field_role VARCHAR(50) DEFAULT 'DIMENSION',
    binding_requirement VARCHAR(50) DEFAULT 'REQUIRED', -- REQUIRED, OPTIONAL, BACKEND_SPECIFIC, CALCULATED, INTERNAL
    binding_status VARCHAR(50) DEFAULT 'UNRESOLVED',      -- RESOLVED, PARTIALLY_RESOLVED, UNRESOLVED, NOT_APPLICABLE
    eligibility_source VARCHAR(50) DEFAULT 'DIRECT',     -- DIRECT, RELATED, CALCULATED, MANUAL, INTERNAL
    eligibility_path JSONB DEFAULT '{}'::jsonb,           -- Diagnostic trace back to driving node / graph path
    is_exposed BOOLEAN DEFAULT TRUE,
    override_reason TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_bo_term UNIQUE (tenant_id, bo_id, term_node_id)
);

-- 4. Ensure public.field_bindings exists
CREATE TABLE IF NOT EXISTS public.field_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES public.business_object_fields(field_id) ON DELETE CASCADE,
    source_node_id UUID,
    source_type VARCHAR(50) DEFAULT 'COLUMN',
    transformation_type VARCHAR(50) DEFAULT 'NONE',
    transformation_sql TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_bo_binding_field UNIQUE (tenant_id, bo_id, binding_id, field_id)
);

-- 5. Ensure catalog_edge table exists
CREATE TABLE IF NOT EXISTS public.catalog_edge (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    edge_type VARCHAR(100),
    edge_type_id UUID,
    from_node_id UUID,
    to_node_id UUID,
    source_node_id UUID,
    target_node_id UUID,
    relationship_type VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    properties JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Alter columns if tables already exist
ALTER TABLE public.business_object_fields 
ADD COLUMN IF NOT EXISTS binding_requirement VARCHAR(50) DEFAULT 'REQUIRED',
ADD COLUMN IF NOT EXISTS binding_status VARCHAR(50) DEFAULT 'UNRESOLVED',
ADD COLUMN IF NOT EXISTS eligibility_source VARCHAR(50) DEFAULT 'DIRECT',
ADD COLUMN IF NOT EXISTS eligibility_path JSONB DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS is_exposed BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS override_reason TEXT;

ALTER TABLE public.business_object_bindings
ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_bof_eligibility 
ON public.business_object_fields (tenant_id, bo_id, eligibility_source, binding_status);

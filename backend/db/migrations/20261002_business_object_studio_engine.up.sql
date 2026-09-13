-- backend/db/migrations/20261002_business_object_studio_engine.up.sql
-- Single-Screen Multi-Binding Semantic Schema, Field Bindings, and Graph Topology

CREATE SCHEMA IF NOT EXISTS public;

DROP TABLE IF EXISTS public.relationship_bindings CASCADE;
DROP TABLE IF EXISTS public.field_bindings CASCADE;
DROP TABLE IF EXISTS public.business_object_fields CASCADE;
DROP TABLE IF EXISTS public.business_object_bindings CASCADE;
DROP TABLE IF EXISTS public.business_object_relationships CASCADE;
DROP TABLE IF EXISTS public.business_objects CASCADE;

-- 1. Business Object Entity Master (Semantic Contract)
CREATE TABLE IF NOT EXISTS public.business_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    model_id UUID NOT NULL,
    bo_key VARCHAR(100) NOT NULL,
    bo_name VARCHAR(255) NOT NULL,
    description TEXT,
    bo_type VARCHAR(50) NOT NULL DEFAULT 'ENTITY' CHECK (bo_type IN ('ENTITY', 'FACT', 'DIMENSION', 'BRIDGE', 'REFERENCE')),
    classification_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    business_key_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    semantic_id_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    grain_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    sti_discriminator_column VARCHAR(100),
    active_subtype_filter VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_core BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bo_tenant_key UNIQUE (tenant_id, bo_key)
);

CREATE INDEX IF NOT EXISTS idx_bo_tenant_lookup ON public.business_objects (tenant_id, is_active);

-- 2. Business Object Backend Execution Bindings (Scope Fence)
CREATE TABLE IF NOT EXISTS public.business_object_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    backend_id UUID NOT NULL,
    backend_type VARCHAR(50) NOT NULL DEFAULT 'POSTGRES' CHECK (backend_type IN ('POSTGRES', 'STARROCKS', 'SNOWFLAKE', 'ICEBERG', 'CRIMS')),
    driving_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    temporal_override VARCHAR(50) NOT NULL DEFAULT 'NONE' CHECK (temporal_override IN ('NONE', 'BITEMPORAL', 'AS_OF', 'DELTA_STREAM')),
    base_sql TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bo_backend_binding UNIQUE (tenant_id, bo_id, backend_id)
);

CREATE INDEX IF NOT EXISTS idx_bo_binding_lookup ON public.business_object_bindings (tenant_id, bo_id, is_default);

-- 3. Business Object Fields (Logical Terms Attached to BO)
CREATE TABLE IF NOT EXISTS public.business_object_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    term_node_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE RESTRICT,
    field_name VARCHAR(100) NOT NULL,
    field_role VARCHAR(50) NOT NULL DEFAULT 'DIMENSION' CHECK (field_role IN ('KEY', 'DIMENSION', 'MEASURE', 'TIME_DIMENSION', 'ATTRIBUTE', 'CALCULATION')),
    aggregation_type VARCHAR(50) NOT NULL DEFAULT 'NONE' CHECK (aggregation_type IN ('NONE', 'SUM', 'AVG', 'MIN', 'MAX', 'COUNT', 'COUNT_DISTINCT')),
    binding_requirement VARCHAR(50) NOT NULL DEFAULT 'REQUIRED' CHECK (binding_requirement IN ('REQUIRED', 'OPTIONAL', 'BACKEND_SPECIFIC', 'CALCULATED', 'INTERNAL')),
    eligibility_source VARCHAR(50) NOT NULL DEFAULT 'DIRECT' CHECK (eligibility_source IN ('DIRECT', 'RELATED', 'CALCULATED', 'MANUAL', 'INTERNAL')),
    subtype_scope VARCHAR(100) DEFAULT 'ALL',
    is_exposed BOOLEAN NOT NULL DEFAULT TRUE,
    inherits_defaults BOOLEAN NOT NULL DEFAULT TRUE,
    override_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bo_field_tenant UNIQUE (tenant_id, bo_id, field_name)
);

CREATE INDEX IF NOT EXISTS idx_bof_bo_term ON public.business_object_fields (tenant_id, bo_id, term_node_id);

-- 4. Physical Field Bindings (Concrete Column / Expression Mappings)
CREATE TABLE IF NOT EXISTS public.field_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES public.business_object_fields(id) ON DELETE CASCADE,
    source_node_id UUID REFERENCES catalog_node(id) ON DELETE RESTRICT,
    source_type VARCHAR(50) NOT NULL DEFAULT 'COLUMN' CHECK (source_type IN ('COLUMN', 'EXPRESSION', 'FUNCTION', 'JSON_PATH')),
    transformation_type VARCHAR(50) NOT NULL DEFAULT 'NONE' CHECK (transformation_type IN ('NONE', 'NORMALIZE', 'SQL_EXPRESSION', 'WASM_KERNEL', 'CODE_TRANSLATION')),
    transformation_sql TEXT,
    json_path TEXT,
    binding_status VARCHAR(50) NOT NULL DEFAULT 'RESOLVED' CHECK (binding_status IN ('RESOLVED', 'PARTIALLY_RESOLVED', 'UNRESOLVED', 'NOT_APPLICABLE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_field_binding_unique UNIQUE (tenant_id, binding_id, field_id)
);

CREATE INDEX IF NOT EXISTS idx_fb_search ON public.field_bindings (tenant_id, binding_id, binding_status);

-- 5. Business Object Relationships & Physical Join Bindings
CREATE TABLE IF NOT EXISTS public.business_object_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    from_bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    to_bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE RESTRICT,
    rel_key VARCHAR(100) NOT NULL,
    rel_name VARCHAR(255) NOT NULL,
    cardinality VARCHAR(20) NOT NULL DEFAULT '1:M' CHECK (cardinality IN ('1:1', '1:M', 'M:1', 'M:M')),
    join_type VARCHAR(20) NOT NULL DEFAULT 'LEFT' CHECK (join_type IN ('INNER', 'LEFT', 'RIGHT', 'FULL')),
    relationship_basis_node_id UUID REFERENCES catalog_node(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bor_tenant_key UNIQUE (tenant_id, from_bo_id, rel_key)
);

CREATE TABLE IF NOT EXISTS public.relationship_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rel_id UUID NOT NULL REFERENCES public.business_object_relationships(id) ON DELETE CASCADE,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    join_condition_sql TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_rel_binding_unique UNIQUE (tenant_id, rel_id, binding_id)
);

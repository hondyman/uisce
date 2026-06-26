-- Migration: Create core metadata tables
-- Description: Tables for storing business object definitions and metadata

CREATE TABLE IF NOT EXISTS core_bo (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    storage VARCHAR(50) NOT NULL CHECK (storage IN ('row', 'wide_jsonb', 'eav')),
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL CHECK (status IN ('draft', 'active', 'deprecated')),
    fields JSONB NOT NULL DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name, version)
);

CREATE INDEX idx_core_bo_tenant ON core_bo(tenant_id);
CREATE INDEX idx_core_bo_status ON core_bo(status);

CREATE TABLE IF NOT EXISTS core_bo_field (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    business_object_id UUID NOT NULL REFERENCES core_bo(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('string', 'decimal', 'date', 'enum', 'ref', 'json')),
    is_required BOOLEAN DEFAULT FALSE,
    is_unique BOOLEAN DEFAULT FALSE,
    enum_id UUID,
    ref_object_id UUID,
    validation_json JSONB DEFAULT '{}',
    visibility_json JSONB DEFAULT '{}',
    default_value TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_core_bo_field_tenant ON core_bo_field(tenant_id);
CREATE INDEX idx_core_bo_field_bo ON core_bo_field(business_object_id);

CREATE TABLE IF NOT EXISTS core_bo_relationship (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    parent_object_id UUID NOT NULL REFERENCES core_bo(id),
    child_object_id UUID NOT NULL REFERENCES core_bo(id),
    cardinality VARCHAR(10) NOT NULL CHECK (cardinality IN ('1:N', 'N:M', '1:1')),
    cascade_rules JSONB DEFAULT '{}',
    aggregation_type VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_core_bo_rel_parent ON core_bo_relationship(parent_object_id);
CREATE INDEX idx_core_bo_rel_child ON core_bo_relationship(child_object_id);

CREATE TABLE IF NOT EXISTS core_enum (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    values JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE IF NOT EXISTS core_policy (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('object', 'field', 'workflow', 'ai_tool')),
    expression TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('authorization', 'data_residency', 'retention', 'visibility')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_core_policy_tenant ON core_policy(tenant_id);
CREATE INDEX idx_core_policy_scope ON core_policy(scope);

-- Migration to create catalog_node_types table for node type management
-- This table stores node types (entities) in the business glossary

CREATE TABLE IF NOT EXISTS catalog_node_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    catalog_type_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    parent_type_id UUID REFERENCES catalog_node_types(id) ON DELETE SET NULL,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, catalog_type_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_catalog_node_types_tenant ON catalog_node_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_node_types_parent ON catalog_node_types(parent_type_id);
CREATE INDEX IF NOT EXISTS idx_catalog_node_types_active ON catalog_node_types(is_active);
CREATE INDEX IF NOT EXISTS idx_catalog_node_types_config ON catalog_node_types USING GIN(config);

-- Comments
COMMENT ON TABLE catalog_node_types IS 'Defines node types for the business glossary catalog';
COMMENT ON COLUMN catalog_node_types.catalog_type_name IS 'Name of the node type (e.g., business_term, semantic_column)';
COMMENT ON COLUMN catalog_node_types.parent_type_id IS 'Optional parent type for inheritance';
COMMENT ON COLUMN catalog_node_types.config IS 'JSON configuration including properties, validation rules, etc.';

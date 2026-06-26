-- Migration to create catalog_edge_types table for edge type management
-- This table stores relationship types (edges) between nodes in the business glossary

CREATE TABLE IF NOT EXISTS catalog_edge_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    edge_type_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    source_node_type_id UUID REFERENCES catalog_node_types(id) ON DELETE SET NULL,
    target_node_type_id UUID REFERENCES catalog_node_types(id) ON DELETE SET NULL,
    is_directed BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, edge_type_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_tenant ON catalog_edge_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_source ON catalog_edge_types(source_node_type_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_target ON catalog_edge_types(target_node_type_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_active ON catalog_edge_types(is_active);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_config ON catalog_edge_types USING GIN(config);

-- Comments
COMMENT ON TABLE catalog_edge_types IS 'Defines edge types (relationships) for the business glossary catalog';
COMMENT ON COLUMN catalog_edge_types.edge_type_name IS 'Name of the edge type (e.g., has_parent, relates_to)';
COMMENT ON COLUMN catalog_edge_types.source_node_type_id IS 'Optional constraint on source node type';
COMMENT ON COLUMN catalog_edge_types.target_node_type_id IS 'Optional constraint on target node type';
COMMENT ON COLUMN catalog_edge_types.is_directed IS 'Whether the edge has direction (source -> target)';
COMMENT ON COLUMN catalog_edge_types.config IS 'JSON configuration including properties, validation rules, etc.';

-- Create catalog_node_type table
CREATE TABLE IF NOT EXISTS catalog_node_type (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    catalog_type_name TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    parent_type_id TEXT REFERENCES catalog_node_type(id),
    config JSONB,
    properties JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add index on tenant_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_catalog_node_type_tenant_id ON catalog_node_type(tenant_id);

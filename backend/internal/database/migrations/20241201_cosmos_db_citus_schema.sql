-- =============================================================================
-- Azure Cosmos DB for PostgreSQL (Citus) Schema for Semlayer
-- Distributed multi-tenant semantic layer metadata
-- =============================================================================

-- This migration sets up sharding for a globally distributed system
-- Run on Azure Cosmos DB for PostgreSQL (formerly Hyperscale/Citus)

-- =============================================================================
-- Reference Tables (Replicated to all nodes)
-- Small, frequently-joined lookup tables
-- =============================================================================

-- Tenants table - replicated for fast joins
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT 'us-east', -- Primary region
    tier TEXT NOT NULL DEFAULT 'standard', -- standard, business, enterprise
    status TEXT NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Mark as reference table (replicated to all nodes)
SELECT create_reference_table('tenants');

-- Products table - replicated
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_reference_table('products');

-- Data types and object types - replicated
CREATE TABLE IF NOT EXISTS object_types (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL, -- dimension, measure, attribute
    description TEXT
);

SELECT create_reference_table('object_types');

-- Compliance frameworks - replicated
CREATE TABLE IF NOT EXISTS compliance_frameworks (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE, -- SOC2, GDPR, HIPAA
    name TEXT NOT NULL,
    description TEXT,
    requirements JSONB
);

SELECT create_reference_table('compliance_frameworks');

-- =============================================================================
-- Distributed Tables (Sharded by tenant_id)
-- These tables are distributed across nodes
-- =============================================================================

-- Datasources - sharded by tenant_id
CREATE TABLE IF NOT EXISTS datasources (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL, -- postgres, snowflake, bigquery, etc
    connection_config JSONB NOT NULL, -- Encrypted at rest
    status TEXT NOT NULL DEFAULT 'active',
    last_sync_at TIMESTAMPTZ,
    sync_status TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

-- Distribute by tenant_id for colocation
SELECT create_distributed_table('datasources', 'tenant_id');

-- Catalog tables - sharded by tenant_id
CREATE TABLE IF NOT EXISTS catalog_tables (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    table_type TEXT NOT NULL DEFAULT 'table', -- table, view, materialized_view
    description TEXT,
    row_count BIGINT,
    size_bytes BIGINT,
    column_count INT,
    metadata JSONB DEFAULT '{}',
    search_vector tsvector,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('catalog_tables', 'tenant_id', colocate_with => 'datasources');

-- Catalog columns - sharded by tenant_id, colocated with tables
CREATE TABLE IF NOT EXISTS catalog_columns (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    table_id UUID NOT NULL,
    column_name TEXT NOT NULL,
    data_type TEXT NOT NULL,
    nullable BOOLEAN DEFAULT true,
    is_primary_key BOOLEAN DEFAULT false,
    is_foreign_key BOOLEAN DEFAULT false,
    description TEXT,
    sample_values JSONB,
    statistics JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('catalog_columns', 'tenant_id', colocate_with => 'datasources');

-- Semantic objects - sharded by tenant_id
CREATE TABLE IF NOT EXISTS semantic_objects (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    object_type TEXT NOT NULL, -- dimension, measure, entity, relationship
    definition JSONB NOT NULL,
    source_table_id UUID,
    source_column_id UUID,
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    search_vector tsvector,
    version INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'draft', -- draft, published, deprecated
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('semantic_objects', 'tenant_id', colocate_with => 'datasources');

-- Bundles - sharded by tenant_id
CREATE TABLE IF NOT EXISTS bundles (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    name TEXT NOT NULL,
    description TEXT,
    bundle_type TEXT NOT NULL DEFAULT 'standard', -- standard, curated, generated
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB DEFAULT '{}',
    search_vector tsvector,
    version INT NOT NULL DEFAULT 1,
    created_by UUID,
    updated_by UUID,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('bundles', 'tenant_id', colocate_with => 'datasources');

-- Bundle items (semantic objects in bundles) - sharded
CREATE TABLE IF NOT EXISTS bundle_items (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bundle_id UUID NOT NULL,
    semantic_object_id UUID NOT NULL,
    position INT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('bundle_items', 'tenant_id', colocate_with => 'bundles');

-- Policies - sharded by tenant_id
CREATE TABLE IF NOT EXISTS policies (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    policy_type TEXT NOT NULL, -- access, data_masking, row_filter, column_filter
    priority INT NOT NULL DEFAULT 100,
    conditions JSONB NOT NULL, -- ABAC conditions
    actions JSONB NOT NULL, -- What to do when conditions match
    target_objects JSONB, -- Which objects this applies to
    status TEXT NOT NULL DEFAULT 'draft',
    search_vector tsvector,
    effective_from TIMESTAMPTZ,
    effective_until TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('policies', 'tenant_id', colocate_with => 'datasources');

-- Audit logs - sharded by tenant_id
-- Consider time-based partitioning as well for large volumes
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id, created_at) -- Include time for partitioning
) PARTITION BY RANGE (created_at);

SELECT create_distributed_table('audit_logs', 'tenant_id', colocate_with => 'datasources');

-- Create monthly partitions for audit logs
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... continue for each month

-- Search analytics - sharded
CREATE TABLE IF NOT EXISTS search_analytics (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    query TEXT NOT NULL,
    result_count INT,
    clicked_result_id UUID,
    clicked_result_type TEXT,
    search_duration_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('search_analytics', 'tenant_id', colocate_with => 'datasources');

-- =============================================================================
-- User/Permission Tables (sharded by tenant_id)
-- =============================================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    email TEXT NOT NULL,
    name TEXT,
    role TEXT NOT NULL DEFAULT 'viewer',
    status TEXT NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, email)
);

SELECT create_distributed_table('users', 'tenant_id', colocate_with => 'datasources');

CREATE TABLE IF NOT EXISTS user_attributes (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    attribute_name TEXT NOT NULL,
    attribute_value JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

SELECT create_distributed_table('user_attributes', 'tenant_id', colocate_with => 'users');

-- =============================================================================
-- Indexes (Created on each shard)
-- =============================================================================

-- Full-text search indexes
CREATE INDEX IF NOT EXISTS idx_semantic_objects_search 
ON semantic_objects USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_bundles_search 
ON bundles USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_policies_search 
ON policies USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_catalog_tables_search 
ON catalog_tables USING GIN(search_vector);

-- Trigram indexes for fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_semantic_objects_name_trgm 
ON semantic_objects USING GIN(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_bundles_name_trgm 
ON bundles USING GIN(name gin_trgm_ops);

-- Foreign key-like indexes (no actual FK on distributed tables)
CREATE INDEX IF NOT EXISTS idx_semantic_objects_datasource 
ON semantic_objects(tenant_id, datasource_id);

CREATE INDEX IF NOT EXISTS idx_bundles_datasource 
ON bundles(tenant_id, datasource_id);

CREATE INDEX IF NOT EXISTS idx_catalog_tables_datasource 
ON catalog_tables(tenant_id, datasource_id);

CREATE INDEX IF NOT EXISTS idx_catalog_columns_table 
ON catalog_columns(tenant_id, table_id);

CREATE INDEX IF NOT EXISTS idx_bundle_items_bundle 
ON bundle_items(tenant_id, bundle_id);

CREATE INDEX IF NOT EXISTS idx_bundle_items_object 
ON bundle_items(tenant_id, semantic_object_id);

CREATE INDEX IF NOT EXISTS idx_policies_status_type 
ON policies(tenant_id, status, policy_type);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity 
ON audit_logs(tenant_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time 
ON audit_logs(tenant_id, user_id, created_at DESC);

-- =============================================================================
-- Search Vector Update Triggers
-- =============================================================================

CREATE OR REPLACE FUNCTION update_search_vector() 
RETURNS trigger AS $$
BEGIN
    IF TG_TABLE_NAME = 'semantic_objects' THEN
        NEW.search_vector := 
            setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(NEW.display_name, '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
            setweight(to_tsvector('english', COALESCE(NEW.object_type, '')), 'C');
    ELSIF TG_TABLE_NAME = 'bundles' THEN
        NEW.search_vector := 
            setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    ELSIF TG_TABLE_NAME = 'policies' THEN
        NEW.search_vector := 
            setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
            setweight(to_tsvector('english', COALESCE(NEW.policy_type, '')), 'C');
    ELSIF TG_TABLE_NAME = 'catalog_tables' THEN
        NEW.search_vector := 
            setweight(to_tsvector('english', COALESCE(NEW.table_name, '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(NEW.schema_name, '')), 'B') ||
            setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
CREATE TRIGGER semantic_objects_search_trigger
    BEFORE INSERT OR UPDATE ON semantic_objects
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

CREATE TRIGGER bundles_search_trigger
    BEFORE INSERT OR UPDATE ON bundles
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

CREATE TRIGGER policies_search_trigger
    BEFORE INSERT OR UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

CREATE TRIGGER catalog_tables_search_trigger
    BEFORE INSERT OR UPDATE ON catalog_tables
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

-- =============================================================================
-- Distributed Search Function
-- =============================================================================

CREATE OR REPLACE FUNCTION search_tenant_objects(
    p_tenant_id UUID,
    p_query TEXT,
    p_datasource_id UUID DEFAULT NULL,
    p_object_types TEXT[] DEFAULT NULL,
    p_limit INT DEFAULT 50,
    p_offset INT DEFAULT 0
) RETURNS TABLE (
    entity_type TEXT,
    entity_id UUID,
    name TEXT,
    display_name TEXT,
    description TEXT,
    rank REAL
) AS $$
DECLARE
    v_tsquery tsquery;
BEGIN
    v_tsquery := websearch_to_tsquery('english', p_query);
    
    RETURN QUERY
    SELECT 
        'semantic_object'::TEXT,
        so.id,
        so.name,
        so.display_name,
        so.description,
        ts_rank(so.search_vector, v_tsquery)
    FROM semantic_objects so
    WHERE so.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR so.datasource_id = p_datasource_id)
      AND (p_object_types IS NULL OR so.object_type = ANY(p_object_types))
      AND so.search_vector @@ v_tsquery
      AND so.deleted_at IS NULL
    
    UNION ALL
    
    SELECT 
        'bundle'::TEXT,
        b.id,
        b.name,
        b.name,
        b.description,
        ts_rank(b.search_vector, v_tsquery)
    FROM bundles b
    WHERE b.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR b.datasource_id = p_datasource_id)
      AND b.search_vector @@ v_tsquery
      AND b.deleted_at IS NULL
    
    ORDER BY rank DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- Citus Specific: Rebalancing and Maintenance
-- =============================================================================

-- View shard distribution
-- SELECT * FROM citus_shards;

-- View table distribution
-- SELECT * FROM citus_tables;

-- Rebalance shards after adding nodes
-- SELECT rebalance_table_shards();

-- Move specific shard
-- SELECT citus_move_shard_placement(shard_id, 'source_node', 5432, 'target_node', 5432);

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON TABLE tenants IS 'Reference table: replicated to all nodes for fast joins';
COMMENT ON TABLE datasources IS 'Distributed by tenant_id: tenant data stays together';
COMMENT ON TABLE semantic_objects IS 'Colocated with datasources for efficient tenant queries';
COMMENT ON FUNCTION search_tenant_objects IS 'Distributed search function - runs on coordinator and shards';

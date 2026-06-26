-- PostgreSQL Metadata Schema (Multi-Tenant)
-- 
-- This database holds:
--   - Tenant registry
--   - Semantic object definitions
--   - Bundles, policies, mappings
--   - Connection strings to tenant-specific databases
--
-- Uses Citus for horizontal scaling (or runs on Cosmos DB for PostgreSQL)

-- =============================================================================
-- TENANT MANAGEMENT
-- =============================================================================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(128) UNIQUE NOT NULL,
    display_name VARCHAR(256) NOT NULL,
    status VARCHAR(32) DEFAULT 'active',
    tier VARCHAR(32) DEFAULT 'standard', -- free, standard, enterprise
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenants_external_id ON tenants(external_id);
CREATE INDEX idx_tenants_status ON tenants(status);

-- Tenant database connections (isolated PostgreSQL instances)
CREATE TABLE IF NOT EXISTS tenant_databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    database_type VARCHAR(32) NOT NULL DEFAULT 'financial', -- financial, analytics, etc.
    connection_name VARCHAR(128) NOT NULL,
    host VARCHAR(256) NOT NULL,
    port INTEGER NOT NULL DEFAULT 5432,
    database_name VARCHAR(128) NOT NULL,
    username VARCHAR(128) NOT NULL,
    -- Password stored in secrets manager, reference here
    secret_ref VARCHAR(256) NOT NULL,
    ssl_mode VARCHAR(32) DEFAULT 'require',
    max_connections INTEGER DEFAULT 10,
    is_primary BOOLEAN DEFAULT false,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, database_type, connection_name)
);

CREATE INDEX idx_tenant_databases_tenant ON tenant_databases(tenant_id);

-- =============================================================================
-- DATASOURCES (Analytics Connections)
-- =============================================================================

CREATE TABLE IF NOT EXISTS datasources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_name VARCHAR(128) NOT NULL,
    source_type VARCHAR(32) NOT NULL, -- starrocks, postgres, snowflake, etc.
    connection_config JSONB NOT NULL DEFAULT '{}',
    -- Secrets stored externally
    secret_ref VARCHAR(256),
    is_default BOOLEAN DEFAULT false,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, source_name)
);

CREATE INDEX idx_datasources_tenant ON datasources(tenant_id);

-- =============================================================================
-- SEMANTIC OBJECTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS semantic_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    datasource_id UUID REFERENCES datasources(id) ON DELETE SET NULL,
    object_type VARCHAR(32) NOT NULL, -- dimension, measure, cube, view
    name VARCHAR(256) NOT NULL,
    display_name VARCHAR(256),
    description TEXT,
    definition JSONB NOT NULL DEFAULT '{}',
    sql_expression TEXT,
    data_type VARCHAR(64),
    is_published BOOLEAN DEFAULT false,
    version INTEGER DEFAULT 1,
    tags TEXT[] DEFAULT '{}',
    -- Full-text search
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(display_name, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'C')
    ) STORED,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id, name)
);

CREATE INDEX idx_semantic_objects_tenant ON semantic_objects(tenant_id);
CREATE INDEX idx_semantic_objects_type ON semantic_objects(object_type);
CREATE INDEX idx_semantic_objects_search ON semantic_objects USING gin(search_vector);
CREATE INDEX idx_semantic_objects_tags ON semantic_objects USING gin(tags);

-- =============================================================================
-- BUNDLES
-- =============================================================================

CREATE TABLE IF NOT EXISTS bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    display_name VARCHAR(256),
    description TEXT,
    bundle_type VARCHAR(32) DEFAULT 'standard', -- standard, template, system
    status VARCHAR(32) DEFAULT 'draft', -- draft, published, archived
    settings JSONB DEFAULT '{}',
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_bundles_tenant ON bundles(tenant_id);
CREATE INDEX idx_bundles_status ON bundles(status);

-- Bundle to semantic object mapping
CREATE TABLE IF NOT EXISTS bundle_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bundle_id UUID NOT NULL REFERENCES bundles(id) ON DELETE CASCADE,
    semantic_object_id UUID NOT NULL REFERENCES semantic_objects(id) ON DELETE CASCADE,
    include_mode VARCHAR(32) DEFAULT 'include', -- include, exclude
    order_index INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(bundle_id, semantic_object_id)
);

CREATE INDEX idx_bundle_items_bundle ON bundle_items(bundle_id);
CREATE INDEX idx_bundle_items_object ON bundle_items(semantic_object_id);

-- =============================================================================
-- POLICIES
-- =============================================================================

CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    policy_type VARCHAR(32) NOT NULL, -- row_filter, column_mask, access_control
    scope VARCHAR(32) DEFAULT 'bundle', -- bundle, datasource, global
    target_type VARCHAR(32), -- bundle, semantic_object, column
    target_id UUID,
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '{}',
    priority INTEGER DEFAULT 100,
    is_enabled BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_policies_tenant ON policies(tenant_id);
CREATE INDEX idx_policies_type ON policies(policy_type);
CREATE INDEX idx_policies_target ON policies(target_type, target_id);

-- =============================================================================
-- AUDIT LOG
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    actor_id VARCHAR(128),
    actor_type VARCHAR(32), -- user, service, system
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id UUID,
    resource_name VARCHAR(256),
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Partition by month for efficient cleanup
CREATE INDEX idx_audit_log_tenant_time ON audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Search semantic objects within a tenant
CREATE OR REPLACE FUNCTION search_semantic_objects(
    p_tenant_id UUID,
    p_query TEXT,
    p_object_types TEXT[] DEFAULT NULL,
    p_limit INTEGER DEFAULT 50
)
RETURNS TABLE (
    id UUID,
    name VARCHAR,
    display_name VARCHAR,
    object_type VARCHAR,
    description TEXT,
    rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        so.id,
        so.name,
        so.display_name,
        so.object_type,
        so.description,
        ts_rank(so.search_vector, websearch_to_tsquery('english', p_query)) AS rank
    FROM semantic_objects so
    WHERE so.tenant_id = p_tenant_id
      AND so.search_vector @@ websearch_to_tsquery('english', p_query)
      AND (p_object_types IS NULL OR so.object_type = ANY(p_object_types))
    ORDER BY rank DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Get tenant database connection
CREATE OR REPLACE FUNCTION get_tenant_database(
    p_tenant_id UUID,
    p_database_type VARCHAR DEFAULT 'financial'
)
RETURNS TABLE (
    host VARCHAR,
    port INTEGER,
    database_name VARCHAR,
    username VARCHAR,
    secret_ref VARCHAR,
    ssl_mode VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        td.host,
        td.port,
        td.database_name,
        td.username,
        td.secret_ref,
        td.ssl_mode
    FROM tenant_databases td
    WHERE td.tenant_id = p_tenant_id
      AND td.database_type = p_database_type
      AND td.status = 'active'
      AND td.is_primary = true
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- SAMPLE DATA
-- =============================================================================

-- Demo tenant
INSERT INTO tenants (external_id, display_name, tier) 
VALUES ('demo', 'Demo Organization', 'enterprise')
ON CONFLICT (external_id) DO NOTHING;

-- Demo datasource (StarRocks hot store)
INSERT INTO datasources (tenant_id, source_name, source_type, connection_config, is_default)
SELECT 
    id,
    'starrocks-hot',
    'starrocks',
    '{"host": "starrocks-fe", "port": 9030, "database": "cube_hot"}',
    true
FROM tenants WHERE external_id = 'demo'
ON CONFLICT (tenant_id, source_name) DO NOTHING;

-- Demo tenant database (isolated PostgreSQL)
INSERT INTO tenant_databases (tenant_id, database_type, connection_name, host, port, database_name, username, secret_ref, is_primary)
SELECT 
    id,
    'financial',
    'primary',
    'postgres-tenant-demo',
    5432,
    'tenant_demo',
    'tenant_demo',
    'secrets/tenant-demo/db-password',
    true
FROM tenants WHERE external_id = 'demo'
ON CONFLICT (tenant_id, database_type, connection_name) DO NOTHING;

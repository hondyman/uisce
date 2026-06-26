-- Semantic Layer Platform Schema Extension
-- Extends existing cube model builder with multi-tenant semantic layer functionality

-- Enhanced semantic cubes table (extends existing if present)
CREATE TABLE IF NOT EXISTS semantic_cubes_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    sql TEXT NOT NULL,  -- Base SQL for the cube
    refresh_key TEXT,   -- For cache invalidation
    pre_aggregations JSONB DEFAULT '[]'::jsonb,
    joins JSONB DEFAULT '[]'::jsonb,  -- Cube joins
    metadata JSONB DEFAULT '{}'::jsonb,
    status TEXT DEFAULT 'draft',  -- draft, active, deprecated
    version INTEGER DEFAULT 1,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, name, version)
);

-- Semantic dimensions
CREATE TABLE IF NOT EXISTS semantic_dimensions_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cube_id UUID NOT NULL REFERENCES semantic_cubes_v2(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    type TEXT NOT NULL,  -- string, number, time, geo, boolean
    sql TEXT NOT NULL,
    format TEXT,
    case_sensitive BOOLEAN DEFAULT false,
    primary_key BOOLEAN DEFAULT false,
    shown BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(cube_id, name)
);

-- Semantic measures
CREATE TABLE IF NOT EXISTS semantic_measures_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cube_id UUID NOT NULL REFERENCES semantic_cubes_v2(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    type TEXT NOT NULL,  -- count, sum, avg, min, max, countDistinct, countDistinctApprox, runningTotal
    sql TEXT NOT NULL,
    format TEXT,
    rolling_window TEXT,
    drill_members TEXT[] DEFAULT '{}',
    filters JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(cube_id, name)
);

-- Pre-aggregations for performance
CREATE TABLE IF NOT EXISTS semantic_pre_aggregations_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cube_id UUID NOT NULL REFERENCES semantic_cubes_v2(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- rollup, originalSql, rollupJoin, rollupLambda
    dimensions TEXT[] DEFAULT '{}',
    measures TEXT[] DEFAULT '{}',
    segments TEXT[] DEFAULT '{}',
    time_dimension TEXT,
    granularity TEXT,  -- second, minute, hour, day, week, month, quarter, year
    partition_granularity TEXT,
    refresh_key TEXT,
    indexes JSONB DEFAULT '[]'::jsonb,
    build_range_start TEXT,
    build_range_end TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    last_built_at TIMESTAMPTZ,
    status TEXT DEFAULT 'pending',  -- pending, building, ready, failed
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(cube_id, name)
);

-- Query cache for fast results
CREATE TABLE IF NOT EXISTS semantic_query_cache_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_hash TEXT NOT NULL,
    query JSONB NOT NULL,
    result JSONB NOT NULL,
    result_rows INTEGER,
    execution_time_ms INTEGER,
    cache_key TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_accessed_at TIMESTAMPTZ DEFAULT now(),
    access_count INTEGER DEFAULT 1,
    UNIQUE(tenant_id, query_hash)
);

-- Query execution history
CREATE TABLE IF NOT EXISTS semantic_query_history_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    cube_name TEXT,
    query JSONB NOT NULL,
    generated_sql TEXT,
    execution_time_ms INTEGER,
    result_rows INTEGER,
    cache_hit BOOLEAN DEFAULT false,
    pre_agg_used TEXT,  -- Name of pre-aggregation used
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Cube metadata cache (for fast lookups)
CREATE TABLE IF NOT EXISTS semantic_cube_cache (
    tenant_id UUID NOT NULL,
    cube_name TEXT NOT NULL,
    metadata JSONB NOT NULL,
    dimensions JSONB NOT NULL,
    measures JSONB NOT NULL,
    pre_aggregations JSONB NOT NULL,
    cached_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (tenant_id, cube_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_semantic_cubes_v2_tenant ON semantic_cubes_v2(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_semantic_dimensions_v2_cube ON semantic_dimensions_v2(cube_id);
CREATE INDEX IF NOT EXISTS idx_semantic_measures_v2_cube ON semantic_measures_v2(cube_id);
CREATE INDEX IF NOT EXISTS idx_semantic_pre_aggs_v2_cube ON semantic_pre_aggregations_v2(cube_id, status);
CREATE INDEX IF NOT EXISTS idx_semantic_query_cache_v2_tenant_hash ON semantic_query_cache_v2(tenant_id, query_hash);
CREATE INDEX IF NOT EXISTS idx_semantic_query_cache_v2_expires ON semantic_query_cache_v2(expires_at);
CREATE INDEX IF NOT EXISTS idx_semantic_query_history_v2_tenant ON semantic_query_history_v2(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_history_v2_cube ON semantic_query_history_v2(cube_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_cube_cache_tenant ON semantic_cube_cache(tenant_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_semantic_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS semantic_cubes_v2_updated_at ON semantic_cubes_v2;
CREATE TRIGGER semantic_cubes_v2_updated_at
    BEFORE UPDATE ON semantic_cubes_v2
    FOR EACH ROW
    EXECUTE FUNCTION update_semantic_updated_at();

-- Function to invalidate cube cache on changes
CREATE OR REPLACE FUNCTION invalidate_semantic_cube_cache()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM semantic_cube_cache 
    WHERE tenant_id = COALESCE(NEW.tenant_id, OLD.tenant_id)
    AND cube_name IN (
        SELECT name FROM semantic_cubes_v2 
        WHERE id = COALESCE(NEW.cube_id, OLD.cube_id, NEW.id, OLD.id)
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for cache invalidation
DROP TRIGGER IF EXISTS semantic_cubes_v2_cache_invalidate ON semantic_cubes_v2;
CREATE TRIGGER semantic_cubes_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON semantic_cubes_v2
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_semantic_cube_cache();

DROP TRIGGER IF EXISTS semantic_dimensions_v2_cache_invalidate ON semantic_dimensions_v2;
CREATE TRIGGER semantic_dimensions_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON semantic_dimensions_v2
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_semantic_cube_cache();

DROP TRIGGER IF EXISTS semantic_measures_v2_cache_invalidate ON semantic_measures_v2;
CREATE TRIGGER semantic_measures_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON semantic_measures_v2
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_semantic_cube_cache();

-- Function to clean expired query cache
CREATE OR REPLACE FUNCTION clean_expired_query_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM semantic_query_cache_v2
    WHERE expires_at < now();
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE semantic_cubes_v2 IS 'Cube definitions for semantic layer (Cube.dev-style)';
COMMENT ON TABLE semantic_dimensions_v2 IS 'Dimensions for semantic cubes';
COMMENT ON TABLE semantic_measures_v2 IS 'Measures (metrics) for semantic cubes';
COMMENT ON TABLE semantic_pre_aggregations_v2 IS 'Pre-aggregations for query performance';
COMMENT ON TABLE semantic_query_cache_v2 IS 'Query result cache';
COMMENT ON TABLE semantic_query_history_v2 IS 'Query execution history and analytics';
COMMENT ON TABLE semantic_cube_cache IS 'Compiled cube metadata cache';

-- Cost-Based Optimizer Schema
-- Tracks query execution patterns for adaptive optimization

-- Query execution log for workload analysis
CREATE TABLE IF NOT EXISTS cbo_query_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_hash TEXT NOT NULL,
    query_pattern TEXT,
    execution_path TEXT NOT NULL CHECK (execution_path IN ('direct', 'preagg', 'cache', 'materialized')),
    estimated_cost FLOAT,
    actual_duration_ms INT,
    cache_hit BOOLEAN DEFAULT FALSE,
    preagg_used UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for query analysis
CREATE INDEX IF NOT EXISTS idx_cbo_query_log_tenant_created ON cbo_query_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cbo_query_log_pattern ON cbo_query_log(query_pattern);
CREATE INDEX IF NOT EXISTS idx_cbo_query_log_hash ON cbo_query_log(query_hash);
CREATE INDEX IF NOT EXISTS idx_cbo_query_log_path ON cbo_query_log(execution_path);
CREATE INDEX IF NOT EXISTS idx_cbo_query_log_duration ON cbo_query_log(actual_duration_ms DESC);

-- Table statistics for cost estimation
CREATE TABLE IF NOT EXISTS cbo_table_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    table_name TEXT NOT NULL,
    schema_name TEXT DEFAULT 'public',
    row_count BIGINT,
    avg_row_size INT,
    total_size_bytes BIGINT,
    analyzed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, table_name)
);

CREATE INDEX IF NOT EXISTS idx_cbo_table_stats_tenant ON cbo_table_stats(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbo_table_stats_table ON cbo_table_stats(table_name);

-- Column statistics for selectivity estimation
CREATE TABLE IF NOT EXISTS cbo_column_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_stats_id UUID REFERENCES cbo_table_stats(id) ON DELETE CASCADE,
    column_name TEXT NOT NULL,
    data_type TEXT,
    null_fraction FLOAT DEFAULT 0,
    distinct_count BIGINT,
    avg_width INT,
    min_value TEXT,
    max_value TEXT,
    histogram JSONB, -- For distribution-aware estimation
    analyzed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(table_stats_id, column_name)
);

CREATE INDEX IF NOT EXISTS idx_cbo_column_stats_table ON cbo_column_stats(table_stats_id);

-- CBO recommendations log
CREATE TABLE IF NOT EXISTS cbo_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    recommendation_type TEXT NOT NULL CHECK (recommendation_type IN ('create_preagg', 'add_index', 'partition_table', 'enable_caching', 'tune_query')),
    priority TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    description TEXT NOT NULL,
    impact TEXT,
    sql_hint TEXT,
    query_pattern TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'implemented')),
    accepted_at TIMESTAMPTZ,
    accepted_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cbo_recommendations_tenant ON cbo_recommendations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_cbo_recommendations_type ON cbo_recommendations(recommendation_type);
CREATE INDEX IF NOT EXISTS idx_cbo_recommendations_priority ON cbo_recommendations(priority);

-- Query plan cache for complex plans
CREATE TABLE IF NOT EXISTS cbo_plan_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_hash TEXT NOT NULL,
    plan_json JSONB NOT NULL,
    estimated_cost FLOAT,
    execution_path TEXT NOT NULL,
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, query_hash)
);

CREATE INDEX IF NOT EXISTS idx_cbo_plan_cache_hash ON cbo_plan_cache(query_hash);
CREATE INDEX IF NOT EXISTS idx_cbo_plan_cache_valid ON cbo_plan_cache(valid_until);

-- Resource utilization snapshots for load-aware routing
CREATE TABLE IF NOT EXISTS cbo_resource_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    cpu_usage FLOAT,
    memory_usage FLOAT,
    active_queries INT,
    queue_depth INT,
    avg_query_time_ms FLOAT,
    snapshot_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cbo_resource_snapshots_tenant ON cbo_resource_snapshots(tenant_id, snapshot_at DESC);

-- Cleanup policy: auto-delete old logs (can be run by scheduler)
-- DELETE FROM cbo_query_log WHERE created_at < NOW() - INTERVAL '30 days';
-- DELETE FROM cbo_resource_snapshots WHERE snapshot_at < NOW() - INTERVAL '7 days';

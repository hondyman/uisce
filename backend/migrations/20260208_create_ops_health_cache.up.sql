-- 20260208_create_ops_health_cache.up.sql
-- Cached health scores for tenants and endpoints

CREATE TABLE IF NOT EXISTS ops_tenant_health_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,
    health_score INTEGER NOT NULL, -- 0-100
    components JSONB NOT NULL, -- {"availability": 95, "latency": 80, ...}
    computed_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_tenant_health_cache_tenant_id ON ops_tenant_health_cache(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ops_tenant_health_cache_health_score ON ops_tenant_health_cache(health_score);

CREATE TABLE IF NOT EXISTS ops_endpoint_health_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint TEXT NOT NULL UNIQUE,
    health_score INTEGER NOT NULL, -- 0-100
    error_rate DOUBLE PRECISION,
    p95_ms INTEGER,
    requests_1h BIGINT,
    components JSONB,
    computed_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_endpoint_health_cache_endpoint ON ops_endpoint_health_cache(endpoint);
CREATE INDEX IF NOT EXISTS idx_ops_endpoint_health_cache_health_score ON ops_endpoint_health_cache(health_score);

-- Heatmap data: time-series latency buckets per dimension (region, tenant, endpoint)
CREATE TABLE IF NOT EXISTS ops_latency_heatmap (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_time TIMESTAMPTZ NOT NULL,
    dimension_type TEXT NOT NULL, -- 'region', 'tenant', 'endpoint'
    dimension_value TEXT NOT NULL,
    p50_ms INTEGER,
    p95_ms INTEGER,
    p99_ms INTEGER,
    request_count INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ops_latency_heatmap_bucket_time ON ops_latency_heatmap(bucket_time DESC);
CREATE INDEX IF NOT EXISTS idx_ops_latency_heatmap_dimension ON ops_latency_heatmap(dimension_type, dimension_value);
CREATE INDEX IF NOT EXISTS idx_ops_latency_heatmap_dimension_time ON ops_latency_heatmap(dimension_type, dimension_value, bucket_time DESC);

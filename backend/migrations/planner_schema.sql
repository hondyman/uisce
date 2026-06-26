-- Planner schema for query optimization and decision persistence
-- Database: alpha (100.84.126.19)
-- Created: 2026-02-10

-- Create planner schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS planner;

-- planner_decisions: Persists all planner decisions for audit and monitoring
CREATE TABLE IF NOT EXISTS planner.planner_decisions (
    id BIGSERIAL PRIMARY KEY,
    plan_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(255) NOT NULL,
    query_type VARCHAR(50) NOT NULL, -- feature|metric|ts|drift|importance|discovery
    semantic_target VARCHAR(255) NOT NULL,
    selected_regions TEXT[] NOT NULL DEFAULT '{}', -- array of region codes
    plan_type VARCHAR(50) NOT NULL, -- single_region|multi_region_fanout|global_federated
    estimated_cost FLOAT8 NOT NULL DEFAULT 0.0,
    estimated_latency_ms FLOAT8 NOT NULL DEFAULT 0.0,
    degradation_strategy JSONB, -- Serialized DegradationStrategy
    explain TEXT, -- Human-readable explanation
    raw_request JSONB, -- Original QueryRequest
    raw_plan JSONB, -- Serialized QueryPlan
    region_health_snapshot JSONB, -- Region health at decision time
    executed_at TIMESTAMP,
    actual_latency_ms FLOAT8,
    actual_cost FLOAT8,
    execution_status VARCHAR(50), -- pending|success|partial|failed
    execution_error TEXT,
    
    CONSTRAINT fk_tenant CHECK (tenant_id != ''),
    CONSTRAINT fk_query_type CHECK (query_type IN ('feature', 'metric', 'ts', 'drift', 'importance', 'discovery')),
    CONSTRAINT fk_plan_type CHECK (plan_type IN ('single_region', 'multi_region_fanout', 'global_federated'))
);

CREATE INDEX idx_planner_decisions_tenant_created ON planner.planner_decisions(tenant_id, created_at DESC);
CREATE INDEX idx_planner_decisions_plan_id ON planner.planner_decisions(plan_id);
CREATE INDEX idx_planner_decisions_query_type ON planner.planner_decisions(query_type);
CREATE INDEX idx_planner_decisions_status ON planner.planner_decisions(execution_status);
CREATE INDEX idx_planner_decisions_region ON planner.planner_decisions USING GIN (selected_regions);

-- planner_metrics: Tracks decision accuracy and performance
CREATE TABLE IF NOT EXISTS planner.planner_metrics (
    id BIGSERIAL PRIMARY KEY,
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    query_type VARCHAR(50) NOT NULL,
    plan_type VARCHAR(50) NOT NULL,
    estimated_latency_ms FLOAT8,
    actual_latency_ms FLOAT8,
    latency_error_pct FLOAT8, -- (actual - estimated) / estimated * 100
    estimated_cost FLOAT8,
    actual_cost FLOAT8,
    regions_used INT DEFAULT 1,
    execution_status VARCHAR(50),
    degraded BOOLEAN DEFAULT FALSE,
    
    CONSTRAINT fk_metric_query_type CHECK (query_type IN ('feature', 'metric', 'ts', 'drift', 'importance', 'discovery'))
);

CREATE INDEX idx_planner_metrics_ts ON planner.planner_metrics(ts DESC);
CREATE INDEX idx_planner_metrics_query_type_ts ON planner.planner_metrics(query_type, ts DESC);
CREATE INDEX idx_planner_metrics_status ON planner.planner_metrics(execution_status);

-- region_performance: Real-time health metrics for each region
CREATE TABLE IF NOT EXISTS planner.region_performance (
    id BIGSERIAL PRIMARY KEY,
    region VARCHAR(50) UNIQUE NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_healthy BOOLEAN DEFAULT TRUE,
    latency_ms_p50 FLOAT8,
    latency_ms_p95 FLOAT8,
    latency_ms_p99 FLOAT8,
    error_rate FLOAT8,
    active_features INT DEFAULT 0,
    materialization_freshness_pct FLOAT8,
    cache_hit_rate FLOAT8,
    
    CONSTRAINT positive_latency CHECK (
        (latency_ms_p50 IS NULL OR latency_ms_p50 >= 0) AND
        (latency_ms_p95 IS NULL OR latency_ms_p95 >= 0) AND
        (latency_ms_p99 IS NULL OR latency_ms_p99 >= 0)
    ),
    CONSTRAINT valid_rates CHECK (
        (error_rate IS NULL OR (error_rate >= 0 AND error_rate <= 1)) AND
        (cache_hit_rate IS NULL OR (cache_hit_rate >= 0 AND cache_hit_rate <= 1))
    )
);

CREATE INDEX idx_region_performance_region ON planner.region_performance(region);
CREATE INDEX idx_region_performance_health ON planner.region_performance(is_healthy, last_updated DESC);

-- feature_planner_config: Configuration preferences per feature
CREATE TABLE IF NOT EXISTS planner.feature_planner_config (
    id BIGSERIAL PRIMARY KEY,
    feature_id VARCHAR(255) UNIQUE NOT NULL,
    preferred_regions TEXT[] DEFAULT '{}',
    disallowed_regions TEXT[] DEFAULT '{}',
    default_consistency VARCHAR(50) DEFAULT 'eventual', -- strong|eventual|region_preferred
    default_freshness VARCHAR(50) DEFAULT '1h',
    interactive_latency_budget_ms INT DEFAULT 500,
    batch_latency_budget_ms INT DEFAULT 30000,
    use_cache_if_stale BOOLEAN DEFAULT FALSE,
    max_cache_staleness VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_consistency CHECK (default_consistency IN ('strong', 'eventual', 'region_preferred')),
    CONSTRAINT positive_budgets CHECK (
        interactive_latency_budget_ms >= 0 AND
        batch_latency_budget_ms >= 0
    )
);

CREATE INDEX idx_feature_planner_config_feature_id ON planner.feature_planner_config(feature_id);
CREATE INDEX idx_feature_planner_config_updated ON planner.feature_planner_config(updated_at DESC);

-- Seed region_performance with common regions
INSERT INTO planner.region_performance (region, is_healthy, latency_ms_p50, latency_ms_p95, latency_ms_p99, error_rate, cache_hit_rate)
VALUES
    ('us-east', TRUE, 40.0, 80.0, 120.0, 0.001, 0.85),
    ('eu-west', TRUE, 80.0, 150.0, 200.0, 0.002, 0.82),
    ('apac', TRUE, 200.0, 300.0, 350.0, 0.005, 0.78)
ON CONFLICT (region) DO UPDATE SET
    latency_ms_p50 = EXCLUDED.latency_ms_p50,
    latency_ms_p95 = EXCLUDED.latency_ms_p95,
    latency_ms_p99 = EXCLUDED.latency_ms_p99,
    error_rate = EXCLUDED.error_rate,
    cache_hit_rate = EXCLUDED.cache_hit_rate;

-- Grant permissions
GRANT ALL PRIVILEGES ON SCHEMA planner TO postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA planner TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA planner TO postgres;

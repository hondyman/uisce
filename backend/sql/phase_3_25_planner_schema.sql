-- Phase 3.25: Global Query Planner Schema
-- Planner decisions audit log and configuration

-- ============================================================================
-- PLANNER DECISIONS TABLE (Audit log of all planner decisions)
-- ============================================================================
CREATE TABLE IF NOT EXISTS planner_decisions (
    plan_id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT now(),
    tenant_id TEXT,
    query_type TEXT NOT NULL CHECK (query_type IN ('feature', 'metric', 'ts', 'drift', 'importance', 'discovery')),
    semantic_target TEXT NOT NULL,
    selected_regions TEXT[] NOT NULL,
    plan_type TEXT NOT NULL CHECK (plan_type IN ('single_region', 'multi_region_fanout', 'global_federated')),
    estimated_cost DOUBLE PRECISION NOT NULL,
    estimated_latency_ms DOUBLE PRECISION NOT NULL,
    degradation_strategy JSONB NOT NULL,
    explain TEXT NOT NULL,
    
    -- Raw request and plan for full visibility
    raw_request JSONB NOT NULL,
    raw_plan JSONB NOT NULL,
    
    -- Execution metadata (populated after query runs)
    executed_at TIMESTAMPTZ,
    actual_latency_ms DOUBLE PRECISION,
    actual_cost DOUBLE PRECISION,
    execution_status TEXT CHECK (execution_status IN ('success', 'partial_failure', 'failed', 'pending')),
    execution_error TEXT,
    
    -- Query-time region health snapshot
    region_health_snapshot JSONB
);

CREATE INDEX idx_planner_decisions_target ON planner_decisions(semantic_target);
CREATE INDEX idx_planner_decisions_query_type ON planner_decisions(query_type);
CREATE INDEX idx_planner_decisions_created_at ON planner_decisions(created_at DESC);
CREATE INDEX idx_planner_decisions_tenant_id ON planner_decisions(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_planner_decisions_plan_type ON planner_decisions(plan_type);

-- ============================================================================
-- PLANNER FEATURE CONFIG (Per-feature planner preferences)
-- ============================================================================
CREATE TABLE IF NOT EXISTS planner_feature_config (
    feature_id TEXT PRIMARY KEY,
    preferred_regions TEXT[] DEFAULT ARRAY[]::TEXT[],
    disallowed_regions TEXT[] DEFAULT ARRAY[]::TEXT[],
    default_consistency TEXT DEFAULT 'region_preferred' CHECK (default_consistency IN ('strong', 'eventual', 'region_preferred')),
    default_freshness TEXT DEFAULT '15m',
    interactive_latency_budget_ms INTEGER DEFAULT 2000,
    batch_latency_budget_ms INTEGER DEFAULT 600000,
    use_cache_if_stale BOOLEAN DEFAULT TRUE,
    max_cache_staleness TEXT DEFAULT '1h',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_planner_feature_config_preferred_regions ON planner_feature_config USING GIN(preferred_regions);
CREATE INDEX idx_planner_feature_config_disallowed_regions ON planner_feature_config USING GIN(disallowed_regions);

-- ============================================================================
-- PLANNER METRICS (Aggregated statistics for SLO tracking)
-- ============================================================================
CREATE TABLE IF NOT EXISTS planner_metrics (
    id SERIAL PRIMARY KEY,
    ts TIMESTAMPTZ DEFAULT now(),
    query_type TEXT NOT NULL,
    plan_type TEXT NOT NULL,
    estimated_latency_ms DOUBLE PRECISION,
    actual_latency_ms DOUBLE PRECISION,
    latency_error_pct DOUBLE PRECISION, -- abs(actual - estimated) / estimated * 100
    estimated_cost DOUBLE PRECISION,
    actual_cost DOUBLE PRECISION,
    regions_used INTEGER,
    execution_status TEXT,
    degraded BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_planner_metrics_ts ON planner_metrics(ts DESC);
CREATE INDEX idx_planner_metrics_query_type_plan_type ON planner_metrics(query_type, plan_type);
CREATE INDEX idx_planner_metrics_efficiency ON planner_metrics(latency_error_pct) WHERE latency_error_pct IS NOT NULL;

-- ============================================================================
-- PLANNER REGION PERFORMANCE (Cached region health snapshots)
-- ============================================================================
CREATE TABLE IF NOT EXISTS planner_region_performance (
    region TEXT PRIMARY KEY,
    last_updated TIMESTAMPTZ DEFAULT now(),
    is_healthy BOOLEAN DEFAULT TRUE,
    latency_ms_p50 DOUBLE PRECISION,
    latency_ms_p95 DOUBLE PRECISION,
    latency_ms_p99 DOUBLE PRECISION,
    error_rate DOUBLE PRECISION,
    active_features INTEGER DEFAULT 0,
    materialization_freshness_pct DOUBLE PRECISION, -- % features materialized within freshness window
    cache_hit_rate DOUBLE PRECISION
);

CREATE INDEX idx_planner_region_performance_health ON planner_region_performance(is_healthy, latency_ms_p99);

-- ============================================================================
-- STORED PROCEDURES (Planner utilities)
-- ============================================================================

-- Record a planner decision
CREATE OR REPLACE FUNCTION record_planner_decision(
    p_plan_id TEXT,
    p_tenant_id TEXT,
    p_query_type TEXT,
    p_semantic_target TEXT,
    p_selected_regions TEXT[],
    p_plan_type TEXT,
    p_estimated_cost DOUBLE PRECISION,
    p_estimated_latency_ms DOUBLE PRECISION,
    p_degradation_strategy JSONB,
    p_explain TEXT,
    p_raw_request JSONB,
    p_raw_plan JSONB,
    p_region_health_snapshot JSONB
) RETURNS TEXT AS $$
BEGIN
    INSERT INTO planner_decisions (
        plan_id, tenant_id, query_type, semantic_target, selected_regions,
        plan_type, estimated_cost, estimated_latency_ms, degradation_strategy,
        explain, raw_request, raw_plan, region_health_snapshot
    ) VALUES (
        p_plan_id, p_tenant_id, p_query_type, p_semantic_target, p_selected_regions,
        p_plan_type, p_estimated_cost, p_estimated_latency_ms, p_degradation_strategy,
        p_explain, p_raw_request, p_raw_plan, p_region_health_snapshot
    );
    RETURN p_plan_id;
END;
$$ LANGUAGE plpgsql;

-- Update planner decision with execution metrics
CREATE OR REPLACE FUNCTION update_planner_decision_execution(
    p_plan_id TEXT,
    p_executed_at TIMESTAMPTZ,
    p_actual_latency_ms DOUBLE PRECISION,
    p_actual_cost DOUBLE PRECISION,
    p_execution_status TEXT,
    p_execution_error TEXT
) RETURNS VOID AS $$
BEGIN
    UPDATE planner_decisions
    SET 
        executed_at = p_executed_at,
        actual_latency_ms = p_actual_latency_ms,
        actual_cost = p_actual_cost,
        execution_status = p_execution_status,
        execution_error = p_execution_error
    WHERE plan_id = p_plan_id;
    
    -- Record metric for SLO tracking
    IF p_execution_status IS NOT NULL THEN
        INSERT INTO planner_metrics (
            query_type, plan_type, estimated_latency_ms, actual_latency_ms,
            latency_error_pct, estimated_cost, actual_cost,
            execution_status, degraded
        ) SELECT
            query_type, plan_type, estimated_latency_ms, p_actual_latency_ms,
            CASE WHEN estimated_latency_ms > 0 
                THEN ABS(p_actual_latency_ms - estimated_latency_ms) * 100.0 / estimated_latency_ms
                ELSE NULL
            END,
            estimated_cost, p_actual_cost,
            p_execution_status,
            p_execution_status = 'partial_failure'
        FROM planner_decisions WHERE plan_id = p_plan_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Get planner decisions for a semantic target (debugging)
CREATE OR REPLACE FUNCTION get_planner_decisions_for_target(
    p_semantic_target TEXT,
    p_limit INTEGER DEFAULT 10
) RETURNS TABLE (
    plan_id TEXT,
    created_at TIMESTAMPTZ,
    query_type TEXT,
    plan_type TEXT,
    selected_regions TEXT[],
    estimated_latency_ms DOUBLE PRECISION,
    actual_latency_ms DOUBLE PRECISION,
    execution_status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        planner_decisions.plan_id,
        planner_decisions.created_at,
        planner_decisions.query_type,
        planner_decisions.plan_type,
        planner_decisions.selected_regions,
        planner_decisions.estimated_latency_ms,
        planner_decisions.actual_latency_ms,
        planner_decisions.execution_status
    FROM planner_decisions
    WHERE semantic_target = p_semantic_target
    ORDER BY created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Query planner SLO compliance
CREATE OR REPLACE FUNCTION planner_slo_compliance(
    p_query_type TEXT,
    p_hours_back INTEGER DEFAULT 24
) RETURNS TABLE (
    metric_name TEXT,
    query_count INTEGER,
    latency_error_avg_pct DOUBLE PRECISION,
    success_rate DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        'latency_estimation' AS metric_name,
        COUNT(*)::INTEGER,
        AVG(latency_error_pct),
        COUNT(CASE WHEN execution_status = 'success' THEN 1 END)::DOUBLE PRECISION * 100.0 / COUNT(*)
    FROM planner_metrics
    WHERE query_type = p_query_type
      AND ts > now() - (p_hours_back || ' hours')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- INITIAL DATA (Cost model baseline)
-- ============================================================================

-- Update region performance baseline
UPDATE planner_region_performance SET
    latency_ms_p50 = 50,
    latency_ms_p95 = 80,
    latency_ms_p99 = 120,
    error_rate = 0.001,
    materialization_freshness_pct = 0.98
WHERE region = 'us-east';

UPDATE planner_region_performance SET
    latency_ms_p50 = 90,
    latency_ms_p95 = 140,
    latency_ms_p99 = 200,
    error_rate = 0.002,
    materialization_freshness_pct = 0.95
WHERE region = 'eu-west';

UPDATE planner_region_performance SET
    latency_ms_p50 = 180,
    latency_ms_p95 = 250,
    latency_ms_p99 = 350,
    error_rate = 0.003,
    materialization_freshness_pct = 0.92
WHERE region = 'apac';

-- Seed some planner feature configs (examples)
INSERT INTO planner_feature_config (feature_id, preferred_regions, default_consistency, default_freshness)
VALUES 
    ('customer_lifetime_value', ARRAY['us-east', 'eu-west'], 'region_preferred', '1h'),
    ('transaction_anomaly_score', ARRAY['us-east'], 'strong', '5m'),
    ('global_fraud_pattern', ARRAY['us-east', 'eu-west', 'apac'], 'eventual', '2h')
ON CONFLICT (feature_id) DO NOTHING;

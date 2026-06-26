-- Phase 9: ASO ML Feature Store
-- Feature tables for ML-powered optimization scoring

-- Create aso schema if not exists
CREATE SCHEMA IF NOT EXISTS aso;

-- ============================================================================
-- BO Features - Business Object workload aggregates
-- ============================================================================
CREATE TABLE IF NOT EXISTS aso.bo_features (
    env text NOT NULL,
    tenant_id uuid NOT NULL,
    bo_id uuid NOT NULL,
    bo_name text NOT NULL,
    window_interval text NOT NULL,  -- '7d', '30d', '90d'
    
    -- Query volume
    queries int NOT NULL DEFAULT 0,
    queries_per_day float NOT NULL DEFAULT 0,
    distinct_users int NOT NULL DEFAULT 0,
    distinct_query_patterns int NOT NULL DEFAULT 0,
    
    -- Latency
    avg_latency_ms float NOT NULL DEFAULT 0,
    p50_latency_ms float NOT NULL DEFAULT 0,
    p95_latency_ms float NOT NULL DEFAULT 0,
    p99_latency_ms float NOT NULL DEFAULT 0,
    
    -- Scan metrics
    avg_scan_bytes float NOT NULL DEFAULT 0,
    avg_rows_scanned float NOT NULL DEFAULT 0,
    
    -- Pre-agg coverage
    preagg_hit_rate float NOT NULL DEFAULT 0,
    preagg_miss_rate float NOT NULL DEFAULT 0,
    preagg_miss_queries int NOT NULL DEFAULT 0,
    
    -- Time patterns
    peak_hour int,  -- 0-23
    peak_day_of_week int,  -- 0=Sunday
    
    -- Metadata
    last_updated timestamptz NOT NULL DEFAULT now(),
    
    PRIMARY KEY (env, tenant_id, bo_id, window_interval)
);

CREATE INDEX IF NOT EXISTS idx_bo_features_tenant ON aso.bo_features(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bo_features_updated ON aso.bo_features(last_updated);

-- ============================================================================
-- Pre-Agg Features - Pre-aggregation performance metrics
-- ============================================================================
CREATE TABLE IF NOT EXISTS aso.preagg_features (
    env text NOT NULL,
    tenant_id uuid,  -- NULL for core pre-aggs
    preagg_id uuid NOT NULL,
    preagg_name text NOT NULL,
    window_interval text NOT NULL,
    
    -- Acceleration metrics
    queries_accelerated int NOT NULL DEFAULT 0,
    avg_speedup float NOT NULL DEFAULT 1.0,
    hit_rate float NOT NULL DEFAULT 0,
    
    -- Storage
    storage_bytes bigint NOT NULL DEFAULT 0,
    row_count bigint NOT NULL DEFAULT 0,
    
    -- Refresh costs
    refresh_cost_ms float NOT NULL DEFAULT 0,
    refresh_frequency_sec int NOT NULL DEFAULT 3600,
    last_refresh_at timestamptz,
    refresh_failure_count int NOT NULL DEFAULT 0,
    
    -- Usage trend
    usage_trend_pct float NOT NULL DEFAULT 0,  -- + growth, - decline
    days_since_last_use int NOT NULL DEFAULT 0,
    
    -- Metadata
    last_updated timestamptz NOT NULL DEFAULT now(),
    
    PRIMARY KEY (env, preagg_id, window_interval)
);

CREATE INDEX IF NOT EXISTS idx_preagg_features_tenant ON aso.preagg_features(tenant_id);
CREATE INDEX IF NOT EXISTS idx_preagg_features_updated ON aso.preagg_features(last_updated);

-- ============================================================================
-- Optimization Features - ML training/inference feature vectors
-- ============================================================================
CREATE TABLE IF NOT EXISTS aso.optimization_features (
    optimization_id uuid PRIMARY KEY,
    env text NOT NULL,
    tenant_id uuid,
    
    -- Optimization metadata
    type text NOT NULL,  -- create_preagg, tune_refresh, retire_asset, prewarm
    target_type text NOT NULL,  -- preagg, bo, calc
    target_id uuid NOT NULL,
    target_name text NOT NULL,
    bo_name text,
    window_interval text NOT NULL DEFAULT '7d',
    
    -- BO Features (copied at proposal time)
    bo_queries int,
    bo_queries_per_day float,
    bo_distinct_users int,
    bo_avg_latency_ms float,
    bo_p95_latency_ms float,
    bo_avg_scan_bytes float,
    bo_preagg_miss_rate float,
    
    -- PreAgg Features (if applicable)
    preagg_queries_accelerated int,
    preagg_avg_speedup float,
    preagg_storage_bytes bigint,
    preagg_refresh_cost_ms float,
    preagg_refresh_frequency_sec int,
    preagg_hit_rate float,
    preagg_usage_trend_pct float,
    
    -- Simulation Predictions
    sim_expected_speedup float,
    sim_expected_cost_savings float,
    sim_queries_improved int,
    sim_queries_regressed int,
    sim_hit_rate_before float,
    sim_hit_rate_after float,
    
    -- ML Predictions (at proposal time)
    ml_score float,
    ml_predicted_speedup float,
    ml_predicted_cost_savings float,
    ml_risk_score float,
    ml_confidence float,
    ml_top_factors jsonb,  -- [{"feature":"...", "weight":0.3, "direction":"positive"}]
    
    -- Labels (filled after optimization is applied and evaluated)
    realized_speedup float,
    realized_cost_savings float,
    realized_regression boolean,
    realized_latency_change_pct float,
    realized_hit_rate_change float,
    label_recorded_at timestamptz,
    label_ready boolean NOT NULL DEFAULT false,
    
    -- Timestamps
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_opt_features_type ON aso.optimization_features(type);
CREATE INDEX IF NOT EXISTS idx_opt_features_label ON aso.optimization_features(label_ready);
CREATE INDEX IF NOT EXISTS idx_opt_features_env ON aso.optimization_features(env, tenant_id);

-- ============================================================================
-- ML Model Registry - Track trained models
-- ============================================================================
CREATE TABLE IF NOT EXISTS aso.ml_model_registry (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    model_type text NOT NULL,  -- create_preagg, tune_refresh, retire_asset
    version int NOT NULL,
    
    -- Training metadata
    training_samples int NOT NULL,
    training_started_at timestamptz NOT NULL,
    training_completed_at timestamptz,
    
    -- Performance metrics
    accuracy_r2 float,
    mae float,
    auc float,
    
    -- Model artifact
    model_json jsonb,  -- Serialized model weights/trees
    feature_columns jsonb NOT NULL,  -- Feature column names
    
    -- Status
    status text NOT NULL DEFAULT 'training',  -- training, active, retired
    activated_at timestamptz,
    
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ml_model_active 
ON aso.ml_model_registry(model_type) WHERE status = 'active';

-- ============================================================================
-- View: Latest BO Features (7d window)
-- ============================================================================
CREATE OR REPLACE VIEW aso.v_bo_features_7d AS
SELECT * FROM aso.bo_features WHERE window_interval = '7d';

-- ============================================================================
-- View: ML Training Dataset
-- ============================================================================
CREATE OR REPLACE VIEW aso.v_ml_training_data AS
SELECT 
    optimization_id,
    type,
    -- Features
    bo_queries,
    bo_queries_per_day,
    bo_distinct_users,
    bo_avg_latency_ms,
    bo_p95_latency_ms,
    bo_avg_scan_bytes,
    bo_preagg_miss_rate,
    preagg_queries_accelerated,
    preagg_avg_speedup,
    preagg_storage_bytes,
    preagg_refresh_cost_ms,
    preagg_refresh_frequency_sec,
    preagg_hit_rate,
    preagg_usage_trend_pct,
    -- Labels
    realized_speedup,
    realized_cost_savings,
    realized_regression
FROM aso.optimization_features
WHERE label_ready = true;

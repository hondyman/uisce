-- Phase 3.24: Complete Multi-Region Feature Platform Schema
-- Global control plane + region-scoped + federation views
-- Deploy to: global control plane PostgreSQL

-- ============================================================================
-- PART 1: GLOBAL REGISTRY TABLES (Control Plane)
-- ============================================================================

-- Region Registry: Define all regions in the federation
CREATE TABLE IF NOT EXISTS region_registry (
    region_id SERIAL PRIMARY KEY,
    region_code TEXT UNIQUE NOT NULL,
    region_name TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    temporal_namespace TEXT NOT NULL,
    temporal_address TEXT NOT NULL,
    trino_catalog TEXT NOT NULL,
    trino_endpoint TEXT NOT NULL,
    iceberg_catalog TEXT NOT NULL,
    iceberg_s3_bucket TEXT NOT NULL,
    iceberg_warehouse_path TEXT NOT NULL,
    api_endpoint TEXT NOT NULL,
    api_port INTEGER DEFAULT 8080,
    mTLS_ca_cert TEXT,
    mTLS_client_cert TEXT,
    mTLS_client_key TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_region_registry_code ON region_registry(region_code);
CREATE INDEX idx_region_registry_active ON region_registry(is_active);

-- Global Feature Catalog (Unified Definition)
CREATE TABLE IF NOT EXISTS global_feature_catalog (
    feature_id TEXT PRIMARY KEY,
    feature_name TEXT UNIQUE NOT NULL,
    description TEXT,
    source_table TEXT,
    source_database TEXT,
    data_type TEXT,
    materialization_policy TEXT DEFAULT 'on-demand',
    materialization_frequency TEXT,
    materialize_in_regions TEXT[] DEFAULT ARRAY['us-east', 'eu-west', 'apac'],
    region_overrides JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    owner_team TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_feature_catalog_active ON global_feature_catalog(is_active);
CREATE INDEX idx_feature_catalog_regions ON global_feature_catalog USING GIN(materialize_in_regions);

-- Global Feature Status (Track per-region health)
CREATE TABLE IF NOT EXISTS global_feature_status (
    feature_id TEXT NOT NULL,
    region_code TEXT NOT NULL,
    last_materialized TIMESTAMPTZ,
    last_materialization_duration_ms BIGINT,
    last_drift_score DOUBLE PRECISION,
    last_drift_timestamp TIMESTAMPTZ,
    last_importance_score DOUBLE PRECISION,
    last_importance_timestamp TIMESTAMPTZ,
    status TEXT DEFAULT 'healthy', -- healthy, degraded, failed
    error_message TEXT,
    PRIMARY KEY (feature_id, region_code),
    FOREIGN KEY (region_code) REFERENCES region_registry(region_code)
);

CREATE INDEX idx_feature_status_region ON global_feature_status(region_code);
CREATE INDEX idx_feature_status_status ON global_feature_status(status);

-- Global Workflow Execution (Track region-routed workflows)
CREATE TABLE IF NOT EXISTS global_workflow_execution (
    execution_id TEXT PRIMARY KEY,
    workflow_name TEXT NOT NULL,
    feature_id TEXT,
    target_regions TEXT[] NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, running, success, partial_failure, failed
    started_at TIMESTAMPTZ DEFAULT now(),
    completed_at TIMESTAMPTZ,
    total_regions INTEGER,
    successful_regions INTEGER,
    failed_regions INTEGER,
    error_message TEXT
);

CREATE INDEX idx_workflow_execution_feature ON global_workflow_execution(feature_id);
CREATE INDEX idx_workflow_execution_status ON global_workflow_execution(status);
CREATE INDEX idx_workflow_execution_started ON global_workflow_execution(started_at DESC);

-- Region-Level Workflow Execution (Per-region tracking)
CREATE TABLE IF NOT EXISTS region_workflow_execution (
    execution_id TEXT NOT NULL,
    region_code TEXT NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, running, success, failed
    started_at TIMESTAMPTZ DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    error_message TEXT,
    PRIMARY KEY (execution_id, region_code),
    FOREIGN KEY (region_code) REFERENCES region_registry(region_code)
);

CREATE INDEX idx_region_workflow_status ON region_workflow_execution(region_code, status);

-- Global SLO Configuration
CREATE TABLE IF NOT EXISTS global_slo_config (
    slo_id SERIAL PRIMARY KEY,
    metric_name TEXT NOT NULL,
    target_value DOUBLE PRECISION NOT NULL,
    threshold_unit TEXT, -- latency_ms, percentage, count
    region_code TEXT,
    is_global BOOLEAN DEFAULT FALSE,
    alerting_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================================
-- PART 2: REGION-SCOPED OPERATIONAL TABLES (Template)
-- Create one set per region by replacing <region> with region_code
-- ============================================================================

-- Feature Watermarks: Track last processed timestamp per feature
CREATE TABLE IF NOT EXISTS feature_watermarks_us_east (
    feature_id TEXT PRIMARY KEY,
    last_processed TIMESTAMPTZ NOT NULL,
    last_materialized TIMESTAMPTZ,
    next_scheduled TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_watermarks_us_east_processed ON feature_watermarks_us_east(last_processed DESC);

-- Drift Metrics: Per-region drift detection results
CREATE TABLE IF NOT EXISTS feature_drift_metrics_us_east (
    feature_id TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    method TEXT NOT NULL, -- ks_test, js_distance, wasserstein, md_statistic
    score DOUBLE PRECISION NOT NULL,
    p_value DOUBLE PRECISION,
    baseline_window TEXT,
    eval_window TEXT,
    is_drift BOOLEAN,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (feature_id, ts, method)
);

CREATE INDEX idx_drift_us_east_feature ON feature_drift_metrics_us_east(feature_id);
CREATE INDEX idx_drift_us_east_ts ON feature_drift_metrics_us_east(ts DESC);

-- Feature Importance: Per-region importance scores
CREATE TABLE IF NOT EXISTS feature_importance_us_east (
    feature_id TEXT NOT NULL,
    model_id TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    method TEXT NOT NULL, -- shap, permutation, correlation, mutual_information
    importance DOUBLE PRECISION NOT NULL,
    stability DOUBLE PRECISION,
    trend DOUBLE PRECISION,
    rank_position INTEGER,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (feature_id, model_id, ts, method)
);

CREATE INDEX idx_importance_us_east_feature ON feature_importance_us_east(feature_id, ts DESC);
CREATE INDEX idx_importance_us_east_model ON feature_importance_us_east(model_id, ts DESC);

-- Time-Series Features: Per-region TS analysis results
CREATE TABLE IF NOT EXISTS feature_ts_features_us_east (
    feature_id TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    horizon TEXT NOT NULL, -- 1h, 6h, 24h, 7d, 30d
    forecast_value DOUBLE PRECISION,
    lower_bound DOUBLE PRECISION,
    upper_bound DOUBLE PRECISION,
    anomaly BOOLEAN DEFAULT FALSE,
    anomaly_score DOUBLE PRECISION,
    trend TEXT, -- up, down, stable
    seasonality_strength DOUBLE PRECISION,
    acf_lag1 DOUBLE PRECISION,
    pacf_lag1 DOUBLE PRECISION,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (feature_id, ts, horizon)
);

CREATE INDEX idx_ts_features_us_east_feature ON feature_ts_features_us_east(feature_id, ts DESC);
CREATE INDEX idx_ts_features_us_east_anomaly ON feature_ts_features_us_east(feature_id) WHERE anomaly = TRUE;

-- Discovery Candidates: Per-region discovered features
CREATE TABLE IF NOT EXISTS feature_discovery_us_east (
    candidate_id TEXT PRIMARY KEY,
    feature_name TEXT NOT NULL,
    source_database TEXT NOT NULL,
    source_field TEXT NOT NULL,
    data_type TEXT,
    completeness DOUBLE PRECISION,
    cardinality DOUBLE PRECISION,
    business_value DOUBLE PRECISION NOT NULL,
    technical_score DOUBLE PRECISION,
    discovery_method TEXT, -- schema_scan, log_parser, metric_extractor
    status TEXT DEFAULT 'candidate', -- candidate, approved, rejected
    properties JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    approved_at TIMESTAMPTZ,
    approved_by TEXT
);

CREATE INDEX idx_discovery_us_east_feature ON feature_discovery_us_east(feature_name);
CREATE INDEX idx_discovery_us_east_score ON feature_discovery_us_east(business_value DESC);
CREATE INDEX idx_discovery_us_east_status ON feature_discovery_us_east(status);

-- Materialization Log: Track feature materialization per region
CREATE TABLE IF NOT EXISTS feature_materialization_log_us_east (
    log_id SERIAL PRIMARY KEY,
    feature_id TEXT NOT NULL,
    started_at TIMESTAMPTZ DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    row_count BIGINT,
    status TEXT, -- success, failed, timeout
    error_message TEXT,
    output_path TEXT
);

CREATE INDEX idx_materialization_log_us_east_feature ON feature_materialization_log_us_east(feature_id);
CREATE INDEX idx_materialization_log_us_east_timestamp ON feature_materialization_log_us_east(started_at DESC);

-- Audit Log: Track region-scoped changes
CREATE TABLE IF NOT EXISTS audit_log_us_east (
    audit_id SERIAL PRIMARY KEY,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    user_id TEXT,
    changes JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_audit_us_east_entity ON audit_log_us_east(entity_type, entity_id);
CREATE INDEX idx_audit_us_east_user ON audit_log_us_east(user_id);

-- ============================================================================
-- PART 3: FEDERATION VIEWS (Read-Only, Global Queries)
-- These unify all regions into single global queryable datasets
-- ============================================================================

-- Global Drift Metrics View
CREATE OR REPLACE VIEW global_feature_drift AS
SELECT *, 'us-east' AS region FROM feature_drift_metrics_us_east
UNION ALL
SELECT *, 'eu-west' AS region FROM feature_drift_metrics_eu_west
UNION ALL
SELECT *, 'apac' AS region FROM feature_drift_metrics_apac;

-- Global Feature Importance View
CREATE OR REPLACE VIEW global_feature_importance AS
SELECT *, 'us-east' AS region FROM feature_importance_us_east
UNION ALL
SELECT *, 'eu-west' AS region FROM feature_importance_eu_west
UNION ALL
SELECT *, 'apac' AS region FROM feature_importance_apac;

-- Global Time-Series Features View
CREATE OR REPLACE VIEW global_ts_features AS
SELECT *, 'us-east' AS region FROM feature_ts_features_us_east
UNION ALL
SELECT *, 'eu-west' AS region FROM feature_ts_features_eu_west
UNION ALL
SELECT *, 'apac' AS region FROM feature_ts_features_apac;

-- Global Discovery Candidates View
CREATE OR REPLACE VIEW global_feature_discovery AS
SELECT *, 'us-east' AS region FROM feature_discovery_us_east
UNION ALL
SELECT *, 'eu-west' AS region FROM feature_discovery_eu_west
UNION ALL
SELECT *, 'apac' AS region FROM feature_discovery_apac;

-- Global Feature Freshness View (Last materialization by region)
CREATE OR REPLACE VIEW global_feature_freshness AS
SELECT 
    f.feature_id,
    f.feature_name,
    r.region_code,
    fs.last_materialized,
    EXTRACT(EPOCH FROM (now() - fs.last_materialized))/3600.0 AS hours_since_materialized,
    fs.status,
    CASE WHEN fs.last_materialized < now() - INTERVAL '24 hours' THEN 'stale'
         WHEN fs.last_materialized < now() - INTERVAL '6 hours' THEN 'aging'
         ELSE 'fresh' END AS freshness_status
FROM global_feature_catalog f
CROSS JOIN region_registry r
LEFT JOIN global_feature_status fs ON f.feature_id = fs.feature_id AND r.region_code = fs.region_code
WHERE r.is_active = TRUE;

-- Global Feature Health Dashboard View
CREATE OR REPLACE VIEW global_feature_health_dashboard AS
SELECT 
    f.feature_id,
    f.feature_name,
    COUNT(DISTINCT r.region_code) AS total_regions,
    SUM(CASE WHEN fs.status = 'healthy' THEN 1 ELSE 0 END) AS healthy_regions,
    ROUND(100.0 * SUM(CASE WHEN fs.status = 'healthy' THEN 1 ELSE 0 END) / NULLIF(COUNT(DISTINCT r.region_code), 0), 1) AS health_percentage,
    MAX(fs.last_materialized) AS most_recent_materialization,
    AVG(fs.last_drift_score) AS avg_drift_score,
    MAX(fs.last_drift_score) AS max_drift_score,
    AVG(fs.last_importance_score) AS avg_importance_score
FROM global_feature_catalog f
CROSS JOIN region_registry r
LEFT JOIN global_feature_status fs ON f.feature_id = fs.feature_id AND r.region_code = fs.region_code
WHERE f.is_active = TRUE AND r.is_active = TRUE
GROUP BY f.feature_id, f.feature_name;

-- ============================================================================
-- PART 4: STORED PROCEDURES FOR REGION MANAGEMENT
-- ============================================================================

-- Register a new region
CREATE OR REPLACE FUNCTION register_region(
    p_region_code TEXT,
    p_region_name TEXT,
    p_temporal_namespace TEXT,
    p_temporal_address TEXT,
    p_trino_endpoint TEXT,
    p_api_endpoint TEXT
) RETURNS BOOLEAN AS $$
BEGIN
    INSERT INTO region_registry(
        region_code, region_name, temporal_namespace, temporal_address,
        trino_catalog, trino_endpoint, iceberg_catalog, iceberg_s3_bucket,
        iceberg_warehouse_path, api_endpoint
    ) VALUES(
        p_region_code, p_region_name, p_temporal_namespace, p_temporal_address,
        'iceberg_' || p_region_code, p_trino_endpoint, 'iceberg_' || p_region_code,
        'semlayer-' || p_region_code, 's3://semlayer-' || p_region_code || '/warehouse'
    );
    RETURN TRUE;
EXCEPTION WHEN UNIQUE_VIOLATION THEN
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- Get all active regions
CREATE OR REPLACE FUNCTION get_active_regions()
RETURNS TABLE(region_code TEXT, temporal_namespace TEXT, api_endpoint TEXT) AS $$
BEGIN
    RETURN QUERY
    SELECT r.region_code, r.temporal_namespace, r.api_endpoint
    FROM region_registry r
    WHERE r.is_active = TRUE
    ORDER BY r.region_code;
END;
$$ LANGUAGE plpgsql;

-- Update feature status for a region
CREATE OR REPLACE FUNCTION update_feature_status(
    p_feature_id TEXT,
    p_region_code TEXT,
    p_status TEXT,
    p_drift_score DOUBLE PRECISION,
    p_importance_score DOUBLE PRECISION
) RETURNS VOID AS $$
BEGIN
    INSERT INTO global_feature_status(feature_id, region_code, last_drift_score, last_importance_score, status)
    VALUES(p_feature_id, p_region_code, p_drift_score, p_importance_score, p_status)
    ON CONFLICT (feature_id, region_code)
    DO UPDATE SET
        status = p_status,
        last_drift_score = p_drift_score,
        last_drift_timestamp = now(),
        last_importance_score = p_importance_score,
        last_importance_timestamp = now();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PART 5: INITIALIZATION DATA
-- ============================================================================

-- Insert primary regions
INSERT INTO region_registry(
    region_code, region_name, display_name, temporal_namespace,
    temporal_address, trino_catalog, trino_endpoint, iceberg_catalog,
    iceberg_s3_bucket, iceberg_warehouse_path, api_endpoint, is_active
) VALUES
('us-east', 'US East', 'US East (Virginia)', 'us-east-namespace',
 'temporal-us-east.semlayer.internal:7233', 'iceberg_us_east',
 'https://trino-us-east.semlayer.internal:8443', 'iceberg_us_east',
 'semlayer-us-east', 's3://semlayer-us-east/warehouse',
 'https://api-us-east.semlayer.internal', TRUE),

('eu-west', 'EU West', 'EU West (Ireland)', 'eu-west-namespace',
 'temporal-eu-west.semlayer.internal:7233', 'iceberg_eu_west',
 'https://trino-eu-west.semlayer.internal:8443', 'iceberg_eu_west',
 'semlayer-eu-west', 's3://semlayer-eu-west/warehouse',
 'https://api-eu-west.semlayer.internal', TRUE),

('apac', 'APAC', 'Asia Pacific (Singapore)', 'apac-namespace',
 'temporal-apac.semlayer.internal:7233', 'iceberg_apac',
 'https://trino-apac.semlayer.internal:8443', 'iceberg_apac',
 'semlayer-apac', 's3://semlayer-apac/warehouse',
 'https://api-apac.semlayer.internal', TRUE)
ON CONFLICT DO NOTHING;

-- Insert global SLO targets
INSERT INTO global_slo_config(metric_name, target_value, threshold_unit, is_global, alerting_enabled) VALUES
('materialization_latency_ms', 5000, 'latency_ms', TRUE, TRUE),
('drift_detection_latency_ms', 3000, 'latency_ms', TRUE, TRUE),
('discovery_pipeline_latency_ms', 10000, 'latency_ms', TRUE, TRUE),
('api_response_latency_p99', 2000, 'latency_ms', TRUE, TRUE),
('feature_health_percentage', 99.0, 'percentage', TRUE, TRUE),
('region_availability', 99.5, 'percentage', FALSE, TRUE)
ON CONFLICT DO NOTHING;

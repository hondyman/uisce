-- =============================================================================
-- Phase 3.15: Materialized Analytics Tables for Dashboard Performance
-- =============================================================================
-- Run these CREATE TABLE statements in Trino to establish the analytics tables
-- used for SLA trending, health reporting, and predictive scoring.
-- Assume these tables already exist: ops_events, chain_config, audit_log
-- =============================================================================

-- ============================================================================
-- Table 1: Hourly Chain Rollup
-- Aggregates metrics at hourly granularity for trend analysis
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.hourly_chain_rollup (
    tenant_id VARCHAR,
    chain_id VARCHAR,
    region VARCHAR,
    window_hour TIMESTAMP,
    success_count BIGINT DEFAULT 0,
    failure_count BIGINT DEFAULT 0,
    avg_latency_ms DOUBLE,
    p95_latency_ms DOUBLE,
    p99_latency_ms DOUBLE,
    incident_count BIGINT DEFAULT 0,
    computed_at TIMESTAMP DEFAULT current_timestamp
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['region', 'year(window_hour)', 'month(window_hour)', 'day(window_hour)', 'hour(window_hour)'],
    write_compression = 'snappy'
);

-- Create indexes for common dashboard queries
CREATE TABLE IF NOT EXISTS iceberg.ops.hourly_chain_rollup_idx
WITH (
    format = 'PARQUET'
) AS
SELECT 
    tenant_id, chain_id, region, window_hour,
    success_count, failure_count, avg_latency_ms,
    p95_latency_ms, p99_latency_ms, incident_count, computed_at
FROM iceberg.ops.hourly_chain_rollup;

-- ============================================================================
-- Table 2: Daily SLA Compliance Rollup
-- Aggregates full-day SLA metrics; used for trend visualization
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.daily_chain_sla (
    tenant_id VARCHAR,
    chain_id VARCHAR,
    region VARCHAR,
    day DATE,
    success_rate_pct DOUBLE,
    avg_latency_ms DOUBLE,
    p95_latency_ms DOUBLE,
    p99_latency_ms DOUBLE,
    incident_count BIGINT DEFAULT 0,
    sla_met BOOLEAN,
    computed_at TIMESTAMP DEFAULT current_timestamp
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['region', 'year(day)', 'month(day)', 'day(day)'],
    write_compression = 'snappy'
);

-- ============================================================================
-- Table 3: Chain Health Report (Daily snapshot)
-- Stores computed health scores, recommendations, and action status
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.chain_health_report (
    id VARCHAR,
    chain_id VARCHAR,
    tenant_id VARCHAR,
    region VARCHAR,
    overall_health INTEGER, -- 0-100
    last_execution_status VARCHAR,
    consecutive_failures INTEGER DEFAULT 0,
    is_healthy BOOLEAN,
    recommended_action VARCHAR, -- 'investigate', 'retry', 'disable', 'none'
    action_executed BOOLEAN DEFAULT FALSE,
    reported_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT current_timestamp
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['region', 'year(reported_at)', 'month(reported_at)', 'day(reported_at)'],
    write_compression = 'snappy'
);

-- ============================================================================
-- Table 4: Chain Predictions (ML batch scoring results)
-- Stores failure probability and related metadata for predictive features
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.chain_predictions (
    id VARCHAR,
    chain_id VARCHAR,
    tenant_id VARCHAR,
    region VARCHAR,
    prediction_ts TIMESTAMP,
    failure_prob DOUBLE, -- 0.0-1.0
    recommended_action VARCHAR,
    model_version VARCHAR,
    top_features VARCHAR, -- JSON: [{name: "feature", importance: 0.23}, ...]
    created_at TIMESTAMP DEFAULT current_timestamp
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['region', 'year(prediction_ts)', 'month(prediction_ts)', 'day(prediction_ts)'],
    write_compression = 'snappy'
);

-- ============================================================================
-- Table 5: Chain Features (for ML training)
-- Feature store table with lagged metrics used in model training
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.chain_features (
    feature_id VARCHAR,
    chain_id VARCHAR,
    tenant_id VARCHAR,
    region VARCHAR,
    feature_date DATE,
    -- Historical lag features
    success_rate_1h DOUBLE,
    success_rate_6h DOUBLE,
    success_rate_24h DOUBLE,
    success_rate_7d DOUBLE,
    avg_latency_1h DOUBLE,
    avg_latency_6h DOUBLE,
    -- Incident metrics
    incident_count_24h INTEGER,
    critical_incident_count_24h INTEGER,
    -- Configuration changes
    config_changed_24h BOOLEAN DEFAULT FALSE,
    -- External signals
    region_healthy BOOLEAN DEFAULT TRUE,
    -- Label: next 24h failure
    label_failure_24h BOOLEAN,
    created_at TIMESTAMP DEFAULT current_timestamp
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['region', 'year(feature_date)', 'month(feature_date)', 'day(feature_date)'],
    write_compression = 'snappy'
);

-- ============================================================================
-- Table 6: Model Registry
-- Tracks trained models, performance metrics, and deployment status
-- ============================================================================
CREATE TABLE IF NOT EXISTS iceberg.ops.model_registry (
    model_id VARCHAR,
    model_name VARCHAR,
    model_version INTEGER,
    model_type VARCHAR, -- 'xgboost', 'lightgbm', 'logistic_regression'
    created_at TIMESTAMP,
    training_date DATE,
    -- Performance metrics
    auc_roc DOUBLE,
    precision_at_90 DOUBLE,
    recall_at_90 DOUBLE,
    f1_score DOUBLE,
    -- Deployment status
    deployed BOOLEAN DEFAULT FALSE,
    deployment_date TIMESTAMP,
    -- Location
    model_artifact_s3_path VARCHAR,
    -- Drift monitoring
    is_active BOOLEAN DEFAULT TRUE
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['year(created_at)', 'month(created_at)'],
    write_compression = 'snappy'
);

-- ============================================================================
-- Materialized Views for Dashboard Queries
-- ============================================================================

-- View: Daily chain SLA trends (last 30 days)
CREATE OR REPLACE VIEW iceberg.ops.sla_trend_30d AS
SELECT 
    tenant_id,
    chain_id,
    region,
    day,
    success_rate_pct,
    LAG(success_rate_pct) OVER (PARTITION BY chain_id ORDER BY day) AS prev_success_rate_pct,
    success_rate_pct - LAG(success_rate_pct) OVER (PARTITION BY chain_id ORDER BY day) AS success_rate_change_pct,
    CASE 
        WHEN success_rate_pct >= 99.0 THEN 'excellent'
        WHEN success_rate_pct >= 95.0 THEN 'good'
        WHEN success_rate_pct >= 90.0 THEN 'acceptable'
        ELSE 'critical'
    END AS sla_status,
    computed_at
FROM iceberg.ops.daily_chain_sla
WHERE day >= current_date - INTERVAL '30' DAY;

-- View: Health score distribution (all active chains)
CREATE OR REPLACE VIEW iceberg.ops.health_distribution AS
SELECT 
    region,
    CASE 
        WHEN overall_health >= 90 THEN 'Excellent'
        WHEN overall_health >= 70 THEN 'Good'
        WHEN overall_health >= 50 THEN 'Fair'
        ELSE 'Poor'
    END AS health_category,
    COUNT(*) AS chain_count,
    COUNT(*) FILTER (WHERE is_healthy = true) AS healthy_chains,
    100.0 * COUNT(*) FILTER (WHERE is_healthy = true) / COUNT(*) AS healthy_pct
FROM iceberg.ops.chain_health_report
WHERE reported_at >= current_timestamp - INTERVAL '1' HOUR
GROUP BY region;

-- View: Recommended actions pending (actionable insights)
CREATE OR REPLACE VIEW iceberg.ops.actions_pending AS
SELECT 
    chain_id,
    tenant_id,
    region,
    recommended_action,
    overall_health,
    COUNT(*) AS recommendation_count,
    MAX(reported_at) AS latest_recommendation
FROM iceberg.ops.chain_health_report
WHERE action_executed = FALSE 
  AND recommended_action != 'none'
  AND reported_at >= current_timestamp - INTERVAL '24' HOUR
GROUP BY chain_id, tenant_id, region, recommended_action, overall_health;

-- View: Failure predictions (high risk chains in next 24h)
CREATE OR REPLACE VIEW iceberg.ops.high_risk_predictions AS
SELECT 
    chain_id,
    tenant_id,
    region,
    failure_prob,
    model_version,
    top_features,
    prediction_ts,
    CASE 
        WHEN failure_prob >= 0.9 THEN 'Critical'
        WHEN failure_prob >= 0.7 THEN 'High'
        WHEN failure_prob >= 0.5 THEN 'Medium'
        ELSE 'Low'
    END AS risk_level
FROM iceberg.ops.chain_predictions
WHERE prediction_ts >= current_timestamp - INTERVAL '1' DAY
  AND failure_prob >= 0.5
ORDER BY failure_prob DESC, prediction_ts DESC;

-- ============================================================================
-- Refresh Procedures (call these from Temporal workflows)
-- ============================================================================

CREATE OR REPLACE PROCEDURE iceberg.ops.refresh_hourly_rollup()
LANGUAGE SQL
PARAMETER STYLE SQL
DETERMINISTIC
READS SQL DATA
BEGIN
  -- Insert new hourly rollup data from raw events
  INSERT INTO iceberg.ops.hourly_chain_rollup
  SELECT
    tenant_id,
    chain_id,
    region,
    date_trunc('hour', event_timestamp) AS window_hour,
    COUNT(*) FILTER (WHERE status = 'success') AS success_count,
    COUNT(*) FILTER (WHERE status != 'success') AS failure_count,
    AVG(latency_ms) AS avg_latency_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) AS p95_latency_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) AS p99_latency_ms,
    COUNT(DISTINCT incident_id) AS incident_count,
    current_timestamp AS computed_at
  FROM iceberg.ops.ops_events
  WHERE event_timestamp >= date_trunc('hour', current_timestamp) - INTERVAL '1' HOUR
    AND event_timestamp < date_trunc('hour', current_timestamp)
  GROUP BY tenant_id, chain_id, region, date_trunc('hour', event_timestamp);
END;

CREATE OR REPLACE PROCEDURE iceberg.ops.refresh_daily_sla(date_str VARCHAR)
LANGUAGE SQL
PARAMETER STYLE SQL
DETERMINISTIC
READS SQL DATA
BEGIN
  -- Insert daily SLA metrics from hourly rollup
  INSERT INTO iceberg.ops.daily_chain_sla
  SELECT
    tenant_id,
    chain_id,
    region,
    CAST(date_str AS DATE) AS day,
    100.0 * SUM(success_count) / (SUM(success_count) + SUM(failure_count)) AS success_rate_pct,
    AVG(avg_latency_ms) AS avg_latency_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY p95_latency_ms) AS p95_latency_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY p99_latency_ms) AS p99_latency_ms,
    SUM(incident_count) AS incident_count,
    100.0 * SUM(success_count) / (SUM(success_count) + SUM(failure_count)) >= 99.0 AS sla_met,
    current_timestamp AS computed_at
  FROM iceberg.ops.hourly_chain_rollup
  WHERE DATE(window_hour) = CAST(date_str AS DATE)
  GROUP BY tenant_id, chain_id, region;
END;

-- ============================================================================
-- Indexes for common query patterns
-- ============================================================================

-- Index on hourly rollup for typical dashboard queries
CREATE INDEX IF NOT EXISTS idx_hourly_rollup_tenant_time 
ON iceberg.ops.hourly_chain_rollup (tenant_id, window_hour DESC);

CREATE INDEX IF NOT EXISTS idx_hourly_rollup_chain_time 
ON iceberg.ops.hourly_chain_rollup (chain_id, window_hour DESC);

-- Index on daily SLA for trending queries
CREATE INDEX IF NOT EXISTS idx_daily_sla_tenant_day 
ON iceberg.ops.daily_chain_sla (tenant_id, day DESC);

-- Index on health reports for active chains
CREATE INDEX IF NOT EXISTS idx_health_report_active 
ON iceberg.ops.chain_health_report (tenant_id, reported_at DESC) 
WHERE is_healthy = FALSE;

-- Index on predictions for risk assessment
CREATE INDEX IF NOT EXISTS idx_predictions_risk 
ON iceberg.ops.chain_predictions (tenant_id, failure_prob DESC, prediction_ts DESC);

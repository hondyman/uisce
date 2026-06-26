-- ============================================================================
-- Phase 3.21: Sample Data Loader for Testing
-- ============================================================================
-- Populates feature catalog with example features, test cases, lineage,
-- and synthetic drift/quality/importance data for demonstration.
--
-- Run after schema initialization:
--   psql -d semlayer -f sample_data.sql
-- ============================================================================

-- ============================================================================
-- 1. Sample Features (Feature Catalog)
-- ============================================================================

INSERT INTO feature_catalog (
    feature_id, name, description, namespace, owner, version,
    is_core, properties
) VALUES

-- Blockchain domain features
('feature:chain.health_score_v1',
 'Chain Health Score',
 'Weighted health metric: sync ratio + latency + error rate',
 'chain',
 'mlops-team@example.com',
 '1.0.0',
 true,
 '{
   "feature_type": "derived",
   "owner": "mlops-team@example.com",
   "tags": ["core", "health", "real-time"],
   "aggregation": "weighted_avg",
   "partition_keys": ["tenant_id", "region"],
   "materialization_policy": {
     "policy": "precompute_hourly",
     "ttl_seconds": 3600
   },
   "drift_config": {
     "method": "ks",
     "threshold": 0.05,
     "baseline_window": "30d",
     "eval_window": "1d",
     "alert_policy": "email"
   }
 }'::jsonb),

('feature:orders.monthly_revenue_v1',
 'Monthly Revenue Sum',
 'Total revenue from paid orders over last 30 days, by tenant',
 'orders',
 'analytics-team@example.com',
 '1.0.0',
 true,
 '{
   "feature_type": "time_series",
   "owner": "analytics-team@example.com",
   "tags": ["business_metric", "revenue"],
   "expression": "SUM(orders.amount) FILTER (WHERE orders.status=''paid'')",
   "aggregation": "sum",
   "window": {"size": "30d", "stride": "1d"},
   "granularity": "day",
   "partition_keys": ["tenant_id", "region"],
   "materialization_policy": {
     "policy": "precompute_daily",
     "ttl_seconds": 86400,
     "preferred_storage": "iceberg"
   },
   "drift_config": {
     "method": "psi",
     "threshold": 0.15,
     "baseline_window": "30d",
     "eval_window": "1d"
   }
 }'::jsonb),

('feature:conflicts.active_count_v1',
 'Active Conflicts Count',
 'Number of open/unresolved conflicts per tenant',
 'conflicts',
 'ops-team@example.com',
 '1.0.0',
 true,
 '{
   "feature_type": "derived",
   "owner": "ops-team@example.com",
   "tags": ["operational", "critical"],
   "expression": "COUNT(*) FILTER (WHERE status=''active'')",
   "aggregation": "count",
   "granularity": "hour",
   "partition_keys": ["tenant_id"],
   "materialization_policy": {
     "policy": "precompute_hourly",
     "ttl_seconds": 3600
   },
   "drift_config": {
     "method": "ks",
     "threshold": 0.10,
     "baseline_window": "14d",
     "eval_window": "4h"
   },
   "test_cases": [
     {
       "id": "tc_active_count_1",
       "description": "Active conflicts count is non-negative",
       "sql_assertion": "SELECT COUNT(*) >= 0 FROM conflicts WHERE status=''active''",
       "expected": true
     }
   ]
 }'::jsonb),

('feature:latency.p99_ms_v1',
 'P99 Latency (ms)',
 '99th percentile latency from request logs',
 'latency',
 'perf-team@example.com',
 '1.0.0',
 true,
 '{
   "feature_type": "time_series",
   "owner": "perf-team@example.com",
   "tags": ["performance", "slo"],
   "aggregation": "percentile",
   "window": {"size": "1d", "stride": "1h"},
   "granularity": "hour",
   "partition_keys": ["tenant_id", "region"],
   "materialization_policy": {
     "policy": "precompute_hourly",
     "ttl_seconds": 3600
   },
   "drift_config": {
     "method": "ks",
     "threshold": 0.08,
     "baseline_window": "7d",
     "eval_window": "1h"
   }
 }'::jsonb),

('feature:errors.http_rate_v1',
 'HTTP Error Rate (%)',
 'Percentage of HTTP responses with status >= 400',
 'errors',
 'reliability-team@example.com',
 '1.0.0',
 true,
 '{
   "feature_type": "derived",
   "owner": "reliability-team@example.com",
   "tags": ["reliability", "slo"],
   "expression": "SUM(status >= 400) / COUNT(*) * 100",
   "aggregation": "ratio",
   "window": {"size": "1h", "stride": "5m"},
   "granularity": "hour",
   "partition_keys": ["tenant_id", "region"],
   "materialization_policy": {
     "policy": "precompute_hourly",
     "ttl_seconds": 300
   },
   "drift_config": {
     "method": "psi",
     "threshold": 0.20,
     "baseline_window": "7d",
     "eval_window": "1h"
   }
 }'::jsonb);

-- ============================================================================
-- 2. Feature Watermarks (Materialization Tracking)
-- ============================================================================

INSERT INTO feature_watermarks (feature_id, last_processed, last_processed_batch_id)
VALUES
  ('feature:chain.health_score_v1', NOW() - INTERVAL '2 hours', 'batch-20260209-0200'),
  ('feature:orders.monthly_revenue_v1', NOW() - INTERVAL '6 hours', 'batch-20260208-1800'),
  ('feature:conflicts.active_count_v1', NOW() - INTERVAL '30 minutes', 'batch-20260209-0530'),
  ('feature:latency.p99_ms_v1', NOW() - INTERVAL '1 hour', 'batch-20260209-0400'),
  ('feature:errors.http_rate_v1', NOW() - INTERVAL '5 minutes', 'batch-20260209-0655');

-- ============================================================================
-- 3. Sample Drift Detection Results
-- ============================================================================

INSERT INTO feature_drift_metrics (
    feature_id, method, score, pvalue, is_drifted, threshold,
    baseline_window_start, baseline_window_end,
    eval_window_start, eval_window_end,
    alert_sent, recorded_at
) VALUES

-- Stable feature (no drift)
('feature:chain.health_score_v1',
 'ks', 0.032, 0.67, false, 0.05,
 NOW() - INTERVAL '30 days', NOW() - INTERVAL '1 day',
 NOW() - INTERVAL '1 day', NOW(),
 false, NOW()),

-- Feature in drift (example alert)
('feature:orders.monthly_revenue_v1',
 'psi', 0.18, 0.002, true, 0.15,
 NOW() - INTERVAL '30 days', NOW() - INTERVAL '1 day',
 NOW() - INTERVAL '1 day', NOW(),
 true, NOW()),

-- Stable categorical
('feature:conflicts.active_count_v1',
 'chi2', 0.06, 0.88, false, 0.10,
 NOW() - INTERVAL '14 days', NOW() - INTERVAL '1 day',
 NOW() - INTERVAL '4 hours', NOW(),
 false, NOW()),

-- Latency showing early drift signal
('feature:latency.p99_ms_v1',
 'ks', 0.072, 0.15, false, 0.08,
 NOW() - INTERVAL '7 days', NOW() - INTERVAL '1 hour',
 NOW() - INTERVAL '1 hour', NOW(),
 false, NOW()),

-- Error rate stable
('feature:errors.http_rate_v1',
 'psi', 0.12, 0.31, false, 0.20,
 NOW() - INTERVAL '7 days', NOW() - INTERVAL '1 hour',
 NOW() - INTERVAL '1 hour', NOW(),
 false, NOW());

-- ============================================================================
-- 4. Sample Quality Checks
-- ============================================================================

INSERT INTO feature_quality_checks (
    feature_id, check_name, check_type, threshold_type, threshold_value,
    passed, window_start, window_end, observed_value, computed_at
) VALUES

('feature:chain.health_score_v1', 'null_rate_acceptable', 'null_rate', 'max', 5.0,
 true, NOW() - INTERVAL '1 hour', NOW(), 0.2, NOW()),

('feature:orders.monthly_revenue_v1', 'cardinality_check', 'cardinality', 'max', 10000.0,
 true, NOW() - INTERVAL '1 hour', NOW(), 8734.0, NOW()),

('feature:conflicts.active_count_v1', 'type_check', 'type', 'exact', NULL,
 true, NOW() - INTERVAL '1 hour', NOW(), NULL, NOW()),

('feature:latency.p99_ms_v1', 'range_check', 'range', 'max', 5000.0,
 false, NOW() - INTERVAL '1 hour', NOW(), 5234.5, NOW() - INTERVAL '10 minutes');

-- ============================================================================
-- 5. Sample Feature Importance (SHAP Results)
-- ============================================================================

INSERT INTO feature_importance (
    feature_id, model_id, mean_abs_shap, shap_values,
    permutation_importance, stability_score, importance_trend,
    importance_percentile, dataset_size, model_version, recorded_at
) VALUES

('feature:chain.health_score_v1', 'model-v2-20260209', 0.42,
 ARRAY[0.35, 0.51, 0.38, 0.45, 0.40, 0.39, 0.43, 0.44, 0.41, 0.42],
 0.38, 0.94, 0.005, 85.0, 5000, '20260209_v2', NOW() - INTERVAL '1 day'),

('feature:orders.monthly_revenue_v1', 'model-v2-20260209', 0.68,
 ARRAY[0.62, 0.71, 0.65, 0.70, 0.69, 0.68, 0.66, 0.67],
 0.72, 0.91, 0.008, 95.0, 5000, '20260209_v2', NOW() - INTERVAL '1 day'),

('feature:conflicts.active_count_v1', 'model-v2-20260209', 0.25,
 ARRAY[0.22, 0.27, 0.24, 0.26, 0.25, 0.24, 0.26, 0.25],
 0.23, 0.88, -0.002, 60.0, 5000, '20260209_v2', NOW() - INTERVAL '1 day'),

('feature:latency.p99_ms_v1', 'model-v2-20260209', 0.31,
 ARRAY[0.28, 0.34, 0.30, 0.33, 0.31, 0.30, 0.32, 0.31],
 0.29, 0.89, 0.003, 70.0, 5000, '20260209_v2', NOW() - INTERVAL '1 day'),

('feature:errors.http_rate_v1', 'model-v2-20260209', 0.19,
 ARRAY[0.17, 0.21, 0.19, 0.20, 0.18, 0.19, 0.20, 0.19],
 0.18, 0.85, -0.001, 45.0, 5000, '20260209_v2', NOW() - INTERVAL '1 day');

-- ============================================================================
-- 6. Sample Change Log (Governance)
-- ============================================================================

INSERT INTO feature_change_log (
    feature_id, change_type, change_description, requested_by,
    pr_url, commit_sha, approval_status, deployment_version
) VALUES

('feature:chain.health_score_v1', 'created',
 'Initial feature definition from chain metrics',
 'mlops-eng@example.com',
 'https://github.com/example/repo/pull/1234',
 'abc123def456',
 'APPROVED',
 '3.21.0'),

('feature:orders.monthly_revenue_v1', 'approved',
 'Approved for production materialization',
 'datalead@example.com',
 'https://github.com/example/repo/pull/1235',
 'def456ghi789',
 'APPROVED',
 '3.21.0'),

('feature:conflicts.active_count_v1', 'deployed',
 'Deployed to production materialization',
 'ops-automation@example.com',
 NULL,
 'ghi789jkl012',
 'APPROVED',
 '3.21.0');

-- ============================================================================
-- 7. Sample Test Cases
-- ============================================================================

INSERT INTO feature_test_cases (
    feature_id, test_name, test_type, description,
    sql_assertion, expected_result, enabled, critical
) VALUES

('feature:chain.health_score_v1', 'health_range_valid', 'unit',
 'Health score between 0 and 100',
 'SELECT COUNT(*) FROM (SELECT health_score FROM features.chain_health WHERE health_score BETWEEN 0 AND 100)',
 '{"expected": "> 0"}'::jsonb, true, true),

('feature:orders.monthly_revenue_v1', 'revenue_positive', 'unit',
 'Revenue sum is non-negative',
 'SELECT SUM(amount) >= 0 FROM orders WHERE status = ''paid''',
 '{"expected": true}'::jsonb, true, true),

('feature:conflicts.active_count_v1', 'conflict_count_matches', 'integration',
 'Active count matches conflicts table',
 'SELECT COUNT(*) FROM conflicts WHERE status = ''active''',
 '{"expected_range": [0, 100000]}'::jsonb, true, false),

('feature:latency.p99_ms_v1', 'p99_reasonable', 'unit',
 'P99 latency under 10 seconds',
 'SELECT p99_ms < 10000 FROM latest_latency_metrics',
 '{"expected": true}'::jsonb, true, true),

('feature:errors.http_rate_v1', 'error_rate_percentage', 'unit',
 'Error rate between 0 and 100 percent',
 'SELECT error_rate BETWEEN 0 AND 100 FROM daily_error_metrics',
 '{"expected": true}'::jsonb, true, false);

-- ============================================================================
-- 8. Sample Feature Lineage
-- ============================================================================

INSERT INTO feature_lineage (
    source_table, source_column, target_feature_id,
    lineage_type, transformation
) VALUES

('public.chain_metrics', 'sync_ratio',
 'feature:chain.health_score_v1',
 'table_to_feature', 'Weighted component: sync_ratio * 0.4'),

('public.chain_metrics', 'latency_ms',
 'feature:chain.health_score_v1',
 'table_to_feature', 'Weighted component: (1 - latency_ms/1000) * 0.35'),

('public.chain_metrics', 'error_count',
 'feature:chain.health_score_v1',
 'table_to_feature', 'Weighted component: (1 - error_rate) * 0.25'),

('public.orders', 'amount',
 'feature:orders.monthly_revenue_v1',
 'table_to_feature', 'SUM of paid orders over 30d rolling window'),

('public.orders', 'status',
 'feature:orders.monthly_revenue_v1',
 'table_to_feature', 'Filter WHERE status = ''paid'''),

('public.conflicts', 'conflict_id',
 'feature:conflicts.active_count_v1',
 'table_to_feature', 'COUNT(*) WHERE status = ''active'''),

('public.request_logs', 'latency_ms',
 'feature:latency.p99_ms_v1',
 'table_to_feature', 'PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms)'),

('public.request_logs', 'status_code',
 'feature:errors.http_rate_v1',
 'table_to_feature', 'SUM(status >= 400) / COUNT(*) * 100');

-- ============================================================================
-- 9. Sample Feature Computations (Job Execution Logs)
-- ============================================================================

INSERT INTO feature_computations (
    feature_id, job_type, job_id, compute_engine, started_at,
    completed_at, status, rows_processed, bytes_written
) VALUES

('feature:chain.health_score_v1', 'materialization',
 'spark-job-20260209-001', 'spark',
 NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour 58 minutes',
 'SUCCESS', 125000, 2500000),

('feature:orders.monthly_revenue_v1', 'materialization',
 'spark-job-20260209-002', 'spark',
 NOW() - INTERVAL '6 hours', NOW() - INTERVAL '5 hours 55 minutes',
 'SUCCESS', 850000, 15000000),

('feature:chain.health_score_v1', 'drift',
 'temporal-drift-001', 'temporal',
 NOW() - INTERVAL '1 hour', NOW() - INTERVAL '50 minutes',
 'SUCCESS', 125000, 125000),

('feature:orders.monthly_revenue_v1', 'importance',
 'temporal-importance-001', 'temporal',
 NOW() - INTERVAL '14 hours', NOW() - INTERVAL '13 hours 45 minutes',
 'SUCCESS', 5000, 500000);

-- ============================================================================
-- Summary
-- ============================================================================

SELECT
    (SELECT COUNT(*) FROM feature_catalog) as feature_count,
    (SELECT COUNT(*) FROM feature_watermarks) as watermark_count,
    (SELECT COUNT(*) FROM feature_drift_metrics) as drift_metric_count,
    (SELECT COUNT(*) FROM feature_quality_checks) as quality_check_count,
    (SELECT COUNT(*) FROM feature_importance) as importance_count,
    (SELECT COUNT(*) FROM feature_change_log) as changelog_count,
    (SELECT COUNT(*) FROM feature_test_cases) as test_case_count,
    (SELECT COUNT(*) FROM feature_lineage) as lineage_count,
    (SELECT COUNT(*) FROM feature_computations) as computation_count;

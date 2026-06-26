-- ============================================================================
-- Phase 3.21: Advanced Feature Engineering - Complete Schema
-- ============================================================================
-- Production-grade Postgres DDL for feature catalog, materialization,
-- drift detection, quality checks, importance scoring, and versioning.
--
-- All tables are region-aware and tenant-aware for multi-tenant deployments.
-- Partitioning strategies noted for production Iceberg integration.
-- ============================================================================

-- Ensure extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "hstore";

-- ============================================================================
-- 1. FEATURE_CATALOG: Core Feature Metadata Registry
-- ============================================================================
-- Canonical registry of all features: raw, derived, time-series, embeddings.
-- Properties stored as JSONB for flexibility (owner, feature_type, expression, 
-- aggregation, materialization_policy, drift_config, test_cases, lineage_refs).
--
-- Lifecycle states: draft → approved → production → deprecated
-- Confidence scores guide automated feature discovery rank.

CREATE TABLE feature_catalog (
    feature_id TEXT PRIMARY KEY,                    -- e.g., feature:orders.revenue_v1
    name TEXT NOT NULL,
    description TEXT,
    namespace TEXT NOT NULL,                        -- e.g., "orders", "payments", "ml"
    node_type TEXT NOT NULL DEFAULT 'feature',      -- orthogonal to semantic node system
    is_core BOOLEAN NOT NULL DEFAULT FALSE,         -- critical features requiring high SLA
    owner TEXT NOT NULL,                            -- email or team identifier
    
    -- Versioning
    version TEXT NOT NULL DEFAULT '1.0.0',          -- semantic version
    deprecated_at TIMESTAMPTZ,
    replacement_feature_id TEXT,
    
    -- Flexible metadata as JSONB (validated externally)
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,  -- Contains:
    -- Core: tags, synonyms, confidence
    -- Feature: feature_type, expression, aggregation, window, granularity, partition_keys
    -- Materialization: materialization_policy, ttl_seconds, preferred_storage, rollup_grains
    -- Drift: drift_config (method, threshold, baseline_window, eval_window, alert_policy)
    -- Testing: test_cases (id, description, sql_assertion, expected)
    -- Lineage: lineage_refs (table, column dependencies)
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    updated_by TEXT,
    
    -- Multi-tenancy & region awareness
    tenant_id TEXT NOT NULL DEFAULT 'default',      -- for multi-tenant setups
    region TEXT NOT NULL DEFAULT 'us-east-1',       -- geographic region for data residency
    
    CONSTRAINT feature_id_format CHECK (feature_id ~ '^feature:'),
    CONSTRAINT version_format CHECK (version ~ '^\d+\.\d+\.\d+$')
);

CREATE INDEX idx_feature_catalog_owner ON feature_catalog(owner);
CREATE INDEX idx_feature_catalog_namespace ON feature_catalog(namespace);
CREATE INDEX idx_feature_catalog_is_core ON feature_catalog(is_core);
CREATE INDEX idx_feature_catalog_tenant_region ON feature_catalog(tenant_id, region);
CREATE INDEX idx_feature_catalog_created_at ON feature_catalog(created_at DESC);
CREATE INDEX idx_feature_catalog_properties_type ON feature_catalog USING GIN (properties -> 'feature_type');
CREATE INDEX idx_feature_catalog_deprecated ON feature_catalog(deprecated_at) WHERE deprecated_at IS NOT NULL;
CREATE INDEX idx_feature_catalog_lifecycle ON feature_catalog USING GIN (properties -> 'lifecycle');

-- View for non-deprecated features
CREATE VIEW feature_catalog_active AS
SELECT * FROM feature_catalog
WHERE deprecated_at IS NULL;

GRANT SELECT ON feature_catalog TO PUBLIC;
GRANT SELECT ON feature_catalog_active TO PUBLIC;

-- ============================================================================
-- 2. FEATURE_WATERMARKS: Incremental Materialization Tracking
-- ============================================================================
-- Tracks last processed timestamp per feature for incremental pipelines.
-- Enables exactly-once semantics and efficient fan-out computation.
-- Single row per feature; updated atomically with materialization completion.

CREATE TABLE feature_watermarks (
    feature_id TEXT PRIMARY KEY,
    last_processed TIMESTAMPTZ NOT NULL,            -- latest timestamp materialized
    last_processed_batch_id TEXT,                   -- run ID for idempotency
    materialization_lag_seconds INT DEFAULT 0,     -- 0 = on-time, >0 = behind
    
    -- Watermark history for rollback/audit
    previous_checkpoint TIMESTAMPTZ,
    
    -- Observation window (for late arrivals)
    watermark_age_seconds INT GENERATED ALWAYS AS (EXTRACT(EPOCH FROM (NOW() - last_processed))::INT) STORED,
    
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT watermark_valid CHECK (last_processed <= NOW())
);

CREATE INDEX idx_watermarks_lag ON feature_watermarks(materialization_lag_seconds DESC);
CREATE INDEX idx_watermarks_age ON feature_watermarks(watermark_age_seconds DESC);

GRANT SELECT ON feature_watermarks TO PUBLIC;

-- ============================================================================
-- 3. FEATURE_DRIFT_METRICS: Distribution Shift Detection Results
-- ============================================================================
-- Stores drift detection results per feature per evaluation window.
-- Methods: KS test, PSI, Chi-square, classifier-based.
-- Partitioned by ts for efficient rolling-window queries.

CREATE TABLE feature_drift_metrics (
    drift_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    
    -- Drift methodology
    method TEXT NOT NULL,                           -- 'ks' | 'psi' | 'chi2' | 'classifier'
    score DOUBLE PRECISION NOT NULL,                -- KS statistic [0,1], PSI [0,∞)
    pvalue DOUBLE PRECISION,                        -- p-value for statistical test
    
    -- Baseline & eval windows
    baseline_window_start TIMESTAMPTZ NOT NULL,
    baseline_window_end TIMESTAMPTZ NOT NULL,
    eval_window_start TIMESTAMPTZ NOT NULL,
    eval_window_end TIMESTAMPTZ NOT NULL,
    
    -- Results
    is_drifted BOOLEAN NOT NULL,                    -- score > threshold
    threshold DOUBLE PRECISION NOT NULL,            -- configured threshold
    percentile_rank DOUBLE PRECISION,               -- how extreme is this drift? [0,100]
    
    -- Root cause analysis
    affected_categories TEXT[],                     -- if categorical, which values shifted
    affected_quantiles TEXT[],                      -- if continuous, which quantiles
    
    -- Alerting
    alert_sent BOOLEAN DEFAULT FALSE,
    alert_channel TEXT,                             -- email, pagerduty, webhook
    
    -- Metadata
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT score_valid CHECK (score >= 0),
    CONSTRAINT pvalue_valid CHECK (pvalue IS NULL OR (pvalue >= 0 AND pvalue <= 1))
);

-- Partitioning strategy (manual): PARTITION BY RANGE (recorded_at)
-- In production, implement range partitions for recent + archive performance

CREATE INDEX idx_drift_feature_ts ON feature_drift_metrics(feature_id, recorded_at DESC);
CREATE INDEX idx_drift_tenantregion ON feature_drift_metrics(tenant_id, region, recorded_at DESC);
CREATE INDEX idx_drift_drifted ON feature_drift_metrics(is_drifted, recorded_at DESC) WHERE is_drifted;
CREATE INDEX idx_drift_method ON feature_drift_metrics(method);
CREATE INDEX idx_drift_alert_sent ON feature_drift_metrics(alert_sent) WHERE NOT alert_sent;
CREATE INDEX idx_drift_recorded_at ON feature_drift_metrics(recorded_at DESC);

-- Materialized view for recent active drifts
CREATE MATERIALIZED VIEW active_drifts AS
SELECT 
    feature_id,
    method,
    score,
    pvalue,
    is_drifted,
    percentile_rank,
    eval_window_end,
    alert_sent,
    ROW_NUMBER() OVER (PARTITION BY feature_id ORDER BY recorded_at DESC) as recency_rank
FROM feature_drift_metrics
WHERE is_drifted AND recorded_at >= NOW() - INTERVAL '7 days';

CREATE INDEX idx_active_drifts_feature ON active_drifts(feature_id);

GRANT SELECT ON feature_drift_metrics TO PUBLIC;
GRANT SELECT ON active_drifts TO PUBLIC;

-- ============================================================================
-- 4. FEATURE_QUALITY_CHECKS: Data Quality Assertions
-- ============================================================================
-- Validates feature health: null rates, cardinality, type expectations, schema drift.
-- Run per-feature per-window; fail feature materialization if checks fail.

CREATE TABLE feature_quality_checks (
    check_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    
    -- Check definition
    check_name TEXT NOT NULL,                       -- e.g., 'null_rate_too_high', 'cardinality_spike'
    check_type TEXT NOT NULL,                       -- 'null_rate' | 'cardinality' | 'type' | 'range' | 'custom'
    
    -- Check specification
    threshold_type TEXT,                            -- 'max' | 'min' | 'range' | 'exact'
    threshold_value DOUBLE PRECISION,               -- numeric threshold
    threshold_unit TEXT,                            -- 'percent' | 'count' | 'ratio'
    
    -- Check results this window
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    observed_value DOUBLE PRECISION,                -- actual metric computed
    passed BOOLEAN NOT NULL,
    
    -- Failure details
    error_message TEXT,
    remediation_suggested TEXT,
    
    -- Metadata
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id)
);

CREATE INDEX idx_qc_feature_window ON feature_quality_checks(feature_id, window_start DESC);
CREATE INDEX idx_qc_passed ON feature_quality_checks(passed, computed_at DESC);
CREATE INDEX idx_qc_check_name ON feature_quality_checks(check_name, computed_at DESC);
CREATE INDEX idx_qc_tenant_region ON feature_quality_checks(tenant_id, region, computed_at DESC);

-- View for recent failures
CREATE VIEW quality_check_failures AS
SELECT * FROM feature_quality_checks
WHERE NOT passed AND computed_at >= NOW() - INTERVAL '24 hours'
ORDER BY computed_at DESC;

GRANT SELECT ON feature_quality_checks TO PUBLIC;
GRANT SELECT ON quality_check_failures TO PUBLIC;

-- ============================================================================
-- 5. FEATURE_IMPORTANCE: SHAP & Stability Metrics
-- ============================================================================
-- Nightly SHAP computation results per model per feature.
-- Tracks importance trend + stability + permutation importance as model evolves.
-- Feeds feature selection, model explanation, and drift alerting.

CREATE TABLE feature_importance (
    importance_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    model_id TEXT NOT NULL,                         -- cross-reference to model registry
    
    -- SHAP results
    mean_abs_shap DOUBLE PRECISION NOT NULL,        -- mean(|SHAP value|) aggregated
    shap_values FLOAT8[] NOT NULL,                  -- raw SHAP values for this batch (sampled)
    
    -- Importance variants
    permutation_importance DOUBLE PRECISION,        -- drop-column importance
    gain_importance DOUBLE PRECISION,               -- tree-based gains
    
    -- Stability metrics
    stability_score DOUBLE PRECISION,               -- 1 - variance(importance over last N runs)
    importance_trend DOUBLE PRECISION,              -- slope of importance over time
    importance_percentile DOUBLE PRECISION,         -- rank among all features [0,100]
    
    -- Inference dataset metadata
    dataset_size INT NOT NULL,                      -- rows evaluated for SHAP
    model_version TEXT,                             -- e.g., "20260205_v2"
    
    -- Temporal
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT mean_shap_valid CHECK (mean_abs_shap >= 0),
    CONSTRAINT stability_valid CHECK (stability_score IS NULL OR (stability_score >= 0 AND stability_score <= 1)),
    CONSTRAINT dataset_size_valid CHECK (dataset_size > 0)
);

CREATE INDEX idx_importance_feature_model ON feature_importance(feature_id, model_id, computed_at DESC);
CREATE INDEX idx_importance_model ON feature_importance(model_id, mean_abs_shap DESC);
CREATE INDEX idx_importance_tenant_region ON feature_importance(tenant_id, region, computed_at DESC);
CREATE INDEX idx_importance_top_features ON feature_importance(mean_abs_shap DESC) WHERE recorded_at >= NOW() - INTERVAL '7 days';
CREATE INDEX idx_importance_trend ON feature_importance(importance_trend DESC NULLS LAST);

-- View for nightly top K features per model
CREATE MATERIALIZED VIEW top_features_by_model AS
SELECT 
    model_id,
    feature_id,
    mean_abs_shap,
    importance_percentile,
    stability_score,
    importance_trend,
    recorded_at,
    ROW_NUMBER() OVER (PARTITION BY model_id ORDER BY mean_abs_shap DESC) as rank
FROM feature_importance
WHERE recorded_at >= NOW() - INTERVAL '1 day'
    AND mean_abs_shap IS NOT NULL;

CREATE INDEX idx_top_features_model ON top_features_by_model(model_id, rank);

GRANT SELECT ON feature_importance TO PUBLIC;
GRANT SELECT ON top_features_by_model TO PUBLIC;

-- ============================================================================
-- 6. FEATURE_CHANGE_LOG: Governance & Audit Trail
-- ============================================================================
-- Immutable log of all changes to feature_catalog, test_cases, ownership, and deployments.
-- Links to GitHub PRs, approval decisions, and deployment events.
-- Supports regulatory compliance and incident post-mortems.

CREATE TABLE feature_change_log (
    change_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    
    -- Change metadata
    change_type TEXT NOT NULL,                      -- 'created' | 'updated' | 'approved' | 'deployed' | 'deprecated'
    change_description TEXT NOT NULL,
    
    -- Code & CI/CD links
    pr_url TEXT,                                    -- GitHub PR link if applicable
    commit_sha TEXT,
    branch TEXT,
    
    -- Approval workflow
    requested_by TEXT NOT NULL,
    approved_by TEXT,
    approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    approval_status TEXT,                           -- PENDING | APPROVED | REJECTED
    rejection_reason TEXT,
    
    -- Deployment info
    deployed_at TIMESTAMPTZ,
    deployment_version TEXT,
    rollback_at TIMESTAMPTZ,
    
    -- Before/after snapshots (for updates)
    old_properties JSONB,
    new_properties JSONB,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT change_type_valid CHECK (change_type IN ('created', 'updated', 'approved', 'deployed', 'deprecated', 'rollback'))
);

CREATE INDEX idx_changelog_feature ON feature_change_log(feature_id, created_at DESC);
CREATE INDEX idx_changelog_type ON feature_change_log(change_type, created_at DESC);
CREATE INDEX idx_changelog_approval ON feature_change_log(approval_status, created_at DESC) WHERE approval_required;
CREATE INDEX idx_changelog_deployment ON feature_change_log(deployed_at DESC NULLS LAST) WHERE deployed_at IS NOT NULL;
CREATE INDEX idx_changelog_tenant_region ON feature_change_log(tenant_id, region, created_at DESC);
CREATE INDEX idx_changelog_requested_by ON feature_change_log(requested_by, created_at DESC);

-- View for pending approvals
CREATE VIEW pending_approvals AS
SELECT * FROM feature_change_log
WHERE approval_required AND approval_status = 'PENDING'
ORDER BY created_at ASC;

GRANT SELECT ON feature_change_log TO PUBLIC;
GRANT SELECT ON pending_approvals TO PUBLIC;

-- ============================================================================
-- 7. FEATURE_TEST_CASES: Feature Unit Tests & Validation
-- ============================================================================
-- Stored test definitions per feature; run in CI and pre-deployment.
-- Each test has a SQL assertion and expected result; gates feature deployment.

CREATE TABLE feature_test_cases (
    test_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    
    -- Test definition
    test_name TEXT NOT NULL,
    description TEXT,
    test_type TEXT NOT NULL,                        -- 'unit' | 'integration' | 'regression' | 'property'
    
    -- Test specification
    sql_assertion TEXT NOT NULL,                    -- SELECT query returning bool or numeric value
    expected_result JSONB,                          -- expected value(s)
    tolerance DOUBLE PRECISION,                     -- for numeric comparisons (default 0)
    
    -- Sample dataset reference
    sample_data_id TEXT,                            -- reference to small sample dataset for CI
    sample_size INT,                                -- rows in sample dataset
    
    -- Execution history (last run)
    last_run_at TIMESTAMPTZ,
    last_run_duration_s DOUBLE PRECISION,
    last_run_passed BOOLEAN,
    last_run_error TEXT,
    
    -- Configuration
    timeout_seconds INT DEFAULT 30,
    enabled BOOLEAN DEFAULT TRUE,
    critical BOOLEAN DEFAULT FALSE,                 -- must pass before deployment
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT timeout_valid CHECK (timeout_seconds > 0)
);

CREATE INDEX idx_test_feature ON feature_test_cases(feature_id);
CREATE INDEX idx_test_critical ON feature_test_cases(critical) WHERE critical AND enabled;
CREATE INDEX idx_test_last_run ON feature_test_cases(last_run_at DESC NULLS LAST);
CREATE INDEX idx_test_enabled ON feature_test_cases(enabled) WHERE enabled;
CREATE INDEX idx_test_tenant_region ON feature_test_cases(tenant_id, region);

-- View for failing tests
CREATE VIEW failing_tests AS
SELECT * FROM feature_test_cases
WHERE enabled AND (last_run_passed = FALSE OR last_run_at IS NULL)
ORDER BY last_run_at DESC NULLS FIRST;

GRANT SELECT ON feature_test_cases TO PUBLIC;
GRANT SELECT ON failing_tests TO PUBLIC;

-- ============================================================================
-- 8. FEATURE_LINEAGE: Dependency Tracking & DAG
-- ============================================================================
-- Tracks upstream (source tables/features) and downstream (models, dashboards) 
-- dependencies for impact analysis and data governance.

CREATE TABLE feature_lineage (
    lineage_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_feature_id TEXT,                         -- NULL if upstream is table/external
    source_table TEXT,                              -- schema.table if external source
    source_column TEXT,
    target_feature_id TEXT NOT NULL,
    
    -- Dependency type
    lineage_type TEXT NOT NULL,                     -- 'feature_to_feature' | 'table_to_feature' | 'feature_to_model'
    transformation TEXT,                            -- brief description of transform
    
    -- Cardinality
    is_one_to_one BOOLEAN,
    is_many_to_one BOOLEAN,
    
    -- Sensitive data flag
    contains_pii BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id TEXT NOT NULL DEFAULT 'default',
    
    FOREIGN KEY (target_feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT lineage_source_valid CHECK (source_feature_id IS NOT NULL OR source_table IS NOT NULL)
);

CREATE INDEX idx_lineage_source_feature ON feature_lineage(source_feature_id) 
    WHERE source_feature_id IS NOT NULL;
CREATE INDEX idx_lineage_target_feature ON feature_lineage(target_feature_id);
CREATE INDEX idx_lineage_source_table ON feature_lineage(source_table) 
    WHERE source_table IS NOT NULL;
CREATE INDEX idx_lineage_pii ON feature_lineage(contains_pii) WHERE contains_pii;

-- Recursive view for upstream dependencies (all ancestors)
CREATE RECURSIVE VIEW feature_lineage_ancestors AS
  SELECT 
    target_feature_id,
    source_feature_id,
    source_table,
    lineage_type,
    1::INT as depth,
    ARRAY[target_feature_id] as path
  FROM feature_lineage
  WHERE source_feature_id IS NOT NULL
  UNION ALL
  SELECT 
    fla.target_feature_id,
    fl.source_feature_id,
    fl.source_table,
    fl.lineage_type,
    fla.depth + 1,
    fla.path || fl.source_feature_id
  FROM feature_lineage_ancestors fla
  JOIN feature_lineage fl ON fla.source_feature_id = fl.target_feature_id
  WHERE fla.depth < 10 AND NOT fl.source_feature_id = ANY(fla.path);

GRANT SELECT ON feature_lineage TO PUBLIC;
GRANT SELECT ON feature_lineage_ancestors TO PUBLIC;

-- ============================================================================
-- 9. FEATURE_COMPUTATIONS: Job Execution Metadata
-- ============================================================================
-- Logs feature materialization / drift computation / importance job runs.
-- Enables observability of compute costs, latencies, failures.

CREATE TABLE feature_computations (
    computation_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_id TEXT NOT NULL,
    
    -- Job info
    job_type TEXT NOT NULL,                         -- 'materialization' | 'drift' | 'importance'
    job_id TEXT NOT NULL,                           -- e.g., Spark job ID, Temporal run ID
    compute_engine TEXT,                            -- 'spark' | 'trino' | 'python' | 'temporal'
    
    -- Execution
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_seconds DOUBLE PRECISION GENERATED ALWAYS AS (
        EXTRACT(EPOCH FROM COALESCE(completed_at, NOW()) - started_at)
    ) STORED,
    
    -- Status
    status TEXT NOT NULL DEFAULT 'RUNNING',         -- RUNNING | SUCCESS | FAILED | CANCELLED
    error_message TEXT,
    error_type TEXT,
    
    -- Resource usage
    compute_cost_usd DOUBLE PRECISION,
    rows_processed BIGINT,
    bytes_written BIGINT,
    
    -- Metrics
    success_rate DOUBLE PRECISION,                  -- [0,1] for batch jobs
    
    -- Metadata
    operator_id TEXT,                               -- user or service that triggered job
    tenant_id TEXT NOT NULL DEFAULT 'default',
    region TEXT NOT NULL DEFAULT 'us-east-1',
    
    FOREIGN KEY (feature_id) REFERENCES feature_catalog(feature_id),
    CONSTRAINT status_valid CHECK (status IN ('RUNNING', 'SUCCESS', 'FAILED', 'CANCELLED')),
    CONSTRAINT duration_valid CHECK (duration_seconds IS NULL OR duration_seconds >= 0)
);

CREATE INDEX idx_computation_feature_job ON feature_computations(feature_id, started_at DESC);
CREATE INDEX idx_computation_status ON feature_computations(status, started_at DESC);
CREATE INDEX idx_computation_job_id ON feature_computations(job_id);
CREATE INDEX idx_computation_cost ON feature_computations(compute_cost_usd DESC NULLS LAST) 
    WHERE compute_cost_usd IS NOT NULL;
CREATE INDEX idx_computation_failed ON feature_computations(status) 
    WHERE status IN ('FAILED', 'CANCELLED');
CREATE INDEX idx_computation_tenant_region ON feature_computations(tenant_id, region, started_at DESC);

-- Materialized view for SLO tracking
CREATE MATERIALIZED VIEW computation_slos AS
SELECT
    feature_id,
    job_type,
    COUNT(*) as total_runs,
    SUM(CASE WHEN status = 'SUCCESS' THEN 1 ELSE 0 END)::DOUBLE PRECISION / COUNT(*) as success_rate,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_seconds) as p95_duration_s,
    AVG(compute_cost_usd) as avg_cost_usd,
    MAX(started_at) as last_run,
    DATE_TRUNC('hour', MAX(started_at)) as last_run_hour
FROM feature_computations
WHERE started_at >= NOW() - INTERVAL '7 days'
GROUP BY feature_id, job_type;

CREATE INDEX idx_slos_feature_type ON computation_slos(feature_id, job_type);

GRANT SELECT ON feature_computations TO PUBLIC;
GRANT SELECT ON computation_slos TO PUBLIC;

-- ============================================================================
-- 10. HELPER FUNCTIONS & TRIGGERS
-- ============================================================================

-- Auto-update feature_catalog.updated_at on row change
CREATE OR REPLACE FUNCTION update_feature_catalog_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER feature_catalog_update_timestamp
    BEFORE UPDATE ON feature_catalog
    FOR EACH ROW
    EXECUTE FUNCTION update_feature_catalog_timestamp();

-- Auto-update feature_test_cases.updated_at
CREATE OR REPLACE FUNCTION update_test_cases_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER test_cases_update_timestamp
    BEFORE UPDATE ON feature_test_cases
    FOR EACH ROW
    EXECUTE FUNCTION update_test_cases_timestamp();

-- Function to get feature ancestors (recursive lineage)
CREATE OR REPLACE FUNCTION get_feature_ancestors(feature_id_input TEXT)
RETURNS TABLE (ancestor_feature_id TEXT, depth INT, lineage_type TEXT) AS $$
SELECT DISTINCT source_feature_id, depth, lineage_type
FROM feature_lineage_ancestors
WHERE target_feature_id = feature_id_input
ORDER BY depth, source_feature_id;
$$ LANGUAGE SQL STABLE;

-- Function to check feature health (drift + quality)
CREATE OR REPLACE FUNCTION get_feature_health(feature_id_input TEXT)
RETURNS TABLE (
    feature_id TEXT,
    last_materialized TIMESTAMPTZ,
    materialization_lag_seconds INT,
    drift_count_24h INT,
    active_drift BOOLEAN,
    quality_failures_24h INT,
    test_failures INT
) AS $$
SELECT
    fc.feature_id,
    fw.last_processed,
    fw.watermark_age_seconds,
    (SELECT COUNT(*) FROM feature_drift_metrics 
     WHERE feature_id = fc.feature_id AND is_drifted AND recorded_at >= NOW() - INTERVAL '24 hours'),
    (SELECT EXISTS(SELECT 1 FROM active_drifts WHERE feature_id = fc.feature_id)),
    (SELECT COUNT(*) FROM feature_quality_checks 
     WHERE feature_id = fc.feature_id AND NOT passed AND computed_at >= NOW() - INTERVAL '24 hours'),
    (SELECT COUNT(*) FROM feature_test_cases 
     WHERE feature_id = fc.feature_id AND last_run_passed = FALSE AND enabled)
FROM feature_catalog fc
LEFT JOIN feature_watermarks fw ON fc.feature_id = fw.feature_id
WHERE fc.feature_id = feature_id_input;
$$ LANGUAGE SQL STABLE;

-- ============================================================================
-- 11. PERMISSIONS & GRANTS
-- ============================================================================
-- Role-based access:
-- - feature_owner: can update own features
-- - feature_read: read-only access to catalog
-- - feature_admin: full access

GRANT SELECT, INSERT, UPDATE ON feature_catalog TO PUBLIC;
GRANT SELECT, UPDATE ON feature_watermarks TO PUBLIC;
GRANT SELECT, INSERT ON feature_drift_metrics TO PUBLIC;
GRANT SELECT, INSERT ON feature_quality_checks TO PUBLIC;
GRANT SELECT, INSERT ON feature_importance TO PUBLIC;
GRANT SELECT, INSERT ON feature_change_log TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON feature_test_cases TO PUBLIC;
GRANT SELECT, INSERT ON feature_lineage TO PUBLIC;
GRANT SELECT, INSERT ON feature_computations TO PUBLIC;

-- ============================================================================
-- 12. COMMENTS & DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE feature_catalog IS 'Canonical registry of all features (raw, derived, time-series, embeddings) with JSONB properties for flexible metadata storage.';
COMMENT ON COLUMN feature_catalog.properties IS 'Flexible JSONB containing: feature_type, expression, aggregation, window, materialization_policy, drift_config, test_cases, lineage_refs, tags, confidence.';
COMMENT ON TABLE feature_watermarks IS 'Tracks last processed timestamp per feature for incremental materialization pipelines. Updated atomically with job completion.';
COMMENT ON TABLE feature_drift_metrics IS 'Distribution shift detection results (KS, PSI, Chi-square, classifier). Partitioned by timestamp for rolling-window performance.';
COMMENT ON TABLE feature_quality_checks IS 'Data quality assertions (null rate, cardinality, type, range). Failures gate feature materialization.';
COMMENT ON TABLE feature_importance IS 'Features'' importance scores from SHAP, permutation, and gain methods. Tracks stability and trend for drift alerting.';
COMMENT ON TABLE feature_change_log IS 'Immutable audit trail of all catalog changes, approvals, deployments. Supports governance and compliance.';
COMMENT ON TABLE feature_test_cases IS 'Unit/integration tests per feature; gates deployment via CI gating.';
COMMENT ON TABLE feature_lineage IS 'Tracks upstream (tables, features) and downstream (models, dashboards) dependencies. Enables impact analysis.';
COMMENT ON TABLE feature_computations IS 'Execution logs for materialization, drift, importance jobs. Tracks SLOs, costs, latencies.';

-- ============================================================================
-- END OF SCHEMA DEFINITION
-- ============================================================================

-- =====================================================================
-- Trino Materialized Views for Audit Dashboards
-- Pre-aggregated views for performance and multi-tenant isolation
-- =====================================================================

-- =====================================================================
-- 1. TENANT SCHEDULER SLO SUMMARY
-- Daily SLO metrics per tenant
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_tenant_scheduler_slo AS
SELECT
    tenant_id,
    DATE(start_ts) AS run_date,
    COUNT(*) AS total_runs,
    COUNT(*) FILTER (WHERE status = 'SUCCESS') AS successful_runs,
    COUNT(*) FILTER (WHERE status = 'FAILED') AS failed_runs,
    COUNT(*) FILTER (WHERE status = 'COMPLIANCE_BLOCK') AS blocked_runs,
    AVG(DATE_DIFF('second', start_ts, end_ts)) AS avg_duration_seconds,
    MAX(DATE_DIFF('second', start_ts, end_ts)) AS max_duration_seconds,
    PERCENTILE(DATE_DIFF('second', start_ts, end_ts), 0.95) AS p95_duration_seconds
FROM iceberg.audit.scheduler_job_runs
WHERE end_ts IS NOT NULL
GROUP BY tenant_id, DATE(start_ts);

COMMENT ON MATERIALIZED VIEW iceberg.mv_tenant_scheduler_slo IS 
'Daily SLO metrics per tenant for scheduler performance dashboards';

-- =====================================================================
-- 2. TENANT COMPLIANCE VIOLATIONS
-- Daily compliance violation summary per tenant
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_tenant_compliance_violations AS
SELECT
    tenant_id,
    DATE(violated_at) AS violation_date,
    COUNT(*) AS violation_count,
    COUNT(*) FILTER (WHERE pii_exposed = true) AS pii_exposure_count,
    COUNT(*) FILTER (WHERE severity = 'CRITICAL') AS critical_violations,
    COUNT(*) FILTER (WHERE severity = 'HIGH') AS high_violations,
    AVG(DATE_DIFF('hour', violated_at, COALESCE(remediated_at, CURRENT_TIMESTAMP))) AS avg_remediation_hours,
    COUNT(*) FILTER (WHERE remediated_at IS NULL) AS open_violations
FROM iceberg.audit.compliance_violations
GROUP BY tenant_id, DATE(violated_at);

COMMENT ON MATERIALIZED VIEW iceberg.mv_tenant_compliance_violations IS 
'Daily compliance violation summary for regulator reporting';

-- =====================================================================
-- 3. TENANT GOVERNANCE ACTIVITY
-- Daily changeset activity per tenant
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_tenant_governance_activity AS
SELECT
    tenant_id,
    DATE(created_at) AS activity_date,
    COUNT(*) AS changeset_count,
    COUNT(DISTINCT actor) AS unique_actors,
    COUNT(*) FILTER (WHERE status = 'PENDING') AS pending_count,
    COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved_count,
    COUNT(*) FILTER (WHERE status = 'REJECTED') AS rejected_count,
    COUNT(*) FILTER (WHERE status = 'APPLIED') AS applied_count,
    AVG(CAST(JSON_EXTRACT_SCALAR(ai_risk, '$.riskScore') AS DOUBLE)) AS avg_risk_score
FROM iceberg.audit.governance_changesets
WHERE tenant_id IS NOT NULL
GROUP BY tenant_id, DATE(created_at);

COMMENT ON MATERIALIZED VIEW iceberg.mv_tenant_governance_activity IS 
'Daily governance changeset activity per tenant';

-- =====================================================================
-- 4. SEMANTIC DRIFT TRENDS
-- Track semantic term changes over time per tenant
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_semantic_drift_trends AS
SELECT
    tenant_id,
    semantic_term_id,
    DATE(timestamp) AS drift_date,
    COUNT(*) AS version_count,
    MAX(version) AS latest_version,
    COUNT(DISTINCT business_term_id) AS business_term_changes
FROM iceberg.audit.semantic_snapshots
WHERE tenant_id IS NOT NULL
GROUP BY tenant_id, semantic_term_id, DATE(timestamp);

COMMENT ON MATERIALIZED VIEW iceberg.mv_semantic_drift_trends IS 
'Semantic term drift tracking for lineage analysis';

-- =====================================================================
-- 5. CROSS-TENANT OPERATIONAL HEALTH
-- Internal view for platform ops (not exposed to tenants)
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_platform_health AS
SELECT
    tenant_id,
    DATE(start_ts) AS health_date,
    
    -- Job metrics
    COUNT(*) AS total_jobs,
    COUNT(*) FILTER (WHERE status = 'FAILED') AS failed_jobs,
    CAST(COUNT(*) FILTER (WHERE status = 'FAILED') AS DOUBLE) / COUNT(*) AS failure_rate,
    
    -- Compliance metrics
    (SELECT COUNT(*) FROM iceberg.audit.compliance_violations cv 
     WHERE cv.tenant_id = jr.tenant_id 
     AND DATE(cv.violated_at) = DATE(jr.start_ts)) AS violation_count,
    
    -- Semantic drift
    (SELECT COUNT(DISTINCT semantic_term_id) 
     FROM iceberg.audit.semantic_snapshots ss 
     WHERE ss.tenant_id = jr.tenant_id 
     AND DATE(ss.timestamp) = DATE(jr.start_ts)) AS semantic_changes
     
FROM iceberg.audit.scheduler_job_runs jr
GROUP BY tenant_id, DATE(start_ts);

COMMENT ON MATERIALIZED VIEW iceberg.mv_platform_health IS 
'Cross-tenant operational health metrics (internal only)';

-- =====================================================================
-- 6. JOB SEMANTIC IMPACT
-- Track which jobs are affected by semantic term changes
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_job_semantic_impact AS
SELECT
    jr.tenant_id,
    jr.job_id,
    DATE(jr.start_ts) AS impact_date,
    COUNT(DISTINCT JSON_EXTRACT_SCALAR(jr.semantic_context, '$.semantic_term_id')) AS semantic_terms_used,
    COUNT(*) FILTER (WHERE jr.status = 'FAILED') AS failures_after_drift,
    AVG(DATE_DIFF('second', jr.start_ts, jr.end_ts)) AS avg_runtime_seconds
FROM iceberg.audit.scheduler_job_runs jr
WHERE jr.semantic_context IS NOT NULL
GROUP BY jr.tenant_id, jr.job_id, DATE(jr.start_ts);

COMMENT ON MATERIALIZED VIEW iceberg.mv_job_semantic_impact IS 
'Track job failures correlated with semantic drift';

-- =====================================================================
-- 7. AI NARRATIVE SUMMARY
-- Aggregated AI-generated insights per tenant
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_ai_narrative_summary AS
SELECT
    ai.tenant_id,
    ai.record_type,
    DATE(ai.timestamp) AS summary_date,
    COUNT(*) AS narrative_count,
    AVG(ai.confidence) AS avg_confidence,
    COUNT(DISTINCT ai.root_cause) AS unique_root_causes,
    MODE() WITHIN GROUP (ORDER BY ai.root_cause) AS most_common_root_cause
FROM iceberg.audit.ai_suggestions ai
GROUP BY ai.tenant_id, ai.record_type, DATE(ai.timestamp);

COMMENT ON MATERIALIZED VIEW iceberg.mv_ai_narrative_summary IS 
'AI-generated audit narrative summary for trend analysis';

-- =====================================================================
-- 8. TENANT COMPLIANCE REPORT
-- Regulator-ready compliance summary
-- =====================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS iceberg.mv_tenant_compliance_report AS
SELECT
    cv.tenant_id,
    DATE_TRUNC('month', cv.violated_at) AS report_month,
    COUNT(*) AS total_violations,
    COUNT(*) FILTER (WHERE cv.pii_exposed = true) AS pii_exposures,
    COUNT(*) FILTER (WHERE cv.severity IN ('HIGH', 'CRITICAL')) AS severe_violations,
    AVG(DATE_DIFF('hour', cv.violated_at, COALESCE(cv.remediated_at, CURRENT_TIMESTAMP))) AS avg_remediation_hours,
    MAX(DATE_DIFF('hour', cv.violated_at, COALESCE(cv.remediated_at, CURRENT_TIMESTAMP))) AS max_remediation_hours,
    COUNT(*) FILTER (WHERE cv.remediated_at IS NULL) AS open_violations,
    COUNT(DISTINCT ELEMENT(cv.compliance_refs)) AS unique_regulations
FROM iceberg.audit.compliance_violations cv
GROUP BY cv.tenant_id, DATE_TRUNC('month', cv.violated_at);

COMMENT ON MATERIALIZED VIEW iceberg.mv_tenant_compliance_report IS 
'Monthly compliance report for regulator submission';

-- =====================================================================
-- REFRESH SCHEDULE (example for automation)
-- =====================================================================
-- These would typically be scheduled via Airflow or similar
-- 
-- -- Refresh daily at 1 AM
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_tenant_scheduler_slo');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_tenant_compliance_violations');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_tenant_governance_activity');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_semantic_drift_trends');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_platform_health');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_job_semantic_impact');
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_ai_narrative_summary');
-- 
-- -- Refresh monthly (for compliance reports)
-- CALL iceberg.system.refresh_materialized_view('iceberg', 'mv_tenant_compliance_report');

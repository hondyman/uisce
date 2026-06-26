-- Semantic Layer Integration Tables
-- Run this migration to add monitoring, auditing, drift detection, and event tracking

-- ============================================================================
-- AUDIT & QUERY TRACKING
-- ============================================================================

-- Semantic Query Audit Log - tracks every compiled and executed query
CREATE TABLE IF NOT EXISTS public.semantic_query_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    user_id uuid,
  CREATE TABLE IF NOT EXISTS analytics.semantic_query_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID NOT NULL,
    query_hash TEXT NOT NULL,
    generated_sql TEXT,
    execution_time_ms INTEGER,
    row_count INTEGER,
    error_message TEXT,
    user_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source_application TEXT,
    query_metadata JSONB,
    cache_hit BOOLEAN DEFAULT FALSE,
    duration_ms INTEGER,
    status TEXT,
    CONSTRAINT semantic_query_audit_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_tenant_model ON analytics.semantic_query_audit(tenant_id, model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_duration ON analytics.semantic_query_audit(duration_ms DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_cache_hit ON analytics.semantic_query_audit(cache_hit, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_status ON analytics.semantic_query_audit(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_user ON analytics.semantic_query_audit(user_id, created_at DESC);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS analytics.semantic_query_performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID NOT NULL,
    hour_bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    avg_execution_time_ms FLOAT,
    p95_execution_time_ms FLOAT,
    query_count INTEGER,
    error_count INTEGER,
    cache_hit_rate FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_query_performance_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_query_performance_tenant_bucket ON analytics.semantic_query_performance(tenant_id, hour_bucket DESC);
CREATE INDEX IF NOT EXISTS idx_query_performance_model_bucket ON analytics.semantic_query_performance(model_id, hour_bucket DESC);

-- ============================================================================
-- SEMANTIC LAYER CHANGES & EVENT SOURCING
-- ============================================================================

-- Audit log for semantic layer changes
CREATE TABLE IF NOT EXISTS analytics.semantic_layer_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID,
    entity_type TEXT NOT NULL, -- 'metric', 'dimension', 'view', etc.
    change_type TEXT NOT NULL, -- 'create', 'update', 'delete', 'publish'
    user_id UUID,
    previous_state JSONB,
    new_state JSONB,
    change_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_layer_audit_log_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_audit_tenant_created ON analytics.semantic_layer_audit_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_audit_model ON analytics.semantic_layer_audit_log(model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_audit_change_type ON analytics.semantic_layer_audit_log(change_type);
CREATE INDEX IF NOT EXISTS idx_semantic_audit_user ON analytics.semantic_layer_audit_log(user_id, created_at DESC);

-- Semantic change events for downstream consumers
CREATE TABLE IF NOT EXISTS analytics.semantic_change_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL, -- 'metric_updated', 'schema_changed', etc.
    payload JSONB NOT NULL,
    aggregate_id UUID,
    aggregate_type TEXT,
    version INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT semantic_change_events_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_events_aggregate ON analytics.semantic_change_events(aggregate_id, aggregate_type, version);
CREATE INDEX IF NOT EXISTS idx_semantic_events_tenant ON analytics.semantic_change_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_events_type ON analytics.semantic_change_events(event_type, created_at DESC);

-- Event delivery log
CREATE TABLE IF NOT EXISTS analytics.semantic_event_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    subscriber_queue TEXT NOT NULL,
    status TEXT NOT NULL, -- 'pending', 'delivered', 'failed'
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_event_delivery_fk_event FOREIGN KEY (event_id) REFERENCES analytics.semantic_change_events(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_delivery_status ON analytics.semantic_event_delivery_log(status, created_at);
CREATE INDEX IF NOT EXISTS idx_event_delivery_queue ON analytics.semantic_event_delivery_log(subscriber_queue, created_at DESC);

-- ============================================================================
-- DRIFT DETECTION & MANAGEMENT
-- ============================================================================

-- Semantic Drift Reports
CREATE TABLE IF NOT EXISTS analytics.semantic_drift_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID,
    drift_severity TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    status TEXT NOT NULL, -- 'open', 'investigating', 'resolved', 'ignored'
    report_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    detected_changes JSONB,
    affected_downstream_systems TEXT[],
    resolution_notes TEXT,
    resolved_by UUID,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_drift_reports_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_drift_reports_tenant_model ON analytics.semantic_drift_reports(tenant_id, model_id, report_time DESC);
CREATE INDEX IF NOT EXISTS idx_drift_reports_severity ON analytics.semantic_drift_reports(drift_severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_drift_reports_status ON analytics.semantic_drift_reports(status, created_at DESC);

-- Semantic Drift Issues
CREATE TABLE IF NOT EXISTS analytics.semantic_drift_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL,
    issue_type TEXT NOT NULL, -- 'schema_mismatch', 'data_type_change', 'missing_column', etc.
    description TEXT,
    severity TEXT NOT NULL,
    source_element TEXT,
    target_element TEXT,
    suggested_fix TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_drift_issues_fk_report FOREIGN KEY (report_id) REFERENCES analytics.semantic_drift_reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_drift_issues_type ON analytics.semantic_drift_issues(issue_type);
CREATE INDEX IF NOT EXISTS idx_drift_issues_severity ON analytics.semantic_drift_issues(severity);
CREATE INDEX IF NOT EXISTS idx_drift_issues_report ON analytics.semantic_drift_issues(report_id);

-- ============================================================================
-- AI SUGGESTIONS
-- ============================================================================

-- Semantic Suggestions (AI/ML generated)
CREATE TABLE IF NOT EXISTS analytics.semantic_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID,
    suggestion_type TEXT NOT NULL, -- 'optimization', 'naming', 'relationship', 'documentation'
    confidence_score FLOAT,
    suggestion_payload JSONB,
    rationale TEXT,
    status TEXT DEFAULT 'pending', -- 'pending', 'accepted', 'rejected', 'implemented'
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_suggestions_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_suggestions_tenant_model ON analytics.semantic_suggestions(tenant_id, model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_suggestions_type ON analytics.semantic_suggestions(suggestion_type);
CREATE INDEX IF NOT EXISTS idx_suggestions_status ON analytics.semantic_suggestions(status, priority DESC);
CREATE INDEX IF NOT EXISTS idx_suggestions_confidence ON analytics.semantic_suggestions(confidence_score DESC);

-- Suggestion Feedback
CREATE TABLE IF NOT EXISTS analytics.semantic_suggestion_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suggestion_id UUID NOT NULL,
    user_id UUID,
    action TEXT NOT NULL, -- 'accept', 'reject', 'modify'
    feedback_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_suggestion_feedback_fk_suggestion FOREIGN KEY (suggestion_id) REFERENCES analytics.semantic_suggestions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_suggestion_feedback_suggestion ON analytics.semantic_suggestion_feedback(suggestion_id);
CREATE INDEX IF NOT EXISTS idx_suggestion_feedback_action ON analytics.semantic_suggestion_feedback(action);

-- ============================================================================
-- CACHE INVALIDATION TRACKING
-- ============================================================================

-- Cache Invalidation Log
CREATE TABLE IF NOT EXISTS analytics.semantic_cache_invalidation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_id UUID,
    cache_key TEXT,
    invalidation_reason TEXT,
    invalidated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT semantic_cache_invalidation_fk_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cache_inval_tenant ON analytics.semantic_cache_invalidation_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cache_inval_model ON analytics.semantic_cache_invalidation_log(model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cache_inval_success ON public.semantic_cache_invalidation_log(success);

-- ============================================================================
-- MATERIALIZED VIEWS FOR ANALYTICS
-- ============================================================================

-- Query Performance Summary (refreshed hourly)
CREATE MATERIALIZED VIEW IF NOT EXISTS semantic_query_performance_summary AS
SELECT
    tenant_id,
    model_id,
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as query_count,
    AVG(duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99_duration_ms,
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / COUNT(*) as cache_hit_rate,
    SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)::float / COUNT(*) as error_rate,
    AVG(rows_scanned) as avg_rows_scanned
FROM semantic_query_audit
WHERE created_at > now() - interval '7 days'
GROUP BY tenant_id, model_id, DATE_TRUNC('hour', created_at);

CREATE INDEX IF NOT EXISTS idx_query_perf_summary_tenant_model ON semantic_query_performance_summary(tenant_id, model_id);

-- Model Change Summary (count changes by type)
CREATE MATERIALIZED VIEW IF NOT EXISTS semantic_model_change_summary AS
SELECT
    tenant_id,
    model_id,
    change_type,
    element_type,
    DATE_TRUNC('day', created_at) as day,
    COUNT(*) as change_count,
    COUNT(DISTINCT user_id) as unique_users
FROM semantic_layer_audit_log
WHERE created_at > now() - interval '30 days'
GROUP BY tenant_id, model_id, change_type, element_type, DATE_TRUNC('day', created_at);

-- Drift Summary (latest issues by model)
CREATE MATERIALIZED VIEW IF NOT EXISTS semantic_drift_summary AS
SELECT
    r.tenant_id,
    r.model_id,
    r.report_time,
    r.drift_severity,
    COUNT(DISTINCT i.id) as issue_count,
    COUNT(DISTINCT CASE WHEN i.severity = 'critical' THEN i.id END) as critical_issues,
    COUNT(DISTINCT CASE WHEN i.severity = 'high' THEN i.id END) as high_issues
FROM semantic_drift_reports r
LEFT JOIN semantic_drift_issues i ON r.id = i.report_id
WHERE r.report_time > now() - interval '30 days'
GROUP BY r.tenant_id, r.model_id, r.report_time, r.drift_severity;

-- ============================================================================
-- PERMISSIONS & RBAC UPDATES
-- ============================================================================

-- Add semantic layer permissions if role-based security is in place
-- Uncomment if your system uses RBAC

-- INSERT INTO public.role_permissions (id, role_id, resource, action, created_at)
-- VALUES
--   (gen_random_uuid(), 'analyst', 'semantic_queries', 'read', now()),
--   (gen_random_uuid(), 'analyst', 'semantic_audit_log', 'read', now()),
--   (gen_random_uuid(), 'data_engineer', 'semantic_models', 'write', now()),
--   (gen_random_uuid(), 'data_engineer', 'semantic_drift', 'write', now()),
--   (gen_random_uuid(), 'admin', 'semantic_suggestions', 'approve', now())
-- ON CONFLICT DO NOTHING;

-- ============================================================================
-- FUNCTION: Automatic audit trigger
-- ============================================================================

CREATE OR REPLACE FUNCTION trigger_semantic_change_audit()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO semantic_change_events (
        tenant_id, aggregate_id, aggregate_type, event_type, event_data, version, created_at
    ) VALUES (
        NEW.tenant_id,
        NEW.element_id,
        NEW.element_type,
        CASE WHEN OLD.id IS NULL THEN 'created' ELSE 'updated' END,
        jsonb_build_object(
            'old_definition', OLD.new_definition,
            'new_definition', NEW.new_definition,
            'change_reason', NEW.change_reason
        ),
        1,
        now()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER semantic_audit_trigger
AFTER INSERT OR UPDATE ON semantic_layer_audit_log
FOR EACH ROW EXECUTE FUNCTION trigger_semantic_change_audit();

-- ============================================================================
-- CLEANUP & MAINTENANCE PROCEDURES
-- ============================================================================

-- Stored procedure for archiving old audit logs (run monthly)
CREATE OR REPLACE PROCEDURE cleanup_old_semantic_audits(days_old INT DEFAULT 90)
LANGUAGE plpgsql
AS $$
DECLARE
    rows_deleted BIGINT;
BEGIN
    DELETE FROM semantic_query_audit
    WHERE created_at < now() - (days_old || ' days')::interval;
    
    GET DIAGNOSTICS rows_deleted = ROW_COUNT;
    
    RAISE NOTICE 'Cleaned up % old query audit records', rows_deleted;
END;
$$;

-- Stored procedure for refreshing materialized views
CREATE OR REPLACE PROCEDURE refresh_semantic_analytics()
LANGUAGE plpgsql
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY semantic_query_performance_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY semantic_model_change_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY semantic_drift_summary;
    RAISE NOTICE 'Semantic analytics views refreshed';
END;
$$;

-- ============================================================================
-- DONE
-- ============================================================================

-- Run this to verify all tables were created:
-- SELECT table_name FROM information_schema.tables 
-- WHERE table_schema = 'public' AND table_name LIKE 'semantic_%' 
-- ORDER BY table_name;

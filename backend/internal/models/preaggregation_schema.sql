-- Preaggregated Metrics Schema for Semantic Layer
-- This schema stores precomputed financial metrics for fast query performance

CREATE SCHEMA IF NOT EXISTS semantic_layer;

-- Main table for preaggregated metrics
CREATE TABLE IF NOT EXISTS semantic_layer.preaggregated_metrics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    value DECIMAL(20, 8) NOT NULL,
    grain JSONB NOT NULL, -- Array of grain dimensions (e.g., ["fund_id", "month"])
    grain_values JSONB NOT NULL, -- Key-value pairs for grain values
    last_refresh TIMESTAMP WITH TIME ZONE NOT NULL,
    refresh_schedule VARCHAR(50) NOT NULL, -- "daily", "weekly", "monthly"
    source_formula TEXT NOT NULL, -- Original Excel formula
    data_quality JSONB NOT NULL, -- Completeness, freshness, validation metrics
    business_context TEXT, -- Description of metric purpose
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_preagg_node_id ON semantic_layer.preaggregated_metrics(node_id);
CREATE INDEX IF NOT EXISTS idx_preagg_grain ON semantic_layer.preaggregated_metrics USING GIN(grain);
CREATE INDEX IF NOT EXISTS idx_preagg_grain_values ON semantic_layer.preaggregated_metrics USING GIN(grain_values);
CREATE INDEX IF NOT EXISTS idx_preagg_last_refresh ON semantic_layer.preaggregated_metrics(last_refresh);
CREATE INDEX IF NOT EXISTS idx_preagg_refresh_schedule ON semantic_layer.preaggregated_metrics(refresh_schedule);

-- Partitioning by month for better performance (optional, for large datasets)
-- CREATE TABLE semantic_layer.preaggregated_metrics_y2024m01 PARTITION OF semantic_layer.preaggregated_metrics
-- FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Audit table for tracking preaggregation runs
CREATE TABLE IF NOT EXISTS semantic_layer.preaggregation_audit (
    id SERIAL PRIMARY KEY,
    job_name VARCHAR(255) NOT NULL,
    metric_node_id VARCHAR(255) NOT NULL,
    grain JSONB NOT NULL,
    records_processed INTEGER NOT NULL,
    execution_time_ms INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL, -- "success", "partial_failure", "failure"
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Indexes for audit table
CREATE INDEX IF NOT EXISTS idx_audit_job_name ON semantic_layer.preaggregation_audit(job_name);
CREATE INDEX IF NOT EXISTS idx_audit_metric_node_id ON semantic_layer.preaggregation_audit(metric_node_id);
CREATE INDEX IF NOT EXISTS idx_audit_started_at ON semantic_layer.preaggregation_audit(started_at);

-- Data quality monitoring table
CREATE TABLE IF NOT EXISTS semantic_layer.data_quality_monitoring (
    id SERIAL PRIMARY KEY,
    metric_id uuid NOT NULL REFERENCES semantic_layer.preaggregated_metrics(id),
    check_type VARCHAR(100) NOT NULL, -- "completeness", "freshness", "accuracy"
    check_value DECIMAL(10, 4) NOT NULL,
    threshold DECIMAL(10, 4) NOT NULL,
    status VARCHAR(50) NOT NULL, -- "pass", "warning", "fail"
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL,
    details JSONB
);

-- Indexes for data quality
CREATE INDEX IF NOT EXISTS idx_dq_metric_id ON semantic_layer.data_quality_monitoring(metric_id);
CREATE INDEX IF NOT EXISTS idx_dq_status ON semantic_layer.data_quality_monitoring(status);
CREATE INDEX IF NOT EXISTS idx_dq_checked_at ON semantic_layer.data_quality_monitoring(checked_at);

-- Refresh schedule configuration
CREATE TABLE IF NOT EXISTS semantic_layer.refresh_schedules (
    id SERIAL PRIMARY KEY,
    metric_node_id VARCHAR(255) UNIQUE NOT NULL,
    schedule_type VARCHAR(50) NOT NULL, -- "cron", "interval"
    schedule_expression VARCHAR(255) NOT NULL, -- "0 6 * * *" or "24h"
    is_active BOOLEAN DEFAULT true,
    last_successful_run TIMESTAMP WITH TIME ZONE,
    next_scheduled_run TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default refresh schedules for preaggregated metrics
INSERT INTO semantic_layer.refresh_schedules (metric_node_id, schedule_type, schedule_expression, is_active)
VALUES
    ('private_markets_net_irr', 'cron', '0 6 * * *', true),
    ('private_markets_xirr', 'cron', '0 6 * * *', true),
    ('private_markets_gross_irr', 'cron', '0 6 * * *', true),
    ('private_markets_gross_moic', 'cron', '0 6 * * 1', true),
    ('private_markets_fee_ratio', 'cron', '0 6 * * *', true),
    ('private_markets_deployment_pace', 'cron', '0 6 * * *', true)
ON CONFLICT (metric_node_id) DO NOTHING;

-- Function to get preaggregated metric with freshness check
CREATE OR REPLACE FUNCTION semantic_layer.get_preaggregated_metric(
    p_node_id VARCHAR(255),
    p_grain_values JSONB,
    p_max_age_hours INTEGER DEFAULT 24
) RETURNS TABLE (
    id uuid,
    value DECIMAL(20, 8),
    last_refresh TIMESTAMP WITH TIME ZONE,
    is_fresh BOOLEAN,
    hours_old DECIMAL(10, 2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pam.id,
        pam.value,
        pam.last_refresh,
        (EXTRACT(EPOCH FROM (NOW() - pam.last_refresh)) / 3600) <= p_max_age_hours AS is_fresh,
        EXTRACT(EPOCH FROM (NOW() - pam.last_refresh)) / 3600 AS hours_old
    FROM semantic_layer.preaggregated_metrics pam
    WHERE pam.node_id = p_node_id
    AND pam.grain_values @> p_grain_values
    ORDER BY pam.last_refresh DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to get data quality summary
CREATE OR REPLACE FUNCTION semantic_layer.get_data_quality_summary(
    p_days_back INTEGER DEFAULT 7
) RETURNS TABLE (
    metric_node_id VARCHAR(255),
    avg_completeness DECIMAL(5, 4),
    avg_freshness_hours DECIMAL(10, 2),
    check_count INTEGER,
    last_check TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pam.node_id,
        AVG((pam.data_quality->>'completeness_score')::DECIMAL) as avg_completeness,
        AVG((pam.data_quality->>'freshness_hours')::DECIMAL) as avg_freshness_hours,
        COUNT(dqm.id) as check_count,
        MAX(dqm.checked_at) as last_check
    FROM semantic_layer.preaggregated_metrics pam
    LEFT JOIN semantic_layer.data_quality_monitoring dqm ON pam.id = dqm.metric_id
    WHERE dqm.checked_at >= NOW() - INTERVAL '1 day' * p_days_back
    GROUP BY pam.node_id
    ORDER BY pam.node_id;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON SCHEMA semantic_layer IS 'Schema for preaggregated financial metrics in the semantic layer';
COMMENT ON TABLE semantic_layer.preaggregated_metrics IS 'Main table storing precomputed financial metrics for fast query performance';
COMMENT ON TABLE semantic_layer.preaggregation_audit IS 'Audit trail for preaggregation job executions';
COMMENT ON TABLE semantic_layer.data_quality_monitoring IS 'Data quality checks and monitoring results';
COMMENT ON TABLE semantic_layer.refresh_schedules IS 'Configuration for automated metric refresh schedules';
COMMENT ON FUNCTION semantic_layer.get_preaggregated_metric IS 'Retrieve preaggregated metric with freshness validation';
COMMENT ON FUNCTION semantic_layer.get_data_quality_summary IS 'Get data quality summary for monitoring dashboard';

-- Semantic Layer Configuration Schema for Alpha Database
-- Migration: 000017_semantic_layer_config.sql
-- Created: 2025-09-13
-- Description: Configuration schema for semantic layer metadata and schedules

-- ===========================================
-- SEMANTIC LAYER CONFIGURATION SCHEMA
-- =========================================--

-- Create semantic layer schema (already created, but ensuring it exists)
CREATE SCHEMA IF NOT EXISTS semantic_layer;

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

-- Indexes for refresh schedules
CREATE INDEX IF NOT EXISTS idx_refresh_metric_node_id ON semantic_layer.refresh_schedules(metric_node_id);
CREATE INDEX IF NOT EXISTS idx_refresh_next_run ON semantic_layer.refresh_schedules(next_scheduled_run);
CREATE INDEX IF NOT EXISTS idx_refresh_active ON semantic_layer.refresh_schedules(is_active);

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

-- ===========================================
-- HELPER FUNCTIONS
-- =========================================--

-- Function to get next scheduled runs
CREATE OR REPLACE FUNCTION semantic_layer.get_next_scheduled_runs(
    p_limit INTEGER DEFAULT 10
) RETURNS TABLE (
    metric_node_id VARCHAR(255),
    schedule_expression VARCHAR(255),
    next_scheduled_run TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        rs.metric_node_id,
        rs.schedule_expression,
        rs.next_scheduled_run
    FROM semantic_layer.refresh_schedules rs
    WHERE rs.is_active = true
    AND rs.next_scheduled_run IS NOT NULL
    ORDER BY rs.next_scheduled_run ASC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to update last successful run
CREATE OR REPLACE FUNCTION semantic_layer.update_last_successful_run(
    p_metric_node_id VARCHAR(255),
    p_next_run TIMESTAMP WITH TIME ZONE DEFAULT NULL
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE semantic_layer.refresh_schedules
    SET
        last_successful_run = NOW(),
        next_scheduled_run = COALESCE(p_next_run, next_scheduled_run),
        updated_at = NOW()
    WHERE metric_node_id = p_metric_node_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- GRANT PERMISSIONS
-- =========================================--

-- Grant permissions to the application user (adjust as needed)
-- GRANT USAGE ON SCHEMA semantic_layer TO your_app_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA semantic_layer TO your_app_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA semantic_layer TO your_app_user;

-- ===========================================
-- MIGRATION COMPLETE
-- =========================================--

-- Add comment for documentation
COMMENT ON SCHEMA semantic_layer IS 'Schema for semantic layer configuration and metadata';
COMMENT ON TABLE semantic_layer.refresh_schedules IS 'Configuration for automated metric refresh schedules';
COMMENT ON FUNCTION semantic_layer.get_next_scheduled_runs IS 'Get upcoming scheduled metric refresh runs';
COMMENT ON FUNCTION semantic_layer.update_last_successful_run IS 'Update timestamp for successful metric refresh run';

-- ============================================================================
-- Migration 013: Advanced Analytics
-- ============================================================================
-- Purpose: Analytics views and materialized tables for dashboards
-- ============================================================================

-- Create materialized view for sync analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.sync_daily_stats AS
SELECT 
    DATE(sj.started_at) as date,
    sj.tenant_id,
    COUNT(*) as total_syncs,
    COUNT(*) FILTER (WHERE sj.status = 'completed') as successful_syncs,
    COUNT(*) FILTER (WHERE sj.status = 'failed') as failed_syncs,
    COUNT(*) FILTER (WHERE sj.status = 'completed') * 100.0 / COUNT(*) as success_rate,
    AVG(EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at))) as avg_duration_seconds,
    SUM(sj.processed_events) as total_events_synced,
    AVG(sj.processed_events) as avg_events_per_sync
FROM sync_jobs sj
WHERE sj.started_at > NOW() - INTERVAL '90 days'
GROUP BY DATE(sj.started_at), sj.tenant_id
ORDER BY date DESC;

-- Create index for fast queries
CREATE UNIQUE INDEX idx_sync_daily_stats_date_tenant 
ON analytics.sync_daily_stats(date, tenant_id);

-- Create materialized view for user engagement
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.user_engagement_stats AS
SELECT 
    u.tenant_id,
    u.id as user_id,
    COUNT(DISTINCT DATE(sj.started_at)) as active_days,
    COUNT(sj.id) as total_syncs,
    COUNT(DISTINCT sj.calendar_id) as calendars_connected,
    MAX(sj.started_at) as last_sync_at,
    u.created_at as user_since,
    EXTRACT(DAY FROM (NOW() - u.created_at)) as user_tenure_days
FROM users u
LEFT JOIN sync_jobs sj ON u.id = sj.user_id
GROUP BY u.tenant_id, u.id, u.created_at;

CREATE UNIQUE INDEX idx_user_engagement_user 
ON analytics.user_engagement_stats(user_id);

-- Create materialized view for conflict analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.conflict_stats AS
SELECT 
    DATE(sc.detected_at) as date,
    sc.tenant_id,
    sc.conflict_type,
    sc.severity,
    COUNT(*) as total_conflicts,
    COUNT(*) FILTER (WHERE sc.resolution_status IN ('auto_resolved', 'manually_resolved')) as resolved_conflicts,
    COUNT(*) FILTER (WHERE sc.resolution_status IN ('auto_resolved', 'manually_resolved')) * 100.0 / COUNT(*) as resolution_rate,
    COUNT(*) FILTER (WHERE sc.auto_resolved = TRUE) as auto_resolved,
    COUNT(*) FILTER (WHERE sc.user_overrode_ml = TRUE) as user_overrides,
    AVG(sc.ml_confidence) as avg_ml_confidence
FROM sync_conflicts sc
WHERE sc.detected_at > NOW() - INTERVAL '90 days'
GROUP BY DATE(sc.detected_at), sc.tenant_id, sc.conflict_type, sc.severity
ORDER BY date DESC;

CREATE UNIQUE INDEX idx_conflict_stats_date_tenant_type 
ON analytics.conflict_stats(date, tenant_id, conflict_type, severity);

-- Create function to refresh materialized views
CREATE OR REPLACE FUNCTION analytics.refresh_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.sync_daily_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.user_engagement_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.conflict_stats;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh every hour (using pg_cron extension)
-- SELECT cron.schedule('refresh-analytics', '0 * * * *', 'SELECT analytics.refresh_analytics_views()');

-- Create view for executive dashboard
CREATE OR REPLACE VIEW analytics.executive_dashboard AS
SELECT 
    'Total Users' as metric,
    COUNT(*)::text as value,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '7 days') as new_this_week
FROM users
UNION ALL
SELECT 
    'Active Users (7d)',
    COUNT(DISTINCT user_id)::text,
    NULL
FROM sync_jobs
WHERE started_at > NOW() - INTERVAL '7 days'
UNION ALL
SELECT 
    'Total Syncs (24h)',
    COUNT(*)::text,
    NULL
FROM sync_jobs
WHERE started_at > NOW() - INTERVAL '24 hours'
UNION ALL
SELECT 
    'Sync Success Rate',
    ROUND(AVG(success_rate), 2)::text || '%',
    NULL
FROM analytics.sync_daily_stats
WHERE date > NOW() - INTERVAL '7 days'
UNION ALL
SELECT 
    'Auto-Resolved Conflicts',
    ROUND(AVG(auto_resolved * 100.0 / NULLIF(total_conflicts, 0)), 2)::text || '%',
    NULL
FROM analytics.conflict_stats
WHERE date > NOW() - INTERVAL '7 days';

-- Create view for cohort analysis
CREATE OR REPLACE VIEW analytics.user_cohorts AS
SELECT 
    DATE_TRUNC('month', u.created_at) as cohort_month,
    EXTRACT(DAY FROM (sj.started_at - u.created_at)) / 7 as week_number,
    COUNT(DISTINCT u.id) as users_in_cohort,
    COUNT(DISTINCT sj.user_id) as active_users,
    COUNT(DISTINCT sj.user_id) * 100.0 / COUNT(DISTINCT u.id) as retention_rate
FROM users u
LEFT JOIN sync_jobs sj ON u.id = sj.user_id
WHERE u.created_at > NOW() - INTERVAL '12 months'
GROUP BY DATE_TRUNC('month', u.created_at), EXTRACT(DAY FROM (sj.started_at - u.created_at)) / 7
ORDER BY cohort_month, week_number;

-- Grant permissions
GRANT SELECT ON ALL TABLES IN SCHEMA analytics TO calendar_app;

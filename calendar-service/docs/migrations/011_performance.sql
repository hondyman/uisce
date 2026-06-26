-- ============================================================================
-- Migration 011: Performance Optimization
-- Description: Add missing indexes and optimize slow queries for large datasets
-- ============================================================================

-- Track calendar listings by tenant (Composite Index)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_created_at 
ON calendar.calendars(tenant_id, created_at DESC);

-- Optimize blackout occurrences lookup
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_calendar_time_range 
ON calendar.blackouts(calendar_id, start_time, end_time) 
WHERE valid_to IS NULL;

-- Optimize synced internal events search by tenant and time
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_synced_internal_events_tenant_time 
ON calendar.synced_google_events(tenant_id, start_time DESC);

-- Optimize conflict detection lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sync_conflicts_tenant_status 
ON calendar.sync_conflicts(tenant_id, resolution_status, detected_at DESC);

-- Composite index for events status and time
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_tenant_status_time 
ON calendar.events(tenant_id, status, start_time DESC);

-- BRIN Index for large time-series tables (Audit logs)
-- BRIN (Block Range Index) is extremely efficient for large sequential data
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_brin_time 
ON calendar.audit_log USING BRIN (created_at) 
WITH (pages_per_range = 128);

-- Update table statistics for the query planner
ANALYZE calendar.calendars;
ANALYZE calendar.blackouts;
ANALYZE calendar.synced_google_events;
ANALYZE calendar.sync_conflicts;
ANALYZE calendar.audit_log;
ANALYZE calendar.events;

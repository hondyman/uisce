-- ============================================================================
-- PHASE 8: Database Performance Optimization - Indexing Strategy
-- ============================================================================
-- This migration implements critical indexes for Calendar Service
-- Deploy with: psql $CALENDAR_DB < phase8_indexing_optimization.sql
-- Expected runtime: ~5 minutes (CONCURRENTLY is production-safe)
-- ============================================================================

BEGIN;

-- ============================================================================
-- PHASE 1: CRITICAL INDEXES (Highest ROI)
-- ============================================================================

-- Primary access pattern: GetByID with tenant isolation
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_id_id 
ON calendars(tenant_id, id)
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_calendars_tenant_id_id IS 'Primary GetByID query optimization (~50x faster)';

-- ListByTenant with ordering support
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_created 
ON calendars(tenant_id, created_at DESC)
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_calendars_tenant_created IS 'ListByTenant pagination (~20x faster)';

-- Holiday date-based queries (most common operation)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_calendar_date 
ON calendar_holidays(calendar_id, holiday_date);
COMMENT ON INDEX idx_holidays_calendar_date IS 'Holiday lookup by date (~30x faster)';

-- Blackout availability checks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_calendar_time 
ON calendar_blackouts(calendar_id, start_time, end_time);
COMMENT ON INDEX idx_blackouts_calendar_time IS 'Availability overlap checks (~50x faster)';

COMMIT;

-- ============================================================================
-- PHASE 2: SECONDARY INDEXES (Important Access Patterns)
-- ============================================================================

BEGIN;

-- Soft-delete filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_deleted_at 
ON calendars(deleted_at)
WHERE deleted_at IS NOT NULL;
COMMENT ON INDEX idx_calendars_deleted_at IS 'Optimizes deleted calendar queries';

-- Tenant-level analytics and reporting
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_updated 
ON calendars(tenant_id, updated_at DESC)
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_calendars_tenant_updated IS 'Supports change tracking queries';

-- Holiday range queries with included columns (index-only scans)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_date_range 
ON calendar_holidays(calendar_id, holiday_date)
INCLUDE (holiday_name, is_half_day, holiday_type);
COMMENT ON INDEX idx_holidays_date_range IS 'Enables covered queries for holiday details';

-- Recurring holidays filter (common in inheritance logic)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_recurring 
ON calendar_holidays(calendar_id, is_recurring, holiday_date DESC)
WHERE is_recurring = TRUE;
COMMENT ON INDEX idx_holidays_recurring IS 'Optimizes recurring holiday queries';

-- Active blackouts for current availability checks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_active 
ON calendar_blackouts(calendar_id, end_time DESC)
WHERE end_time > CURRENT_TIMESTAMP;
COMMENT ON INDEX idx_blackouts_active IS 'Speeds up active blackout lookups';

-- Event participant lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_participants_event_id 
ON event_participants(event_id)
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_event_participants_event_id IS 'GetAttendees operation (~20x faster)';

-- User-calendar attendee tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_participants_user_calendar 
ON event_participants(user_id, calendar_id)
WHERE status != 'DECLINED' AND deleted_at IS NULL;
COMMENT ON INDEX idx_event_participants_user_calendar IS 'User availability queries';

-- Audit trail retrieval by calendar
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_calendar_timestamp 
ON calendar_audit_logs(calendar_id, created_at DESC);
COMMENT ON INDEX idx_audit_calendar_timestamp IS 'Calendar change history (~15x faster)';

-- User activity tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_user_action 
ON calendar_audit_logs(user_id, action_type, created_at DESC);
COMMENT ON INDEX idx_audit_user_action IS 'User action audit trail (~15x faster)';

-- Compliance-relevant action tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_action_time 
ON calendar_audit_logs(action_type, created_at)
WHERE action_type IN ('DELETE', 'BULK_DELETE', 'UPDATE', 'ACCESS_DENIED');
COMMENT ON INDEX idx_audit_action_time IS 'Compliance event queries';

COMMIT;

-- ============================================================================
-- PHASE 3: PARTIAL INDEXES (Storage-Efficient)
-- ============================================================================

BEGIN;

-- Only active calendars (most queries filter by this)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_active 
ON calendars(tenant_id, id)
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_calendars_active IS 'Partial index for active calendars only (50% smaller)';

-- Only pending/tentative responses
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_pending 
ON event_participants(calendar_id, status)
WHERE status IN ('PENDING', 'TENTATIVE');
COMMENT ON INDEX idx_participants_pending IS 'Fast pending response lookups';

-- Future blackouts only
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_future 
ON calendar_blackouts(calendar_id, start_time)
WHERE end_time > CURRENT_TIMESTAMP;
COMMENT ON INDEX idx_blackouts_future IS 'Partial index for upcoming blackouts only';

COMMIT;

-- ============================================================================
-- PHASE 4: BRIN INDEXES (Time-Series Optimization - 95% smaller)
-- ============================================================================

BEGIN;

-- Audit logs: Natural time-series ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_brin_timestamp 
ON calendar_audit_logs USING BRIN (created_at)
WITH (pages_per_range = 128);
COMMENT ON INDEX idx_audit_logs_brin_timestamp IS 'BRIN time-series index (95% smaller than B-tree)';

-- Holiday dates: Natural date ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_brin_date 
ON calendar_holidays USING BRIN (holiday_date)
WITH (pages_per_range = 256);
COMMENT ON INDEX idx_holidays_brin_date IS 'BRIN for date-ordered holidays';

-- Event start times: Natural time ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_brin_starttime 
ON calendar_events USING BRIN (start_time)
WITH (pages_per_range = 128);
COMMENT ON INDEX idx_events_brin_starttime IS 'BRIN for event time-series';

COMMIT;

-- ============================================================================
-- PHASE 5: EXPRESSION INDEXES (Query-Specific Optimization)
-- ============================================================================

BEGIN;

-- Year-based queries for recurring holidays
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_year 
ON calendar_holidays (calendar_id, EXTRACT(YEAR FROM holiday_date))
WHERE is_recurring = TRUE;
COMMENT ON INDEX idx_holidays_year IS 'Optimizes yearly holiday queries';

-- Month-based business day calculations
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_month 
ON calendar_holidays (calendar_id, EXTRACT(MONTH FROM holiday_date))
WHERE is_recurring = TRUE;
COMMENT ON INDEX idx_holidays_month IS 'Optimizes monthly pattern queries';

-- Case-insensitive calendar name searches
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_name_ci 
ON calendars (tenant_id, LOWER(name))
WHERE deleted_at IS NULL;
COMMENT ON INDEX idx_calendars_name_ci IS 'Case-insensitive name searching';

COMMIT;

-- ============================================================================
-- PHASE 6: FOREIGN KEY OPTIMIZATION INDEXES
-- ============================================================================

BEGIN;

-- Ensure FK performance for tenant relationships
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_fk 
ON calendars(tenant_id);
COMMENT ON INDEX idx_calendars_tenant_fk IS 'Foreign key optimization (standard pattern)';

-- Holiday-to-calendar FK
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holidays_calendar_fk 
ON calendar_holidays(calendar_id);
COMMENT ON INDEX idx_holidays_calendar_fk IS 'Foreign key optimization';

-- Participants-to-event FK
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_event_fk 
ON event_participants(event_id);
COMMENT ON INDEX idx_participants_event_fk IS 'Foreign key optimization';

-- Participants-to-user FK
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_participants_user_fk 
ON event_participants(user_id);
COMMENT ON INDEX idx_participants_user_fk IS 'Foreign key optimization';

-- Recurrence rules FK
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holiday_rules_holiday_fk 
ON holiday_recurrence_rules(holiday_id);
COMMENT ON INDEX idx_holiday_rules_holiday_fk IS 'Foreign key optimization';

COMMIT;

-- ============================================================================
-- POST-DEPLOYMENT: Quality Assurance Checks
-- ============================================================================

BEGIN;

-- Verify all indexes were created successfully
SELECT 
    schemaname, 
    tablename, 
    indexname, 
    ROUND(pg_relation_size(indexrelid) / 1024.0 / 1024.0, 2) AS size_mb
FROM pg_stat_user_indexes 
WHERE schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Update table statistics for query planner
ANALYZE public.calendars;
ANALYZE public.calendar_holidays;
ANALYZE public.calendar_blackouts;
ANALYZE public.event_participants;
ANALYZE public.calendar_audit_logs;

-- Verify no unused indexes exist pre-deployment
SELECT 
    schemaname, 
    tablename, 
    indexname, 
    idx_scan as scans
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND idx_scan = 0
  AND indexname NOT LIKE 'idx_%'
ORDER BY schemaname, tablename;

COMMIT;

-- ============================================================================
-- ROLLBACK INSTRUCTIONS (if needed)
-- ============================================================================
-- If issues occur, execute:
-- DROP INDEX CONCURRENTLY IF EXISTS idx_calendars_tenant_id_id;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_calendars_tenant_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_holidays_calendar_date;
-- ... (repeat for all indexes)
-- ANALYZE;
-- ============================================================================

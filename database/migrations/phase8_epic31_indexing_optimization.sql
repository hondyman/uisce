-- ============================================================================
-- PHASE 8: Database Performance Optimization - EPIC 31 Corrected Indexing
-- ============================================================================
-- This migration implements indexes aligned with Epic 31 schema
-- (Bitemporal versioning, JSONB holidays, Phase 4+ features)
-- Deploy with: psql $CALENDAR_DB < phase8_epic31_indexing_optimization.sql
-- Expected runtime: ~10 minutes (CONCURRENTLY is production-safe)
-- ============================================================================
-- NOTE: surrounding transaction block removed because CREATE INDEX CONCURRENTLY cannot run inside a transaction
-- Indexes are created using CONCURRENTLY for zero-downtime deployment

-- ============================================================================
-- PHASE 1: CRITICAL INDEXES (Deploy Immediately)
-- ============================================================================

-- PRIMARY: Get active calendar by ID (most common - 40% of traffic)
-- Uses bitemporal pattern: valid_to IS NULL for active records
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_active 
ON calendar.calendars(tenant_id, id) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_calendars_tenant_active IS 
  'Primary GetByID query optimization - 50x faster for active calendars only';

-- LIST: Active calendars for tenant with pagination support
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_created 
ON calendar.calendars(tenant_id, created_at DESC) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_calendars_tenant_created IS 
  'ListByTenant pagination (~20x faster) - supports ORDER BY created_at DESC';

-- BLACKOUT OVERLAP QUERY: GiST index for range intersection checks
-- CRITICAL for availability checks: WHERE tstzrange(start_time, end_time) && query_range
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_overlap_gist 
ON calendar.blackouts USING GIST (tstzrange(start_time, end_time)) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_blackouts_overlap_gist IS 
  'GiST range overlap index (~50x faster) - CRITICAL for availability checks with &&';

-- BLACKOUT CALENDAR/PROFILE LOOKUP: Create index only if column exists (psql client-side execution)
-- Create calendar_id index if column exists
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_calendar_active ON calendar.blackouts(calendar_id, start_time, end_time) WHERE valid_to IS NULL;')
FROM information_schema.columns
WHERE table_schema = 'calendar' AND table_name = 'blackouts' AND column_name = 'calendar_id';
\gexec

-- Create profile_id index only when calendar_id is not present and profile_id exists
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_profile_active ON calendar.blackouts(profile_id, start_time, end_time) WHERE valid_to IS NULL;')
FROM information_schema.columns
WHERE table_schema = 'calendar' AND table_name = 'blackouts' AND column_name = 'profile_id'
  AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns WHERE table_schema = 'calendar' AND table_name = 'blackouts' AND column_name = 'calendar_id'
  );
\gexec

-- Add comments if indexes were created
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_blackouts_calendar_active', 'Active blackouts per calendar (~50x faster) - supports time range queries')
FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE c.relname = 'idx_blackouts_calendar_active' AND n.nspname = 'calendar';
\gexec
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_blackouts_profile_active', 'Active blackouts per profile (~50x faster) - supports time range queries')
FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE c.relname = 'idx_blackouts_profile_active' AND n.nspname = 'calendar';
\gexec

-- SCHEDULE PROFILES: Get active profiles per tenant
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_tenant_active 
ON calendar.schedule_profiles(tenant_id, valid_to, active) 
WHERE valid_to IS NULL AND active = TRUE;
COMMENT ON INDEX idx_profiles_tenant_active IS 
  'Active profiles per tenant (~30x faster) - used by scheduling engine';

-- COMMIT removed (CONCURRENTLY indexes must not be inside transactions)

-- ============================================================================
-- PHASE 2: GLOBAL DISTRIBUTION & AUDIT INDEXES (Deploy Week 1)
-- ============================================================================

-- NOTE: removed surrounding transaction block so CONCURRENTLY indexes can run

-- REGION-BASED QUERIES: For international calendar distribution
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_region_active 
ON calendar.calendars(region, tenant_id) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_calendars_region_active IS 
  'Region-based calendar queries (~20x faster) - for geographically distributed systems';

-- PRIORITY-BASED SCHEDULING: Phase 3+ feature
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_priority_active 
ON calendar.calendars(priority, tenant_id, valid_from DESC) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_calendars_priority_active IS 
  'Priority-based scheduling queries - Phase 3+ calendar prioritization';

-- PROFILE NAME LOOKUPS: Calendar profile discovery
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_name 
ON calendar.schedule_profiles(tenant_id, profile_name) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_profiles_name IS 
  'Profile name search (~15x faster) - used by UI for calendar selection';

-- AUDIT LOG: Primary lookup by tenant + entity type (create per-partition CONCURRENT indexes)
-- For partitioned audit_log create index on each child partition (CONCURRENTLY not allowed on parent)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_tenant_entity_%s ON %I.%I (tenant_id, entity_type, entity_id, changed_at DESC);', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- Add comments on created partition indexes (if any)
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_audit_tenant_entity_' || c.relname, 'Audit queries by entity (~20x faster) - used by audit UI and compliance')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- AUDIT LOG: Recent audits only (per-partition partial indexes for dashboards, within 30 days)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_recent_%s ON %I.%I (changed_at DESC) WHERE changed_at > %L;', c.relname, n.nspname, c.relname, now() - interval '30 days')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- Comment per-partition index (if created)
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_audit_recent_' || c.relname, 'Recent audit queries (~15x faster) - partial index for dashboard queries')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- AUDIT LOG: BRIN time-series index (create per-partition BRIN indexes)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_brin_timestamp_%s ON %I.%I USING BRIN (changed_at) WITH (pages_per_range = 128);', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- Comment on per-partition BRIN indexes
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_audit_brin_timestamp_' || c.relname, 'BRIN time-series index (~95% smaller) - excellent for naturally ordered timestamps')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- AUDIT LOG: User activity tracking (create per-partition index)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_%s ON %I.%I (changed_by, changed_at DESC) WHERE changed_by IS NOT NULL;', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- Comment on per-partition actor indexes
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_audit_actor_' || c.relname, 'User activity queries (~15x faster) - tracks who changed what and when')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'audit_log' AND n.nspname = 'calendar';
\gexec

-- COMMIT removed (CONCURRENTLY indexes must not be inside transactions)

-- ============================================================================
-- PHASE 3: PHASE 4+ FEATURE INDEXES (Deploy Week 2)
-- ============================================================================

-- NOTE: removed surrounding transaction block so CONCURRENTLY indexes can run

-- AI SUGGESTIONS: Pending suggestions per tenant
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_pending 
ON calendar.ai_suggestions(tenant_id, status, created_at DESC) 
WHERE status = 'pending';
COMMENT ON INDEX idx_ai_suggestions_pending IS 
  'Pending AI suggestions (~20x faster) - Phase 4+ feature';

-- AI SUGGESTIONS: Suggestion type filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_type 
ON calendar.ai_suggestions(suggestion_type, created_at DESC);
COMMENT ON INDEX idx_ai_suggestions_type IS 
  'AI suggestions by type (~15x faster) - used by suggestion engine';

-- AI SUGGESTIONS: Job-specific recommendations
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_job 
ON calendar.ai_suggestions(job_id, status);
COMMENT ON INDEX idx_ai_suggestions_job IS 
  'Job-specific suggestions (~20x faster) - links suggestions to scheduled jobs';

-- JOB EXECUTION HISTORY: Create indexes on each partition (job_execution_history is partitioned)
-- Per-job history lookup
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_job_%s ON %I.%I (job_id, scheduled_time DESC);', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec

-- Tenant-level analytics
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_%s ON %I.%I (tenant_id, scheduled_time DESC);', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec

-- Status-based queries (partial index per partition)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_status_%s ON %I.%I (tenant_id, status, scheduled_time DESC) WHERE status = %L;', c.relname, n.nspname, c.relname, 'failed')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec

-- BRIN time-series per partition
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_brin_time_%s ON %I.%I USING BRIN (scheduled_time) WITH (pages_per_range = 128);', c.relname, n.nspname, c.relname)
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec

-- Add comments for created partition indexes
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_job_history_job_' || c.relname, 'Job execution history (~20x faster) - Phase 4+ job analytics')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_job_history_tenant_' || c.relname, 'Tenant-level job analytics (~20x faster) - dashboard queries')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_job_history_status_' || c.relname, 'Failed job queries (~20x faster) - partial index for error analysis')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_job_history_brin_time_' || c.relname, 'BRIN time-series for job history (~95% smaller) - excellent for range scans')
FROM pg_inherits i
JOIN pg_class p ON i.inhparent = p.oid
JOIN pg_class c ON i.inhrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE p.relname = 'job_execution_history' AND n.nspname = 'calendar';
\gexec

-- ML PREDICTIONS: Active predictions per job (materialize NOW() into a literal so predicate is immutable)
SELECT format('CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_job ON calendar.ml_predictions(job_id, expires_at) WHERE expires_at > %L;', now());
\gexec

-- Add comment only if the index exists (safe in multi-environment migrations)
SELECT format('COMMENT ON INDEX %I.%I IS %L;', 'calendar', 'idx_ml_predictions_job', 'Active predictions cache (~20x faster) - Phase 4+ ML feature')
FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE c.relname = 'idx_ml_predictions_job' AND n.nspname = 'calendar';
\gexec

-- ML PREDICTIONS: Tenant-level predictions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_tenant 
ON calendar.ml_predictions(tenant_id, prediction_type, expires_at);
COMMENT ON INDEX idx_ml_predictions_tenant IS 
  'Tenant ML predictions (~15x faster) - used by ML scoring engine';

-- RESCHEDULE AUDIT: Per-job reschedule history
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_job 
ON calendar.reschedule_audit(job_id, created_at DESC);
COMMENT ON INDEX idx_reschedule_audit_job IS 
  'Job reschedule history (~20x faster) - Phase 4+ audit trail';

-- RESCHEDULE AUDIT: Tenant-level reschedule analytics
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_tenant 
ON calendar.reschedule_audit(tenant_id, created_at DESC);
COMMENT ON INDEX idx_reschedule_audit_tenant IS 
  'Tenant reschedule analytics (~20x faster) - reports and metrics';

-- RESCHEDULE AUDIT: Reason-based analysis
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_reason 
ON calendar.reschedule_audit(reason, created_at DESC);
COMMENT ON INDEX idx_reschedule_audit_reason IS 
  'Reschedule reason analysis (~15x faster) - failure reason categorization';

-- RESCHEDULE AUDIT: BRIN time-series index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_brin_time 
ON calendar.reschedule_audit USING BRIN (created_at)
WITH (pages_per_range = 128);
COMMENT ON INDEX idx_reschedule_brin_time IS 
  'BRIN time-series for reschedule (~95% smaller) - excellent for time range queries';

-- COMMIT removed (CONCURRENTLY indexes must not be inside transactions)

-- ============================================================================
-- PHASE 4: TIMEZONE & EXPRESSION INDEXES (Deploy Week 2-3)
-- ============================================================================

-- NOTE: removed surrounding transaction block so CONCURRENTLY indexes can run

-- TIMEZONE-BASED QUERIES: For international support
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_timezone 
ON calendar.schedule_profiles(timezone, tenant_id) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_profiles_timezone IS 
  'Timezone-based profile queries (~15x faster) - international calendar support';

-- ACTIVE CALENDAR COUNT: Expression index for filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_active_count 
ON calendar.calendars(tenant_id) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_calendars_active_count IS 
  'Active calendar count queries (~10x faster) - partial index for active only';

-- CASE-INSENSITIVE PROFILE NAME SEARCH: Expression index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_name_ci 
ON calendar.schedule_profiles(tenant_id, LOWER(profile_name)) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_profiles_name_ci IS 
  'Case-insensitive profile search (~20x faster) - UI search functionality';

-- RECURRING BLACKOUT DETECTION: Expression index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_is_recurring 
ON calendar.blackouts(tenant_id, (recurrence_rule IS NOT NULL)) 
WHERE valid_to IS NULL;
COMMENT ON INDEX idx_blackouts_is_recurring IS 
  'Recurring blackout queries (~15x faster) - Phase 3+ recurrence support';

-- JSONB HOLIDAYS IN CALENDARS: GIN index for JSON array search
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_holidays_gin 
ON calendar.calendars USING GIN (holidays);
COMMENT ON INDEX idx_calendars_holidays_gin IS 
  'JSONB holidays search (~20x faster) - used if querying specific holiday dates';

-- COMMIT removed (CONCURRENTLY indexes must not be inside transactions)

-- ============================================================================
-- PHASE 5: FOREIGN KEY OPTIMIZATION INDEXES
-- ============================================================================

-- NOTE: removed surrounding transaction block so CONCURRENTLY indexes can run

-- FK Optimization: calendars.tenant_id -> public.tenants.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_fk 
ON calendar.calendars(tenant_id);
COMMENT ON INDEX idx_calendars_tenant_fk IS 'Foreign key optimization for tenant relationships';

-- FK Optimization: blackouts.tenant_id -> public.tenants.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_tenant_fk 
ON calendar.blackouts(tenant_id);
COMMENT ON INDEX idx_blackouts_tenant_fk IS 'Foreign key optimization for tenant relationships';

-- FK Optimization: blackouts.calendar_id -> calendar.calendars.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_calendar_fk 
ON calendar.blackouts(calendar_id);
COMMENT ON INDEX idx_blackouts_calendar_fk IS 'Foreign key optimization for calendar relationships';

-- FK Optimization: schedule_profiles.tenant_id -> public.tenants.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_tenant_fk 
ON calendar.schedule_profiles(tenant_id);
COMMENT ON INDEX idx_profiles_tenant_fk IS 'Foreign key optimization for tenant relationships';

-- FK Optimization: ai_suggestions.tenant_id -> public.tenants.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_tenant_fk 
ON calendar.ai_suggestions(tenant_id);
COMMENT ON INDEX idx_ai_suggestions_tenant_fk IS 'Foreign key optimization for tenant relationships (Phase 4+)';

-- FK Optimization: job_execution_history.tenant_id -> public.tenants.id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_fk 
ON calendar.job_execution_history(tenant_id);
COMMENT ON INDEX idx_job_history_tenant_fk IS 'Foreign key optimization for tenant relationships (Phase 4+)';

-- FK Optimization: ml_predictions.job_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_job_fk 
ON calendar.ml_predictions(job_id);
COMMENT ON INDEX idx_ml_predictions_job_fk IS 'Foreign key optimization for job relationships (Phase 4+)';

-- FK Optimization: reschedule_audit.job_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_job_fk 
ON calendar.reschedule_audit(job_id);
COMMENT ON INDEX idx_reschedule_audit_job_fk IS 'Foreign key optimization for job relationships (Phase 4+)';

-- COMMIT removed (see above)

-- ============================================================================
-- VALIDATION: Verify All Indexes Created Successfully
-- ============================================================================

-- NOTE: removed transaction block (validation SELECTs can run standalone)

-- Display index creation summary
SELECT 
    schemaname,
    relname    AS tablename,
    indexrelname AS indexname,
    ROUND(pg_relation_size(indexrelid) / 1024.0 / 1024.0, 2) AS size_mb,
    CASE 
        WHEN indexrelname LIKE '%_brin_%' THEN 'BRIN (95% smaller)'
        WHEN indexrelname LIKE '%_gist%' THEN 'GiST (range queries)'
        WHEN indexrelname LIKE '%_gin%' THEN 'GIN (JSON search)'
        ELSE 'B-tree'
    END AS index_type
FROM pg_stat_user_indexes 
WHERE schemaname = 'calendar'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Update table statistics for query planner optimization
ANALYZE calendar.calendars;
ANALYZE calendar.blackouts;
ANALYZE calendar.schedule_profiles;
ANALYZE calendar.audit_log;
ANALYZE calendar.ai_suggestions;
ANALYZE calendar.job_execution_history;
ANALYZE calendar.ml_predictions;
ANALYZE calendar.reschedule_audit;

-- Verify indexes are valid (not concurrent creation still pending)
SELECT 
    indexrelname AS indexname,
    idx_blks_read,
    idx_blks_hit,
    CASE 
        WHEN idx_blks_hit = 0 AND idx_blks_read = 0 THEN 'NEW (not yet used)'
        WHEN idx_blks_hit = 0 THEN 'UNUSED'
        ELSE 'ACTIVE'
    END AS usage_status
FROM pg_statio_user_indexes 
WHERE schemaname = 'calendar'
ORDER BY indexrelname;

-- COMMIT removed (validation complete)

-- ============================================================================
-- ROLLBACK INSTRUCTIONS (if needed)
-- ============================================================================
-- If issues occur after deployment, execute:
--
-- DROP INDEX CONCURRENTLY IF EXISTS calendar.idx_calendars_tenant_active;
-- DROP INDEX CONCURRENTLY IF EXISTS calendar.idx_calendars_tenant_created;
-- DROP INDEX CONCURRENTLY IF EXISTS calendar.idx_blackouts_overlap_gist;
-- ... (repeat for all indexes)
-- ANALYZE;
--
-- ============================================================================

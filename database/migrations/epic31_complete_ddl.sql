-- ============================================================================
-- EPIC 31: Holiday & Calendar Intelligence - Complete Database Schema
-- ============================================================================
-- PostgreSQL 15+ Compatible
-- Schema: calendar.*
-- Features: Bitemporal versioning, CDC-ready, Partitioned audit logs
-- Deploy: psql $DB_URL -f epic31_complete_ddl.sql
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================================
-- PHASE 1: CREATE SCHEMA
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS calendar;
COMMENT ON SCHEMA calendar IS 'Epic 31: Holiday & Calendar Intelligence (trigger-free, CDC-first)';

-- ============================================================================
-- PHASE 2: CORE TABLES
-- ============================================================================

-- Calendars: Store holiday definitions (bitemporal versioning)
CREATE TABLE IF NOT EXISTS calendar.calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    region VARCHAR(100),
    
    -- Bitemporal versioning (application-enforced, NO TRIGGERS)
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ, -- NULL = active version
    
    -- Content (holidays stored as JSONB array)
    holidays JSONB NOT NULL DEFAULT '[]',
    
    -- Priority & Global Distribution
    priority INT DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    resource_profile VARCHAR(50) DEFAULT 'standard',
    
    -- Metadata
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Audit (application-managed)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    
    -- CDC hint
    cdc_topic_hint VARCHAR(50) DEFAULT 'calendars',
    
    -- Constraints
    CONSTRAINT chk_calendars_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Schedule Profiles: Combine multiple calendars with conflict resolution
CREATE TABLE IF NOT EXISTS calendar.schedule_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    profile_name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(100) DEFAULT 'UTC',
    
    -- Conflict resolution strategy
    conflict_resolution VARCHAR(50) DEFAULT 'UNION' 
        CHECK (conflict_resolution IN ('UNION', 'INTERSECTION', 'PRIORITY')),
    
    -- Bitemporal versioning
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    
    -- Active flag for soft delete
    active BOOLEAN DEFAULT TRUE,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    
    CONSTRAINT chk_profiles_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Profile-Calendars Mapping: Links calendars to profiles with priority weights
CREATE TABLE IF NOT EXISTS calendar.profile_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES calendar.schedule_profiles(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendar.calendars(id) ON DELETE CASCADE,
    
    -- Multiplier for conflict resolution (higher = more important)
    weight INTEGER DEFAULT 100 CHECK (weight BETWEEN 1 AND 1000),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(profile_id, calendar_id)
);

-- Blackouts: Time ranges when nothing should run (supports recurrence)
CREATE TABLE IF NOT EXISTS calendar.blackouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES calendar.schedule_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Time range (UTC)
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    
    -- Recurrence support (RRULE format, NULL = one-time)
    recurrence_rule VARCHAR(255),
    recurrence_timezone VARCHAR(100) DEFAULT 'UTC',
    recurrence_end TIMESTAMPTZ,
    is_recurring BOOLEAN GENERATED ALWAYS AS (recurrence_rule IS NOT NULL) STORED,
    
    -- Metadata
    reason VARCHAR(100),
    severity VARCHAR(50) DEFAULT 'NORMAL' 
        CHECK (severity IN ('CRITICAL', 'HIGH', 'NORMAL', 'LOW')),
    
    -- Bitemporal versioning
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    
    CONSTRAINT chk_blackout_range CHECK (end_time > start_time),
    CONSTRAINT chk_blackout_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Audit Log: Track all changes for compliance (PARTITIONED by month)
CREATE TABLE IF NOT EXISTS calendar.audit_log (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL 
        CHECK (entity_type IN ('calendar', 'profile', 'blackout', 'job')),
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL 
        CHECK (action IN ('CREATE', 'UPDATE', 'DELETE')),
    
    -- Change details
    old_values JSONB,
    new_values JSONB,
    
    -- Actor & context
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    reason TEXT,
    
    -- Partition key
    partition_date DATE NOT NULL,

    -- PRIMARY KEY must include partition key for partitioned tables
    PRIMARY KEY (partition_date, id)
) PARTITION BY RANGE (partition_date);

-- Create initial partitions for audit_log
CREATE TABLE IF NOT EXISTS calendar.audit_log_2026_q1 
    PARTITION OF calendar.audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

CREATE TABLE IF NOT EXISTS calendar.audit_log_2026_q2 
    PARTITION OF calendar.audit_log
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

CREATE TABLE IF NOT EXISTS calendar.audit_log_2026_q3 
    PARTITION OF calendar.audit_log
    FOR VALUES FROM ('2026-07-01') TO ('2026-10-01');

CREATE TABLE IF NOT EXISTS calendar.audit_log_2026_q4 
    PARTITION OF calendar.audit_log
    FOR VALUES FROM ('2026-10-01') TO ('2027-01-01');

-- Jobs Table: Scheduling targets with calendar awareness
CREATE TABLE IF NOT EXISTS calendar.jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    
    -- Scheduling
    schedule_profile_id UUID REFERENCES calendar.schedule_profiles(id),
    next_run TIMESTAMPTZ,
    last_run TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'pending' 
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'rescheduled')),
    
    -- Calendar awareness
    calendar_aware BOOLEAN DEFAULT TRUE,
    
    -- Priority & Region (for global distribution)
    priority INT DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    region VARCHAR(50) DEFAULT 'us-west',
    resource_profile VARCHAR(50) DEFAULT 'standard',
    sla_deadline TIMESTAMPTZ,
    
    -- Metadata
    config JSONB DEFAULT '{}',
    schedule VARCHAR(100),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_jobs_priority CHECK (priority BETWEEN 1 AND 10)
);

-- ============================================================================
-- PHASE 3: PHASE 4+ FEATURE TABLES
-- ============================================================================

-- AI Suggestions: Store AI-generated recommendations pending approval
CREATE TABLE IF NOT EXISTS calendar.ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    job_id UUID,
    suggestion_type VARCHAR(50) NOT NULL 
        CHECK (suggestion_type IN ('holiday', 'blackout', 'reschedule', 'test_plan')),
    
    -- Generated content
    data JSONB NOT NULL,
    
    -- AI context
    prompt TEXT,
    raw_response TEXT,
    model_used VARCHAR(100),
    
    -- Lifecycle
    status VARCHAR(20) DEFAULT 'pending' 
        CHECK (status IN ('pending', 'approved', 'rejected', 'applied')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Review metadata
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- Audit
    created_by UUID,
    applied_at TIMESTAMPTZ,
    
    -- Audit constraint
    CONSTRAINT chk_suggestion_status CHECK (
        (status = 'pending' AND reviewed_by IS NULL) OR
        (status != 'pending' AND reviewed_by IS NOT NULL)
    )
);

-- Job Execution History: Track job runs for AI analysis (PARTITIONED)
CREATE TABLE IF NOT EXISTS calendar.job_execution_history (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    job_id UUID NOT NULL,
    scheduled_time TIMESTAMPTZ NOT NULL,
    actual_start_time TIMESTAMPTZ,
    actual_end_time TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL 
        CHECK (status IN ('success', 'failed', 'rescheduled', 'skipped', 'timeout')),
    delay_reason VARCHAR(100),
    error_message TEXT,
    cpu_seconds FLOAT,
    memory_mb FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- PRIMARY KEY must include partition key for partitioned tables
    PRIMARY KEY (scheduled_time, id)
) PARTITION BY RANGE (scheduled_time);

-- Create initial partitions for job_execution_history
CREATE TABLE IF NOT EXISTS calendar.job_history_2026_q1 
    PARTITION OF calendar.job_execution_history
    FOR VALUES FROM ('2026-01-01 00:00:00+00'::timestamptz) TO ('2026-04-01 00:00:00+00'::timestamptz);

CREATE TABLE IF NOT EXISTS calendar.job_history_2026_q2 
    PARTITION OF calendar.job_execution_history
    FOR VALUES FROM ('2026-04-01 00:00:00+00'::timestamptz) TO ('2026-07-01 00:00:00+00'::timestamptz);

-- ML Predictions: Cache ML model predictions
CREATE TABLE IF NOT EXISTS calendar.ml_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    job_id UUID NOT NULL,
    prediction_type VARCHAR(50) NOT NULL,
    prediction_result JSONB NOT NULL,
    confidence FLOAT NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reschedule Audit: Track rescheduling events
CREATE TABLE IF NOT EXISTS calendar.reschedule_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    job_id UUID NOT NULL,
    original_time TIMESTAMPTZ NOT NULL,
    new_time TIMESTAMPTZ NOT NULL,
    reason VARCHAR(255),
    triggered_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- External Sync Config: External holiday API integration
CREATE TABLE IF NOT EXISTS calendar.external_sync_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendar.calendars(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    country_code VARCHAR(2) NOT NULL,
    api_key VARCHAR(255),
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_frequency VARCHAR(20) DEFAULT 'monthly',
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- External Sync Logs: Track sync operations
CREATE TABLE IF NOT EXISTS calendar.external_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES calendar.external_sync_config(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    holidays_added INT DEFAULT 0,
    holidays_updated INT DEFAULT 0,
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- PHASE 4: ROW-LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE calendar.calendars ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.schedule_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.profile_calendars ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.blackouts ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.ai_suggestions ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.job_execution_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.ml_predictions ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.reschedule_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.external_sync_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.external_sync_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies (tenant isolation)
DROP POLICY IF EXISTS calendars_tenant_isolation ON calendar.calendars;
CREATE POLICY calendars_tenant_isolation ON calendar.calendars
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS profiles_tenant_isolation ON calendar.schedule_profiles;
CREATE POLICY profiles_tenant_isolation ON calendar.schedule_profiles
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS blackouts_tenant_isolation ON calendar.blackouts;
CREATE POLICY blackouts_tenant_isolation ON calendar.blackouts
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS audit_tenant_isolation ON calendar.audit_log;
CREATE POLICY audit_tenant_isolation ON calendar.audit_log
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS jobs_tenant_isolation ON calendar.jobs;
CREATE POLICY jobs_tenant_isolation ON calendar.jobs
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS ai_suggestions_tenant_isolation ON calendar.ai_suggestions;
CREATE POLICY ai_suggestions_tenant_isolation ON calendar.ai_suggestions
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS job_history_tenant_isolation ON calendar.job_execution_history;
CREATE POLICY job_history_tenant_isolation ON calendar.job_execution_history
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS ml_predictions_tenant_isolation ON calendar.ml_predictions;
CREATE POLICY ml_predictions_tenant_isolation ON calendar.ml_predictions
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

DROP POLICY IF EXISTS reschedule_audit_tenant_isolation ON calendar.reschedule_audit;
CREATE POLICY reschedule_audit_tenant_isolation ON calendar.reschedule_audit
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- ============================================================================
-- PHASE 5: VIEWS (for convenience)
-- ============================================================================

-- Active calendars only
CREATE OR REPLACE VIEW calendar.active_calendars AS
SELECT * FROM calendar.calendars WHERE valid_to IS NULL;

-- Active profiles only
CREATE OR REPLACE VIEW calendar.active_profiles AS
SELECT * FROM calendar.schedule_profiles 
WHERE valid_to IS NULL AND active = TRUE;

-- Profile with resolved calendar IDs
CREATE OR REPLACE VIEW calendar.profile_calendar_summary AS
SELECT 
    sp.id AS profile_id,
    sp.tenant_id,
    sp.profile_name,
    sp.timezone,
    sp.conflict_resolution,
    ARRAY_AGG(pc.calendar_id) AS calendar_ids,
    ARRAY_AGG(pc.weight) AS weights
FROM calendar.active_profiles sp
LEFT JOIN calendar.profile_calendars pc ON sp.id = pc.profile_id
GROUP BY sp.id, sp.tenant_id, sp.profile_name, sp.timezone, sp.conflict_resolution;

-- ============================================================================
-- PHASE 6: INDEX OPTIMIZATIONS (Phase 8 - DBeaver Compatible)
-- NO TRANSACTION BLOCKS around CREATE INDEX CONCURRENTLY
-- ============================================================================

-- CRITICAL INDEXES (Phase 1)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_active 
ON calendar.calendars(tenant_id, id) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_created 
ON calendar.calendars(tenant_id, created_at DESC) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_overlap_gist 
ON calendar.blackouts USING GIST (tstzrange(start_time, end_time)) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_profile_active 
ON calendar.blackouts(profile_id, start_time, end_time) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_tenant_active 
ON calendar.schedule_profiles(tenant_id, valid_to, active) 
WHERE valid_to IS NULL AND active = TRUE;

-- GLOBAL DISTRIBUTION & AUDIT INDEXES (Phase 2)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_region_active 
ON calendar.calendars(region, tenant_id) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_priority_active 
ON calendar.calendars(priority, tenant_id, valid_from DESC) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_name 
ON calendar.schedule_profiles(tenant_id, profile_name) 
WHERE valid_to IS NULL;

-- AUDIT LOG: create indexes directly on each partition (CONCURRENTLY allowed per partition)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_tenant_entity_2026_q1 ON calendar.audit_log_2026_q1(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_recent_2026_q1 ON calendar.audit_log_2026_q1(changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_brin_timestamp_2026_q1 ON calendar.audit_log_2026_q1 USING BRIN (changed_at) WITH (pages_per_range = 128);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_2026_q1 ON calendar.audit_log_2026_q1(changed_by, changed_at DESC) WHERE changed_by IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_tenant_entity_2026_q2 ON calendar.audit_log_2026_q2(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_recent_2026_q2 ON calendar.audit_log_2026_q2(changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_brin_timestamp_2026_q2 ON calendar.audit_log_2026_q2 USING BRIN (changed_at) WITH (pages_per_range = 128);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_2026_q2 ON calendar.audit_log_2026_q2(changed_by, changed_at DESC) WHERE changed_by IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_tenant_entity_2026_q3 ON calendar.audit_log_2026_q3(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_recent_2026_q3 ON calendar.audit_log_2026_q3(changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_brin_timestamp_2026_q3 ON calendar.audit_log_2026_q3 USING BRIN (changed_at) WITH (pages_per_range = 128);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_2026_q3 ON calendar.audit_log_2026_q3(changed_by, changed_at DESC) WHERE changed_by IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_tenant_entity_2026_q4 ON calendar.audit_log_2026_q4(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_recent_2026_q4 ON calendar.audit_log_2026_q4(changed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_brin_timestamp_2026_q4 ON calendar.audit_log_2026_q4 USING BRIN (changed_at) WITH (pages_per_range = 128);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_2026_q4 ON calendar.audit_log_2026_q4(changed_by, changed_at DESC) WHERE changed_by IS NOT NULL;

-- PHASE 4+ FEATURE INDEXES (Phase 3)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_pending 
ON calendar.ai_suggestions(tenant_id, status, created_at DESC) 
WHERE status = 'pending';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_type 
ON calendar.ai_suggestions(suggestion_type, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_job 
ON calendar.ai_suggestions(job_id, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_job_2026_q1 ON calendar.job_history_2026_q1(job_id, scheduled_time DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_2026_q1 ON calendar.job_history_2026_q1(tenant_id, scheduled_time DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_status_2026_q1 ON calendar.job_history_2026_q1(tenant_id, status, scheduled_time DESC) WHERE status = 'failed';
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_brin_time_2026_q1 ON calendar.job_history_2026_q1 USING BRIN (scheduled_time) WITH (pages_per_range = 128);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_job_2026_q2 ON calendar.job_history_2026_q2(job_id, scheduled_time DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_2026_q2 ON calendar.job_history_2026_q2(tenant_id, scheduled_time DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_status_2026_q2 ON calendar.job_history_2026_q2(tenant_id, status, scheduled_time DESC) WHERE status = 'failed';
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_brin_time_2026_q2 ON calendar.job_history_2026_q2 USING BRIN (scheduled_time) WITH (pages_per_range = 128);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_job 
ON calendar.ml_predictions(job_id, expires_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_tenant 
ON calendar.ml_predictions(tenant_id, prediction_type, expires_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_job 
ON calendar.reschedule_audit(job_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_tenant 
ON calendar.reschedule_audit(tenant_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_reason 
ON calendar.reschedule_audit(reason, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_brin_time 
ON calendar.reschedule_audit USING BRIN (created_at)
WITH (pages_per_range = 128);

-- TIMEZONE & EXPRESSION INDEXES (Phase 4)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_timezone 
ON calendar.schedule_profiles(timezone, tenant_id) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_active_count 
ON calendar.calendars(tenant_id) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_name_ci 
ON calendar.schedule_profiles(tenant_id, LOWER(profile_name)) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_is_recurring 
ON calendar.blackouts(tenant_id, (recurrence_rule IS NOT NULL)) 
WHERE valid_to IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_holidays_gin 
ON calendar.calendars USING GIN (holidays);

-- FOREIGN KEY OPTIMIZATION INDEXES (Phase 5)

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendars_tenant_fk 
ON calendar.calendars(tenant_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_tenant_fk 
ON calendar.blackouts(tenant_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_blackouts_profile_fk 
ON calendar.blackouts(profile_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_profiles_tenant_fk 
ON calendar.schedule_profiles(tenant_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_suggestions_tenant_fk 
ON calendar.ai_suggestions(tenant_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_fk_2026_q1 ON calendar.job_history_2026_q1(tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_job_history_tenant_fk_2026_q2 ON calendar.job_history_2026_q2(tenant_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ml_predictions_job_fk 
ON calendar.ml_predictions(job_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reschedule_audit_job_fk 
ON calendar.reschedule_audit(job_id);

-- ============================================================================
-- PHASE 7: VALIDATION (CAN run in transaction)
-- ============================================================================

BEGIN;

-- Display index creation summary
SELECT 
    schemaname,
    relname   AS tablename,
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

-- Update table statistics
ANALYZE calendar.calendars;
ANALYZE calendar.blackouts;
ANALYZE calendar.schedule_profiles;
ANALYZE calendar.audit_log;
ANALYZE calendar.ai_suggestions;
ANALYZE calendar.job_execution_history;
ANALYZE calendar.ml_predictions;
ANALYZE calendar.reschedule_audit;

-- Verify indexes are valid
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

COMMIT;

-- ============================================================================
-- DEPLOYMENT COMPLETE
-- ============================================================================

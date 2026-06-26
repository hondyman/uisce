-- Epic 31: Holiday & Calendar Intelligence - Database Schema
-- Phase 3 Deployment (Simplified without optional extensions)
-- Deploy this on remote PostgreSQL instance
-- Architecture: CDC-First, Trigger-Free, Cache-Backed

-- ===========================
-- EXTENSIONS & SETUP
-- ===========================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===========================
-- TENANTS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    allowed_regions TEXT[] DEFAULT '{"us-east-1"}',
    data_residency_policy VARCHAR(50) DEFAULT 'strict',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_name ON tenants(name);

-- ===========================
-- CALENDARS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    logical_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    region VARCHAR(100),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    holidays JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    cdc_topic_hint VARCHAR(50) DEFAULT 'calendars',
    CONSTRAINT chk_calendars_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_calendars_active ON calendars(tenant_id, valid_to) WHERE valid_to IS NULL;
CREATE INDEX idx_calendars_region_active ON calendars(region, tenant_id, valid_to) WHERE valid_to IS NULL;
CREATE INDEX idx_calendars_logical ON calendars(logical_id, valid_from DESC);
CREATE INDEX idx_calendars_logical_tenant ON calendars(tenant_id, logical_id, valid_from DESC) WHERE valid_to IS NULL;
CREATE INDEX idx_calendars_holidays ON calendars USING GIN (holidays);
CREATE INDEX idx_calendars_tags ON calendars USING GIN (tags);

ALTER TABLE calendars ENABLE ROW LEVEL SECURITY;
CREATE POLICY calendars_tenant_isolation ON calendars
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- SCHEDULE PROFILES TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS schedule_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(100) DEFAULT 'UTC',
    conflict_resolution VARCHAR(50) DEFAULT 'UNION',
    priority_calendars UUID[] DEFAULT '{}',
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    CONSTRAINT chk_profiles_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_profiles_tenant_active ON schedule_profiles(tenant_id, valid_to) WHERE valid_to IS NULL;
ALTER TABLE schedule_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY profiles_tenant_isolation ON schedule_profiles
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- PROFILE CALENDARS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS profile_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    weight INTEGER DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, calendar_id)
);

CREATE INDEX idx_profile_calendars_composite ON profile_calendars(profile_id, calendar_id, weight DESC);

-- ===========================
-- BLACKOUTS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS blackouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    reason VARCHAR(100),
    severity VARCHAR(50) DEFAULT 'NORMAL',
    recurrence_rule TEXT,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    CONSTRAINT chk_blackout_range CHECK (end_time > start_time),
    CONSTRAINT chk_blackout_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_blackouts_active_range ON blackouts(tenant_id, profile_id, start_time, end_time) WHERE valid_to IS NULL;
CREATE INDEX idx_blackouts_overlap_query ON blackouts USING GIST (tsrange(start_time, end_time)) WHERE valid_to IS NULL;

ALTER TABLE blackouts ENABLE ROW LEVEL SECURITY;
CREATE POLICY blackouts_tenant_isolation ON blackouts
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- AUDIT LOG (Partitioned)
-- ===========================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    reason TEXT,
    partition_date DATE GENERATED ALWAYS AS (DATE(changed_at)) STORED
) PARTITION BY RANGE (partition_date);

CREATE TABLE IF NOT EXISTS audit_log_2026_q1 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

CREATE INDEX idx_audit_lookup ON audit_log(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX idx_audit_recent ON audit_log(changed_at DESC) WHERE changed_at > NOW() - INTERVAL '30 days';

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_tenant_isolation ON audit_log
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- JOBS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    schedule_profile_id UUID REFERENCES schedule_profiles(id),
    next_run TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'pending',
    calendar_aware BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jobs_tenant_status ON jobs(tenant_id, status);
CREATE INDEX idx_jobs_next_run ON jobs(next_run) WHERE status = 'pending';
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY jobs_tenant_isolation ON jobs
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- EXTERNAL CALENDAR CONNECTIONS
-- ===========================

CREATE TABLE IF NOT EXISTS external_calendar_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_email VARCHAR(255),
    provider_calendar_id VARCHAR(500),
    access_token_encrypted BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    encryption_key_version VARCHAR(50),
    last_rotation_at TIMESTAMPTZ,
    sync_enabled BOOLEAN DEFAULT TRUE,
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(50),
    sync_error_details TEXT,
    sync_interval_minutes INTEGER DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_external_calendars_tenant ON external_calendar_connections(tenant_id);
CREATE INDEX idx_external_calendars_sync_due ON external_calendar_connections(last_sync_at, sync_enabled) WHERE sync_enabled = TRUE;
ALTER TABLE external_calendar_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY external_calendars_tenant_isolation ON external_calendar_connections
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- ANALYTICS TABLES
-- ===========================

CREATE TABLE IF NOT EXISTS calendar_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    metric_date DATE NOT NULL,
    holiday_count INTEGER DEFAULT 0,
    blackout_hours INTEGER DEFAULT 0,
    total_blocked_hours INTEGER DEFAULT 0,
    affected_job_count INTEGER DEFAULT 0,
    reschedule_count INTEGER DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, calendar_id, metric_date),
    partition_date DATE GENERATED ALWAYS AS (metric_date) STORED
) PARTITION BY RANGE (partition_date);

CREATE TABLE IF NOT EXISTS calendar_metrics_2026_q1 PARTITION OF calendar_metrics
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

CREATE INDEX idx_calendar_metrics_tenant_date ON calendar_metrics(tenant_id, metric_date DESC);
CREATE INDEX idx_calendar_metrics_calendar ON calendar_metrics(calendar_id, metric_date DESC);

-- ===========================
-- TEST DATA
-- ===========================

-- Insert test tenant
DELETE FROM tenants WHERE id = '550e8400-e29b-41d4-a716-446655440000';
INSERT INTO tenants (id, name, allowed_regions, data_residency_policy)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'Test Tenant',
    '{"us-east-1"}',
    'strict'
);

-- Insert test calendar
INSERT INTO calendars (tenant_id, name, region, holidays)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'USA Federal Holidays',
    'US',
    jsonb_build_array(
        jsonb_build_object('date', '2026-01-01', 'name', 'New Year', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-07-04', 'name', 'Independence Day', 'severity', 'HIGH'),
        jsonb_build_object('date', '2026-12-25', 'name', 'Christmas', 'severity', 'HIGH')
    )
);

-- Insert test schedule profile
INSERT INTO schedule_profiles (tenant_id, name, timezone, conflict_resolution)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'default',
    'UTC',
    'UNION'
);

-- Get the IDs for linking
DO $$
DECLARE
    test_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000';
    calendar_id UUID;
    profile_id UUID;
BEGIN
    -- Get calendar ID
    SELECT id INTO calendar_id FROM calendars WHERE tenant_id = test_tenant_id AND valid_to IS NULL LIMIT 1;
    
    -- Get profile ID
    SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND valid_to IS NULL LIMIT 1;
    
    -- Link calendar to profile if both exist
    IF calendar_id IS NOT NULL AND profile_id IS NOT NULL THEN
        INSERT INTO profile_calendars (profile_id, calendar_id, weight)
        VALUES (profile_id, calendar_id, 100)
        ON CONFLICT (profile_id, calendar_id) DO NOTHING;
    END IF;
END $$;

-- Insert test blackouts (both recurring and one-time)
DO $$
DECLARE
    test_tenant_id UUID := '550e8400-e29b-41d4-a716-446655440000';
    profile_id UUID;
BEGIN
    -- Get profile ID
    SELECT id INTO profile_id FROM schedule_profiles WHERE tenant_id = test_tenant_id AND valid_to IS NULL LIMIT 1;
    
    IF profile_id IS NOT NULL THEN
        -- One-time blackout (Maintenance window)
        INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity)
        VALUES (
            test_tenant_id,
            profile_id,
            'Monthly Maintenance',
            'Scheduled database maintenance',
            '2026-02-20 02:00:00+00',
            '2026-02-20 04:00:00+00',
            'MAINTENANCE',
            'HIGH'
        );
        
        -- Recurring blackout (Every Monday 11 PM - 1 AM UTC for 52 weeks)
        INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
        VALUES (
            test_tenant_id,
            profile_id,
            'Weekly Batch Job',
            'Recurring batch processing window',
            '2026-02-23 23:00:00+00',
            '2026-02-24 01:00:00+00',
            'PLANNED_DOWNTIME',
            'MEDIUM',
            'FREQ=WEEKLY;BYDAY=MO;COUNT=52'
        );
        
        -- Another recurring blackout (Every Friday 3 PM - 5 PM UTC)
        INSERT INTO blackouts (tenant_id, profile_id, name, description, start_time, end_time, reason, severity, recurrence_rule)
        VALUES (
            test_tenant_id,
            profile_id,
            'Weekly Deployment Window',
            'Scheduled deployments every Friday afternoon',
            '2026-02-20 15:00:00+00',
            '2026-02-20 17:00:00+00',
            'PLANNED_DOWNTIME',
            'LOW',
            'FREQ=WEEKLY;BYDAY=FR;COUNT=52'
        );
    END IF;
END $$;

-- Verify schema was created
SELECT 'Schema deployed successfully - Tables created:' as status;
SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('calendars', 'schedule_profiles', 'profile_calendars', 'blackouts', 'audit_log', 'jobs') ORDER BY tablename;

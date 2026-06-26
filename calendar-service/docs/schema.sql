-- Epic 31: Holiday & Calendar Intelligence - Database Schema
-- Deploy this on remote PostgreSQL instance
-- Architecture: CDC-First, Trigger-Free, Cache-Backed
-- Audit logging: explicit application responsibility (no DB triggers)
-- Caching: Redis via CDC invalidation
-- Analytics: optional StarRocks integration

-- ===========================
-- EXTENSIONS & SETUP
-- ===========================

CREATE EXTENSION IF NOT EXISTS uuid-ossp;
CREATE EXTENSION IF NOT EXISTS pgtle; -- For KMS integration (optional)


-- ===========================
-- TENANTS TABLE
-- ===========================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    allowed_regions TEXT[] DEFAULT '{"us-east-1"}',
    data_residency_policy VARCHAR(50) DEFAULT 'strict', -- 'strict' | 'preferred'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_name ON tenants(name);

-- ===========================
-- CORE TABLES (OPTIMIZED)
-- ===========================

-- Calendars: Store holiday definitions and blackout schedules
-- NOTE: Application layer responsible for audit logging (no DB triggers)
-- CDC: Debezium auto-captures changes → Redpanda for downstream consumption
CREATE TABLE IF NOT EXISTS calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    logical_id UUID, -- Tracks versions of same logical calendar across bitemporal updates
    name VARCHAR(255) NOT NULL,
    description TEXT,
    region VARCHAR(100),
    -- Bitemporal versioning (application-enforced)
    -- NULL = active version; non-NULL = deprecated version
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    -- Content
    holidays JSONB NOT NULL DEFAULT '[]', -- Array of {date, name, severity}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    -- Metadata
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    -- CDC hint for topic routing (optional)
    cdc_topic_hint VARCHAR(50) DEFAULT 'calendars',
    -- Constraints
    CONSTRAINT chk_calendars_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Optimized indexes for common query patterns
CREATE INDEX idx_calendars_active ON calendars(tenant_id, valid_to) 
    WHERE valid_to IS NULL; -- Active version lookup
CREATE INDEX idx_calendars_region_active ON calendars(region, tenant_id, valid_to) 
    WHERE valid_to IS NULL; -- Region filter
CREATE INDEX idx_calendars_logical ON calendars(logical_id, valid_from DESC); 
    -- Version history lookup by logical calendar
CREATE INDEX idx_calendars_logical_tenant ON calendars(tenant_id, logical_id, valid_from DESC) 
    WHERE valid_to IS NULL; -- Active version of logical calendar
CREATE INDEX idx_calendars_holidays ON calendars USING GIN (holidays); 
    -- JSONB content queries
CREATE INDEX idx_calendars_tags ON calendars USING GIN (tags); 
    -- Tag-based filtering

-- Row-Level Security (enforced via Hasura session variable)
ALTER TABLE calendars ENABLE ROW LEVEL SECURITY;
CREATE POLICY calendars_tenant_isolation ON calendars
    USING (tenant_id = current_setting('request.tenant_id')::uuid);
COMMENT ON POLICY calendars_tenant_isolation ON calendars IS 
    'Enforced via Hasura X-Hasura-Tenant-Id session variable';

-- Schedule Profiles: Combine multiple calendars with conflict resolution rules
CREATE TABLE IF NOT EXISTS schedule_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(100) DEFAULT 'UTC',
    -- How to combine multiple calendars
    conflict_resolution VARCHAR(50) DEFAULT 'UNION', -- UNION, INTERSECTION, PRIORITY
    priority_calendars UUID[] DEFAULT '{}', -- For PRIORITY mode
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    CONSTRAINT chk_profiles_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_profiles_tenant_active ON schedule_profiles(tenant_id, valid_to) 
    WHERE valid_to IS NULL;
ALTER TABLE schedule_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY profiles_tenant_isolation ON schedule_profiles
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- Profile-Calendar Mapping: Links calendars to profiles
CREATE TABLE IF NOT EXISTS profile_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    -- Multiplier for conflict resolution (higher priority)
    weight INTEGER DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, calendar_id)
);

CREATE INDEX idx_profile_calendars_composite ON profile_calendars(profile_id, calendar_id, weight DESC);

-- Blackouts: Time ranges when nothing should run
-- Optimized for range queries (most common operation: "is this time available?")
CREATE TABLE IF NOT EXISTS blackouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    reason VARCHAR(100), -- MAINTENANCE, PLANNED_DOWNTIME, INCIDENT, MANUAL, etc.
    severity VARCHAR(50) DEFAULT 'NORMAL', -- CRITICAL, HIGH, NORMAL, LOW
    -- Bitemporal
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    -- Constraints
    CONSTRAINT chk_blackout_range CHECK (end_time > start_time),
    CONSTRAINT chk_blackout_valid_range CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Optimized indexes for availability checks (critical query: "what time slots are available?")
CREATE INDEX idx_blackouts_active_range ON blackouts(tenant_id, profile_id, start_time, end_time) 
    WHERE valid_to IS NULL;
CREATE INDEX idx_blackouts_overlap_query ON blackouts 
    USING GIST (tsrange(start_time, end_time)) 
    WHERE valid_to IS NULL; -- GiST for efficient overlap detection
    
ALTER TABLE blackouts ENABLE ROW LEVEL SECURITY;
CREATE POLICY blackouts_tenant_isolation ON blackouts
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- External Calendar Connections (Phase 5)
-- NOTE: Tokens encrypted at application layer using KMS (AWS KMS, HashiCorp Vault)
CREATE TABLE IF NOT EXISTS external_calendar_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- GOOGLE, OUTLOOK, CALDAV, etc.
    provider_email VARCHAR(255),
    provider_calendar_id VARCHAR(500),
    -- OAuth tokens (encrypted at application layer)
    access_token_encrypted BYTEA,
    refresh_token_encrypted BYTEA,
    token_expires_at TIMESTAMPTZ,
    -- KMS key tracking
    encryption_key_version VARCHAR(50),
    last_rotation_at TIMESTAMPTZ,
    -- Sync configuration
    sync_enabled BOOLEAN DEFAULT TRUE,
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(50), -- SUCCESS, FAILED, IN_PROGRESS
    sync_error_details TEXT, -- For debugging failed syncs
    sync_interval_minutes INTEGER DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_external_calendars_tenant ON external_calendar_connections(tenant_id);
CREATE INDEX idx_external_calendars_sync_due ON external_calendar_connections(last_sync_at, sync_enabled) 
    WHERE sync_enabled = TRUE;
ALTER TABLE external_calendar_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY external_calendars_tenant_isolation ON external_calendar_connections
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- AUDIT LOG (Partitioned, CDC-Friendly)
-- ===========================
-- NOTE: No DB triggers. Application inserts audit entries explicitly after successful mutations
-- CDC: Debezium auto-captures audit_log changes for compliance/analytics

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL, -- CALENDAR, PROFILE, BLACKOUT, etc.
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- CREATE, UPDATE, DELETE
    old_values JSONB, -- Captured only for UPDATE/DELETE
    new_values JSONB, -- Captured only for CREATE/UPDATE
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    reason TEXT,
    -- Partition key (helps with monthly retention policies)
    partition_date DATE GENERATED ALWAYS AS (DATE(changed_at)) STORED
) PARTITION BY RANGE (partition_date);

-- Initial partitions (Q1 2026)
CREATE TABLE IF NOT EXISTS audit_log_2026_q1 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

-- Efficient lookup indexes
CREATE INDEX idx_audit_lookup ON audit_log(tenant_id, entity_type, entity_id, changed_at DESC);
CREATE INDEX idx_audit_recent ON audit_log(changed_at DESC) WHERE changed_at > NOW() - INTERVAL '30 days';

-- RLS
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_tenant_isolation ON audit_log
    USING (tenant_id = current_setting('request.tenant_id')::uuid);


-- ===========================
-- JOBS TABLE (For Scheduling & Metrics)
-- ===========================

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    schedule_profile_id UUID REFERENCES schedule_profiles(id),
    next_run TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, completed, failed, rescheduled
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
-- INTERNAL EVENTS TABLE
-- ===========================
-- Centralized table for internal calendar events
-- CDC captures changes → triggers sync to external providers (Google/MS)
CREATE TABLE IF NOT EXISTS internal_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    location TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    is_all_day BOOLEAN DEFAULT FALSE,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_internal_events_tenant_user ON internal_events(tenant_id, user_id);
ALTER TABLE internal_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY internal_events_tenant_isolation ON internal_events
    USING (tenant_id = current_setting('request.tenant_id')::uuid);

-- ===========================
-- ANALYTICS TABLES
-- ===========================

-- Calendar Metrics: Aggregated impact metrics (partitioned for scale)
CREATE TABLE IF NOT EXISTS calendar_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    metric_date DATE NOT NULL,
    -- Metrics
    holiday_count INTEGER DEFAULT 0,
    blackout_hours INTEGER DEFAULT 0,
    total_blocked_hours INTEGER DEFAULT 0,
    affected_job_count INTEGER DEFAULT 0,
    reschedule_count INTEGER DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, calendar_id, metric_date),
    -- Partition key
    partition_date DATE GENERATED ALWAYS AS (metric_date) STORED
) PARTITION BY RANGE (partition_date);

-- Initial partition (Q1 2026)
CREATE TABLE IF NOT EXISTS calendar_metrics_2026_q1 PARTITION OF calendar_metrics
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

-- Indexes for analytics queries
CREATE INDEX idx_calendar_metrics_tenant_date ON calendar_metrics(tenant_id, metric_date DESC);
CREATE INDEX idx_calendar_metrics_calendar ON calendar_metrics(calendar_id, metric_date DESC);

-- ===========================
-- CACHING STRATEGY (APPLICATION-LAYER)
-- ===========================
-- Use Redis with CDCinvalidation:
--   Key format: cache:resolved:{tenant_id}:{profile_id}
--   Value: JSON of merged holidays + blackouts
--   TTL: 3600s (1 hour)
--   Invalidation: CDC consumer listens to calendars/blackouts topics → invalidates key
--
-- Example Redis key structure:
--   cache:resolved:550e8400-e29b-41d4-a716-446655440000:default
--   {
--     "profile_id": "...",
--     "timezone": "UTC",
--     "merged_holidays": [...],
--     "active_blackouts": [...],
--     "cached_at": 1707042000,
--     "expires_at": 1707045600
--   }

-- ===========================
-- EXAMPLE DATA (for testing)
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

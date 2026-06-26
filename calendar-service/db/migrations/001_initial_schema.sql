-- 001_initial_schema.sql - Core Calendar Service schema
-- Created: 2026-02-17

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search on profile names

-- Tenants table - Multi-tenant isolation
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    region VARCHAR(50) NOT NULL DEFAULT 'us-east-1',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT tenant_region_valid CHECK (region IN ('us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1', 'global'))
);

-- Calendar Profiles table - Named collections of holidays/blackout rules
CREATE TABLE IF NOT EXISTS calendar_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    region VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Cache version for invalidation
    version VARCHAR(64) NOT NULL DEFAULT 'v1',
    
    CONSTRAINT tenant_profile_unique UNIQUE (tenant_id, name, deleted_at IS NULL),
    CONSTRAINT profile_region_valid CHECK (region IN ('us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1', 'global'))
);

-- Holidays table - Specific dates when business is closed
CREATE TABLE IF NOT EXISTS holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    holiday_date DATE NOT NULL,
    name VARCHAR(255) NOT NULL,
    region VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT holiday_unique UNIQUE (profile_id, holiday_date)
);

-- Blackout Windows table - Time ranges when business is closed (recurring or one-time)
CREATE TABLE IF NOT EXISTS blackout_windows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    title VARCHAR(255) NOT NULL,
    reason TEXT,
    
    -- RFC 5545 recurrence rule for recurring blackouts
    -- Example: "FREQ=WEEKLY;BYDAY=SA,SU" for every weekend
    rrule TEXT,
    
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    recurrence_start DATE,
    recurrence_end DATE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT blackout_time_order CHECK (start_time < end_time)
);

-- Resolved Calendars Cache Metadata table
-- Tracks computed profile resolutions for cache validation
CREATE TABLE IF NOT EXISTS resolved_calendar_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
    region VARCHAR(50) NOT NULL,
    
    -- Metadata from last resolution
    resolved_at TIMESTAMP WITH TIME ZONE,
    version VARCHAR(64),
    holidays_count INT DEFAULT 0,
    blackouts_count INT DEFAULT 0,
    
    -- Hash for detecting changes
    content_hash VARCHAR(64),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT resolved_metadata_unique UNIQUE (tenant_id, profile_id, region)
);

-- Audit Log table - Track changes for compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- 'calendar_profile', 'blackout', 'holiday', etc.
    entity_id UUID,
    action VARCHAR(20) NOT NULL, -- 'CREATE', 'UPDATE', 'DELETE'
    changes JSONB,
    performed_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_calendar_profiles_tenant ON calendar_profiles(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_calendar_profiles_name ON calendar_profiles USING GIN(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_holidays_profile ON holidays(profile_id, holiday_date);
CREATE INDEX IF NOT EXISTS idx_holidays_date_range ON holidays USING BRIN(holiday_date);
CREATE INDEX IF NOT EXISTS idx_blackouts_profile ON blackout_windows(profile_id);
CREATE INDEX IF NOT EXISTS idx_blackouts_time_range ON blackout_windows USING BRIN(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_resolved_metadata_tenant ON resolved_calendar_metadata(tenant_id, profile_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for tenants table
CREATE TRIGGER tenants_update_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for calendar_profiles table
CREATE TRIGGER calendar_profiles_update_updated_at
BEFORE UPDATE ON calendar_profiles
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for blackout_windows table
CREATE TRIGGER blackout_windows_update_updated_at
BEFORE UPDATE ON blackout_windows
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for resolved_calendar_metadata
CREATE TRIGGER resolved_metadata_update_updated_at
BEFORE UPDATE ON resolved_calendar_metadata
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

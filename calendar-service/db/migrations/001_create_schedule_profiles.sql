-- Migration: Create schedule_profiles table for Phase 4.3 Profile Management
-- Date: 2026-02-17
-- Description: Adds bitemporal versioning support for schedule profiles with multi-calendar support

-- Create schedule_profiles table with bitemporal design
CREATE TABLE IF NOT EXISTS schedule_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_name VARCHAR(255) NOT NULL,
    description TEXT,
    calendars TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[], -- Array of calendar IDs
    conflict_resolution VARCHAR(50) NOT NULL DEFAULT 'union', -- 'union', 'intersection', 'priority'
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    rules JSONB,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Bitemporal Versioning (SCD Type 2)
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,  -- NULL = current version
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    
    -- Constraints
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT valid_dates CHECK (valid_from < valid_to OR valid_to IS NULL),
    CONSTRAINT non_empty_profile_name CHECK (LENGTH(profile_name) > 0),
    CONSTRAINT non_empty_calendars CHECK (ARRAY_LENGTH(calendars, 1) > 0),
    CONSTRAINT valid_conflict_resolution CHECK (conflict_resolution IN ('union', 'intersection', 'priority'))
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_profiles_tenant_active ON schedule_profiles(tenant_id, active, valid_to) 
    WHERE valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_tenant_id ON schedule_profiles(tenant_id);

CREATE INDEX IF NOT EXISTS idx_profiles_valid_from ON schedule_profiles(valid_from DESC);

CREATE INDEX IF NOT EXISTS idx_profiles_valid_to ON schedule_profiles(valid_to DESC) 
    WHERE valid_to IS NOT NULL;

-- Create composite index for bitemporal queries
CREATE INDEX IF NOT EXISTS idx_profiles_bitemporal ON schedule_profiles(tenant_id, valid_from, valid_to);

-- Create table for external sync configuration (Phase 4.5)
CREATE TABLE IF NOT EXISTS external_sync_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'nager_date', 'calendarific', etc.
    country_code VARCHAR(10) NOT NULL, -- ISO 3166-1 alpha-2 or region code
    api_key_encrypted VARCHAR(255), -- Encrypted API key if needed
    sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sync_frequency VARCHAR(20) NOT NULL DEFAULT 'monthly', -- 'weekly', 'monthly', 'yearly'
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_tenant_sync FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT valid_provider CHECK (provider IN ('nager_date', 'calendarific')),
    CONSTRAINT valid_frequency CHECK (sync_frequency IN ('weekly', 'monthly', 'yearly'))
);

CREATE INDEX IF NOT EXISTS idx_sync_config_tenant ON external_sync_config(tenant_id);

CREATE INDEX IF NOT EXISTS idx_sync_config_profile ON external_sync_config(profile_id);

CREATE INDEX IF NOT EXISTS idx_sync_config_next_run ON external_sync_config(next_sync_at) 
    WHERE sync_enabled = TRUE;

-- Create table for sync logs (Phase 4.5)
CREATE TABLE IF NOT EXISTS external_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES external_sync_config(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL, -- 'success', 'failed', 'partial'
    holidays_added INT DEFAULT 0,
    holidays_updated INT DEFAULT 0,
    error_message TEXT,
    execution_time_ms INT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_config_log FOREIGN KEY (config_id) REFERENCES external_sync_config(id) ON DELETE RESTRICT,
    CONSTRAINT valid_status CHECK (status IN ('success', 'failed', 'partial'))
);

CREATE INDEX IF NOT EXISTS idx_sync_logs_config ON external_sync_logs(config_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON external_sync_logs(status, executed_at DESC);

-- Add audit log table if not already exists (for comprehensive audit trail)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(100) NOT NULL, -- 'profile', 'calendar', 'sync', etc.
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'CREATE', 'UPDATE', 'DELETE'
    old_values JSONB,
    new_values JSONB,
    actor_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_entity ON audit_logs(tenant_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id);

-- Grant permissions to calendar_user
GRANT SELECT, INSERT, UPDATE, DELETE ON schedule_profiles TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON external_sync_config TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON external_sync_logs TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON audit_logs TO calendar_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO calendar_user;

-- Add foreign key constraint for audit logs if needed in future
-- ALTER TABLE audit_logs ADD CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

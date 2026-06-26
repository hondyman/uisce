-- Create calendar_service database
CREATE DATABASE calendar_service;

-- Create calendar_user role with permissions
CREATE USER calendar_user WITH ENCRYPTED PASSWORD 'calendar_password';

-- Connect to calendar_service database and grant permissions
\c calendar_service

-- Grant connection privileges
GRANT CONNECT ON DATABASE calendar_service TO calendar_user;

-- Create schema
CREATE SCHEMA IF NOT EXISTS public;

-- Create basic tables for testing (minimal schema)
CREATE TABLE IF NOT EXISTS calendars (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    timezone VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS availability_slots (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS blackouts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    calendar_id UUID NOT NULL REFERENCES calendars(id),
    name VARCHAR(255) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Phase 4.3: Schedule Profiles (Profile Management)
CREATE TABLE IF NOT EXISTS schedule_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_name VARCHAR(255) NOT NULL,
    description TEXT,
    calendars TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    conflict_resolution VARCHAR(50) NOT NULL DEFAULT 'union',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    rules JSONB,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    CONSTRAINT fk_profile_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT valid_dates CHECK (valid_from < valid_to OR valid_to IS NULL)
);

-- Phase 4.5: External Sync Configuration
CREATE TABLE IF NOT EXISTS external_sync_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_id UUID NOT NULL REFERENCES schedule_profiles(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    country_code VARCHAR(10) NOT NULL,
    api_key_encrypted VARCHAR(255),
    sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sync_frequency VARCHAR(20) NOT NULL DEFAULT 'monthly',
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sync_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Phase 4.5: External Sync Logs
CREATE TABLE IF NOT EXISTS external_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES external_sync_config(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    holidays_added INT DEFAULT 0,
    holidays_updated INT DEFAULT 0,
    error_message TEXT,
    execution_time_ms INT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit Logs (for all operations)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    actor_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX idx_calendars_tenant ON calendars(tenant_id);
CREATE INDEX idx_availability_tenant ON availability_slots(tenant_id);
CREATE INDEX idx_blackouts_tenant ON blackouts(tenant_id);
CREATE INDEX idx_profiles_tenant_active ON schedule_profiles(tenant_id, active, valid_to) WHERE valid_to IS NULL;
CREATE INDEX idx_profile_valid_from ON schedule_profiles(valid_from DESC);
CREATE INDEX idx_sync_config_tenant ON external_sync_config(tenant_id);
CREATE INDEX idx_sync_logs_config ON external_sync_logs(config_id, executed_at DESC);
CREATE INDEX idx_audit_tenant_entity ON audit_logs(tenant_id, entity_type, entity_id);

-- Grant permissions to calendar_user
GRANT USAGE ON SCHEMA public TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON calendars TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON availability_slots TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON blackouts TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON tenants TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON schedule_profiles TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON external_sync_config TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON external_sync_logs TO calendar_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON audit_logs TO calendar_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO calendar_user;

-- Allow sequential access
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO calendar_user;

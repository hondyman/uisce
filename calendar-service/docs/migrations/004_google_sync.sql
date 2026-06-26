-- ==========================================================================
-- Migration 004: Google Calendar Sync Tables
-- Description: Track Google Calendar connections and synced events
-- ==========================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS calendar;

-- Track Google Calendar connections per user/tenant
CREATE TABLE IF NOT EXISTS calendar.google_calendar_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    google_user_id VARCHAR(255) NOT NULL,
    google_email VARCHAR(255) NOT NULL,
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_frequency VARCHAR(20) DEFAULT 'hourly' CHECK (sync_frequency IN ('hourly','daily','weekly','manual')),
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(20) DEFAULT 'pending',
    last_sync_error TEXT,
    mapped_calendars JSONB DEFAULT '[]'::jsonb,
    oauth_scopes TEXT[] DEFAULT ARRAY['calendar.readonly','calendar.events'],
    token_last_refreshed TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, google_user_id)
);

-- Track synced Google events for conflict detection
CREATE TABLE IF NOT EXISTS calendar.synced_google_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES calendar.google_calendar_connections(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    google_event_id VARCHAR(255) NOT NULL,
    google_calendar_id VARCHAR(255) NOT NULL,
    internal_event_id UUID,
    internal_calendar_id UUID,
    title VARCHAR(255),
    description TEXT,
    location TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    is_all_day BOOLEAN DEFAULT FALSE,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_id VARCHAR(255),
    recurrence_rule TEXT,
    attendees JSONB DEFAULT '[]'::jsonb,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sync_hash VARCHAR(64) NOT NULL,
    sync_status VARCHAR(20) DEFAULT 'synced' CHECK (sync_status IN ('synced','conflict','error','deleted','pending')),
    sync_error TEXT,
    conflict_resolution VARCHAR(20) CHECK (conflict_resolution IN ('keep_google','keep_internal','merge','skip','manual')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, google_event_id, google_calendar_id),
    CONSTRAINT chk_time_range CHECK (end_time > start_time),
    CONSTRAINT chk_sync_hash_length CHECK (length(sync_hash) = 64)
);

CREATE INDEX IF NOT EXISTS idx_google_connections_user ON calendar.google_calendar_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_google_connections_tenant ON calendar.google_calendar_connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_google_connections_next_sync ON calendar.google_calendar_connections(next_sync_at);
CREATE INDEX IF NOT EXISTS idx_synced_events_connection ON calendar.synced_google_events(connection_id);
CREATE INDEX IF NOT EXISTS idx_synced_events_tenant ON calendar.synced_google_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_synced_events_internal_event ON calendar.synced_google_events(internal_event_id) WHERE internal_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_synced_events_time_range ON calendar.synced_google_events(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_synced_events_sync_hash ON calendar.synced_google_events(sync_hash);
CREATE INDEX IF NOT EXISTS idx_synced_events_status ON calendar.synced_google_events(sync_status) WHERE sync_status != 'synced';

ALTER TABLE calendar.google_calendar_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.synced_google_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY google_connections_tenant_isolation ON calendar.google_calendar_connections
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);
CREATE POLICY synced_events_tenant_isolation ON calendar.synced_google_events
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY google_connections_user_isolation ON calendar.google_calendar_connections
    FOR SELECT USING (user_id = NULLIF(current_setting('request.user_id', TRUE), '')::UUID OR
                     current_setting('request.is_admin', TRUE) = 'true');

CREATE OR REPLACE FUNCTION calendar.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_google_connections_updated_at
BEFORE UPDATE ON calendar.google_calendar_connections
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();

CREATE TRIGGER update_synced_events_updated_at
BEFORE UPDATE ON calendar.synced_google_events
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();

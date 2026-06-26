-- Migration for Google Calendar Sync

-- 1. Google Calendar Connections (OAuth tokens and settings)
CREATE TABLE IF NOT EXISTS google_calendar_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    google_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    access_token TEXT, -- Encrypted
    refresh_token TEXT, -- Encrypted
    token_expiry TIMESTAMP WITH TIME ZONE,
    scope TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, email)
);

-- 1.5 Internal Events (The system's native event representation)
CREATE TABLE IF NOT EXISTS internal_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    location TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    is_all_day BOOLEAN DEFAULT FALSE,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Synced Google Events (Mapping between Google and Internal events)
CREATE TABLE IF NOT EXISTS synced_google_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID REFERENCES google_calendar_connections(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    google_event_id TEXT NOT NULL,
    google_calendar_id TEXT NOT NULL,
    internal_event_id UUID, -- Can be null if not yet linked
    internal_calendar_id UUID,
    title TEXT,
    description TEXT,
    location TEXT,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    is_all_day BOOLEAN DEFAULT FALSE,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT,
    recurrence_id TEXT, -- For specific instances of recurring events
    sync_status TEXT DEFAULT 'synced', -- synced, pending, conflict, error
    last_synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(connection_id, google_event_id, google_calendar_id)
);

-- 3. Sync Conflicts (Detected conflicts requiring resolution)
CREATE TABLE IF NOT EXISTS sync_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    connection_id UUID REFERENCES google_calendar_connections(id) ON DELETE CASCADE,
    google_event_id TEXT,
    google_calendar_id TEXT,
    internal_event_id UUID,
    conflict_type TEXT NOT NULL, -- time_overlap, title_mismatch, etc.
    severity TEXT NOT NULL, -- info, warning, error, critical
    description TEXT,
    google_event_data JSONB, -- Snapshot of Google event
    internal_event_data JSONB, -- Snapshot of Internal event
    resolution_status TEXT DEFAULT 'pending', -- pending, resolved, skipped
    resolution_strategy TEXT, -- keep_google, keep_internal, merge
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_google_connections_user ON google_calendar_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_synced_events_google_id ON synced_google_events(google_event_id);
CREATE INDEX IF NOT EXISTS idx_synced_events_internal_id ON synced_google_events(internal_event_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_status ON sync_conflicts(resolution_status);

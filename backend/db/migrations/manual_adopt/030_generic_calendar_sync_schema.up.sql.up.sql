-- Migration: Rename Google-specific tables and columns to generic calendar sync naming
-- Adds provider column to support multi-provider (Google, Microsoft, Apple)

-- Rename google_calendar_connections -> calendar_connections
ALTER TABLE google_calendar_connections RENAME TO calendar_connections;
ALTER TABLE calendar_connections RENAME COLUMN google_user_id TO external_user_id;

-- Rename synced_google_events -> synced_calendar_events
ALTER TABLE synced_google_events RENAME TO synced_calendar_events;
ALTER TABLE synced_calendar_events RENAME COLUMN google_event_id TO external_event_id;
ALTER TABLE synced_calendar_events RENAME COLUMN google_calendar_id TO external_calendar_id;
ALTER TABLE synced_calendar_events ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'google';

-- Drop old unique constraint and recreate with new column names
ALTER TABLE synced_calendar_events DROP CONSTRAINT IF EXISTS synced_google_events_google_event_id_google_calendar_id_key;
ALTER TABLE synced_calendar_events ADD CONSTRAINT synced_calendar_events_external_id_external_calendar_id_key
    UNIQUE (provider, external_event_id, external_calendar_id);

-- Rename in sync_conflicts table
ALTER TABLE sync_conflicts RENAME COLUMN google_event_id TO external_event_id;
ALTER TABLE sync_conflicts RENAME COLUMN google_calendar_id TO external_calendar_id;
ALTER TABLE sync_conflicts ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'google';

-- Update indexes
DROP INDEX IF EXISTS idx_synced_events_google_id;
CREATE INDEX IF NOT EXISTS x_synced_events_external_id ON synced_calendar_events(external_event_id);
CREATE INDEX IF NOT EXISTS x_synced_events_provider ON synced_calendar_events(provider);

-- Rename connection_id FK reference in synced_calendar_events
-- (connection_id column name stays the same, but now references calendar_connections)
ALTER TABLE synced_calendar_events DROP CONSTRAINT IF EXISTS synced_google_events_connection_id_fkey;
ALTER TABLE synced_calendar_events ADD CONSTRAINT synced_calendar_events_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES calendar_connections(id) ON DELETE CASCADE;

ALTER TABLE sync_conflicts DROP CONSTRAINT IF EXISTS sync_conflicts_connection_id_fkey;
ALTER TABLE sync_conflicts ADD CONSTRAINT sync_conflicts_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES calendar_connections(id) ON DELETE CASCADE;

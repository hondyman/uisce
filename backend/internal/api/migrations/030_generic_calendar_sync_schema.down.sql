-- Revert: Restore Google-specific naming for calendar sync tables

-- Revert calendar_connections -> google_calendar_connections
ALTER TABLE calendar_connections RENAME TO google_calendar_connections;
ALTER TABLE google_calendar_connections RENAME COLUMN external_user_id TO google_user_id;

-- Revert synced_calendar_events -> synced_google_events
ALTER TABLE synced_calendar_events DROP CONSTRAINT IF EXISTS synced_calendar_events_external_id_external_calendar_id_key;
ALTER TABLE synced_calendar_events RENAME TO synced_google_events;
ALTER TABLE synced_google_events RENAME COLUMN external_event_id TO google_event_id;
ALTER TABLE synced_google_events RENAME COLUMN external_calendar_id TO google_calendar_id;
ALTER TABLE synced_google_events DROP COLUMN IF EXISTS provider;
ALTER TABLE synced_google_events ADD CONSTRAINT synced_google_events_google_event_id_google_calendar_id_key
    UNIQUE (google_event_id, google_calendar_id);

-- Revert sync_conflicts columns
ALTER TABLE sync_conflicts RENAME COLUMN external_event_id TO google_event_id;
ALTER TABLE sync_conflicts RENAME COLUMN external_calendar_id TO google_calendar_id;
ALTER TABLE sync_conflicts DROP COLUMN IF EXISTS provider;

-- Restore indexes
DROP INDEX IF EXISTS idx_synced_events_external_id;
DROP INDEX IF EXISTS idx_synced_events_provider;
CREATE INDEX idx_synced_events_google_id ON synced_google_events(google_event_id);

-- Restore FK constraints
ALTER TABLE synced_google_events DROP CONSTRAINT IF EXISTS synced_calendar_events_connection_id_fkey;
ALTER TABLE synced_google_events ADD CONSTRAINT synced_google_events_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES google_calendar_connections(id) ON DELETE CASCADE;

ALTER TABLE sync_conflicts DROP CONSTRAINT IF EXISTS sync_conflicts_connection_id_fkey;
ALTER TABLE sync_conflicts ADD CONSTRAINT sync_conflicts_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES google_calendar_connections(id) ON DELETE CASCADE;

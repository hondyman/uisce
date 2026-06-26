-- ============================================================================
-- Migration 008: Bi-directional Sync Support
-- ============================================================================
-- Purpose: Track sync direction and prevent infinite loops
-- Deploy: psql $DB_URL -f docs/migrations/008_bi_directional_sync.sql
-- ============================================================================

-- Add sync direction tracking to synced_google_events
ALTER TABLE calendar.synced_google_events 
ADD COLUMN IF NOT EXISTS sync_direction VARCHAR(20) DEFAULT 'google_to_internal'
CHECK (sync_direction IN ('google_to_internal', 'internal_to_google', 'bi_directional'));

-- Add last_pushed_to_google timestamp
ALTER TABLE calendar.synced_google_events
ADD COLUMN IF NOT EXISTS last_pushed_to_google TIMESTAMPTZ;

-- Index for sync direction queries
CREATE INDEX IF NOT EXISTS idx_synced_events_direction 
ON calendar.synced_google_events(sync_direction, last_synced_at DESC);

-- Create sync_queue table for background sync jobs
CREATE TABLE IF NOT EXISTS calendar.sync_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    event_id UUID NOT NULL,
    operation VARCHAR(20) NOT NULL CHECK (operation IN ('create', 'update', 'delete')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    error_message TEXT,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for sync queue
CREATE INDEX idx_sync_queue_status ON calendar.sync_queue(status, scheduled_at);
CREATE INDEX idx_sync_queue_user ON calendar.sync_queue(user_id, status);
CREATE INDEX idx_sync_queue_scheduled ON calendar.sync_queue(scheduled_at) WHERE status = 'pending';

-- Enable RLS
ALTER TABLE calendar.sync_queue ENABLE ROW LEVEL SECURITY;

CREATE POLICY sync_queue_tenant_isolation 
ON calendar.sync_queue
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- Trigger to add sync queue entries on event changes
CREATE OR REPLACE FUNCTION calendar.queue_event_for_google_sync()
RETURNS TRIGGER AS $$
BEGIN
    -- Only queue if event has changes that need sync
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND NEW.updated_at > OLD.updated_at) THEN
        INSERT INTO calendar.sync_queue (
            tenant_id, user_id, event_id, operation, scheduled_at
        ) VALUES (
            NEW.tenant_id, 
            NEW.created_by, 
            NEW.id, 
            CASE WHEN TG_OP = 'INSERT' THEN 'create' ELSE 'update' END,
            NOW() + INTERVAL '5 seconds' -- Delay to batch rapid changes
        );
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        INSERT INTO calendar.sync_queue (
            tenant_id, user_id, event_id, operation, scheduled_at
        ) VALUES (
            OLD.tenant_id,
            OLD.created_by,
            OLD.id,
            'delete',
            NOW() + INTERVAL '5 seconds'
        );
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

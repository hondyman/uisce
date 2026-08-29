-- Notification Outbox Table for CDC
-- Outbox pattern: write events here, Debezium captures and publishes to Kafka

CREATE TABLE IF NOT EXISTS public.notification_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL DEFAULT 'notification',
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    tenant_id UUID NOT NULL,
    user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ
);

-- Index for efficient polling by Debezium
CREATE INDEX IF NOT EXISTS idx_notification_outbox_created_at ON public.notification_outbox(created_at);

-- Index for tenant-scoped queries
CREATE INDEX IF NOT EXISTS idx_notification_outbox_tenant_id ON public.notification_outbox(tenant_id);

-- Index for unprocessed events (for relay consumer)
CREATE INDEX IF NOT EXISTS idx_notification_outbox_unprocessed ON public.notification_outbox(processed_at) WHERE processed_at IS NULL;

-- RLS for outbox table
ALTER TABLE public.notification_outbox ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS notification_outbox_tenant_policy ON public.notification_outbox;
CREATE POLICY notification_outbox_tenant_policy ON public.notification_outbox
    FOR ALL USING (true);

COMMENT ON TABLE public.notification_outbox IS 'Outbox table for notification CDC. Debezium captures events and publishes to Kafka topic.';
COMMENT ON COLUMN public.notification_outbox.aggregate_type IS 'Type of aggregate: notification, email, push, etc.';
COMMENT ON COLUMN public.notification_outbox.event_type IS 'Event type: created, read, delivered, etc.';
COMMENT ON COLUMN public.notification_outbox.payload IS 'JSON payload with event data';
COMMENT ON COLUMN public.notification_outbox.processed_at IS 'When relay consumer processed this event';
COMMENT ON COLUMN public.notification_outbox.published_at IS 'When Debezium published this event (optional)';

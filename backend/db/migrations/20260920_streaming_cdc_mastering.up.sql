-- 20260920_streaming_cdc_mastering.up.sql
-- Live Streaming CDC Ingress, Durable Offset Management & Transactional Outbox

CREATE SCHEMA IF NOT EXISTS catalog_mdm;

-- 1. Streaming Consumer Offsets (Durable at-least-once ingestion)
CREATE TABLE IF NOT EXISTS catalog_mdm.streaming_cdc_offsets (
    consumer_group VARCHAR(100) NOT NULL,
    topic_name VARCHAR(100) NOT NULL,
    partition_id INT NOT NULL,
    last_committed_offset BIGINT NOT NULL DEFAULT 0,
    tenant_id UUID NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (consumer_group, topic_name, partition_id, tenant_id)
);

-- 2. Transactional Outbox for Downstream Fan-Out Dispatching
CREATE TABLE IF NOT EXISTS catalog_mdm.mastering_outbox_events (
    outbox_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    master_entity_sid VARCHAR(100) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- GOLDEN_RECORD_MASTERED, TOLERANCE_EXCEPTION_RAISED
    payload JSONB NOT NULL,
    destination_topic VARCHAR(100) NOT NULL DEFAULT 'mdm.golden.events.v1',
    is_dispatched BOOLEAN DEFAULT FALSE,
    dispatched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mdm_outbox_pending 
ON catalog_mdm.mastering_outbox_events (tenant_id, is_dispatched, created_at ASC);

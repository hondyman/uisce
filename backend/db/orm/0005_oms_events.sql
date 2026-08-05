-- ============================================================================
-- 0005_oms_events.sql
-- Immutable order event audit trail.
-- INSERT only — order_event should be append-only (no UPDATE/DELETE grants
-- for the application role; superuser can still fix broken rows).
-- Run after 0004_oms_orders.sql
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- oms.order_event
-- Every state transition on orders and slices is recorded here.
-- payload JSONB carries full before/after snapshot.
-- --------------------------------------------------------------------------
CREATE TABLE oms.order_event (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    order_id        UUID        NOT NULL REFERENCES oms.orders(id),
    slice_id        UUID        REFERENCES oms.order_slice(id),  -- NULL for parent-level events

    event_type_id   INTEGER     NOT NULL REFERENCES ref.order_event_type(id),
    source_id       INTEGER     NOT NULL REFERENCES ref.event_source(id),

    -- snapshot
    payload         JSONB       DEFAULT '{}',  -- {old_status, new_status, price, qty, reason, actor, …}
    raw_payload     TEXT,                      -- raw FIX/JSON wire message if available

    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    server_ts       TIMESTAMPTZ NOT NULL DEFAULT now(),  -- server receipt time for latency tracking

    -- actor identification
    actor_id        VARCHAR(100),
    actor_name      VARCHAR(200),
    actor_type      VARCHAR(20),  -- 'TRADER','RISK_ENGINE','ALGO','SYSTEM','VENUE'

    -- causal linkage for replay
    correlation_id  UUID,         -- trace ID for causal chain
    causation_id   UUID          -- previous event ID in chain
);

-- Append-only design: no index on id (PK is fine), but heavily index for replay queries.
CREATE INDEX idx_order_event_order     ON oms.order_event (order_id, occurred_at DESC);
CREATE INDEX idx_order_event_slice     ON oms.order_event (slice_id) WHERE slice_id IS NOT NULL;
CREATE INDEX idx_order_event_type     ON oms.order_event (event_type_id);
CREATE INDEX idx_order_event_tenant   ON oms.order_event (tenant_id, occurred_at DESC);
CREATE INDEX idx_order_event_source   ON oms.order_event (source_id);
CREATE INDEX idx_order_event_corr    ON oms.order_event (correlation_id) WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_order_event_causation ON oms.order_event (causation_id) WHERE causation_id IS NOT NULL;
CREATE INDEX idx_order_event_active
    ON oms.order_event (order_id, occurred_at DESC);

-- GIN on payload for structured querying
CREATE INDEX idx_order_event_payload_gin ON oms.order_event USING gin (payload);

COMMIT;

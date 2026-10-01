-- 011_streaming_price.sql
-- Streaming pricing master — schema.
-- Applied to crims. Idempotent.
-- Corrected: plural mdm.source_systems, dev tenant UUID, FORCE RLS.

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS streaming;

-- ═══════════════════════════════════════════════════════════════════════
-- 1. watermark
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS streaming.watermark (
    source_system_id  uuid NOT NULL,
    topic             text NOT NULL,
    partition         int NOT NULL DEFAULT -1,
    high_watermark    timestamptz NOT NULL,
    last_seen_at      timestamptz NOT NULL DEFAULT now(),
    expected_lag_ms   int,
    actual_lag_ms     int,
    status            varchar(20) NOT NULL DEFAULT 'HEALTHY',
    tenant_id         uuid NOT NULL,
    CONSTRAINT watermark_pkey PRIMARY KEY (tenant_id, source_system_id, topic, partition),
    CONSTRAINT chk_wm_status CHECK (status IN ('HEALTHY','LAGGING','STALLED','ERROR'))
);

CREATE INDEX IF NOT EXISTS idx_wm_status
    ON streaming.watermark (status, last_seen_at DESC);

ALTER TABLE streaming.watermark ENABLE ROW LEVEL SECURITY;
ALTER TABLE streaming.watermark FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS wm_tenant_read ON streaming.watermark;
CREATE POLICY wm_tenant_read ON streaming.watermark
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS wm_tenant_write ON streaming.watermark;
CREATE POLICY wm_tenant_write ON streaming.watermark
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 2. price_tick
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS streaming.price_tick (
    id                uuid NOT NULL DEFAULT gen_random_uuid(),
    kafka_topic       text NOT NULL,
    kafka_partition   int NOT NULL,
    kafka_offset      bigint NOT NULL,
    kafka_timestamp   timestamptz NOT NULL,
    source_system_id  uuid NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id   uuid,
    source_entity_ref varchar(200),
    price_type_cd     varchar(30) NOT NULL,
    observation_type  varchar(30),
    event_time        timestamptz NOT NULL,
    ingest_time       timestamptz NOT NULL DEFAULT now(),
    value             numeric(28,12),
    bid               numeric(28,12),
    ask               numeric(28,12),
    mid               numeric(28,12),
    last              numeric(28,12),
    volume            numeric(28,4),
    currency          varchar(3),
    quality_tier_cd   varchar(30),
    is_correction     boolean NOT NULL DEFAULT false,
    corrects_tick_id  uuid,
    raw_payload       jsonb NOT NULL DEFAULT '{}'::jsonb,
    tenant_id         uuid NOT NULL,
    CONSTRAINT price_tick_pkey PRIMARY KEY (id, event_time)
) PARTITION BY RANGE (event_time);

DO $$
DECLARE
    d date;
    pname text;
BEGIN
    FOR d IN SELECT generate_series(CURRENT_DATE - 1, CURRENT_DATE + 1, '1 day')::date LOOP
        pname := 'price_tick_' || to_char(d, 'YYYYMMDD');
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname='streaming' AND c.relname=pname
        ) THEN
            EXECUTE format(
                'CREATE TABLE streaming.%I PARTITION OF streaming.price_tick
                 FOR VALUES FROM (%L) TO (%L)',
                pname, d, d + 1);
        END IF;
    END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_price_tick_dedup
    ON streaming.price_tick (source_system_id, price_entity_type,
        COALESCE(price_entity_id::text, source_entity_ref), price_type_cd,
        event_time, kafka_topic, kafka_offset);

CREATE INDEX IF NOT EXISTS idx_pt_entity_time
    ON streaming.price_tick (price_entity_type, price_entity_id, event_time DESC)
    WHERE price_entity_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pt_source_time
    ON streaming.price_tick (source_system_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_pt_tenant
    ON streaming.price_tick (tenant_id);

ALTER TABLE streaming.price_tick ENABLE ROW LEVEL SECURITY;
ALTER TABLE streaming.price_tick FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pt_tenant_read ON streaming.price_tick;
CREATE POLICY pt_tenant_read ON streaming.price_tick
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS pt_tenant_write ON streaming.price_tick;
CREATE POLICY pt_tenant_write ON streaming.price_tick
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 3. price_intraday
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mdm.price_intraday (
    id                uuid NOT NULL DEFAULT gen_random_uuid(),
    price_entity_type varchar(30) NOT NULL,
    price_entity_id   uuid NOT NULL,
    price_type_cd     varchar(30) NOT NULL,
    window_start      timestamptz NOT NULL,
    window_end        timestamptz NOT NULL,
    window_size_sec   int NOT NULL,
    open_value        numeric(28,12),
    high_value        numeric(28,12),
    low_value         numeric(28,12),
    close_value       numeric(28,12),
    vwap_value        numeric(28,12),
    tick_count        int NOT NULL,
    source_count      int NOT NULL,
    is_provisional    boolean NOT NULL DEFAULT true,
    quality_tier_cd   varchar(30),
    tenant_id         uuid NOT NULL,
    CONSTRAINT price_intraday_pkey PRIMARY KEY (id),
    CONSTRAINT uq_price_intraday UNIQUE (tenant_id, price_entity_type,
        price_entity_id, price_type_cd, window_start, window_size_sec)
);

CREATE INDEX IF NOT EXISTS idx_pi_lookup
    ON mdm.price_intraday (price_entity_type, price_entity_id, window_end DESC);
CREATE INDEX IF NOT EXISTS idx_pi_tenant
    ON mdm.price_intraday (tenant_id);

ALTER TABLE mdm.price_intraday ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.price_intraday FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pi_intraday_read ON mdm.price_intraday;
CREATE POLICY pi_intraday_read ON mdm.price_intraday
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS pi_intraday_write ON mdm.price_intraday;
CREATE POLICY pi_intraday_write ON mdm.price_intraday
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 4. price_quality_event
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mdm.price_quality_event (
    id                uuid NOT NULL DEFAULT gen_random_uuid(),
    price_entity_type varchar(30) NOT NULL,
    price_entity_id   uuid,
    price_type_cd     varchar(30),
    source_system_id  uuid,
    tick_id           uuid,
    event_class       varchar(30) NOT NULL,
    severity          varchar(10) NOT NULL,
    detected_at       timestamptz NOT NULL DEFAULT now(),
    detail            jsonb NOT NULL DEFAULT '{}'::jsonb,
    status            varchar(20) NOT NULL DEFAULT 'OPEN',
    tenant_id         uuid NOT NULL,
    CONSTRAINT chk_pqe_class CHECK (event_class IN (
        'STALE','VARIANCE','GAP','SANITY','LATE','DUPLICATE')),
    CONSTRAINT chk_pqe_severity CHECK (severity IN (
        'INFO','WARNING','ERROR','CRITICAL')),
    CONSTRAINT chk_pqe_status CHECK (status IN (
        'OPEN','IN_REVIEW','RESOLVED','WAIVED'))
);

CREATE INDEX IF NOT EXISTS idx_pqe_entity
    ON mdm.price_quality_event (price_entity_type, price_entity_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_pqe_status
    ON mdm.price_quality_event (status, severity, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_pqe_tenant
    ON mdm.price_quality_event (tenant_id);

ALTER TABLE mdm.price_quality_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.price_quality_event FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pqe_tenant_read ON mdm.price_quality_event;
CREATE POLICY pqe_tenant_read ON mdm.price_quality_event
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS pqe_tenant_write ON mdm.price_quality_event;
CREATE POLICY pqe_tenant_write ON mdm.price_quality_event
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 5. price_stream_health
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mdm.price_stream_health (
    id                uuid NOT NULL DEFAULT gen_random_uuid(),
    source_system_id  uuid NOT NULL,
    topic             text NOT NULL,
    check_timestamp   timestamptz NOT NULL DEFAULT now(),
    msgs_per_sec      numeric(12,2),
    consumer_lag_ms   bigint,
    processing_lag_ms bigint,
    error_count       int NOT NULL DEFAULT 0,
    status            varchar(20) NOT NULL,
    tenant_id         uuid NOT NULL,
    CONSTRAINT chk_psh_status CHECK (status IN ('HEALTHY','LAGGING','STALLED','ERROR'))
);

CREATE INDEX IF NOT EXISTS idx_psh_source
    ON mdm.price_stream_health (source_system_id, check_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_psh_tenant
    ON mdm.price_stream_health (tenant_id);

ALTER TABLE mdm.price_stream_health ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.price_stream_health FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS psh_tenant_read ON mdm.price_stream_health;
CREATE POLICY psh_tenant_read ON mdm.price_stream_health
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS psh_tenant_write ON mdm.price_stream_health;
CREATE POLICY psh_tenant_write ON mdm.price_stream_health
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 6. Partition helper
-- ═══════════════════════════════════════════════════════════════════════
CREATE OR REPLACE FUNCTION streaming.ensure_price_tick_partitions(days_ahead int DEFAULT 3)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    d date;
    pname text;
BEGIN
    FOR d IN SELECT generate_series(CURRENT_DATE, CURRENT_DATE + days_ahead, '1 day')::date LOOP
        pname := 'price_tick_' || to_char(d, 'YYYYMMDD');
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname='streaming' AND c.relname=pname
        ) THEN
            EXECUTE format(
                'CREATE TABLE streaming.%I PARTITION OF streaming.price_tick
                 FOR VALUES FROM (%L) TO (%L)',
                pname, d, d + 1);
        END IF;
    END LOOP;
END $$;

DO $v$
DECLARE
    parts int;
BEGIN
    SELECT count(*) INTO parts FROM pg_class c
    JOIN pg_namespace n ON n.oid=c.relnamespace
    WHERE n.nspname='streaming' AND c.relname LIKE 'price_tick_%';
    RAISE NOTICE '011_streaming_price: 5 tables + % partitions', parts;
END $v$;

COMMIT;

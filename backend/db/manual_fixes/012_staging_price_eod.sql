-- 012_staging_price_eod.sql
-- EOD batch price landing. Complements staging.ff_product from 008.

\set ON_ERROR_STOP on
BEGIN;

-- ═══════════════════════════════════════════════════════════════════════
-- 1. FactSet EOD price landing
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.ff_price_eod (
    _load_run_id      uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    _source_row_num   int  NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    fsym_id           varchar(50),
    isin              varchar(12),
    cusip             varchar(9),
    ticker            varchar(26),
    price_date        date,
    price_currency    varchar(3),
    price             numeric(28,12),
    price_open        numeric(28,12),
    price_high        numeric(28,12),
    price_low         numeric(28,12),
    price_close       numeric(28,12),
    prior_close       numeric(28,12),
    volume            numeric(28,4),
    nav               numeric(28,12),
    nav_currency      varchar(3),
    nav_date          date,
    total_net_assets  numeric(28,4),
    price_type        text,
    price_source_desc text,
    is_official       boolean,
    CONSTRAINT ff_price_eod_pkey PRIMARY KEY (_load_run_id, _source_row_num)
);

CREATE INDEX IF NOT EXISTS idx_ffpe_fsym
    ON staging.ff_price_eod (fsym_id, price_date DESC);
CREATE INDEX IF NOT EXISTS idx_ffpe_isin
    ON staging.ff_price_eod (isin, price_date DESC) WHERE isin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ffpe_date
    ON staging.ff_price_eod (price_date DESC);
CREATE INDEX IF NOT EXISTS idx_ffpe_tenant
    ON staging.ff_price_eod (tenant_id);

ALTER TABLE staging.ff_price_eod ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.ff_price_eod FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ffpe_tenant_read ON staging.ff_price_eod;
CREATE POLICY ffpe_tenant_read ON staging.ff_price_eod
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS ffpe_tenant_write ON staging.ff_price_eod;
CREATE POLICY ffpe_tenant_write ON staging.ff_price_eod
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 2. price_incoming — canonical shape
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.price_incoming (
    id                uuid NOT NULL DEFAULT gen_random_uuid(),
    source_system_id  uuid NOT NULL,
    source_system_cd  varchar(30) NOT NULL,
    source_row_id     text NOT NULL,
    load_run_id       uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    tenant_id         uuid NOT NULL,
    price_entity_type varchar(30) NOT NULL DEFAULT 'SECURITY',
    price_entity_id   uuid,
    source_entity_ref varchar(200),
    price_type_cd     varchar(30) NOT NULL,
    observation_type  varchar(30) NOT NULL DEFAULT 'EOD',
    price_date        date NOT NULL,
    value             numeric(28,12),
    currency          varchar(3),
    is_official       boolean NOT NULL DEFAULT false,
    canonical_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    mapping_version   int NOT NULL DEFAULT 1,
    mapped_at         timestamptz NOT NULL DEFAULT now(),
    is_valid          boolean NOT NULL DEFAULT true,
    validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT chk_price_inc_valid CHECK (
        (is_valid = true AND jsonb_array_length(validation_errors) = 0)
        OR (is_valid = false))
);

CREATE INDEX IF NOT EXISTS idx_price_inc_entity
    ON staging.price_incoming (price_entity_type, price_entity_id, price_date DESC)
    WHERE price_entity_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_price_inc_ref
    ON staging.price_incoming (source_entity_ref, price_date DESC)
    WHERE source_entity_ref IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_price_inc_load
    ON staging.price_incoming (load_run_id);
CREATE INDEX IF NOT EXISTS idx_price_inc_tenant
    ON staging.price_incoming (tenant_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_price_inc_source_row
    ON staging.price_incoming (tenant_id, source_system_id, source_row_id);

ALTER TABLE staging.price_incoming ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.price_incoming FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS priceinc_tenant_read ON staging.price_incoming;
CREATE POLICY priceinc_tenant_read ON staging.price_incoming
    AS PERMISSIVE FOR SELECT
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
            (current_setting('app.shared_reference_tenant'::text, true))::uuid,
            '00000000-0000-0000-0000-000000000001'::uuid)));

DROP POLICY IF EXISTS priceinc_tenant_write ON staging.price_incoming;
CREATE POLICY priceinc_tenant_write ON staging.price_incoming
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $v$
BEGIN
    RAISE NOTICE '012_staging_price_eod: 2 tables';
END $v$;

COMMIT;

-- 008_staging_product.sql
-- Multi-source product master landing zone.
-- Run against crims. Idempotent (IF NOT EXISTS everywhere).
--
-- Design principles:
--   1. Landing is dumb — verbatim vendor columns, no mapping at ingest.
--   2. Every row traces to a _load_run (audit + idempotency).
--   3. Immutable — never UPDATE staged rows; new load = new _load_run_id.
--   4. Tenant-scoped + RLS with the standard shared-reference fallback.
--
-- Apply:
--   psql "$CRIMS_URL" -1 -v ON_ERROR_STOP=1 -f 008_staging_product.sql

\set ON_ERROR_STOP on

BEGIN;

CREATE SCHEMA IF NOT EXISTS staging;

-- ═══════════════════════════════════════════════════════════════════════
-- 1. _load_run — one row per (source, domain, run_ref)
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging._load_run (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_cd  varchar(30) NOT NULL,
    domain            varchar(30) NOT NULL,
    run_ref           varchar(100) NOT NULL,
    file_name         text,
    file_hash         varchar(64),
    expected_rows     int,
    received_rows     int,
    accepted_rows     int,
    rejected_rows     int,
    started_at        timestamptz NOT NULL DEFAULT now(),
    completed_at      timestamptz,
    status            varchar(20) NOT NULL DEFAULT 'RUNNING',
    error_summary     text,
    tenant_id         uuid NOT NULL,
    CONSTRAINT uq_lr UNIQUE (tenant_id, source_system_cd, domain, run_ref),
    CONSTRAINT chk_lr_status CHECK (status IN (
        'RUNNING','COMPLETED','FAILED','PARTIAL','CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_lr_status
    ON staging._load_run (status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_lr_source
    ON staging._load_run (source_system_cd, domain, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_lr_tenant
    ON staging._load_run (tenant_id);

ALTER TABLE staging._load_run ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging._load_run FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS _lr_tenant_read ON staging._load_run;
CREATE POLICY _lr_tenant_read ON staging._load_run
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS _lr_tenant_write ON staging._load_run;
CREATE POLICY _lr_tenant_write ON staging._load_run
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 2. _mapping_error — one row per unrecoverable mapping failure
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging._mapping_error (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    load_run_id       uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    source_row_id     text,
    error_type        varchar(30) NOT NULL,
    field_name        text,
    field_value       text,
    error_message     text,
    detected_at       timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    CONSTRAINT chk_me_error_type CHECK (error_type IN (
        'UNKNOWN_FIELD','TYPE_MISMATCH','MISSING_REQUIRED',
        'ENUM_MISMATCH','DUPLICATE_ROW','TRANSFORM_ERROR'))
);

CREATE INDEX IF NOT EXISTS idx_me_run
    ON staging._mapping_error (load_run_id, error_type);
CREATE INDEX IF NOT EXISTS idx_me_tenant
    ON staging._mapping_error (tenant_id);

ALTER TABLE staging._mapping_error ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging._mapping_error FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS _me_tenant_read ON staging._mapping_error;
CREATE POLICY _me_tenant_read ON staging._mapping_error
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS _me_tenant_write ON staging._mapping_error;
CREATE POLICY _me_tenant_write ON staging._mapping_error
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 3. ff_product — FactSet raw landing
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.ff_product (
    _load_run_id      uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    _source_row_num   int  NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,

    -- FactSet verbatim columns
    fsym_id           varchar(50),
    isin              varchar(12),
    cusip             varchar(9),
    sedol             varchar(7),
    ticker            varchar(26),
    fund_name         text,
    fund_name_short   text,
    fund_type         text,
    fund_category     text,
    asset_class       text,
    domicile_country  varchar(2),
    base_currency     varchar(3),
    inception_date    date,
    fund_manager      text,
    management_company text,
    legal_structure   text,
    is_active         boolean,
    benchmark_name    text,
    aum               numeric(20,4),
    aum_currency      varchar(3),
    aum_date          date,

    CONSTRAINT ff_product_pkey PRIMARY KEY (_load_run_id, _source_row_num)
);

CREATE INDEX IF NOT EXISTS idx_ffprod_fsym
    ON staging.ff_product (fsym_id);
CREATE INDEX IF NOT EXISTS idx_ffprod_isin
    ON staging.ff_product (isin) WHERE isin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ffprod_cusip
    ON staging.ff_product (cusip) WHERE cusip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ffprod_ticker
    ON staging.ff_product (ticker) WHERE ticker IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ffprod_tenant
    ON staging.ff_product (tenant_id);

ALTER TABLE staging.ff_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.ff_product FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ffprod_tenant_read ON staging.ff_product;
CREATE POLICY ffprod_tenant_read ON staging.ff_product
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS ffprod_tenant_write ON staging.ff_product;
CREATE POLICY ffprod_tenant_write ON staging.ff_product
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 4. bbg_product — Bloomberg raw landing
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.bbg_product (
    _load_run_id      uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    _source_row_num   int  NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,

    bbg_id            varchar(30),
    bbg_ticker        varchar(50),
    isin              varchar(12),
    cusip             varchar(9),
    ticker            varchar(26),
    long_name         text,
    short_name        text,
    security_des      text,
    fund_type         text,
    asset_class       text,
    country_domicile  varchar(2),
    crncy             varchar(3),
    inception_dt      date,
    fund_manager      text,
    issuer            text,
    benchmark         text,
    total_assets      numeric(20,4),
    fund_status       varchar(20),

    CONSTRAINT bbg_product_pkey PRIMARY KEY (_load_run_id, _source_row_num)
);

CREATE INDEX IF NOT EXISTS idx_bbgprod_bbgid
    ON staging.bbg_product (bbg_id);
CREATE INDEX IF NOT EXISTS idx_bbgprod_isin
    ON staging.bbg_product (isin) WHERE isin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_bbgprod_cusip
    ON staging.bbg_product (cusip) WHERE cusip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_bbgprod_tenant
    ON staging.bbg_product (tenant_id);

ALTER TABLE staging.bbg_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.bbg_product FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS bbgprod_tenant_read ON staging.bbg_product;
CREATE POLICY bbgprod_tenant_read ON staging.bbg_product
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS bbgprod_tenant_write ON staging.bbg_product;
CREATE POLICY bbgprod_tenant_write ON staging.bbg_product
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 5. rdp_product — Refinitiv raw landing
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.rdp_product (
    _load_run_id      uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    _source_row_num   int  NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,

    ric               varchar(50),
    isin              varchar(12),
    cusip             varchar(9),
    sedol             varchar(7),
    lipper_id         varchar(50),
    fund_legal_name   text,
    fund_display_name text,
    lipper_class      text,
    lipper_global_class text,
    asset_class       text,
    domicile          varchar(2),
    currency          varchar(3),
    launch_date       date,
    manager_name      text,
    promoter          text,
    umbrella          text,
    nav_date          date,
    total_net_assets  numeric(20,4),

    CONSTRAINT rdp_product_pkey PRIMARY KEY (_load_run_id, _source_row_num)
);

CREATE INDEX IF NOT EXISTS idx_rdpprod_ric
    ON staging.rdp_product (ric);
CREATE INDEX IF NOT EXISTS idx_rdpprod_isin
    ON staging.rdp_product (isin) WHERE isin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rdpprod_cusip
    ON staging.rdp_product (cusip) WHERE cusip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rdpprod_tenant
    ON staging.rdp_product (tenant_id);

ALTER TABLE staging.rdp_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.rdp_product FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rdpprod_tenant_read ON staging.rdp_product;
CREATE POLICY rdpprod_tenant_read ON staging.rdp_product
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS rdpprod_tenant_write ON staging.rdp_product;
CREATE POLICY rdpprod_tenant_write ON staging.rdp_product
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- 6. product_incoming — canonical shape, post-mapping
-- ═══════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS staging.product_incoming (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_id  uuid NOT NULL,
    source_system_cd  varchar(30) NOT NULL,
    source_row_id     text NOT NULL,
    load_run_id       uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    tenant_id         uuid NOT NULL,

    -- Canonical columns (mirrors mdm.product)
    product_cd        varchar(50),
    name              text,
    legal_name        text,
    short_name        text,
    product_type_cd   varchar(30),
    product_category_cd varchar(50),
    manager_name      text,
    inception_date    date,
    base_currency     varchar(3),
    domicile          varchar(2),
    benchmark_name    text,
    is_active         boolean,
    aum               numeric(20,4),
    aum_currency      varchar(3),

    -- Full mapped payload
    canonical_payload jsonb NOT NULL DEFAULT '{}'::jsonb,

    -- Traceability
    mapping_version   int NOT NULL DEFAULT 1,
    mapped_at         timestamptz NOT NULL DEFAULT now(),

    -- Validation state
    is_valid          boolean NOT NULL DEFAULT true,
    validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb,

    CONSTRAINT chk_pi_valid CHECK (
        (is_valid = true AND jsonb_array_length(validation_errors) = 0)
        OR (is_valid = false)
    )
);

CREATE INDEX IF NOT EXISTS idx_pi_source
    ON staging.product_incoming (source_system_id, product_cd);
CREATE INDEX IF NOT EXISTS idx_pi_isin_lookup
    ON staging.product_incoming ((canonical_payload->>'isin'))
    WHERE canonical_payload ? 'isin';
CREATE INDEX IF NOT EXISTS idx_pi_valid
    ON staging.product_incoming (is_valid, load_run_id)
    WHERE is_valid = false;
CREATE INDEX IF NOT EXISTS idx_pi_load
    ON staging.product_incoming (load_run_id);
CREATE INDEX IF NOT EXISTS idx_pi_tenant
    ON staging.product_incoming (tenant_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pi_source_row
    ON staging.product_incoming (tenant_id, source_system_id, source_row_id);

ALTER TABLE staging.product_incoming ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.product_incoming FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pi_tenant_read ON staging.product_incoming;
CREATE POLICY pi_tenant_read ON staging.product_incoming
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS pi_tenant_write ON staging.product_incoming;
CREATE POLICY pi_tenant_write ON staging.product_incoming
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

-- ═══════════════════════════════════════════════════════════════════════
-- Verification
-- ═══════════════════════════════════════════════════════════════════════
DO $verify$
DECLARE
    t text;
    tables text[] := ARRAY[
        '_load_run','_mapping_error',
        'ff_product','bbg_product','rdp_product',
        'product_incoming'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema='staging' AND table_name=t
        ) THEN
            RAISE EXCEPTION 'staging.% was not created', t;
        END IF;
    END LOOP;
    RAISE NOTICE '008_staging_product: 6 tables created/verified';
END
$verify$;

COMMIT;

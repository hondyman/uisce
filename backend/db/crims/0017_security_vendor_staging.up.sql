-- 0017_security_vendor_staging.up.sql
-- Landing tables for security reference data from Bloomberg, Refinitiv and ICE, one per vendor in the
-- vendor's own layout (as staging.bbg_product / rdp_product / ff_product). Loaded by data pipelines,
-- bound to the Security business object by staging bindings, mastered into mdm.security_master.
-- Run against crims. Additive and idempotent.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0017_security_vendor_staging.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TABLE IF NOT EXISTS staging.bbg_security (
    _load_run_id      uuid NOT NULL,
    _source_row_num   integer NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    id_bb_global      varchar(12),      -- Bloomberg global id (FIGI)
    id_bb_unique      varchar(30),      -- Bloomberg unique id (record key)
    ticker            varchar(40),
    exch_code         varchar(10),
    id_isin           varchar(12),
    id_cusip          varchar(9),
    id_sedol1         varchar(7),
    name              text,
    security_des      text,
    security_typ      text,
    asset_class       varchar(20),
    crncy             varchar(3),
    cntry_of_domicile varchar(2),
    cntry_of_risk     varchar(2),
    gics_sector_name  text,
    gics_industry_name text,
    issue_dt          date,
    maturity          date,
    first_trade_dt    date,
    mic_primary       varchar(10),
    market_status     varchar(20)
);

CREATE TABLE IF NOT EXISTS staging.rdp_security (
    _load_run_id      uuid NOT NULL,
    _source_row_num   integer NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    perm_id           varchar(20),      -- Refinitiv instrument PermID (record key)
    ric               varchar(30),
    isin              varchar(12),
    cusip             varchar(9),
    sedol             varchar(7),
    instrument_name   text,
    short_name        text,
    instrument_type   text,
    asset_class       varchar(20),
    currency          varchar(3),
    country_of_issue  varchar(2),
    trbc_sector       text,
    issue_date        date,
    maturity_date     date,
    exchange_mic      varchar(10),
    status            varchar(20),
    as_of_date        date
);

CREATE TABLE IF NOT EXISTS staging.ice_security (
    _load_run_id      uuid NOT NULL,
    _source_row_num   integer NOT NULL,
    _ingested_at      timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    ice_id            varchar(20),      -- ICE instrument id (record key)
    isin              varchar(12),
    cusip             varchar(9),
    issuer_name       text,
    description       text,
    instrument_type   text,
    asset_class       varchar(20),
    currency          varchar(3),
    country_of_risk   varchar(2),
    issue_date        date,
    first_trade_date  date,
    maturity_date     date,
    callable_date     date,
    settlement_currency varchar(3),
    status            varchar(20)
);

DO $rls$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['bbg_security', 'rdp_security', 'ice_security'] LOOP
        EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON staging.%I (tenant_id, _load_run_id)', 'ix_' || t || '_run', t);
        EXECUTE format('ALTER TABLE staging.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE staging.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON staging.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON staging.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$, t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON staging.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON staging.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$, t || '_tenant_write', t);
    END LOOP;
END
$rls$;

COMMIT;

-- 0018_price_master_profile.up.sql
-- The price master on the mastering engine (docs/mdm-price-mastering-design.md, slices 1-3): end-of-day
-- prices from Bloomberg, Refinitiv and ICE resolved to golden securities, observed with restatements,
-- survived per security x price type x valuation date, published as bitemporal golden prices. Run against
-- crims. Additive and idempotent.
--
--   * mdm.mastering_entity.kind - RECORD (product, security: one golden record per entity) or
--     TIMESERIES (price: one golden value per entity x series x date, mastered set-based per date).
--   * One vendor registry for prices too: every foreign key to mdm.price_source now references
--     mdm.source_systems (the referencing tables are empty); mdm.price_source keeps each source's price
--     settings (cutoff, time zone, type) and points at its registry source (source_system_id).
--   * mdm.price_golden_record: one current version per key (partial unique index).
--   * Vendor price staging tables staging.bbg_price / rdp_price / ice_price, in each vendor's layout.
--   * PRICE profile (gold copy), source priority by asset class, variance thresholds, the price
--     survivorship rule and the override/merge policy.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0018_price_master_profile.up.sql

\set ON_ERROR_STOP on
BEGIN;

-- 1. Profile kind -----------------------------------------------------------------------------------
ALTER TABLE mdm.mastering_entity ADD COLUMN IF NOT EXISTS kind varchar(20) NOT NULL DEFAULT 'RECORD';
DO $k$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_mastering_entity_kind') THEN
        ALTER TABLE mdm.mastering_entity ADD CONSTRAINT chk_mastering_entity_kind CHECK (kind IN ('RECORD', 'TIMESERIES'));
    END IF;
END
$k$;

-- 2. One vendor registry for prices --------------------------------------------------------------------
-- Price sources that are the same vendor as a registry source point at it (IDC is ICE Data Services, WM
-- fixings are Refinitiv's, BFIX is Bloomberg's); the others join the registry under their own code.
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
INSERT INTO mdm.source_systems (id, tenant_id, code, display_name)
SELECT md5('source-system:' || p.source_cd)::uuid, '99e99e99-99e9-49e9-89e9-99e99e99e999', p.source_cd, p.name
  FROM (SELECT DISTINCT ON (source_cd) source_cd, name FROM mdm.price_source ORDER BY source_cd) p
 WHERE p.source_cd NOT IN ('BLOOMBERG', 'REFINITIV', 'ICE', 'IDC', 'WM_REFINITIV', 'BFIX')
   AND NOT EXISTS (SELECT 1 FROM mdm.source_systems s WHERE s.code = p.source_cd);

ALTER TABLE mdm.price_source ADD COLUMN IF NOT EXISTS source_system_id uuid REFERENCES mdm.source_systems(id);
UPDATE mdm.price_source ps
   SET source_system_id = s.id
  FROM mdm.source_systems s
 WHERE ps.source_system_id IS NULL
   AND s.code = CASE ps.source_cd WHEN 'IDC' THEN 'ICE' WHEN 'WM_REFINITIV' THEN 'REFINITIV' WHEN 'BFIX' THEN 'BLOOMBERG'
                                  ELSE ps.source_cd END;
COMMENT ON COLUMN mdm.price_source.source_system_id IS
    'The registry source (mdm.source_systems) these price settings belong to; prices reference the registry.';

DO $fk$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT con.conname, con.conrelid::regclass AS tbl,
               (SELECT string_agg(quote_ident(a.attname), ', ' ORDER BY k.ord)
                  FROM unnest(con.conkey) WITH ORDINALITY k(attnum, ord)
                  JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.attnum) AS cols,
               con.confdeltype
          FROM pg_constraint con
         WHERE con.contype = 'f' AND con.confrelid = 'mdm.price_source'::regclass
           AND con.conrelid <> 'mdm.price_source'::regclass
    LOOP
        EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', c.tbl, c.conname);
        EXECUTE format('ALTER TABLE %s ADD CONSTRAINT %I FOREIGN KEY (%s) REFERENCES mdm.source_systems(id)%s',
                       c.tbl, c.conname, c.cols,
                       CASE c.confdeltype WHEN 'c' THEN ' ON DELETE CASCADE' WHEN 'n' THEN ' ON DELETE SET NULL' ELSE '' END);
    END LOOP;
END
$fk$;

-- 3. Golden prices: one current version per key -----------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS uq_pgr_current ON mdm.price_golden_record
    (tenant_id, price_entity_type, price_entity_id, price_date, price_type_cd) WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_pgr_key_version ON mdm.price_golden_record
    (tenant_id, price_entity_type, price_entity_id, price_type_cd, price_date, golden_version DESC);
CREATE INDEX IF NOT EXISTS idx_price_key ON mdm.price
    (tenant_id, price_entity_type, price_entity_id, price_type_id, price_date) WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_pex_open ON mdm.price_exception (tenant_id, price_date, exception_type) WHERE status = 'OPEN';

-- 4. Vendor price staging, in each vendor's layout ------------------------------------------------------
CREATE TABLE IF NOT EXISTS staging.bbg_price (
    _load_run_id    uuid NOT NULL,
    _source_row_num integer NOT NULL,
    _ingested_at    timestamptz NOT NULL DEFAULT now(),
    tenant_id       uuid NOT NULL,
    id_bb_global    varchar(12),       -- FIGI
    id_isin         varchar(12),
    id_cusip        varchar(9),
    ticker          varchar(40),
    exch_code       varchar(10),
    px_date         date,
    crncy           varchar(3),
    px_last         numeric,
    px_bid          numeric,
    px_ask          numeric,
    px_mid          numeric,
    last_update_dt  timestamptz
);

CREATE TABLE IF NOT EXISTS staging.rdp_price (
    _load_run_id    uuid NOT NULL,
    _source_row_num integer NOT NULL,
    _ingested_at    timestamptz NOT NULL DEFAULT now(),
    tenant_id       uuid NOT NULL,
    ric             varchar(30),
    isin            varchar(12),
    cusip           varchar(9),
    trade_date      date,
    currency        varchar(3),
    trdprc_1        numeric,           -- last trade
    official_close  numeric,
    bid             numeric,
    ask             numeric,
    mid_price       numeric,
    value_ts        timestamptz
);

CREATE TABLE IF NOT EXISTS staging.ice_price (
    _load_run_id    uuid NOT NULL,
    _source_row_num integer NOT NULL,
    _ingested_at    timestamptz NOT NULL DEFAULT now(),
    tenant_id       uuid NOT NULL,
    ice_id          varchar(20),
    isin            varchar(12),
    cusip           varchar(9),
    pricing_date    date,
    currency        varchar(3),
    bid_price       numeric,           -- evaluated
    mid_price       numeric,
    ask_price       numeric,
    evaluated_at    timestamptz
);

DO $rls$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['bbg_price', 'rdp_price', 'ice_price'] LOOP
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

-- 5. Gold-copy configuration --------------------------------------------------------------------------
INSERT INTO mdm.mastering_entity
    (tenant_id, entity_cd, display_name, bo_key, table_prefix, anchor_table, anchor_code_column,
     identifier_table, incoming_table, code_prefix, kind, settings)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRICE', 'Price', 'price', 'price', 'mdm.price_golden_record',
        'price_type_cd', NULL, 'staging.price_incoming', 'PRC-', 'TIMESERIES', '{
          "series": {
            "entity_type": "SECURITY",
            "instrument": "SECURITY",
            "identifier_order": ["FIGI", "ISIN", "CUSIP", "SEDOL", "BLOOMBERG_ID", "RIC"],
            "date_field": "ValuationDate",
            "currency_field": "Currency",
            "value_field": "Price",
            "type_field": "PriceTypeCd",
            "observation_type": "EOD",
            "controls": {
              "day_over_day": {"error": "FLAG", "critical": "HOLD"},
              "cross_source": true,
              "currency_mismatch": "EXCLUDE"
            }
          }
        }'::jsonb)
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

-- Source priority: Bloomberg first by default; ICE's evaluated prices first for fixed income.
-- Unscoped columns (NULL) apply to every asset class, price type and currency.
INSERT INTO mdm.price_source_priority
    (tenant_id, price_entity_type, asset_class_cd, source_id, priority, is_fallback, max_staleness_minutes)
SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999', 'SECURITY', g.ac, s.id, g.prio, false, 4320
  FROM (VALUES
        (NULL, 'BLOOMBERG', 10), (NULL, 'REFINITIV', 20), (NULL, 'ICE', 30),
        ('FixedIncome', 'ICE', 10), ('FixedIncome', 'BLOOMBERG', 20), ('FixedIncome', 'REFINITIV', 30)
       ) AS g(ac, code, prio)
  JOIN mdm.source_systems s ON s.code = g.code
 WHERE NOT EXISTS (SELECT 1 FROM mdm.price_source_priority x
                    WHERE x.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND x.price_entity_type = 'SECURITY'
                      AND x.asset_class_cd IS NOT DISTINCT FROM g.ac AND x.price_type_cd IS NULL AND x.source_id = s.id);

-- Variance thresholds (percent): day-over-day moves and cross-source disagreement.
INSERT INTO mdm.price_variance_threshold
    (tenant_id, asset_class_cd, threshold_type, warning_threshold, error_threshold, critical_threshold)
SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999', t.ac, 'PERCENTAGE', t.w, t.e, t.c
  FROM (VALUES ('*', 5, 10, 25), ('Equity', 5, 10, 25), ('FixedIncome', 1, 3, 10)) AS t(ac, w, e, c)
 WHERE NOT EXISTS (SELECT 1 FROM mdm.price_variance_threshold x
                    WHERE x.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND x.asset_class_cd = t.ac
                      AND x.sec_sub_typ_cd IS NULL AND x.price_type_cd IS NULL);

-- The golden value: the source ranking, a source older than three days loses to a fresh one. A consensus
-- selection rule (catalog domain survivorship) can be named here once written in the rule builder.
INSERT INTO mdm.survivorship_rule (tenant_id, entity_type, attribute_name, strategy, staleness_max_age_sec)
SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRICE', 'value', 'SOURCE_PRIORITY', 259200
 WHERE NOT EXISTS (SELECT 1 FROM mdm.survivorship_rule
                    WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_type = 'PRICE' AND attribute_name = 'value');

INSERT INTO mdm.mastering_policy (tenant_id, entity_cd, override_mode, approvals_required, updated_by)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRICE', 'APPROVAL', 1, 'migration 0018')
ON CONFLICT (tenant_id, entity_cd) DO NOTHING;

COMMIT;

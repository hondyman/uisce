-- preflight_crims.sql
-- Read-only. No DDL. No DML. Run against crims.
\set ON_ERROR_STOP on
\timing on

\echo '=== 0.5 target location check ==='
SELECT
    to_regclass('edm.issuer_master')     AS crims_edm_issuer,
    to_regclass('mdm.issuer_master')     AS crims_mdm_issuer,
    to_regclass('mdm.product')           AS crims_mdm_product,
    to_regclass('mdm.counterparty')      AS crims_mdm_counterparty,
    to_regclass('mdm.benchmark_master')  AS crims_mdm_benchmark,
    to_regclass('mdm.calendar_master')   AS crims_mdm_calendar,
    to_regclass('mdm.ca_event')          AS crims_mdm_ca,
    to_regclass('mdm.price')             AS crims_mdm_price;

\echo '=== 0.6 crims.mdm.party population ==='
SELECT
    count(*)                  AS total_rows,
    count(DISTINCT tenant_id) AS tenant_count
FROM mdm.party;

\echo '=== 0.6b crims.mdm table count (for baseline) ==='
SELECT count(*) AS crims_mdm_tables
FROM information_schema.tables
WHERE table_schema = 'mdm'
  AND table_type = 'BASE TABLE';

\echo '=== 0.6c crims schemas present ==='
SELECT nspname
FROM pg_namespace
WHERE nspname NOT LIKE 'pg_%'
  AND nspname <> 'information_schema'
ORDER BY nspname;

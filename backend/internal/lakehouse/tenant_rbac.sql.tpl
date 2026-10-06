-- Tenant isolation: RBAC + resource groups.
-- Rendered by render_tenant_rbac.sh at startup. Do not run raw.
--
-- Tenant naming convention:
--   tenant_northwinds   — gold-copy workload (Northwind sample data)
--   tenant_crd_bakeoff  — Demo Tenant: CRD Bakeoff
--
-- CRD Bakeoff has SELECT on Northwinds (gold copy reference).
-- DELETE the matching GRANT line below if tenants must be fully opaque.
--
-- !!! StarRocks 3.3 syntax only. !!!
--   - Resource groups are RESOURCE GROUP (3.4+ renamed to WORKLOAD GROUP).
--   - Properties are single-quoted; mem_limit is a percentage string.
--   - CREATE RESOURCE GROUP has no IF NOT EXISTS — re-running this script
--     after a partial first run will fail at the duplicate-group step.
--     Run the cleanup block in render_tenant_rbac.sh (or
--     backend/internal/lakehouse/tenant_rbac_cleanup.sql.tpl) first.
--   - On 4.1 this becomes CREATE WORKLOAD GROUP IF NOT EXISTS ... plus
--     a separate CREATE WORKLOAD FILTER statement keyed on user, which
--     removes the need for SET workload_group on every connection.

-- ============================================================
-- 1. Databases
-- ============================================================
CREATE DATABASE IF NOT EXISTS tenant_northwinds;
CREATE DATABASE IF NOT EXISTS tenant_crd_bakeoff;

-- ============================================================
-- 2. Roles
-- ============================================================
CREATE ROLE IF NOT EXISTS tenant_northwinds_rw;
GRANT ALL ON ALL TABLES IN DATABASE tenant_northwinds TO ROLE tenant_northwinds_rw;
GRANT CREATE TABLE ON DATABASE tenant_northwinds TO ROLE tenant_northwinds_rw;

CREATE ROLE IF NOT EXISTS tenant_crd_bakeoff_rw;
GRANT ALL ON ALL TABLES IN DATABASE tenant_crd_bakeoff TO ROLE tenant_crd_bakeoff_rw;
GRANT CREATE TABLE ON DATABASE tenant_crd_bakeoff TO ROLE tenant_crd_bakeoff_rw;

-- Gold copy: CRD Bakeoff gets READ-ONLY on northwinds.
-- DELETE these two lines if tenants must be fully opaque to each other.
GRANT SELECT ON ALL TABLES IN DATABASE tenant_northwinds TO ROLE tenant_crd_bakeoff_rw;

-- ============================================================
-- 3. Service users (passwords injected by render, never hardcoded)
-- ============================================================
CREATE USER IF NOT EXISTS tenant_northwinds_svc IDENTIFIED BY '${TENANT_NORTHWINDS_PASSWORD}';
GRANT tenant_northwinds_rw TO USER tenant_northwinds_svc;
-- ALTER USER ... DEFAULT ROLE is the form StarRocks 3.3 accepts; SET
-- DEFAULT ROLE is rejected by the parser in 3.3.
ALTER USER tenant_northwinds_svc DEFAULT ROLE tenant_northwinds_rw;

CREATE USER IF NOT EXISTS tenant_crd_bakeoff_svc IDENTIFIED BY '${TENANT_CRD_BAKEOFF_PASSWORD}';
GRANT tenant_crd_bakeoff_rw TO USER tenant_crd_bakeoff_svc;
ALTER USER tenant_crd_bakeoff_svc DEFAULT ROLE tenant_crd_bakeoff_rw;

-- ============================================================
-- 4. Stream loader: least privilege (no more root for loads)
-- ============================================================
CREATE USER IF NOT EXISTS stream_loader IDENTIFIED BY '${STREAM_LOADER_PASSWORD}';
CREATE ROLE IF NOT EXISTS loader;
GRANT CREATE TABLE ON DATABASE tenant_northwinds TO ROLE loader;
GRANT INSERT ON ALL TABLES IN DATABASE tenant_northwinds TO ROLE loader;
GRANT CREATE TABLE ON DATABASE tenant_crd_bakeoff TO ROLE loader;
GRANT INSERT ON ALL TABLES IN DATABASE tenant_crd_bakeoff TO ROLE loader;
GRANT loader TO USER stream_loader;
ALTER USER stream_loader DEFAULT ROLE loader;

-- ============================================================
-- 5. Resource groups (one per tenant service user)
--    12 cores / 12.5 GB node: cpu_weight is bounded by core count, not %.
--    Each tenant = 4 vs. default_wg = 12 → tenants share ~40% of CPU
--    scheduling weight; default_wg keeps ~60%. sum(mem_limit) = 70%.
-- ============================================================
-- 3.3 has no IF NOT EXISTS, so re-running after a successful apply requires
-- manual cleanup (DROP RESOURCE GROUP name) or the cleanup template.

CREATE RESOURCE GROUP tenant_northwinds_wg
TO (user='tenant_northwinds_svc')
WITH (
    'cpu_weight' = '4',
    'mem_limit' = '35%',
    'concurrency_limit' = '10',
    'big_query_cpu_second_limit' = '300',
    'big_query_scan_rows_limit' = '10000000000',
    'big_query_mem_limit' = '2147483648'
);

CREATE RESOURCE GROUP tenant_crd_bakeoff_wg
TO (user='tenant_crd_bakeoff_svc')
WITH (
    'cpu_weight' = '4',
    'mem_limit' = '35%',
    'concurrency_limit' = '10',
    'big_query_cpu_second_limit' = '300',
    'big_query_scan_rows_limit' = '10000000000',
    'big_query_mem_limit' = '2147483648'
);

-- 3.3 has no workload filters: each service must run on connect:
--   SET workload_group = 'tenant_northwinds_wg';
--   SET workload_group = 'tenant_crd_bakeoff_wg';
-- (NOT workload_group on 3.3 — it's resource_group. SET RESOURCE GROUP is
-- the 3.3 form. Note the SET form is per-session, not persistent.)
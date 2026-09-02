-- 20260901000002_enforce_product_datasource_catalog.sql
-- Phase 3 of the connections/datasource redesign: enforce that a tenant's
-- tenant_product_datasource.alpha_datasource_id must be one the gold-copy
-- tenant has actually catalogued for that alpha_product_id (see
-- 20260901000001_product_datasource_catalog.sql for the catalog table and
-- the Phase 1 violation-report backfill). Verified clean against live data
-- before this was applied — zero violating rows.
--
-- Postgres composite FKs use MATCH SIMPLE by default: a row is exempt from
-- the check if EITHER column is NULL, so datasources still being configured
-- (alpha_datasource_id not yet chosen) are unaffected.

ALTER TABLE tenant_product_datasource
    ADD CONSTRAINT tenant_product_datasource_catalog_fkey
    FOREIGN KEY (alpha_product_id, alpha_datasource_id)
    REFERENCES product_datasource_catalog (alpha_product_id, alpha_datasource_id);

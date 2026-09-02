-- 20260901000003_tenant_product_datasource_slot_unique.sql
-- Phase 4 of the connections/datasource redesign: guarantee exactly one
-- tenant_product_datasource row per (tenant, alpha_product, alpha_datasource)
-- "logical slot" (e.g. tenant X's own "ORM Connection" for product Y). This
-- is what makes per-tenant datasource resolution for shared/core Business
-- Objects unambiguous — see product_datasource_catalog
-- (20260901000001) for the allow-list this slot concept builds on.
--
-- Verified against live data before writing this: zero rows currently
-- violate this uniqueness.

CREATE UNIQUE INDEX IF NOT EXISTS idx_tpd_tenant_product_datasource_slot
    ON tenant_product_datasource (tenant_id, alpha_product_id, alpha_datasource_id)
    WHERE alpha_datasource_id IS NOT NULL;

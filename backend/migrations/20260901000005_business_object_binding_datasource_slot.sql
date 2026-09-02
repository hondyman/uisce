-- 20260901000005_business_object_binding_datasource_slot.sql
-- Lets a Business Object binding declare WHICH logical datasource it needs
-- (e.g. "ORM Connection" = a specific alpha_product_id + alpha_datasource_id
-- pair from product_datasource_catalog), without hardcoding any tenant's
-- specific tenant_product_datasource row. A core binding is authored once,
-- by gold copy, against this slot; each tenant resolves it to their own
-- connection at request time via ResolveBindingDatasource (tenant_id +
-- slot -> that tenant's tenant_product_datasource.id, unique per
-- 20260901000003's index). Query execution itself already routes on
-- secCtx.DatasourceID (validated per-request, per-tenant) — this only feeds
-- the caller the right value to put there.

ALTER TABLE business_object_bindings
    ADD COLUMN IF NOT EXISTS alpha_product_id UUID,
    ADD COLUMN IF NOT EXISTS alpha_datasource_id UUID;

ALTER TABLE business_object_bindings
    ADD CONSTRAINT business_object_bindings_catalog_fkey
    FOREIGN KEY (alpha_product_id, alpha_datasource_id)
    REFERENCES product_datasource_catalog (alpha_product_id, alpha_datasource_id);

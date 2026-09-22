-- Follow-up to fix_orphan_orm_backend.sql: give the orphan backend
-- (e102c0bf-e111-4f9d-81f9-2801ff2a3231) a tenant_product_datasource row,
-- which is the table live record queries actually read connection config
-- from (the same table/shape the catalog scanner uses), not
-- public.connections.
--
-- Why: the previous fix cloned a public.connections row and fixed
-- public.physical_backend's metadata for this id, which was necessary but
-- not sufficient -- BusinessObjectService.resolveRecordsDB (added today to
-- fix "relation \"orm.security\" does not exist") resolves a binding's
-- physical database via tenant_product_datasource.config (JSON), the exact
-- mechanism the catalog scanner already uses successfully for this same
-- CRIMS connection. With no tenant_product_datasource row for
-- e102c0bf-..., the resolver correctly detected that and fell back to the
-- alpha DB, reproducing the original error.
--
-- This clones the working "CRIMS ORM Database" tenant_product_datasource
-- row (441f62c9-...) under the orphan backend's id, keeping every FK
-- (tenant_product_id, alpha_datasource_id, alpha_product_id,
-- tenant_instance_id, tenant_id) identical -- it is the same physical
-- datasource, just needing a second id because business_object_binding
-- enforces one binding per (tenant_id, bo_id, backend_id) and Security
-- already has a binding on 441f62c9 for the MDM golden-record table. Only
-- source_name is changed, to satisfy tenant_product_datasource's
-- (tenant_product_id, datasource_id, source_name) uniqueness. No
-- credential value is ever displayed.
--
-- Usage:
--   psql "$DATABASE_URL" -v mode=plan  -f fix_orphan_orm_backend_datasource.sql
--   psql "$DATABASE_URL" -v mode=apply -f fix_orphan_orm_backend_datasource.sql
\if :{?mode}
\else
  \set mode 'plan'
\endif

\set ON_ERROR_STOP on

\echo '== mode:' :mode

BEGIN;

SELECT (:'mode' = 'apply') AS is_apply \gset

DO $$
BEGIN
  IF (SELECT count(*) FROM public.tenant_product_datasource WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3') = 0 THEN
    RAISE EXCEPTION 'ABORT: source tenant_product_datasource 441f62c9-aad1-481d-9aab-62943fa11cd3 not found';
  END IF;
  IF (SELECT count(*) FROM public.physical_backend WHERE backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231') = 0 THEN
    RAISE EXCEPTION 'ABORT: orphan backend e102c0bf-e111-4f9d-81f9-2801ff2a3231 not found (nothing to fix)';
  END IF;
END $$;

\echo '-- source row (config redacted) --'
SELECT id, source_name, tenant_product_id, alpha_datasource_id, config - 'auth' - 'password' - 'client_cert' - 'private_key' - 'ca_cert' AS config_safe
FROM public.tenant_product_datasource
WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3';

INSERT INTO public.tenant_product_datasource
  (id, tenant_product_id, alpha_tenant_instance_id, is_active, config, source_name,
   connection_id, environment, tags, description, read_only, pool_config, scan_schedule,
   health_config, integrity_checks, sla_config, data_classification,
   tenant_instance_id, core_id, datasource_id, alpha_datasource_id, tenant_id, alpha_product_id)
SELECT
  'e102c0bf-e111-4f9d-81f9-2801ff2a3231',
  tenant_product_id, alpha_tenant_instance_id, is_active, config,
  'CRIMS ORM Database (bo binding consolidation)',
  connection_id, environment, tags, description, read_only, pool_config, scan_schedule,
  health_config, integrity_checks, sla_config, data_classification,
  tenant_instance_id, core_id, datasource_id, alpha_datasource_id, tenant_id, alpha_product_id
FROM public.tenant_product_datasource
WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3'
  AND NOT EXISTS (SELECT 1 FROM public.tenant_product_datasource WHERE id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231');

\echo '-- clone after insert (no credential columns selected) --'
SELECT id, source_name, tenant_product_id, alpha_datasource_id, is_active
FROM public.tenant_product_datasource
WHERE id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231';

\if :is_apply
  \echo '== APPLY: committing'
  COMMIT;
\else
  \echo '== PLAN: rolling back (no changes made)'
  ROLLBACK;
\endif

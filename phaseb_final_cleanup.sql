-- phaseb_final_cleanup.sql
-- FINAL destructive cleanup: DROP legacy *_old columns and update views to canonical tenant_datasource_id.
-- IMPORTANT: Run only in a maintenance window after a verified backup.
-- Preview and review carefully before executing.

-- Recommended snapshot command (run manually or uncomment to run automatically):
-- export PGPASSWORD=postgres
-- pg_dump -Fc -h localhost -p 5432 -U postgres -d alpha -f ~/alpha_pre_phaseb_final_cleanup_$(date +%Y%m%d%H%M%S).dump

-- ======= Begin cleanup =======
BEGIN;

-- 1) Drop legacy backup columns (these were left as tenant_datasource_id_old)
ALTER TABLE public.catalog_node DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.catalog_node_type DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.tenant_chart DROP COLUMN IF EXISTS tenant_datasource_id_old;

-- 2) Drop compatibility/text id if present
ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS id_text;

-- 3) Update views to reference canonical tenant_datasource_id
-- Example: catalog_node_vw
CREATE OR REPLACE VIEW public.catalog_node_vw AS
SELECT
  cn.tenant_datasource_id,
  tpd.source_name,
  cn.id AS node_id,
  cn.node_name,
  cnt.catalog_type_name,
  COALESCE(cnt.config, jsonb_build_object('properties', cnt.properties)) AS catalog_defn,
  cn.node_type_id,
  cn.description,
  cn.qualified_path,
  cn.properties,
  cn.parent_id
FROM public.catalog_node cn
JOIN public.catalog_node_type cnt ON cnt.id = cn.node_type_id
JOIN public.tenant_product_datasource tpd ON tpd.id = cn.tenant_datasource_id;

-- Add other view updates here if needed. Example placeholders:
-- CREATE OR REPLACE VIEW public.other_view AS
-- SELECT ... FROM ... JOIN ... ON ... = tenant_datasource_id;

COMMIT;

-- ======= Post-run verification (run after the above completes) =======
-- SELECT table_name, column_name FROM information_schema.columns WHERE table_name IN ('catalog_node','catalog_node_type','tenant_chart') AND column_name LIKE '%tenant_datasource%';
-- SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id IS NULL;
-- SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id IS NULL;
-- SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id IS NULL;
-- SELECT viewname FROM pg_views WHERE definition ILIKE '%tenant_datasource_id_old%';

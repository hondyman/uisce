-- phaseb_final_cleanup_fix.sql
-- Recreate views that reference tenant_datasource_id_old to use canonical tenant_datasource_id,
-- then drop legacy *_old columns and id_text.
-- Run in maintenance window. This script will attempt a conservative text-replace in view definitions.

BEGIN;

-- Recreate affected views by replacing occurrences of tenant_datasource_id_old with tenant_datasource_id
DO $$ DECLARE
  r RECORD;
  newdef TEXT;
BEGIN
  FOR r IN SELECT viewname, definition FROM pg_views WHERE schemaname='public' AND definition ILIKE '%tenant_datasource_id_old%' LOOP
    newdef := replace(r.definition, 'tenant_datasource_id_old', 'tenant_datasource_id');
    EXECUTE format('CREATE OR REPLACE VIEW public.%I AS %s', r.viewname, newdef);
    RAISE NOTICE 'Recreated view %', r.viewname;
  END LOOP;
END $$ LANGUAGE plpgsql;

-- Now safe to drop legacy columns and compatibility text id
ALTER TABLE public.catalog_node DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.catalog_node_type DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.tenant_chart DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS id_text;

COMMIT;

-- Post-run verification suggestions (run after this completes):
-- SELECT viewname FROM pg_views WHERE definition ILIKE '%tenant_datasource_id_old%';
-- SELECT table_name, column_name FROM information_schema.columns WHERE table_name IN ('catalog_node','catalog_node_type','tenant_chart') AND column_name LIKE '%tenant_datasource%';

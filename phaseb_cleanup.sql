-- phaseb_cleanup.sql
-- Conservative destructive cleanup: promote tenant_datasource_id_uuid to tenant_datasource_id
-- and rename legacy tenant_datasource_id -> tenant_datasource_id_old. Does NOT DROP the _old columns.
-- Run inside maintenance window. This script performs DDL changes and will acquire locks.

BEGIN;

DO $$ DECLARE
  t text;
BEGIN
  FOR t IN SELECT unnest(ARRAY['catalog_node','catalog_node_type','tenant_chart']) LOOP
    -- If both legacy and uuid columns exist, rename legacy -> _old and promote uuid -> canonical
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id_uuid') THEN

      EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id TO tenant_datasource_id_old', t);
      RAISE NOTICE 'Renamed % tenant_datasource_id -> tenant_datasource_id_old', t;

      EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id', t);
      RAISE NOTICE 'Promoted % tenant_datasource_id_uuid -> tenant_datasource_id', t;

    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id_uuid')
          AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id') THEN

      EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id', t);
      RAISE NOTICE 'Promoted % tenant_datasource_id_uuid -> tenant_datasource_id (no legacy present)', t;

    ELSE
      RAISE NOTICE 'Skipping %: no action needed or missing columns', t;
    END IF;
  END LOOP;
END $$ LANGUAGE plpgsql;

-- Optionally drop compatibility/text id on tenant_product_datasource
ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS id_text;

COMMIT;

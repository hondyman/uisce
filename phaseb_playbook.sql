-- Phase-B production playbook (parameterized)
-- IMPORTANT: Review and replace placeholders before running in production.
-- Usage: edit this file to set mapping rules and environment-specific options. Run in a maintenance window.

-- =========================
-- 0) CONFIG
-- Replace these values as needed
\set TARGET_TABLES 'catalog_node,catalog_node_type,tenant_chart'
\set BATCH_SIZE 10000
-- Provide mapping rules by uncommenting and filling entries in the temporary table below

-- =========================
-- 1) PRE-CHECKS - collect diagnostics
-- (run these and capture output to a file)

-- show candidate tables and column types
SELECT table_name,
       MAX(CASE WHEN column_name = 'tenant_datasource_id' THEN data_type END) AS legacy_type,
       MAX(CASE WHEN column_name = 'tenant_datasource_id_uuid' THEN data_type END) AS uuid_type
FROM information_schema.columns
WHERE table_schema = 'public' AND column_name IN ('tenant_datasource_id','tenant_datasource_id_uuid')
GROUP BY table_name
ORDER BY table_name;

-- per-table null counts
-- replace <table> for each target
SELECT 'catalog_node' AS tbl, COUNT(*) FILTER (WHERE tenant_datasource_id_uuid IS NULL) AS null_uuid_count, COUNT(*) FILTER (WHERE tenant_datasource_id IS NULL) AS null_legacy_count FROM public.catalog_node;
SELECT 'catalog_node_type' AS tbl, COUNT(*) FILTER (WHERE tenant_datasource_id_uuid IS NULL) AS null_uuid_count, COUNT(*) FILTER (WHERE tenant_datasource_id IS NULL) AS null_legacy_count FROM public.catalog_node_type;
SELECT 'tenant_chart' AS tbl, COUNT(*) FILTER (WHERE tenant_datasource_id_uuid IS NULL) AS null_uuid_count, COUNT(*) FILTER (WHERE tenant_datasource_id IS NULL) AS null_legacy_count FROM public.tenant_chart;

-- find constraints referencing tenant_product_datasource
SELECT conname, conrelid::regclass AS table, pg_get_constraintdef(oid) FROM pg_constraint WHERE contype = 'f' AND confrelid = 'public.tenant_product_datasource'::regclass;

-- show views that use legacy column
SELECT view_schema, view_name FROM information_schema.view_column_usage WHERE column_name = 'tenant_datasource_id';

-- =========================
-- 2) MAPPING
-- Option A: provide an explicit mapping table (legacy_value -> tpd.id)
-- If you create new tpd rows, insert them into tenant_product_datasource and reference their ids here.

-- Example (commented):
-- CREATE TEMP TABLE tmp_phaseb_map (legacy_text text, tpd_id uuid);
-- INSERT INTO tmp_phaseb_map VALUES ('legacy-abc', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid);

-- Option B: create a default tpd for unmapped rows (example)
-- INSERT INTO public.tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, is_active, source_name, created_at, updated_at)
-- VALUES (gen_random_uuid(), NULL, NULL, TRUE, 'phaseb-default-<env>-<ts>', now(), now());

-- =========================
-- 3) BACKFILL
-- Text-mapped backfill (only run if legacy column is text)
-- Update using mapping table (if you created one)
-- Example using tmp_phaseb_map (uncomment to use):
-- UPDATE public.<table> t SET tenant_datasource_id_uuid = m.tpd_id
-- FROM tmp_phaseb_map m
-- WHERE t.tenant_datasource_id_uuid IS NULL AND t.tenant_datasource_id::text = m.legacy_text;

-- Direct cast backfill when legacy column is uuid typed
-- (runs quickly, no WAL explosion beyond the updates)
UPDATE public.catalog_node_type SET tenant_datasource_id_uuid = tenant_datasource_id WHERE tenant_datasource_id_uuid IS NULL AND tenant_datasource_id IS NOT NULL;
UPDATE public.tenant_chart SET tenant_datasource_id_uuid = tenant_datasource_id WHERE tenant_datasource_id_uuid IS NULL AND tenant_datasource_id IS NOT NULL;
UPDATE public.catalog_node SET tenant_datasource_id_uuid = tenant_datasource_id WHERE tenant_datasource_id_uuid IS NULL AND tenant_datasource_id IS NOT NULL;

-- Bulk map to a specific tpd id if desired
-- UPDATE public.catalog_node_type SET tenant_datasource_id_uuid = '<TPD_UUID>'::uuid WHERE tenant_datasource_id_uuid IS NULL AND id IN (...);

-- =========================
-- 4) ADD AUTHORITATIVE FK (NOT VALID), then VALIDATE
-- Use NOT VALID to avoid long validations during business hours.
DO $$ DECLARE tbl text; BEGIN
  FOR tbl IN SELECT unnest(ARRAY['catalog_node','catalog_node_type','tenant_chart']) LOOP
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_uuid') THEN
      BEGIN
        EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_datasource_id_uuid) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE NOT VALID', tbl, tbl||'_tenant_product_datasource_uuid_fk');
      EXCEPTION WHEN duplicate_object THEN NULL; WHEN others THEN RAISE NOTICE 'Could not add uuid FK on %: %', tbl, SQLERRM; END;
      BEGIN
        EXECUTE format('ALTER TABLE public.%I VALIDATE CONSTRAINT %I', tbl, tbl||'_tenant_product_datasource_uuid_fk');
      EXCEPTION WHEN others THEN RAISE NOTICE 'Validate deferred for %: %', tbl, SQLERRM; END;
    END IF;
  END LOOP;
END $$ LANGUAGE plpgsql;

-- Verify zero-or-expected-null counts before destructive rename/drop
SELECT 'post-backfill', (SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id_uuid IS NULL), (SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id_uuid IS NULL), (SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id_uuid IS NULL);

-- =========================
-- 5) UPDATE VIEWS / DEPENDENT OBJECTS
-- For any view that referenced legacy tenant_datasource_id text column, update to use tenant_datasource_id_uuid::text or coalesce as appropriate.
-- Example: (replace view DDL as necessary)
-- DROP VIEW IF EXISTS public.catalog_node_vw CASCADE;
-- CREATE VIEW public.catalog_node_vw AS SELECT cn.tenant_datasource_id_uuid::text AS tenant_datasource_id, tpd.source_name, cn.id AS node_id, cn.node_name, cnt.catalog_type_name, COALESCE(cnt.config, jsonb_build_object('properties', cnt.properties)) AS catalog_defn, cn.node_type_id, cn.description, cn.qualified_path, cn.properties, cn.parent_id FROM ((catalog_node cn JOIN catalog_node_type cnt ON ((cnt.id)::text = (cn.node_type_id)::text)) JOIN tenant_product_datasource tpd ON ((tpd.id)::text = (cn.tenant_datasource_id_uuid)::text));

-- =========================
-- 6) DESTRUCTIVE CLEANUP (RUN DURING MAINTENANCE WINDOW)
-- Drop old constraints that reference legacy columns then drop legacy columns and rename canonical columns if required.
-- Use DROP CONSTRAINT IF EXISTS and DROP COLUMN IF EXISTS.

-- Example: drop any FK that references tenant_datasource_id_old
ALTER TABLE public.catalog_node_type DROP CONSTRAINT IF EXISTS catalog_node_type_tenant_product_datasource_fk;
ALTER TABLE public.tenant_chart DROP CONSTRAINT IF EXISTS tenant_chart_tenant_product_datasource_fk;
ALTER TABLE public.catalog_node DROP CONSTRAINT IF EXISTS catalog_node_tenant_product_datasource_old_fk; -- placeholder

-- Drop legacy columns (these acquire ACCESS EXCLUSIVE locks)
ALTER TABLE public.catalog_node_type DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.tenant_chart DROP COLUMN IF EXISTS tenant_datasource_id_old;
ALTER TABLE public.catalog_node DROP COLUMN IF EXISTS tenant_datasource_id_old;

-- Optionally rename canonical column into exact legacy name if you prefer (we already have tenant_datasource_id_uuid or tenant_datasource_id depending on earlier steps)
-- ALTER TABLE public.<table> RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id;

-- Cleanup compatibility column on tenant_product_datasource
ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS id_text;

-- =========================
-- 7) POST-CHECKS
SELECT table_name, COUNT(*) FILTER (WHERE tenant_datasource_id IS NULL) AS null_final_count FROM (VALUES ('catalog_node'),('catalog_node_type'),('tenant_chart')) AS t(table_name) LEFT JOIN LATERAL (SELECT * FROM public."" || t.table_name || "" LIMIT 0) x ON true;
-- Note: adjust the above if your psql client can't evaluate dynamic lateral. Alternatively run manual counts:
SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id IS NULL;
SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id IS NULL;
SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id IS NULL;

-- =========================
-- 8) ROLLBACK SNIPPETS (if anything goes wrong)
-- Undo a mapping to a newly-created tpd_id
-- BEGIN; UPDATE public.catalog_node_type SET tenant_datasource_id_uuid = NULL WHERE tenant_datasource_id_uuid = '<NEW_TPD_UUID>'::uuid; DELETE FROM public.tenant_product_datasource WHERE id = '<NEW_TPD_UUID>'::uuid; COMMIT;

-- Full restore: restore from backup snapshot (recommended if any destructive drop was accidental)

-- End of playbook

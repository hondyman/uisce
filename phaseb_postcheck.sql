-- phaseb_postcheck.sql
-- Post-cleanup verification

-- Show columns now
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_name IN ('catalog_node','catalog_node_type','tenant_chart')
  AND column_name IN ('tenant_datasource_id','tenant_datasource_id_old','tenant_datasource_id_uuid')
ORDER BY table_name, column_name;

-- Counts: how many NULLs in canonical tenant_datasource_id now
SELECT 'catalog_node' AS tbl, (SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id IS NULL) AS canonical_nulls;
SELECT 'catalog_node_type' AS tbl, (SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id IS NULL) AS canonical_nulls;
SELECT 'tenant_chart' AS tbl, (SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id IS NULL) AS canonical_nulls;

-- Show sample rows where canonical is the default tpd
SELECT t.id, t.tenant_datasource_id FROM public.catalog_node t WHERE t.tenant_datasource_id = (SELECT id FROM public.tenant_product_datasource WHERE source_name LIKE 'phaseb-default-%' LIMIT 1) LIMIT 5;
SELECT t.id, t.tenant_datasource_id FROM public.catalog_node_type t WHERE t.tenant_datasource_id = (SELECT id FROM public.tenant_product_datasource WHERE source_name LIKE 'phaseb-default-%' LIMIT 1) LIMIT 5;
SELECT t.id, t.tenant_datasource_id FROM public.tenant_chart t WHERE t.tenant_datasource_id = (SELECT id FROM public.tenant_product_datasource WHERE source_name LIKE 'phaseb-default-%' LIMIT 1) LIMIT 5;

-- Any constraints referencing tenant_datasource_id_uuid? (should be none if promoted)
SELECT conname, conrelid::regclass AS table, pg_get_constraintdef(oid) AS def FROM pg_constraint WHERE pg_get_constraintdef(oid) ILIKE '%tenant_datasource_id_uuid%';

-- Views referencing tenant_datasource_id in definition
SELECT viewname, definition FROM pg_views WHERE definition ILIKE '%tenant_datasource_id%';

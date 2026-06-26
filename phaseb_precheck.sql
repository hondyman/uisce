-- Pre-cleanup verification for Phase-B destructive cleanup
-- Safe, read-only checks. Does not modify data.

-- Show default TPD stored in temp table (if present in this session)
SELECT 'tmp_phaseb_defaults' AS src, * FROM pg_temp.tmp_phaseb_defaults LIMIT 10;

-- Show the default tenant_product_datasource (safe guard if tmp table absent)
SELECT 'default_tpd' AS src, t.* FROM public.tenant_product_datasource t
WHERE t.id = (SELECT tpd_id FROM pg_temp.tmp_phaseb_defaults LIMIT 1)
LIMIT 1;

-- Show columns related to tenant datasource on target tables
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_name IN ('catalog_node','catalog_node_type','tenant_chart')
  AND column_name IN ('tenant_datasource_id','tenant_datasource_id_old','tenant_datasource_id_uuid')
ORDER BY table_name, column_name;

-- Counts (guarded) for each target table
SELECT 'catalog_node' AS tbl,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='tenant_datasource_id_uuid') THEN (SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id_uuid IS NULL) ELSE NULL END) AS null_uuid_count,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='tenant_datasource_id') THEN (SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id IS NOT NULL) ELSE NULL END) AS legacy_count;

SELECT 'catalog_node_type' AS tbl,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node_type' AND column_name='tenant_datasource_id_uuid') THEN (SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id_uuid IS NULL) ELSE NULL END) AS null_uuid_count,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node_type' AND column_name='tenant_datasource_id') THEN (SELECT COUNT(*) FROM public.catalog_node_type WHERE tenant_datasource_id IS NOT NULL) ELSE NULL END) AS legacy_count;

SELECT 'tenant_chart' AS tbl,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='tenant_chart' AND column_name='tenant_datasource_id_uuid') THEN (SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id_uuid IS NULL) ELSE NULL END) AS null_uuid_count,
  (SELECT CASE WHEN EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='tenant_chart' AND column_name='tenant_datasource_id') THEN (SELECT COUNT(*) FROM public.tenant_chart WHERE tenant_datasource_id IS NOT NULL) ELSE NULL END) AS legacy_count;

-- Sample rows assigned to the default TPD (if available)
SELECT 'catalog_node_sample' AS src, * FROM public.catalog_node WHERE tenant_datasource_id_uuid = (SELECT tpd_id FROM pg_temp.tmp_phaseb_defaults LIMIT 1) LIMIT 5;
SELECT 'catalog_node_type_sample' AS src, * FROM public.catalog_node_type WHERE tenant_datasource_id_uuid = (SELECT tpd_id FROM pg_temp.tmp_phaseb_defaults LIMIT 1) LIMIT 5;
SELECT 'tenant_chart_sample' AS src, * FROM public.tenant_chart WHERE tenant_datasource_id_uuid = (SELECT tpd_id FROM pg_temp.tmp_phaseb_defaults LIMIT 1) LIMIT 5;

-- Any constraints referencing tenant_datasource_id (legacy name) or tenant_datasource_id_old
SELECT conname, conrelid::regclass AS table, pg_get_constraintdef(oid) AS def
FROM pg_constraint
WHERE pg_get_constraintdef(oid) ILIKE '%tenant_datasource_id%'
ORDER BY conname;

-- Views that reference legacy tenant_datasource_id in their definition
SELECT viewname, definition FROM pg_views WHERE definition ILIKE '%tenant_datasource_id%';

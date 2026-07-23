-- Remove orphaned database columns (columns not in "public" schema and columns with no relationships)
-- Run with: PGPASSWORD=postgres psql -h 100.84.50.65 -U postgres -d alpha

BEGIN;

-- 1. Find database columns not in "public" schema
-- These have qualified_path like 'schema_name.table_name.column_name' where schema != 'public'
SELECT cn.id, cn.node_name, cn.qualified_path, cnt.catalog_type_name
FROM catalog_node cn
LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name IN ('column', 'database_column')
  AND cn.qualified_path IS NOT NULL
  AND cn.qualified_path NOT LIKE 'public.%'
  AND cn.qualified_path NOT LIKE 'public/'
ORDER BY cn.qualified_path;

-- 2. Find orphaned columns (columns with no edges at all)
SELECT cn.id, cn.node_name, cn.qualified_path, cnt.catalog_type_name
FROM catalog_node cn
LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name IN ('column', 'database_column')
  AND cn.id NOT IN (
    SELECT DISTINCT source_node_id FROM catalog_edge WHERE source_node_id IS NOT NULL
    UNION
    SELECT DISTINCT target_node_id FROM catalog_edge WHERE target_node_id IS NOT NULL
  );

-- 3. Delete columns not in public schema
DELETE FROM catalog_node
WHERE id IN (
    SELECT cn.id
    FROM catalog_node cn
    LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
    WHERE cnt.catalog_type_name IN ('column', 'database_column')
      AND cn.qualified_path IS NOT NULL
      AND cn.qualified_path NOT LIKE 'public.%'
      AND cn.qualified_path NOT LIKE 'public/%'
);

-- 4. Delete orphaned columns (columns with no edges)
DELETE FROM catalog_node
WHERE id IN (
    SELECT cn.id
    FROM catalog_node cn
    LEFT JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
    WHERE cnt.catalog_type_name IN ('column', 'database_column')
      AND cn.id NOT IN (
        SELECT DISTINCT source_node_id FROM catalog_edge WHERE source_node_id IS NOT NULL
        UNION
        SELECT DISTINCT target_node_id FROM catalog_edge WHERE target_node_id IS NOT NULL
      )
);

COMMIT;

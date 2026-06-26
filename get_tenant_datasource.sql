-- Get available tenants and datasources for testing
SELECT 
  t.id as tenant_id,
  t.display_name as tenant_name,
  td.id as datasource_id,
  td.source_name as datasource_name
FROM alpha_tenant t
JOIN alpha_tenant_datasource td ON td.tenant_id = t.id
ORDER BY t.display_name, td.source_name
LIMIT 20;

-- If you need to see what schemas are available for a specific datasource:
-- (Replace the UUID with an actual datasource_id from above)
/*
SELECT 
  cn.id,
  cn.node_name as schema_name,
  cnt.catalog_type_name,
  cn.parent_id
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'schema'
  AND cn.tenant_datasource_id = 'YOUR-DATASOURCE-ID-HERE'
ORDER BY cn.node_name;
*/

-- To see tables for a schema:
/*
SELECT 
  cn.id,
  cn.node_name as table_name,
  cn.parent_id as schema_id
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'table'
  AND cn.parent_id = 'YOUR-SCHEMA-ID-HERE'
ORDER BY cn.node_name
LIMIT 20;
*/

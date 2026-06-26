-- Fix missing driver_table_id for 'customers' BO
UPDATE business_objects
SET driver_table_id = (SELECT id FROM catalog_node WHERE node_name = 'customers' AND node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192' LIMIT 1)
WHERE name = 'customers' AND driver_table_id IS NULL;

-- Fix missing semantic_term_id in bo_fields by matching field name to term name
UPDATE bo_fields bf
SET semantic_term_id = st.id
FROM catalog_node st
INNER JOIN catalog_node_type cnt ON st.node_type_id = cnt.id
WHERE (bf.semantic_term_id IS NULL OR bf.semantic_term_id::text = '')
  AND st.node_name = bf.field_name
  AND cnt.catalog_type_name IN ('semantic_term', 'business_term');

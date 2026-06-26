-- Verification queries for semantic term SQL property backfill

-- 1. Check semantic terms with and without SQL property
SELECT 
    CASE 
        WHEN properties::jsonb ->> 'sql' IS NOT NULL THEN 'Has SQL property'
        ELSE 'Missing SQL property'
    END as property_status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM catalog_node
WHERE node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
GROUP BY property_status
ORDER BY property_status;

-- 2. Show sample semantic terms with their SQL properties
SELECT 
    st.node_name as semantic_term,
    st.properties::jsonb ->> 'sql' as sql_property,
    st.properties::jsonb ->> 'data_type' as data_type,
    COUNT(e.id) as mapped_columns
FROM catalog_node st
LEFT JOIN catalog_edge e ON e.target_node_id = st.id AND e.edge_type_id = '99c86836-98ef-45a3-82df-4c62b5730ac6'
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
GROUP BY st.id, st.node_name, st.properties
ORDER BY st.node_name
LIMIT 20;

-- 3. Check for semantic terms that have mappings but no SQL property
SELECT 
    st.node_name as semantic_term,
    st.properties,
    COUNT(e.id) as mapped_column_count,
    string_agg(c.node_name, ', ') as mapped_columns
FROM catalog_node st
INNER JOIN catalog_edge e ON e.target_node_id = st.id AND e.edge_type_id = '99c86836-98ef-45a3-82df-4c62b5730ac6'
INNER JOIN catalog_node c ON c.id = e.source_node_id
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
    AND (st.properties IS NULL OR st.properties::jsonb ->> 'sql' IS NULL)
GROUP BY st.id, st.node_name, st.properties
ORDER BY mapped_column_count DESC
LIMIT 10;

-- 4. Show semantic term to column mappings with SQL property
SELECT 
    st.node_name as semantic_term,
    st.properties::jsonb ->> 'sql' as sql_property,
    c.node_name as column_name,
    t.node_name as table_name
FROM catalog_node st
INNER JOIN catalog_edge e ON e.target_node_id = st.id AND e.edge_type_id = '99c86836-98ef-45a3-82df-4c62b5730ac6'
INNER JOIN catalog_node c ON c.id = e.source_node_id
LEFT JOIN catalog_node t ON c.parent_id = t.id
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
    AND st.properties::jsonb ->> 'sql' IS NOT NULL
ORDER BY st.node_name, c.node_name
LIMIT 30;

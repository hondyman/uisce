BEGIN;
-- This script backfills the sql property for semantic terms that don't have it
-- The SQL property is generated in the format {CUBE}.<column_name> for use with Cube.js

-- Update semantic terms that have column mappings but missing sql property
UPDATE catalog_node st
SET 
    properties = jsonb_set(
        COALESCE(st.properties::jsonb, '{}'::jsonb),
        '{sql}',
        to_jsonb('{CUBE}.' || column_mappings.column_name),
        true
    ),
    updated_at = NOW()
FROM (
    -- Find the first column that maps to each semantic term
    SELECT DISTINCT ON (st.id)
        st.id as semantic_term_id,
        c.node_name as column_name
    FROM catalog_node st
    INNER JOIN catalog_edge e ON e.target_node_id = st.id AND e.edge_type_id = '99c86836-98ef-45a3-82df-4c62b5730ac6'
    INNER JOIN catalog_node c ON c.id = e.source_node_id
    WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
        AND (
            st.properties IS NULL 
            OR st.properties = '{}'
            OR st.properties::jsonb ->> 'sql' IS NULL
        )
    ORDER BY st.id, e.created_at
) AS column_mappings
WHERE st.id = column_mappings.semantic_term_id;

-- Also ensure data_type is set to 'Dimension' if it's missing
UPDATE catalog_node st
SET 
    properties = jsonb_set(
        COALESCE(st.properties::jsonb, '{}'::jsonb),
        '{data_type}',
        to_jsonb('Dimension'),
        true
    ),
    updated_at = NOW()
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
    AND (
        st.properties IS NULL 
        OR st.properties = '{}'
        OR st.properties::jsonb ->> 'data_type' IS NULL
    );

-- Report on the changes
SELECT 
    'Semantic Terms with SQL property' as status,
    COUNT(*) as count
FROM catalog_node st
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
    AND st.properties::jsonb ->> 'sql' IS NOT NULL
UNION ALL
SELECT 
    'Semantic Terms missing SQL property' as status,
    COUNT(*) as count
FROM catalog_node st
WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098'
    AND (st.properties IS NULL OR st.properties::jsonb ->> 'sql' IS NULL);

COMMIT;

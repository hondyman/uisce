-- Function to retrieve calculation DAG with metadata
CREATE OR REPLACE FUNCTION get_calc_dag_with_metadata(start_path text, tenant uuid)
RETURNS jsonb AS $$
WITH RECURSIVE dag AS (
    -- Start from the calculation node
    SELECT
        n.id AS node_id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        n.lineage,
        n.data_quality_contract,
        n.sla,
        e.target_node_id,
        e.relationship_type
    FROM catalog_node n
    LEFT JOIN catalog_edge e ON n.id = e.source_node_id
    WHERE n.qualified_path = start_path
      AND n.tenant_id = tenant

    UNION ALL

    -- Walk dependencies recursively
    SELECT
        n.id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        n.lineage,
        n.data_quality_contract,
        n.sla,
        e.target_node_id,
        e.relationship_type
    FROM dag d
    JOIN catalog_edge e ON d.target_node_id = e.source_node_id
    JOIN catalog_node n ON e.source_node_id = n.id
    WHERE n.tenant_id = tenant
)
SELECT jsonb_build_object(
    'nodes', jsonb_agg(
        DISTINCT jsonb_build_object(
            'id', node_id,
            'name', node_name,
            'path', qualified_path,
            'type', node_type_id,
            'lineage', lineage,
            'data_quality_contract', data_quality_contract,
            'sla', sla
        )
    ),
    'edges', jsonb_agg(
        DISTINCT jsonb_build_object(
            'source', node_id,
            'target', target_node_id,
            'relationship', relationship_type
        )
    )
)
FROM dag;
$$ LANGUAGE sql;

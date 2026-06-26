
-- Migration: Fix Member Of Edges
-- Description: Updates edges between Semantic Terms and Business Objects to use 'member_of'.

DO $$
DECLARE
    member_of_id UUID;
    semantic_term_type_id UUID;
    business_object_type_id UUID;
BEGIN
    -- 1. Get Edge Type IDs
    SELECT id INTO member_of_id FROM catalog_edge_types WHERE edge_type_name = 'member_of' LIMIT 1;

    -- 2. Get Node Type IDs
    SELECT id INTO semantic_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO business_object_type_id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;

    RAISE NOTICE 'Fixing edges to member_of (ID=%)', member_of_id;

    IF member_of_id IS NOT NULL THEN
        -- A. Delete potential duplicates
        DELETE FROM catalog_edge e
        USING catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND (
               (src.node_type_id = semantic_term_type_id AND tgt.node_type_id = business_object_type_id)
            OR (tgt.node_type_id = semantic_term_type_id AND src.node_type_id = business_object_type_id)
          )
          AND e.edge_type_id IS DISTINCT FROM member_of_id
          AND EXISTS (
              SELECT 1 FROM catalog_edge existing
              WHERE existing.source_node_id = e.source_node_id
                AND existing.target_node_id = e.target_node_id
                AND existing.tenant_datasource_id IS NOT DISTINCT FROM e.tenant_datasource_id
                AND existing.edge_type_id = member_of_id
          );

        -- B. Update edges
        UPDATE catalog_edge e
        SET edge_type_id = member_of_id,
            relationship_type = 'member_of',
            edge_type_name = 'member_of'
        FROM catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND e.edge_type_id IS DISTINCT FROM member_of_id
          AND (
               (src.node_type_id = semantic_term_type_id AND tgt.node_type_id = business_object_type_id)
            OR (tgt.node_type_id = semantic_term_type_id AND src.node_type_id = business_object_type_id)
          );
          
        RAISE NOTICE 'Updated edges to member_of.';
    ELSE
        RAISE WARNING 'member_of edge type not found! Skipping update.';
    END IF;

END $$;

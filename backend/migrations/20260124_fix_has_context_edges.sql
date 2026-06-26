
-- Migration: Fix Has Context Edges
-- Description: Updates edges between Semantic Terms and Database Columns to use 'has_context' instead of 'maps_to'.
-- Handles duplicate conflicts by deleting redundant edges first.

DO $$
DECLARE
    has_context_id UUID;
    semantic_term_type_id UUID;
    db_column_type_id UUID;
    column_type_id UUID; -- Legacy/Alternative
BEGIN
    -- 1. Get Edge Type IDs
    SELECT id INTO has_context_id FROM catalog_edge_types WHERE edge_type_name = 'has_context' LIMIT 1;

    -- 2. Get Node Type IDs for classification
    SELECT id INTO semantic_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO db_column_type_id FROM catalog_node_type WHERE catalog_type_name = 'database_column' LIMIT 1;
    SELECT id INTO column_type_id FROM catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1;

    RAISE NOTICE 'Fixing edges to has_context (ID=%)', has_context_id;

    IF has_context_id IS NOT NULL THEN
        
        -- A. Delete edges that would cause a conflict because a has_context edge ALREADY exists
        DELETE FROM catalog_edge e
        USING catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND (
               (src.node_type_id = semantic_term_type_id AND (tgt.node_type_id = db_column_type_id OR tgt.node_type_id = column_type_id))
            OR (tgt.node_type_id = semantic_term_type_id AND (src.node_type_id = db_column_type_id OR src.node_type_id = column_type_id))
          )
          AND e.edge_type_id IS DISTINCT FROM has_context_id
          AND EXISTS (
              SELECT 1 FROM catalog_edge existing
              WHERE existing.source_node_id = e.source_node_id
                AND existing.target_node_id = e.target_node_id
                AND existing.tenant_datasource_id IS NOT DISTINCT FROM e.tenant_datasource_id
                AND existing.edge_type_id = has_context_id
          );
          
        RAISE NOTICE 'Deleted redundant edges that already had a has_context counterpart.';

        -- B. Update the remaining matching edges to has_context
        UPDATE catalog_edge e
        SET edge_type_id = has_context_id,
            relationship_type = 'has_context',
            edge_type_name = 'has_context'
        FROM catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND e.edge_type_id IS DISTINCT FROM has_context_id -- Prevent no-op updates
          AND (
               (src.node_type_id = semantic_term_type_id AND (tgt.node_type_id = db_column_type_id OR tgt.node_type_id = column_type_id))
            OR (tgt.node_type_id = semantic_term_type_id AND (src.node_type_id = db_column_type_id OR src.node_type_id = column_type_id))
          );
          
        RAISE NOTICE 'Updated remaining edges to has_context.';
        
    ELSE
        RAISE WARNING 'has_context edge type not found! Skipping update.';
    END IF;

END $$;

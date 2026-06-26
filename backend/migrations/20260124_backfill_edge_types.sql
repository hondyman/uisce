
-- Migration: Backfill Edge Types
-- Description: Updates existing edges with NULL edge_type_id to use the correct seeded types (maps_to, related_to) based on node types.

DO $$
DECLARE
    maps_to_id UUID;
    related_to_id UUID;
    semantic_term_type_id UUID;
    business_term_type_id UUID;
    db_column_type_id UUID;
    column_type_id UUID; -- Legacy/Alternative
BEGIN
    -- 1. Get Edge Type IDs
    SELECT id INTO maps_to_id FROM catalog_edge_types WHERE edge_type_name = 'maps_to' LIMIT 1;
    SELECT id INTO related_to_id FROM catalog_edge_types WHERE edge_type_name = 'related_to' LIMIT 1;

    -- 2. Get Node Type IDs for classification
    SELECT id INTO semantic_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO business_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'business_term' LIMIT 1;
    SELECT id INTO db_column_type_id FROM catalog_node_type WHERE catalog_type_name = 'database_column' LIMIT 1;
    SELECT id INTO column_type_id FROM catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1;

    RAISE NOTICE 'Backfilling edges. MapsTo=% RelatedTo=%', maps_to_id, related_to_id;

    IF maps_to_id IS NOT NULL THEN
        -- Update 'maps_to' edges: (Semantic Term <-> Database Column)
        UPDATE catalog_edge e
        SET edge_type_id = maps_to_id,
            relationship_type = 'maps_to'
        FROM catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND e.edge_type_id IS NULL
          AND (
               (src.node_type_id = semantic_term_type_id AND (tgt.node_type_id = db_column_type_id OR tgt.node_type_id = column_type_id))
            OR (tgt.node_type_id = semantic_term_type_id AND (src.node_type_id = db_column_type_id OR src.node_type_id = column_type_id))
          );
          
        RAISE NOTICE 'Updated maps_to edges.';
    END IF;

    IF related_to_id IS NOT NULL THEN
        -- Update 'related_to' edges: (Semantic Term <-> Semantic Term) OR (Business Term <-> Semantic Term)
        UPDATE catalog_edge e
        SET edge_type_id = related_to_id,
           relationship_type = 'related_to'
        FROM catalog_node src, catalog_node tgt
        WHERE e.source_node_id = src.id 
          AND e.target_node_id = tgt.id
          AND e.edge_type_id IS NULL
          AND (
               (src.node_type_id IN (semantic_term_type_id, business_term_type_id) AND tgt.node_type_id IN (semantic_term_type_id, business_term_type_id))
          );
          
        RAISE NOTICE 'Updated related_to edges.';
    END IF;
    
    -- Fallback: Update any remaining NULL semantic/business edges to related_to if we couldn't match types exactly but they look like terms
    IF related_to_id IS NOT NULL THEN
         UPDATE catalog_edge e
         SET edge_type_id = related_to_id,
             relationship_type = 'related_to'
         WHERE e.edge_type_id IS NULL
           AND e.relationship_type IS NULL; -- Catch-all for orphans, assuming they are semantic-ish if they exist in this context
           
         RAISE NOTICE 'Updated remaining orphan edges to related_to default.';
    END IF;

END $$;

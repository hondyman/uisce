-- Rollback: Remove member_of edges created by sync

DO $$
DECLARE
    v_bo_node_type_id UUID;
    v_member_of_edge_type_id UUID;
BEGIN
    -- Get the business_object node type ID
    SELECT id INTO v_bo_node_type_id
    FROM catalog_node_type
    WHERE catalog_type_name = 'business_object'
    LIMIT 1;

    -- Get the member_of edge type ID
    SELECT id INTO v_member_of_edge_type_id
    FROM catalog_edge_types
    WHERE edge_type_name = 'member_of'
    LIMIT 1;

    -- Delete member_of edges from BOs
    IF v_bo_node_type_id IS NOT NULL AND v_member_of_edge_type_id IS NOT NULL THEN
        DELETE FROM catalog_edge ce
        USING catalog_node cn
        WHERE ce.target_node_id = cn.id
          AND cn.node_type_id = v_bo_node_type_id
          AND ce.edge_type_id = v_member_of_edge_type_id;

        RAISE NOTICE 'Deleted member_of edges from BOs';
    END IF;
END $$;

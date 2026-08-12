-- Sync all existing Business Objects to the catalog
-- This ensures:
-- 1. All BOs have catalog_node entries with the correct node type
-- 2. BO-to-semantic-term edges are created from business_object_fields

DO $$
DECLARE
    v_bo_node_type_id UUID;
    v_member_of_edge_type_id UUID;
    v_tenant_id UUID;
    v_bo_record RECORD;
    v_edges_created INT := 0;
    v_nodes_created INT := 0;
BEGIN
    -- Get the business_object node type ID
    SELECT id INTO v_bo_node_type_id
    FROM catalog_node_type
    WHERE catalog_type_name = 'business_object'
    LIMIT 1;

    IF v_bo_node_type_id IS NULL THEN
        RAISE NOTICE 'business_object catalog_node_type not found. Run migration 20260812000001_add_business_object_catalog_node_type.up.sql first';
        RETURN;
    END IF;

    -- Get the member_of edge type ID
    SELECT id INTO v_member_of_edge_type_id
    FROM catalog_edge_types
    WHERE edge_type_name = 'member_of'
    LIMIT 1;

    IF v_member_of_edge_type_id IS NULL THEN
        RAISE NOTICE 'member_of edge type not found. Create it first';
        RETURN;
    END IF;

    -- Get the gold_copy tenant ID (most commonly used)
    SELECT id INTO v_tenant_id
    FROM tenants
    WHERE gold_copy = true
    LIMIT 1;

    IF v_tenant_id IS NULL THEN
        -- Fall back to any tenant
        SELECT id INTO v_tenant_id FROM tenants LIMIT 1;
    END IF;

    RAISE NOTICE 'Starting BO catalog sync with bo_node_type_id=%, edge_type_id=%, tenant_id=%',
        v_bo_node_type_id, v_member_of_edge_type_id, v_tenant_id;

    -- ================================================================
    -- PART 1: Ensure all BOs have catalog_node entries
    -- ================================================================
    FOR v_bo_record IN
        SELECT bo.id, bo.key, bo.name, bo.display_name, bo.driver_table_id, bo.driver_table_name, bo.tenant_id, bo.datasource_id
        FROM business_objects bo
    LOOP
        -- Check if BO already has a catalog_node entry
        IF NOT EXISTS (
            SELECT 1 FROM catalog_node
            WHERE id = v_bo_record.id
        ) THEN
            -- Insert catalog_node for this BO
            INSERT INTO catalog_node (
                id, node_name, node_type_id, tenant_id, tenant_datasource_id,
                properties, description, created_at, updated_at
            ) VALUES (
                v_bo_record.id,
                v_bo_record.key,
                v_bo_node_type_id,
                COALESCE(v_bo_record.tenant_id, v_tenant_id),
                v_bo_record.datasource_id,
                jsonb_build_object(
                    'bo_key', v_bo_record.key,
                    'display_name', COALESCE(v_bo_record.display_name, v_bo_record.name),
                    'driver_table_id', v_bo_record.driver_table_id,
                    'driver_table_name', v_bo_record.driver_table_name
                ),
                v_bo_record.name,
                NOW(),
                NOW()
            );
            v_nodes_created := v_nodes_created + 1;
            RAISE NOTICE 'Created catalog_node for BO: % (%)', v_bo_record.key, v_bo_record.id;
        ELSE
            -- Update existing entry to ensure properties are correct
            UPDATE catalog_node SET
                node_name = v_bo_record.key,
                node_type_id = v_bo_node_type_id,
                properties = jsonb_build_object(
                    'bo_key', v_bo_record.key,
                    'display_name', COALESCE(v_bo_record.display_name, v_bo_record.name),
                    'driver_table_id', v_bo_record.driver_table_id,
                    'driver_table_name', v_bo_record.driver_table_name
                ),
                description = v_bo_record.name,
                updated_at = NOW()
            WHERE id = v_bo_record.id;
        END IF;
    END LOOP;

    RAISE NOTICE 'Part 1 complete: % catalog_node entries created/updated', v_nodes_created;

    -- ================================================================
    -- PART 2: Create BO-to-semantic-term edges from business_object_fields
    -- ================================================================

    -- Delete any existing member_of edges from BOs (to avoid duplicates)
    DELETE FROM catalog_edge ce
    USING catalog_node cn
    WHERE ce.target_node_id = cn.id
      AND cn.node_type_id = v_bo_node_type_id
      AND ce.edge_type_id = v_member_of_edge_type_id;

    RAISE NOTICE 'Cleared existing member_of edges from BOs';

    -- Create new edges based on business_object_fields
    FOR v_bo_record IN
        SELECT DISTINCT bo.id as bo_id, bo.key as bo_key, bof.semantic_term_node_id, bo.tenant_id, bo.datasource_id
        FROM business_objects bo
        JOIN business_object_fields bof ON bo.id = bof.bo_id
        WHERE bof.semantic_term_node_id IS NOT NULL
    LOOP
        -- Check edge doesn't already exist
        IF NOT EXISTS (
            SELECT 1 FROM catalog_edge
            WHERE source_node_id = v_bo_record.semantic_term_node_id
              AND target_node_id = v_bo_record.bo_id
              AND edge_type_id = v_member_of_edge_type_id
        ) THEN
            INSERT INTO catalog_edge (
                id, source_node_id, target_node_id, edge_type_id, relationship_type,
                tenant_id, tenant_datasource_id, properties,
                created_at, updated_at, is_active
            ) VALUES (
                gen_random_uuid(),
                v_bo_record.semantic_term_node_id,
                v_bo_record.bo_id,
                v_member_of_edge_type_id,
                'member_of',
                COALESCE(v_bo_record.tenant_id, v_tenant_id),
                v_bo_record.datasource_id,
                '{}',
                NOW(),
                NOW(),
                true
            );
            v_edges_created := v_edges_created + 1;
        END IF;
    END LOOP;

    RAISE NOTICE 'Part 2 complete: % member_of edges created', v_edges_created;

    -- ================================================================
    -- SUMMARY
    -- ================================================================
    RAISE NOTICE '============================================';
    RAISE NOTICE 'BO Catalog Sync Complete';
    RAISE NOTICE '  Nodes created/updated: %', v_nodes_created;
    RAISE NOTICE '  Edges created: %', v_edges_created;
    RAISE NOTICE '============================================';

END $$;

-- Verify the sync
SELECT
    cn.node_name as bo_key,
    cn.properties->>'display_name' as bo_display_name,
    cnt.catalog_type_name as node_type,
    (SELECT COUNT(*) FROM catalog_edge ce WHERE ce.target_node_id = cn.id AND ce.edge_type_id IN (SELECT id FROM catalog_edge_types WHERE edge_type_name = 'member_of')) as member_of_edge_count
FROM catalog_node cn
JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'business_object'
ORDER BY cn.node_name;

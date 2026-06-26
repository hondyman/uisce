
-- Migration: Seed Semantic Layer for dim_account
-- Description: Auto-generates Semantic Terms and mappings for the dim_account table to enable BO Wizard testing.

DO $$
DECLARE
    dim_account_id UUID;
    column_rec RECORD;
    term_type_id UUID;
    column_type_id UUID;
    maps_to_type_id UUID;
    new_term_id UUID;
    tenant_id UUID;
    datasource_id UUID;
BEGIN
    -- 1. Get Node/Edge Type IDs
    SELECT id INTO term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO column_type_id FROM catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1;
    SELECT id INTO maps_to_type_id FROM catalog_edge_types WHERE edge_type_name = 'maps_to' LIMIT 1;

    -- 2. Validate IDs found
    IF term_type_id IS NULL OR column_type_id IS NULL OR maps_to_type_id IS NULL THEN
        RAISE NOTICE 'Missing required node/edge types (term=% column=% maps_to=%). Skipping seed.' , term_type_id, column_type_id, maps_to_type_id;
        RETURN;
    END IF;

    -- 3. Get dim_account Table Node
    SELECT cn.id, cn.tenant_id, cn.tenant_datasource_id INTO dim_account_id, tenant_id, datasource_id
    FROM catalog_node cn
    WHERE cn.node_name = 'dim_account' 
      AND cn.node_type_id IN (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'table')
    LIMIT 1;

    IF dim_account_id IS NULL THEN
        RAISE NOTICE 'dim_account table not found. Skipping seeding.';
        RETURN;
    END IF;

    RAISE NOTICE 'Seeding terms for table: dim_account (%)', dim_account_id;

    -- 4. Iterate over columns that do NOT have mappings yet
    FOR column_rec IN 
        SELECT c.id, c.node_name, c.node_type_id
        FROM catalog_node c
        WHERE c.parent_id = dim_account_id
          AND c.node_type_id = column_type_id
          AND NOT EXISTS (
              SELECT 1 FROM catalog_edge ce 
              WHERE ce.source_node_id = c.id 
                AND ce.edge_type_id = maps_to_type_id
          )
    LOOP
        -- Create new Semantic Term ID
        new_term_id := gen_random_uuid();

        -- Insert Semantic Term (mirroring column name for now)
        INSERT INTO catalog_node (
            id, node_name, node_type_id, tenant_id, tenant_datasource_id, 
            properties, qualified_path, created_at, updated_at
        ) VALUES (
            new_term_id, 
            column_rec.node_name, -- e.g. "acct_cd"
            term_type_id,
            tenant_id,
            datasource_id,
            jsonb_build_object(
                'title', INITCAP(REPLACE(column_rec.node_name, '_', ' ')), -- "Acct Cd"
                'data_type', 'string', -- Default to string, logic could be smarter
                'auto_generated', true
            ),
            column_rec.node_name, -- Use node_name as qualified_path
            NOW(), NOW()
        );

        -- Insert Mapping Edge (Column -> maps_to -> Term)
        INSERT INTO catalog_edge (
            id, source_node_id, target_node_id, edge_type_id, edge_type_name,
            relationship_type, tenant_id, tenant_datasource_id, created_at, updated_at
        ) VALUES (
            gen_random_uuid(),
            column_rec.id, -- Source: Column
            new_term_id,   -- Target: Term
            maps_to_type_id,
            'maps_to',
            'maps_to',
            tenant_id,
            datasource_id,
            NOW(), NOW()
        );

    END LOOP;

    RAISE NOTICE 'Seeding complete for dim_account.';
END $$;

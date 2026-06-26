-- Seed core edge types maps_to and related_to
DO $$
DECLARE
    tenant_id_val UUID;
    semantic_term_id UUID;
    db_column_id UUID;
BEGIN
    -- Get node type IDs
    SELECT id INTO semantic_term_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO db_column_id FROM catalog_node_type WHERE catalog_type_name = 'database_column' LIMIT 1; 
    -- If database_column not found, try 'column'
    IF db_column_id IS NULL THEN
        SELECT id INTO db_column_id FROM catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1;
    END IF;

    RAISE NOTICE 'Debug: Tenant=% Term=% DBCol=%', tenant_id_val, semantic_term_id, db_column_id;

    IF tenant_id_val IS NOT NULL AND semantic_term_id IS NOT NULL THEN
        -- Seed maps_to (Semantic Term -> Database Column)
        IF db_column_id IS NOT NULL THEN
            INSERT INTO public.catalog_edge_type (id, tenant_id, edge_type_name, description, is_active, properties, source_node_type_id, target_node_type_id)
            SELECT 
                gen_random_uuid(), 
                tenant_id_val, 
                'maps_to', 
                'Indicates that a semantic term maps to a physical database column.', 
                true,
                '{"type": "core"}'::jsonb,
                semantic_term_id,
                db_column_id
            WHERE NOT EXISTS (
                SELECT 1 FROM public.catalog_edge_type 
                WHERE edge_type_name = 'maps_to' AND tenant_id = tenant_id_val
            );
        END IF;

        -- Seed related_to (Semantic Term <-> Semantic Term)
        INSERT INTO public.catalog_edge_type (id, tenant_id, edge_type_name, description, is_active, properties, source_node_type_id, target_node_type_id)
        SELECT 
            gen_random_uuid(), 
            tenant_id_val, 
            'related_to', 
            'Indicates a generic semantic relationship between two terms.', 
            true,
            '{"type": "core"}'::jsonb,
            semantic_term_id,
            semantic_term_id
        WHERE NOT EXISTS (
            SELECT 1 FROM public.catalog_edge_type 
            WHERE edge_type_name = 'related_to' AND tenant_id = tenant_id_val
        );
        
        RAISE NOTICE 'Seeded core edge types for tenant %', tenant_id_val;
    ELSE
        RAISE NOTICE 'Skipping seed: Missing tenant or semantic_term type definition';
    END IF;
END $$;

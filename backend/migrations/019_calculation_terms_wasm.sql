-- Migration 019: Calculation Terms & WASM Execution Foundation
-- Standardizes node types and edge types for the Semantic Execution Fabric.

DO $$
DECLARE
    v_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999'; -- uisce tenant
    v_calc_type_id UUID;
    v_term_type_id UUID;
    v_dep_term_edge_type_id UUID;
    v_dep_calc_edge_type_id UUID;
    v_nav_node_id UUID := '7f3c2c4e-0c8e-4c1a-9b8e-3b8c9e6b4f72'; -- Reusing the ID we found or using a fresh one
    v_pos_val_node_id UUID := gen_random_uuid();
BEGIN
    -- 1. Ensure calculation_term node type exists (lowercase standard)
    INSERT INTO public.catalog_node_type (catalog_type_name, description, tenant_id)
    VALUES ('calculation_term', 'Term defined by an executable expression', v_tenant_id)
    ON CONFLICT (tenant_id, catalog_type_name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO v_calc_type_id;

    -- Get semantic_term type ID for dependency edges
    SELECT id INTO v_term_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'semantic_term' AND tenant_id = v_tenant_id LIMIT 1;

    -- 2. Ensure calculation dependency edge types exist
    INSERT INTO public.catalog_edge_type (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
    VALUES ('calc_depends_on_term', 'Calculation term depends on a semantic term', v_calc_type_id, v_term_type_id, v_tenant_id)
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO v_dep_term_edge_type_id;

    INSERT INTO public.catalog_edge_type (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
    VALUES ('calc_depends_on_calc', 'Calculation term depends on another calculation term', v_calc_type_id, v_calc_type_id, v_tenant_id)
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET description = EXCLUDED.description
    RETURNING id INTO v_dep_calc_edge_type_id;

    -- 3. Seed Example: Net Asset Value
    -- First, create the dependency: PositionValue (semantic term)
    INSERT INTO public.catalog_node (node_type_id, node_name, description, qualified_path, tenant_id)
    VALUES (v_term_type_id, 'PositionValue', 'Market value of a single position', 'semantic/PositionValue', v_tenant_id)
    ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO NOTHING
    RETURNING id INTO v_pos_val_node_id;

    -- Create NAV calculation term
    INSERT INTO public.catalog_node (
        node_type_id, 
        node_name, 
        description, 
        properties, 
        qualified_path, 
        tenant_id
    ) VALUES (
        v_calc_type_id,
        'NetAssetValue',
        'Market value of assets minus liabilities',
        jsonb_build_object(
            'expression', 'NAV = sum(PositionValue)',
            'engine', 'wasm',
            'data_type', 'currency',
            'version', '1.0'
        ),
        'calculation/NetAssetValue',
        v_tenant_id
    )
    ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) 
    DO UPDATE SET 
        properties = EXCLUDED.properties,
        description = EXCLUDED.description
    RETURNING id INTO v_nav_node_id;

    -- 4. Create dependency edge: NAV -> PositionValue
    INSERT INTO public.catalog_edge (
        source_node_id,
        target_node_id,
        edge_type_name, -- Note: the table uses edge_type_name as FK? Let's check \d catalog_edge
        tenant_id
    ) VALUES (
        v_nav_node_id,
        v_pos_val_node_id,
        'calc_depends_on_term',
        v_tenant_id
    ) ON CONFLICT DO NOTHING;

    RAISE NOTICE '✓ Calculation Term Fabric initialized for tenant %', v_tenant_id;
END $$;

-- 20260222_001_portfolio_security_spine_intelligence.sql
-- Enriches the semantic spine between Portfolio and Security domains.

DO $$
DECLARE
    v_tenant_id        UUID;
    v_edge_type_id     UUID;
    v_portfolio_bo_id  UUID;
    v_security_term_id UUID;
BEGIN
    -- 1. Resolve default tenant
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- 2. Update holds_security with property schema in config
    UPDATE public.catalog_edge_types
    SET config = jsonb_build_object(
        'properties', jsonb_build_object(
            'weight', jsonb_build_object('type', 'number'),
            'effective_date', jsonb_build_object('type', 'date'),
            'end_date', jsonb_build_object('type', 'date')
        )
    )
    WHERE edge_type_name = 'holds_security' AND tenant_id = v_tenant_id
    RETURNING id INTO v_edge_type_id;

    -- 3. Find Portfolio BO and SecurityID Term
    -- Portfolio BO
    SELECT id INTO v_portfolio_bo_id 
    FROM public.business_objects 
    WHERE key = 'portfolio' AND tenant_id = v_tenant_id 
    LIMIT 1;

    -- SecurityID Term (from catalog_node)
    SELECT id INTO v_security_term_id
    FROM public.catalog_node
    WHERE node_name = 'SecurityID' AND tenant_id = v_tenant_id::text
    LIMIT 1;

    -- 4. Create specialized links
    IF v_portfolio_bo_id IS NOT NULL AND v_security_term_id IS NOT NULL AND v_edge_type_id IS NOT NULL THEN
        -- Portfolio -> SecurityID Link
        INSERT INTO public.catalog_edge (
            id, source_node_id, target_node_id, edge_type_id, edge_type_name, relationship_type, tenant_id, created_at
        ) VALUES (
            gen_random_uuid(),
            v_portfolio_bo_id,
            v_security_term_id,
            v_edge_type_id,
            'holds_security',
            'holds_security',
            v_tenant_id,
            NOW()
        ) ON CONFLICT DO NOTHING;

        -- Equities specific example (using properties field of catalog_edge)
        INSERT INTO public.catalog_edge (
            id, source_node_id, target_node_id, edge_type_id, edge_type_name, relationship_type, properties, tenant_id, created_at
        ) VALUES (
            gen_random_uuid(),
            v_portfolio_bo_id,
            v_security_term_id,
            v_edge_type_id,
            'holds_security',
            'holds_security',
            '{"asset_class": "Equity"}'::jsonb,
            v_tenant_id,
            NOW()
        ) ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✓ Phase 8: Enhanced holds_security with property schemas and specialized links';
END$$;

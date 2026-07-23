-- backend/migrations/020_portfolio_security_graph.sql
-- Phase 5: Portfolio-Security "Semantic Spine"
-- Creates the 'holds_security' edge type and initial bindings.

DO $$
DECLARE
    v_tenant_id        UUID;
    v_edge_type_id     UUID := gen_random_uuid();
    v_portfolio_bo_id  UUID;
    v_security_bo_id   UUID := '11110001-0000-4000-a000-000000000001';
BEGIN
    -- Resolve default tenant
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- 1. Create the 'holds_security' edge type
    INSERT INTO public.catalog_edge_types (
        id, edge_type_name, display_name, description, is_active, tenant_id
    ) VALUES (
        v_edge_type_id,
        'holds_security',
        'Holds Security',
        'Relationship between a Portfolio position and its master Security record',
        true,
        v_tenant_id
    ) ON CONFLICT (edge_type_name, tenant_id) DO UPDATE 
        SET description = EXCLUDED.description;

    -- 2. Bind Portfolio BO to Security BO in the semantic graph
    SELECT id INTO v_portfolio_bo_id FROM public.business_objects WHERE key = 'portfolio' AND tenant_id = v_tenant_id LIMIT 1;

    IF v_portfolio_bo_id IS NOT NULL THEN
        INSERT INTO public.catalog_edge (
            id, source_node_id, target_node_id, edge_type_id, edge_type_name, relationship_type, tenant_id, created_at
        ) VALUES (
            gen_random_uuid(),
            v_portfolio_bo_id,
            v_security_bo_id,
            v_edge_type_id,
            'holds_security',
            'holds_security',
            v_tenant_id,
            NOW()
        ) ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✓ Phase 5: holds_security edge type and BO binding created';
END$$;

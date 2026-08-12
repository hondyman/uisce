-- Add business_object catalog_node_type for BO → Catalog sync
-- This enables the catalog-worker to properly categorize BO nodes
-- that are synced from the business_objects table

DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    -- Get the default tenant UUID or use the zero UUID
    SELECT COALESCE(NULLIF(tenant_id::text, ''), '00000000-0000-0000-0000-000000000000')::UUID
    INTO v_tenant_id
    FROM tenants LIMIT 1;

    -- If no tenant found, use the zero UUID
    IF v_tenant_id IS NULL THEN
        v_tenant_id := '00000000-0000-0000-0000-000000000000'::UUID;
    END IF;

    -- Insert business_object node type if not exists
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, config)
    VALUES (
        gen_random_uuid(),
        v_tenant_id,
        'business_object',
        'Business Object - represents a business entity with semantic mappings and field definitions',
        '{"description": "Represents a business object with fields, subtypes, semantic term bindings, and lineage"}'
    )
    ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING;

    RAISE NOTICE 'business_object catalog_node_type seeded for tenant %', v_tenant_id;
END $$;

-- Also ensure it exists for the zero UUID tenant (for backwards compatibility)
INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, config)
VALUES (
    gen_random_uuid(),
    '00000000-0000-0000-0000-000000000000'::UUID,
    'business_object',
    'Business Object - represents a business entity with semantic mappings and field definitions',
    '{"description": "Represents a business object with fields, subtypes, semantic term bindings, and lineage"}'
)
ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING;

-- Verify the insert
SELECT catalog_type_name, description, tenant_id FROM public.catalog_node_type WHERE catalog_type_name = 'business_object';

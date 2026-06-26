
-- Migration: Seed Member Of Edge Type
-- Description: Adds 'member_of' edge type for Semantic Term -> Business Object relationships.
-- Uses a valid tenant ID if available and skips if prerequisites are missing.

DO $do$
DECLARE
    semantic_term_type_id UUID;
    business_object_type_id UUID;
    new_edge_type_id UUID;
    _tenant UUID;
BEGIN
    -- 1. Get Node Type IDs
    SELECT id INTO semantic_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO business_object_type_id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;

    -- 2. Get a valid Tenant ID
    SELECT id INTO _tenant FROM tenants LIMIT 1;

    IF _tenant IS NULL THEN
        RAISE NOTICE 'No tenant found; skipping seeding of member_of edge type.';
        RETURN;
    END IF;

    -- 3. Only seed if node types exist
    IF semantic_term_type_id IS NOT NULL AND business_object_type_id IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM catalog_edge_types WHERE edge_type_name = 'member_of' AND tenant_id = _tenant) THEN
            INSERT INTO catalog_edge_types (
                edge_type_name,
                description,
                source_node_type_id,
                target_node_type_id,
                is_directed,
                is_active,
                tenant_id
            ) VALUES (
                'member_of',
                'Semantic Term is a member of Business Object',
                semantic_term_type_id,
                business_object_type_id,
                true,
                true,
                _tenant
            ) RETURNING id INTO new_edge_type_id;

            RAISE NOTICE 'Seeded member_of edge type: %', new_edge_type_id;
        ELSE
            RAISE NOTICE 'member_of edge type already exists for tenant %', _tenant;
        END IF;
    ELSE
        RAISE NOTICE 'Required node types missing; skipping member_of seed (semantic_term_id=% business_object_id=%)', semantic_term_type_id, business_object_type_id;
    END IF;
END
$do$;

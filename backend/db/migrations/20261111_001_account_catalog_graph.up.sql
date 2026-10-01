-- Wire Account into the catalog graph so traversal and TERM_MAPS_TO_CUSTOM_FIELD
-- resolution work. Uses live conventions from attribute.Service:
--   custom_field path = attr:{tenant_id}:ACCOUNT:{field_cd}
--   semantic terms     = semantic/account/{field_cd} (already seeded)
--   TERM_MAPS_TO_CUSTOM_FIELD: term → custom_field
--   BO_HAS_ATTRIBUTE: business_object/account → custom_field
-- Idempotent. Run against alpha.

DO $graph$
DECLARE
    _tenant uuid;
    _bo_type uuid;
    _cf_type uuid;
    _term_type uuid;
    _term_edge uuid;
    _bo_attr_edge uuid;
    _maps_to uuid;
    _bo_node uuid;
    _bo_id uuid;
    _rec record;
    _cf_id uuid;
    _term_id uuid;
    _path text;
    _cf_n int := 0;
    _edge_n int := 0;
    _bo_edge_n int := 0;
BEGIN
    SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        RAISE NOTICE 'account catalog graph: no gold_copy tenant — skip';
        RETURN;
    END IF;

    -- RLS helpers used by catalog_edge policies (search_path may prefer vend.*)
    PERFORM set_config('app.current_tenant', _tenant::text, true);
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);

    SELECT id INTO _bo_type FROM public.catalog_node_types
      WHERE catalog_type_name = 'business_object' AND is_active LIMIT 1;
    SELECT id INTO _cf_type FROM public.catalog_node_types
      WHERE catalog_type_name = 'custom_field' AND is_active LIMIT 1;
    SELECT id INTO _term_type FROM public.catalog_node_types
      WHERE catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM') LIMIT 1;
    SELECT id INTO _term_edge FROM public.catalog_edge_types
      WHERE edge_type_name = 'TERM_MAPS_TO_CUSTOM_FIELD' LIMIT 1;
    SELECT id INTO _bo_attr_edge FROM public.catalog_edge_types
      WHERE edge_type_name = 'BO_HAS_ATTRIBUTE' LIMIT 1;
    SELECT id INTO _maps_to FROM public.catalog_edge_types
      WHERE edge_type_name = 'MAPS_TO' LIMIT 1;

    IF _bo_type IS NULL OR _cf_type IS NULL OR _term_type IS NULL THEN
        RAISE NOTICE 'account catalog graph: missing node types — skip';
        RETURN;
    END IF;

    SELECT id INTO _bo_id FROM public.business_objects
      WHERE tenant_id = _tenant AND bo_key = 'account' LIMIT 1;

    -- 1. business_object catalog node
    INSERT INTO public.catalog_node (
        id, tenant_id, node_type_id, node_name, description,
        qualified_path, is_active, node_type, properties, created_at, updated_at
    ) VALUES (
        gen_random_uuid(),
        _tenant,
        _bo_type,
        'Account',
        'Account master business object (mdm.account_master + orm.account driver)',
        'business_object/account',
        true,
        'business_object',
        jsonb_build_object(
            'bo_key', 'account',
            'bo_id', _bo_id,
            'table_ref', 'mdm.account_master',
            'entity_type', 'ACCOUNT',
            'source', 'account_catalog_graph'
        ),
        now(), now()
    )
    ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
        node_name = EXCLUDED.node_name,
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        is_active = true,
        updated_at = now()
    RETURNING id INTO _bo_node;

    IF _bo_node IS NULL THEN
        SELECT id INTO _bo_node FROM public.catalog_node
         WHERE tenant_id = _tenant AND qualified_path = 'business_object/account';
    END IF;

    -- 2. One custom_field node per ACCOUNT field_cd on the gold tenant
    --    (CORE defs use platform zero-UUID tenant which is not in public.tenants).
    FOR _rec IN
        SELECT DISTINCT ON (a.field_cd)
               a.id AS def_id, a.tenant_id AS def_tenant_id, a.field_cd, a.name, a.description,
               a.json_path, a.table_ref, a.semantic_term_id, a.is_active
        FROM public.attribute_def a
        WHERE a.entity_type = 'ACCOUNT'
          AND a.is_active
        ORDER BY a.field_cd,
                 CASE WHEN a.tenant_id = _tenant THEN 0
                      WHEN a.tenant_id = '00000000-0000-0000-0000-000000000000'::uuid THEN 1
                      ELSE 2 END
    LOOP
        _path := 'attr:' || _tenant::text || ':ACCOUNT:' || _rec.field_cd;

        INSERT INTO public.catalog_node (
            id, tenant_id, node_type_id, node_name, description,
            qualified_path, is_active, node_type, properties, created_at, updated_at
        ) VALUES (
            gen_random_uuid(),
            _tenant,
            _cf_type,
            _rec.name,
            COALESCE(NULLIF(_rec.description, ''), 'Account custom field: ' || _rec.field_cd),
            _path,
            true,
            'custom_field',
            jsonb_build_object(
                'field_cd', _rec.field_cd,
                'json_path', _rec.json_path,
                'table_ref', _rec.table_ref,
                'entity_type', 'ACCOUNT',
                'attribute_def_id', _rec.def_id,
                'def_tenant_id', _rec.def_tenant_id,
                'storage_kind', 'JSONB_KEY',
                'jsonb_column', 'custom_attributes',
                'source', 'account_catalog_graph'
            ),
            now(), now()
        )
        ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
            node_name = EXCLUDED.node_name,
            description = EXCLUDED.description,
            properties = EXCLUDED.properties,
            is_active = true,
            updated_at = now()
        RETURNING id INTO _cf_id;

        IF _cf_id IS NULL THEN
            SELECT id INTO _cf_id FROM public.catalog_node
             WHERE tenant_id = _tenant AND qualified_path = _path;
        END IF;
        _cf_n := _cf_n + 1;

        _term_id := _rec.semantic_term_id;
        IF _term_id IS NULL THEN
            SELECT id INTO _term_id FROM public.catalog_node
             WHERE node_type_id = _term_type
               AND qualified_path = 'semantic/account/' || _rec.field_cd
             LIMIT 1;
        END IF;

        IF _term_id IS NOT NULL AND _term_edge IS NOT NULL AND _cf_id IS NOT NULL THEN
            INSERT INTO public.catalog_edge (
                id, tenant_id, source_node_id, target_node_id, edge_type_id,
                properties, is_active, relationship_type, created_at, updated_at
            )
            SELECT gen_random_uuid(), _tenant, _term_id, _cf_id, _term_edge,
                   jsonb_build_object(
                       'storage_kind', 'JSONB_KEY',
                       'json_path', _rec.json_path,
                       'jsonb_column', 'custom_attributes',
                       'field_cd', _rec.field_cd,
                       'table_ref', _rec.table_ref,
                       'entity_type', 'ACCOUNT'
                   ),
                   true, 'TERM_MAPS_TO_CUSTOM_FIELD', now(), now()
            WHERE NOT EXISTS (
                SELECT 1 FROM public.catalog_edge e
                WHERE e.tenant_id = _tenant
                  AND e.source_node_id = _term_id
                  AND e.target_node_id = _cf_id
                  AND e.edge_type_id = _term_edge
            );
            GET DIAGNOSTICS _edge_n = ROW_COUNT;
        END IF;

        IF _bo_attr_edge IS NOT NULL AND _bo_node IS NOT NULL AND _cf_id IS NOT NULL THEN
            INSERT INTO public.catalog_edge (
                id, tenant_id, source_node_id, target_node_id, edge_type_id,
                properties, is_active, relationship_type, created_at, updated_at
            )
            SELECT gen_random_uuid(), _tenant, _bo_node, _cf_id, _bo_attr_edge,
                   jsonb_build_object('field_cd', _rec.field_cd, 'entity_type', 'ACCOUNT'),
                   true, 'BO_HAS_ATTRIBUTE', now(), now()
            WHERE NOT EXISTS (
                SELECT 1 FROM public.catalog_edge e
                WHERE e.tenant_id = _tenant
                  AND e.source_node_id = _bo_node
                  AND e.target_node_id = _cf_id
                  AND e.edge_type_id = _bo_attr_edge
            );
            GET DIAGNOSTICS _bo_edge_n = ROW_COUNT;
        END IF;
    END LOOP;

    RAISE NOTICE 'account catalog graph: custom_fields=% term_edges≈% bo_attr_edges≈% bo_node=%',
        _cf_n, _edge_n, _bo_edge_n, _bo_node;
END
$graph$;

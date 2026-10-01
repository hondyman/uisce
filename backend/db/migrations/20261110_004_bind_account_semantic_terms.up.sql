-- Bind every active ACCOUNT attribute_def to a semantic term at
-- semantic/account/{field_cd}, and seed SOURCE_PRIORITY survivorship rules
-- for the demo tenant so mastering can resolve custom attributes.

DO $bind$
DECLARE
    _gold uuid;
    _demo uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _term_type_id uuid;
    _priority text[] := ARRAY['GOLDENSOURCE', 'MARKET_EDM', 'ASSET_CONTROL', 'INTERNAL'];
    _rec record;
    _term_id uuid;
    _path text;
    _bound int := 0;
    _rules int := 0;
BEGIN
    SELECT id INTO _gold FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _gold IS NULL THEN
        _gold := _demo;
    END IF;

    SELECT id INTO _term_type_id FROM public.catalog_node_types
      WHERE catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM') LIMIT 1;
    IF _term_type_id IS NULL THEN
        RAISE NOTICE 'bind account semantic terms: no semantic_term node type — skip';
        RETURN;
    END IF;

    FOR _rec IN
        SELECT DISTINCT ON (field_cd)
               id, tenant_id, field_cd, name, description, semantic_term_id
        FROM public.attribute_def
        WHERE entity_type = 'ACCOUNT'
          AND is_active
        ORDER BY field_cd,
                 CASE WHEN tenant_id = '00000000-0000-0000-0000-000000000000'::uuid THEN 0 ELSE 1 END,
                 created_at
    LOOP
        _path := 'semantic/account/' || _rec.field_cd;

        SELECT id INTO _term_id
        FROM public.catalog_node
        WHERE tenant_id = _gold
          AND node_type_id = _term_type_id
          AND qualified_path = _path
        LIMIT 1;

        IF _term_id IS NULL THEN
            INSERT INTO public.catalog_node (
                id, tenant_id, node_type_id, node_name, description,
                qualified_path, is_active, properties
            ) VALUES (
                gen_random_uuid(),
                _gold,
                _term_type_id,
                _rec.field_cd,
                COALESCE(NULLIF(_rec.description, ''), 'Account field: ' || _rec.field_cd),
                _path,
                true,
                jsonb_build_object(
                    'entity_type', 'ACCOUNT',
                    'field_cd', _rec.field_cd,
                    'source', 'account_semantic_bind'
                )
            )
            RETURNING id INTO _term_id;
        END IF;

        -- Bind every active ACCOUNT def row with this field_cd that is still unbound.
        UPDATE public.attribute_def
           SET semantic_term_id = _term_id,
               updated_at = now()
         WHERE entity_type = 'ACCOUNT'
           AND field_cd = _rec.field_cd
           AND is_active
           AND semantic_term_id IS NULL;
        GET DIAGNOSTICS _bound = ROW_COUNT;

        INSERT INTO public.semantic_survivorship_rules (
            tenant_id, entity_type, semantic_term_id, strategy,
            priority_order, max_stale_seconds, is_active
        ) VALUES (
            _demo, 'ACCOUNT', _term_id, 'SOURCE_PRIORITY', _priority, 0, true
        )
        ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE
            SET strategy = EXCLUDED.strategy,
                priority_order = EXCLUDED.priority_order,
                is_active = true,
                updated_at = now();
        _rules := _rules + 1;
    END LOOP;

    RAISE NOTICE 'account semantic bind: terms/rules processed=%', _rules;
END
$bind$;

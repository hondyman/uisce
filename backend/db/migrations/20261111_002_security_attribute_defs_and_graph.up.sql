-- SECURITY dual-registry starter: attribute_def + semantic terms + survivorship
-- + catalog graph (same pattern as Account). Run against alpha.

-- ── attribute_def seed (CORE platform tenant) ──────────────────────────────
INSERT INTO public.attribute_def (
    tenant_id, entity_type, table_ref, field_cd, name, description,
    data_type, json_path, is_required, is_searchable, is_pii,
    validation_rules, display_order, section, is_active
) VALUES
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'esg_score', 'ESG Score', 'Composite ESG score', 'decimal', 'esg_score', false, true, false, '{"min":0,"max":100}'::jsonb, 10, 'ESG', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'sfdr_article', 'SFDR Article', 'SFDR article classification', 'enum', 'sfdr_article', false, true, false, '{}'::jsonb, 20, 'ESG', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'mifid_target_market', 'MiFID Target Market', 'MiFID II target market', 'string', 'mifid_target_market', false, true, false, '{}'::jsonb, 30, 'Regulatory', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'lei', 'Issuer LEI', 'Legal Entity Identifier of issuer', 'string', 'lei', false, true, false, '{}'::jsonb, 40, 'Identity', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'cic_code', 'CIC Code', 'Solvency II CIC code', 'string', 'cic_code', false, true, false, '{}'::jsonb, 50, 'Regulatory', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'primary_exchange_mic', 'Primary Exchange MIC', 'ISO MIC of primary listing', 'string', 'primary_exchange_mic', false, true, false, '{}'::jsonb, 60, 'Listing', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'lot_size', 'Lot Size', 'Trading lot size', 'integer', 'lot_size', false, true, false, '{"min":0}'::jsonb, 70, 'Trading', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'tick_size', 'Tick Size', 'Minimum price increment', 'decimal', 'tick_size', false, true, false, '{"min":0}'::jsonb, 80, 'Trading', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'trading_status', 'Trading Status', 'Exchange trading status', 'enum', 'trading_status', false, true, false, '{}'::jsonb, 90, 'Trading', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'gics_sector', 'GICS Sector', 'GICS sector code', 'string', 'gics_sector', false, true, false, '{}'::jsonb, 100, 'Classification', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'gics_industry', 'GICS Industry', 'GICS industry code', 'string', 'gics_industry', false, true, false, '{}'::jsonb, 110, 'Classification', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'coupon_type', 'Coupon Type', 'Fixed/floating/zero coupon type', 'enum', 'coupon_type', false, true, false, '{}'::jsonb, 120, 'Fixed Income', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'day_count', 'Day Count', 'Day-count convention', 'string', 'day_count', false, true, false, '{}'::jsonb, 130, 'Fixed Income', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'payment_frequency', 'Payment Frequency', 'Coupon payment frequency', 'string', 'payment_frequency', false, true, false, '{}'::jsonb, 140, 'Fixed Income', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'callable_flag', 'Callable', 'Issuer call feature', 'boolean', 'callable_flag', false, true, false, '{}'::jsonb, 150, 'Fixed Income', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'puttable_flag', 'Puttable', 'Holder put feature', 'boolean', 'puttable_flag', false, true, false, '{}'::jsonb, 160, 'Fixed Income', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'underlying_security_id', 'Underlying Security', 'Underlying instrument id', 'string', 'underlying_security_id', false, true, false, '{}'::jsonb, 170, 'Derivative', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'option_style', 'Option Style', 'American/European/Bermudan', 'enum', 'option_style', false, true, false, '{}'::jsonb, 180, 'Derivative', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'strike_price', 'Strike Price', 'Option strike', 'decimal', 'strike_price', false, true, false, '{}'::jsonb, 190, 'Derivative', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'contract_size', 'Contract Size', 'Derivative contract size', 'decimal', 'contract_size', false, true, false, '{}'::jsonb, 200, 'Derivative', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'multiplier', 'Multiplier', 'Price multiplier', 'decimal', 'multiplier', false, true, false, '{}'::jsonb, 210, 'Derivative', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'figi_share_class', 'FIGI Share Class', 'Share-class level FIGI', 'string', 'figi_share_class', false, true, false, '{}'::jsonb, 220, 'Identity', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'bloomberg_unique_id', 'Bloomberg Unique ID', 'BB unique instrument id', 'string', 'bloomberg_unique_id', false, true, false, '{}'::jsonb, 230, 'Identity', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'SECURITY', 'mdm.security_master', 'country_of_incorporation', 'Country of Incorporation', 'ISO country of incorporation', 'string', 'country_of_incorporation', false, true, false, '{}'::jsonb, 240, 'Issuer', true)
ON CONFLICT (tenant_id, entity_type, field_cd) DO NOTHING;

-- Update picklists for enums (best-effort)
UPDATE public.attribute_def SET picklist_values = '["ARTICLE_6","ARTICLE_8","ARTICLE_9"]'::jsonb
 WHERE entity_type='SECURITY' AND field_cd='sfdr_article' AND picklist_values IS NULL;
UPDATE public.attribute_def SET picklist_values = '["ACTIVE","HALTED","SUSPENDED","DELISTED"]'::jsonb
 WHERE entity_type='SECURITY' AND field_cd='trading_status' AND picklist_values IS NULL;
UPDATE public.attribute_def SET picklist_values = '["FIXED","FLOATING","ZERO","STEP"]'::jsonb
 WHERE entity_type='SECURITY' AND field_cd='coupon_type' AND picklist_values IS NULL;
UPDATE public.attribute_def SET picklist_values = '["AMERICAN","EUROPEAN","BERMUDAN"]'::jsonb
 WHERE entity_type='SECURITY' AND field_cd='option_style' AND picklist_values IS NULL;

-- ── terms + survivorship + catalog graph ───────────────────────────
DO $sec$
DECLARE
    _tenant uuid;
    _demo uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _term_type uuid;
    _cf_type uuid;
    _bo_type uuid;
    _term_edge uuid;
    _bo_attr_edge uuid;
    _bo_node uuid;
    _bo_id uuid;
    _priority text[] := ARRAY['BLOOMBERG', 'REFINITIV', 'GOLDENSOURCE', 'INTERNAL'];
    _rec record;
    _term_id uuid;
    _cf_id uuid;
    _path text;
    _typed text[] := ARRAY[
        'security_id','primary_identifier','isin','cusip','sedol','figi','ticker',
        'security_name','asset_class','currency','status'
    ];
    _f text;
BEGIN
    SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        RAISE NOTICE 'security seed: no gold tenant';
        RETURN;
    END IF;
    PERFORM set_config('app.current_tenant', _tenant::text, true);
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);

    SELECT id INTO _term_type FROM public.catalog_node_types WHERE catalog_type_name IN ('semantic_term','SEMANTIC_TERM') LIMIT 1;
    SELECT id INTO _cf_type FROM public.catalog_node_types WHERE catalog_type_name='custom_field' AND is_active LIMIT 1;
    SELECT id INTO _bo_type FROM public.catalog_node_types WHERE catalog_type_name='business_object' AND is_active LIMIT 1;
    SELECT id INTO _term_edge FROM public.catalog_edge_types WHERE edge_type_name='TERM_MAPS_TO_CUSTOM_FIELD' LIMIT 1;
    SELECT id INTO _bo_attr_edge FROM public.catalog_edge_types WHERE edge_type_name='BO_HAS_ATTRIBUTE' LIMIT 1;
    SELECT id INTO _bo_id FROM public.business_objects WHERE tenant_id=_tenant AND bo_key='security' LIMIT 1;

    -- typed-column semantic terms (for survivorship on physical columns)
    FOREACH _f IN ARRAY _typed LOOP
        _path := 'semantic/security/' || _f;
        SELECT id INTO _term_id FROM public.catalog_node
         WHERE tenant_id=_tenant AND node_type_id=_term_type AND qualified_path=_path LIMIT 1;
        IF _term_id IS NULL AND _term_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (
                id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
            ) VALUES (
                gen_random_uuid(), _tenant, _term_type, _f, 'Security master field: '||_f, _path, true,
                jsonb_build_object('entity_type','SECURITY','field_cd',_f,'source','security_seed')
            ) RETURNING id INTO _term_id;
        END IF;
        IF _term_id IS NOT NULL THEN
            INSERT INTO public.semantic_survivorship_rules (
                tenant_id, entity_type, semantic_term_id, strategy, priority_order, max_stale_seconds, is_active
            ) VALUES (_demo, 'SECURITY', _term_id, 'SOURCE_PRIORITY', _priority, 0, true)
            ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE
              SET priority_order=EXCLUDED.priority_order, is_active=true, updated_at=now();
        END IF;
    END LOOP;

    -- business_object node
    IF _bo_type IS NOT NULL THEN
        INSERT INTO public.catalog_node (
            id, tenant_id, node_type_id, node_name, description, qualified_path,
            is_active, node_type, properties, created_at, updated_at
        ) VALUES (
            gen_random_uuid(), _tenant, _bo_type, 'Security',
            'Security master (mdm.security_master)', 'business_object/security',
            true, 'business_object',
            jsonb_build_object('bo_key','security','bo_id',_bo_id,'table_ref','mdm.security_master','entity_type','SECURITY','source','security_catalog_graph'),
            now(), now()
        )
        ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
            properties=EXCLUDED.properties, is_active=true, updated_at=now()
        RETURNING id INTO _bo_node;
        IF _bo_node IS NULL THEN
            SELECT id INTO _bo_node FROM public.catalog_node
             WHERE tenant_id=_tenant AND qualified_path='business_object/security';
        END IF;
    END IF;

    -- attribute_def → terms + custom_field nodes + edges
    FOR _rec IN
        SELECT DISTINCT ON (field_cd)
               id AS def_id, field_cd, name, description, json_path, table_ref, semantic_term_id
        FROM public.attribute_def
        WHERE entity_type='SECURITY' AND is_active
        ORDER BY field_cd
    LOOP
        _path := 'semantic/security/' || _rec.field_cd;
        _term_id := _rec.semantic_term_id;
        IF _term_id IS NULL THEN
            SELECT id INTO _term_id FROM public.catalog_node
             WHERE tenant_id=_tenant AND node_type_id=_term_type AND qualified_path=_path LIMIT 1;
        END IF;
        IF _term_id IS NULL AND _term_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (
                id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
            ) VALUES (
                gen_random_uuid(), _tenant, _term_type, _rec.field_cd,
                COALESCE(NULLIF(_rec.description,''), 'Security field: '||_rec.field_cd),
                _path, true,
                jsonb_build_object('entity_type','SECURITY','field_cd',_rec.field_cd,'source','security_seed')
            ) RETURNING id INTO _term_id;
        END IF;

        UPDATE public.attribute_def
           SET semantic_term_id=_term_id, updated_at=now()
         WHERE entity_type='SECURITY' AND field_cd=_rec.field_cd AND semantic_term_id IS NULL;

        INSERT INTO public.semantic_survivorship_rules (
            tenant_id, entity_type, semantic_term_id, strategy, priority_order, max_stale_seconds, is_active
        ) VALUES (_demo, 'SECURITY', _term_id, 'SOURCE_PRIORITY', _priority, 0, true)
        ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE
          SET priority_order=EXCLUDED.priority_order, is_active=true, updated_at=now();

        IF _cf_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (
                id, tenant_id, node_type_id, node_name, description, qualified_path,
                is_active, node_type, properties, created_at, updated_at
            ) VALUES (
                gen_random_uuid(), _tenant, _cf_type, _rec.name,
                COALESCE(NULLIF(_rec.description,''), 'Security custom field: '||_rec.field_cd),
                'attr:'||_tenant::text||':SECURITY:'||_rec.field_cd,
                true, 'custom_field',
                jsonb_build_object(
                    'field_cd',_rec.field_cd,'json_path',_rec.json_path,'table_ref',_rec.table_ref,
                    'entity_type','SECURITY','attribute_def_id',_rec.def_id,
                    'storage_kind','JSONB_KEY','jsonb_column','custom_attributes','source','security_catalog_graph'
                ),
                now(), now()
            )
            ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
                properties=EXCLUDED.properties, is_active=true, updated_at=now()
            RETURNING id INTO _cf_id;
            IF _cf_id IS NULL THEN
                SELECT id INTO _cf_id FROM public.catalog_node
                 WHERE tenant_id=_tenant AND qualified_path='attr:'||_tenant::text||':SECURITY:'||_rec.field_cd;
            END IF;

            IF _term_edge IS NOT NULL AND _term_id IS NOT NULL AND _cf_id IS NOT NULL THEN
                INSERT INTO public.catalog_edge (
                    id, tenant_id, source_node_id, target_node_id, edge_type_id,
                    properties, is_active, relationship_type, created_at, updated_at
                )
                SELECT gen_random_uuid(), _tenant, _term_id, _cf_id, _term_edge,
                       jsonb_build_object('storage_kind','JSONB_KEY','json_path',_rec.json_path,
                           'jsonb_column','custom_attributes','field_cd',_rec.field_cd,'entity_type','SECURITY'),
                       true, 'TERM_MAPS_TO_CUSTOM_FIELD', now(), now()
                WHERE NOT EXISTS (
                    SELECT 1 FROM public.catalog_edge e
                    WHERE e.tenant_id=_tenant AND e.source_node_id=_term_id
                      AND e.target_node_id=_cf_id AND e.edge_type_id=_term_edge
                );
            END IF;

            IF _bo_attr_edge IS NOT NULL AND _bo_node IS NOT NULL AND _cf_id IS NOT NULL THEN
                INSERT INTO public.catalog_edge (
                    id, tenant_id, source_node_id, target_node_id, edge_type_id,
                    properties, is_active, relationship_type, created_at, updated_at
                )
                SELECT gen_random_uuid(), _tenant, _bo_node, _cf_id, _bo_attr_edge,
                       jsonb_build_object('field_cd',_rec.field_cd,'entity_type','SECURITY'),
                       true, 'BO_HAS_ATTRIBUTE', now(), now()
                WHERE NOT EXISTS (
                    SELECT 1 FROM public.catalog_edge e
                    WHERE e.tenant_id=_tenant AND e.source_node_id=_bo_node
                      AND e.target_node_id=_cf_id AND e.edge_type_id=_bo_attr_edge
                );
            END IF;
        END IF;
    END LOOP;
END
$sec$;

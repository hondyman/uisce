-- Party BO #3: attribute_def + semantic terms + survivorship + catalog graph + pipelines.
-- mdm.party already exists with custom_attributes; this completes the Account/Security template.

INSERT INTO public.attribute_def (
    tenant_id, entity_type, table_ref, field_cd, name, description,
    data_type, json_path, is_required, is_searchable, is_pii,
    validation_rules, display_order, section, is_active
) VALUES
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'short_name', 'Short Name', 'Display short name', 'string', 'short_name', false, true, false, '{}'::jsonb, 10, 'Identity', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'email', 'Email', 'Primary email', 'string', 'email', false, true, true, '{}'::jsonb, 20, 'Contact', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'phone', 'Phone', 'Primary phone', 'string', 'phone', false, true, true, '{}'::jsonb, 30, 'Contact', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'website', 'Website', 'Corporate website', 'string', 'website', false, true, false, '{}'::jsonb, 40, 'Contact', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'kyc_status', 'KYC Status', 'Know-your-customer status', 'enum', 'kyc_status', false, true, false, '{}'::jsonb, 50, 'Compliance', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'aml_risk_rating', 'AML Risk Rating', 'AML risk tier', 'enum', 'aml_risk_rating', false, true, false, '{}'::jsonb, 60, 'Compliance', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'fatca_status', 'FATCA Status', 'FATCA classification', 'string', 'fatca_status', false, true, false, '{}'::jsonb, 70, 'Tax', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'crs_status', 'CRS Status', 'CRS classification', 'string', 'crs_status', false, true, false, '{}'::jsonb, 80, 'Tax', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'industry_naics', 'NAICS Industry', 'NAICS industry code', 'string', 'industry_naics', false, true, false, '{}'::jsonb, 90, 'Classification', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'bloomberg_company_id', 'Bloomberg Company ID', 'BB company identifier', 'string', 'bloomberg_company_id', false, true, false, '{}'::jsonb, 100, 'Identity', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'swift_bic', 'SWIFT BIC', 'BIC for banking counterparties', 'string', 'swift_bic', false, true, false, '{}'::jsonb, 110, 'Banking', true),
    ('00000000-0000-0000-0000-000000000000'::uuid, 'PARTY', 'mdm.party', 'custodian_code', 'Custodian Code', 'Internal custodian code', 'string', 'custodian_code', false, true, false, '{}'::jsonb, 120, 'Banking', true)
ON CONFLICT (tenant_id, entity_type, field_cd) DO NOTHING;

UPDATE public.attribute_def SET picklist_values='["PENDING","CLEARED","EXPIRED","REJECTED"]'::jsonb
 WHERE entity_type='PARTY' AND field_cd='kyc_status' AND (picklist_values IS NULL OR picklist_values='{}'::jsonb);
UPDATE public.attribute_def SET picklist_values='["LOW","MEDIUM","HIGH","PROHIBITED"]'::jsonb
 WHERE entity_type='PARTY' AND field_cd='aml_risk_rating' AND (picklist_values IS NULL OR picklist_values='{}'::jsonb);

DO $party$
DECLARE
    _tenant uuid;
    _demo uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _term_type uuid; _cf_type uuid; _bo_type uuid; _term_edge uuid; _bo_attr_edge uuid;
    _bo_node uuid; _bo_id uuid; _term_id uuid; _cf_id uuid; _path text;
    _priority text[] := ARRAY['GOLDENSOURCE','BLOOMBERG','REFINITIV','INTERNAL'];
    _rec record;
    _typed text[] := ARRAY['party_cd','legal_name','party_type','segment','tax_id','domicile','status','lei'];
    _f text;
    _file_id uuid := 'a11c0001-0001-4000-8000-000000000021'::uuid;
    _master_id uuid := 'a11c0001-0001-4000-8000-000000000020'::uuid;
    _fields jsonb;
BEGIN
    SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN RETURN; END IF;
    PERFORM set_config('app.current_tenant', _tenant::text, true);
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);

    SELECT id INTO _term_type FROM public.catalog_node_types WHERE catalog_type_name IN ('semantic_term','SEMANTIC_TERM') LIMIT 1;
    SELECT id INTO _cf_type FROM public.catalog_node_types WHERE catalog_type_name='custom_field' AND is_active LIMIT 1;
    SELECT id INTO _bo_type FROM public.catalog_node_types WHERE catalog_type_name='business_object' AND is_active LIMIT 1;
    SELECT id INTO _term_edge FROM public.catalog_edge_types WHERE edge_type_name='TERM_MAPS_TO_CUSTOM_FIELD' LIMIT 1;
    SELECT id INTO _bo_attr_edge FROM public.catalog_edge_types WHERE edge_type_name='BO_HAS_ATTRIBUTE' LIMIT 1;
    SELECT id INTO _bo_id FROM public.business_objects WHERE tenant_id=_tenant AND bo_key='party' LIMIT 1;

    FOREACH _f IN ARRAY _typed LOOP
        _path := 'semantic/party/' || _f;
        SELECT id INTO _term_id FROM public.catalog_node WHERE tenant_id=_tenant AND node_type_id=_term_type AND qualified_path=_path LIMIT 1;
        IF _term_id IS NULL AND _term_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties)
            VALUES (gen_random_uuid(), _tenant, _term_type, _f, 'Party field: '||_f, _path, true,
                    jsonb_build_object('entity_type','PARTY','field_cd',_f,'source','party_seed'))
            RETURNING id INTO _term_id;
        END IF;
        IF _term_id IS NOT NULL THEN
            INSERT INTO public.semantic_survivorship_rules (tenant_id, entity_type, semantic_term_id, strategy, priority_order, is_active)
            VALUES (_demo, 'PARTY', _term_id, 'SOURCE_PRIORITY', _priority, true)
            ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE SET priority_order=EXCLUDED.priority_order, is_active=true, updated_at=now();
        END IF;
    END LOOP;

    IF _bo_type IS NOT NULL THEN
        INSERT INTO public.catalog_node (id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, node_type, properties, created_at, updated_at)
        VALUES (gen_random_uuid(), _tenant, _bo_type, 'Party', 'Party master (mdm.party)', 'business_object/party', true, 'business_object',
                jsonb_build_object('bo_key','party','bo_id',_bo_id,'table_ref','mdm.party','entity_type','PARTY','source','party_catalog_graph'), now(), now())
        ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET properties=EXCLUDED.properties, is_active=true, updated_at=now()
        RETURNING id INTO _bo_node;
        IF _bo_node IS NULL THEN
            SELECT id INTO _bo_node FROM public.catalog_node WHERE tenant_id=_tenant AND qualified_path='business_object/party';
        END IF;
    END IF;

    FOR _rec IN
        SELECT DISTINCT ON (field_cd) id AS def_id, field_cd, name, description, json_path, table_ref, semantic_term_id
        FROM public.attribute_def WHERE entity_type='PARTY' AND is_active ORDER BY field_cd
    LOOP
        _path := 'semantic/party/' || _rec.field_cd;
        _term_id := _rec.semantic_term_id;
        IF _term_id IS NULL THEN
            SELECT id INTO _term_id FROM public.catalog_node WHERE tenant_id=_tenant AND node_type_id=_term_type AND qualified_path=_path LIMIT 1;
        END IF;
        IF _term_id IS NULL AND _term_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties)
            VALUES (gen_random_uuid(), _tenant, _term_type, _rec.field_cd, COALESCE(NULLIF(_rec.description,''),'Party field: '||_rec.field_cd),
                    _path, true, jsonb_build_object('entity_type','PARTY','field_cd',_rec.field_cd,'source','party_seed'))
            RETURNING id INTO _term_id;
        END IF;
        UPDATE public.attribute_def SET semantic_term_id=_term_id, updated_at=now()
         WHERE entity_type='PARTY' AND field_cd=_rec.field_cd AND semantic_term_id IS NULL;
        INSERT INTO public.semantic_survivorship_rules (tenant_id, entity_type, semantic_term_id, strategy, priority_order, is_active)
        VALUES (_demo, 'PARTY', _term_id, 'SOURCE_PRIORITY', _priority, true)
        ON CONFLICT (tenant_id, entity_type, semantic_term_id) DO UPDATE SET priority_order=EXCLUDED.priority_order, is_active=true, updated_at=now();

        IF _cf_type IS NOT NULL THEN
            INSERT INTO public.catalog_node (id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, node_type, properties, created_at, updated_at)
            VALUES (gen_random_uuid(), _tenant, _cf_type, _rec.name, COALESCE(NULLIF(_rec.description,''),'Party custom: '||_rec.field_cd),
                    'attr:'||_tenant::text||':PARTY:'||_rec.field_cd, true, 'custom_field',
                    jsonb_build_object('field_cd',_rec.field_cd,'json_path',_rec.json_path,'table_ref',_rec.table_ref,
                        'entity_type','PARTY','attribute_def_id',_rec.def_id,'storage_kind','JSONB_KEY','jsonb_column','custom_attributes','source','party_catalog_graph'),
                    now(), now())
            ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET properties=EXCLUDED.properties, is_active=true, updated_at=now()
            RETURNING id INTO _cf_id;
            IF _cf_id IS NULL THEN
                SELECT id INTO _cf_id FROM public.catalog_node WHERE tenant_id=_tenant AND qualified_path='attr:'||_tenant::text||':PARTY:'||_rec.field_cd;
            END IF;
            IF _term_edge IS NOT NULL AND _term_id IS NOT NULL AND _cf_id IS NOT NULL THEN
                INSERT INTO public.catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, is_active, relationship_type, created_at, updated_at)
                SELECT gen_random_uuid(), _tenant, _term_id, _cf_id, _term_edge,
                       jsonb_build_object('storage_kind','JSONB_KEY','json_path',_rec.json_path,'jsonb_column','custom_attributes','field_cd',_rec.field_cd,'entity_type','PARTY'),
                       true, 'TERM_MAPS_TO_CUSTOM_FIELD', now(), now()
                WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge e WHERE e.tenant_id=_tenant AND e.source_node_id=_term_id AND e.target_node_id=_cf_id AND e.edge_type_id=_term_edge);
            END IF;
            IF _bo_attr_edge IS NOT NULL AND _bo_node IS NOT NULL AND _cf_id IS NOT NULL THEN
                INSERT INTO public.catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, is_active, relationship_type, created_at, updated_at)
                SELECT gen_random_uuid(), _tenant, _bo_node, _cf_id, _bo_attr_edge,
                       jsonb_build_object('field_cd',_rec.field_cd,'entity_type','PARTY'),
                       true, 'BO_HAS_ATTRIBUTE', now(), now()
                WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge e WHERE e.tenant_id=_tenant AND e.source_node_id=_bo_node AND e.target_node_id=_cf_id AND e.edge_type_id=_bo_attr_edge);
            END IF;
        END IF;
    END LOOP;

    -- staging binding + pipelines
    _fields := '{
      "party_cd":"party_cd","legal_name":"legal_name","party_type":"party_type","segment":"segment",
      "tax_id":"tax_id","domicile":"domicile","status":"status","lei":"lei",
      "short_name":"custom_attributes->>''short_name''","email":"custom_attributes->>''email''",
      "phone":"custom_attributes->>''phone''","website":"custom_attributes->>''website''",
      "kyc_status":"custom_attributes->>''kyc_status''","aml_risk_rating":"custom_attributes->>''aml_risk_rating''",
      "fatca_status":"custom_attributes->>''fatca_status''","crs_status":"custom_attributes->>''crs_status''",
      "industry_naics":"custom_attributes->>''industry_naics''","bloomberg_company_id":"custom_attributes->>''bloomberg_company_id''",
      "swift_bic":"custom_attributes->>''swift_bic''","custodian_code":"custom_attributes->>''custodian_code''"
    }'::jsonb;

    IF _bo_id IS NOT NULL THEN
        INSERT INTO public.staging_bindings (tenant_id, bo_id, staging_table, fields, version, source_type)
        VALUES (_tenant, _bo_id, 'staging.party_data', _fields, 1, 'JSON_PATH')
        ON CONFLICT (tenant_id, bo_id, staging_table) DO UPDATE
          SET fields=EXCLUDED.fields, source_type=EXCLUDED.source_type, version=public.staging_bindings.version+1, updated_at=now();
    END IF;

    INSERT INTO public.data_pipeline_definitions (id, tenant_id, name, description, mode, target_entity, dag_json, batch_size, error_policy, is_active, created_by)
    VALUES (
      _file_id, _tenant,
      'Party Ingest → staging.party_data (file)',
      'Vendor CSV → staging.party_data. Change source_cd per feed. Unmapped columns fold into custom_attributes. Master merge applies GoldenSource > Bloomberg > Refinitiv > Internal.',
      'loader', 'PARTY',
      '{"version":1,"batch_size":500,"error_policy":"skip_and_log","nodes":[
        {"id":"src","type":"file_source","label":"Party vendor file","position":{"x":80,"y":120},
         "config":{"uri":"","format":"csv","has_header":true,"columns":[
           {"name":"party_cd","type":"string"},{"name":"legal_name","type":"string"},{"name":"party_type","type":"string"},
           {"name":"domicile","type":"string"},{"name":"status","type":"string"},{"name":"lei","type":"string"},
           {"name":"kyc_status","type":"string"},{"name":"aml_risk_rating","type":"string"},{"name":"email","type":"string"}
         ]}},
        {"id":"map","type":"map","label":"Map typed + keep customs","position":{"x":360,"y":120},
         "config":{"keep_unmapped":true,"fields":[
           {"from":"party_cd","to":"party_cd"},{"from":"legal_name","to":"legal_name"},{"from":"party_type","to":"party_type"},
           {"from":"domicile","to":"domicile"},{"from":"status","to":"status"},{"from":"lei","to":"lei"}
         ]}},
        {"id":"stg","type":"staging_sink","label":"Load staging.party_data","position":{"x":640,"y":120},
         "config":{"table":"staging.party_data","source_cd":"GOLDENSOURCE","domain":"PARTY"}}
      ],"edges":[{"from":"src","to":"map"},{"from":"map","to":"stg"}]}'::jsonb,
      500, 'skip_and_log', true, 'seed:party-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, dag_json=EXCLUDED.dag_json, is_active=true, last_modified_at=now();

    INSERT INTO public.data_pipeline_definitions (id, tenant_id, name, description, mode, target_entity, dag_json, batch_size, error_policy, is_active, created_by)
    VALUES (
      _master_id, _tenant,
      'Party Master ← staging (survivorship)',
      'Promote staging.party_data into mdm.party. Party is the reference target for Account.custodian_id / holder_party_id and Security.issuer_id.',
      'loader', 'PARTY',
      '{"version":1,"batch_size":100,"error_policy":"skip_and_log","nodes":[
        {"id":"trigger","type":"bo_source","label":"Trigger (Party BO)","position":{"x":80,"y":120},"config":{"bo_key":"party","limit":1}},
        {"id":"master","type":"master_sink","label":"Master Party from staging","position":{"x":420,"y":120},
         "config":{"entity_type":"PARTY","staging_table":"staging.party_data","batch_size":100,"require_semantic_terms":true}}
      ],"edges":[{"from":"trigger","to":"master"}]}'::jsonb,
      100, 'skip_and_log', true, 'seed:party-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, dag_json=EXCLUDED.dag_json, is_active=true, last_modified_at=now();
END
$party$;

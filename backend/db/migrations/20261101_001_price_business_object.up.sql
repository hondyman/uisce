-- The Price business object: a golden price - one security, price type and valuation date - over the
-- crims golden price table (mdm.price_golden_record), mastered by the price master (crims 0018). Vendor
-- price files bind to it by staging bindings: its fields (ValuationDate, Currency; Price and PriceTypeCd
-- for files with one row per price) plus the mastering keys (id:<TYPE>, @source_key, value:<PRICE TYPE>).
--
-- Fields are named by their semantic term, as in the MDM seeds: existing terms are reused (ValuationDate,
-- Currency, GoldenVersion, ...), the price-specific ones are added, and each column gets its MAPS_TO edge.
-- Gold-copy tenant, is_core. Skips with a NOTICE when the table is not cataloged. Idempotent.
--
-- Also message set 9400 (mastering) 37-40 for the time-series engine.

DO $$
DECLARE
    v_tenant   uuid;
    v_ds       uuid;
    v_table    uuid;
    v_term_ty  uuid;
    v_maps_to  uuid;
    v_bo_id    uuid := md5('mdm-bo:price')::uuid;
    f          record;
    v_term     uuid;
    v_col      uuid;
    v_n        int := 0;
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN
        RAISE NOTICE 'price BO: no gold-copy tenant, skipping';
        RETURN;
    END IF;
    SELECT id, tenant_datasource_id INTO v_table, v_ds FROM public.catalog_node
     WHERE tenant_id = v_tenant AND qualified_path = '/mdm/price_golden_record';
    IF v_table IS NULL THEN
        RAISE NOTICE 'price BO: /mdm/price_golden_record not cataloged, skipping';
        RETURN;
    END IF;
    SELECT id INTO v_term_ty FROM public.catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO v_maps_to FROM public.catalog_edge_type
     WHERE edge_type_name = 'MAPS_TO' AND (tenant_id IS NULL OR tenant_id::text = v_tenant::text)
     ORDER BY (tenant_id IS NULL) LIMIT 1;

    INSERT INTO public.physical_backend
        (backend_id, backend_name, description, storage_tier, dialect_name, driver_class, is_system)
    SELECT tpd.id, tpd.source_name, 'Auto-registered backend', 'oltp', 'postgres', '*sql.DB', false
      FROM public.tenant_product_datasource tpd WHERE tpd.id = v_ds
    ON CONFLICT (backend_id) DO NOTHING;

    INSERT INTO public.business_objects
        (id, tenant_id, model_id, bo_key, bo_name, description, bo_type,
         classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
         is_active, is_core, driver_table_id, driver_table_name)
    VALUES (v_bo_id, v_tenant, v_bo_id, 'price', 'Price',
            'Golden price of a security for a price type and valuation date, with its winning source and controls.',
            'ENTITY', v_table,
            coalesce((SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
                       AND qualified_path = '/mdm/price_golden_record/price_entity_id'), v_table),
            coalesce((SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
                       AND qualified_path = '/mdm/price_golden_record/id'), v_table),
            v_table, true, true, v_table, '/mdm/price_golden_record')
    ON CONFLICT DO NOTHING;

    INSERT INTO public.business_object_binding
        (tenant_id, bo_id, backend_id, driving_node_id, binding_name, is_core, is_active, is_default, temporal_mode)
    SELECT v_tenant, v_bo_id, v_ds, v_table, 'Price Binding', true, true, true, 'NONE'
     WHERE NOT EXISTS (SELECT 1 FROM public.business_object_binding WHERE bo_id = v_bo_id AND binding_name = 'Price Binding');

    FOR f IN
        SELECT * FROM (VALUES
            ('PriceEntityId',      'price_entity_id',     'KEY',            'Price entity',        true,  1),
            ('PriceEntityType',    'price_entity_type',   'DIMENSION',      'Price entity type',   true,  2),
            ('ValuationDate',      'price_date',          'TIME_DIMENSION', 'Valuation date',      true,  3),
            ('PriceTypeCd',        'price_type_cd',       'DIMENSION',      'Price type',          true,  4),
            ('Price',              'golden_value',        'MEASURE',        'Price',               true,  5),
            ('Currency',           'currency',            'DIMENSION',      'Currency',            false, 6),
            ('PriceTime',          'price_time',          'TIME_DIMENSION', 'Price time',          false, 7),
            ('IsOfficialPrice',    'is_official',         'DIMENSION',      'Official price',      true,  8),
            ('IsStalePrice',       'is_stale',            'DIMENSION',      'Stale price',         true,  9),
            ('WinningSourceId',    'winning_source_id',   'DIMENSION',      'Winning source',      false, 10),
            ('SourceCount',        'source_count',        'MEASURE',        'Source count',        true,  11),
            ('VariancePct',        'variance_pct',        'MEASURE',        'Cross-source variance %', false, 12),
            ('Confidence',         'confidence',          'MEASURE',        'Confidence',          false, 13),
            ('DqScore',            'dq_score',            'MEASURE',        'Data quality score',  false, 14),
            ('GoldenVersion',      'golden_version',      'DIMENSION',      'Version',             true,  15),
            ('IsCurrent',          'is_current',          'DIMENSION',      'Current',             true,  16),
            ('Status',             'status',              'DIMENSION',      'Status',              true,  17),
            ('EffectiveDate',      'effective_date',      'TIME_DIMENSION', 'Effective date',      true,  18),
            ('KnowledgeTimestamp', 'knowledge_timestamp', 'TIME_DIMENSION', 'Known at',            true,  19),
            ('PublishedAt',        'published_at',        'TIME_DIMENSION', 'Published at',        false, 20)
        ) AS x(term, col, role, label, required, ord)
    LOOP
        SELECT id INTO v_col FROM public.catalog_node
         WHERE tenant_id = v_tenant AND qualified_path = '/mdm/price_golden_record/' || f.col;
        IF v_col IS NULL THEN
            RAISE NOTICE 'price BO: column % not cataloged, skipping %', f.col, f.term;
            CONTINUE;
        END IF;

        -- The term: reuse, else add.
        SELECT id INTO v_term FROM public.catalog_node
         WHERE tenant_id = v_tenant AND node_type_id = v_term_ty AND node_name = f.term
         ORDER BY (qualified_path = 'semantic_term/' || f.term) DESC LIMIT 1;
        IF v_term IS NULL THEN
            INSERT INTO public.catalog_node (id, node_type_id, node_name, description, properties, qualified_path,
                                             tenant_id, tenant_datasource_id, is_active, governance_status)
            VALUES (md5('semantic-term:' || f.term)::uuid, v_term_ty, f.term, f.label, '{}'::jsonb,
                    'semantic_term/' || f.term, v_tenant, v_ds, true, 'ACTIVE')
            ON CONFLICT (tenant_id, qualified_path) DO NOTHING;
            SELECT id INTO v_term FROM public.catalog_node
             WHERE tenant_id = v_tenant AND qualified_path = 'semantic_term/' || f.term;
        END IF;

        -- Lineage: term MAPS_TO column.
        IF v_maps_to IS NOT NULL AND NOT EXISTS (
            SELECT 1 FROM public.catalog_edge
             WHERE source_node_id = v_term AND target_node_id = v_col AND edge_type_id = v_maps_to) THEN
            INSERT INTO public.catalog_edge (id, source_node_id, target_node_id, edge_type_id, tenant_id, created_at, updated_at)
            VALUES (gen_random_uuid(), v_term, v_col, v_maps_to, v_tenant, now(), now());
        END IF;

        INSERT INTO public.business_object_fields
            (tenant_id, bo_id, term_node_id, field_name, field_role, aggregation_type,
             binding_requirement, eligibility_source, subtype_scope, is_exposed, inherits_defaults,
             display_name, technical_name, data_type, is_required, is_system, display_order)
        SELECT v_tenant, v_bo_id, v_term, f.term, f.role, 'NONE',
               CASE WHEN f.required THEN 'REQUIRED' ELSE 'OPTIONAL' END, 'DIRECT', 'ALL', true, true,
               f.label, f.col, (SELECT properties->>'data_type' FROM public.catalog_node WHERE id = v_col),
               f.required, false, f.ord
         WHERE NOT EXISTS (SELECT 1 FROM public.business_object_fields WHERE bo_id = v_bo_id AND field_name = f.term);
        v_n := v_n + 1;
    END LOOP;
    RAISE NOTICE 'price BO: % fields', v_n;
END
$$;

-- Message set 9400 (mastering): the time-series (price) engine.
INSERT INTO public.message_catalog (set_nbr, message_nbr, language_cd, severity, message_text, description, user_action) VALUES
  (9400, 37, 'en', 'Error', '%1 binds no price values - bind %2 (with a price type) or value:<PRICE TYPE> keys to its columns.', NULL, NULL),
  (9400, 37, 'es', 'Error', '%1 no vincula ningún precio: vincule %2 (con un tipo de precio) o claves value:<TIPO DE PRECIO> a sus columnas.', NULL, NULL),
  (9400, 37, 'fr', 'Error', '%1 ne lie aucun prix : liez %2 (avec un type de prix) ou des clés value:<TYPE DE PRIX> à ses colonnes.', NULL, NULL),
  (9400, 38, 'en', 'Error', '%1 has no valuation date - bind %2 to the column holding it.', NULL, NULL),
  (9400, 38, 'es', 'Error', '%1 no tiene fecha de valoración: vincule %2 a la columna que la contiene.', NULL, NULL),
  (9400, 38, 'fr', 'Error', '%1 n''a pas de date de valorisation : liez %2 à la colonne qui la contient.', NULL, NULL),
  (9400, 39, 'en', 'Error', 'Price types %1 are not defined - add them to the price types or correct the binding.', NULL, NULL),
  (9400, 39, 'es', 'Error', 'Los tipos de precio %1 no están definidos: añádalos a los tipos de precio o corrija el vínculo.', NULL, NULL),
  (9400, 39, 'fr', 'Error', 'Les types de prix %1 ne sont pas définis : ajoutez-les aux types de prix ou corrigez la liaison.', NULL, NULL),
  (9400, 40, 'en', 'Error', '%1 is mastered as a time series; this applies to record masters only.', NULL, NULL),
  (9400, 40, 'es', 'Error', '%1 se maestriza como serie temporal; esto solo se aplica a maestros de registros.', NULL, NULL),
  (9400, 40, 'fr', 'Error', '%1 est maîtrisé en série temporelle ; ceci ne s''applique qu''aux référentiels d''enregistrements.', NULL, NULL)
ON CONFLICT (set_nbr, message_nbr, language_cd) DO NOTHING;

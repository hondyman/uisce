-- Migration: Personalization Vocabulary Graph — Phase 0
-- Seeds typed edge types for synonym/alias resolution and backfills
-- synonym nodes and edges from legacy JSONB/alias sources.
-- Keeps JSONB synonyms and catalog_aliases as authoritative seed;
-- the graph is the canonical read path for AI intent resolution.

DO $$
DECLARE
    gold_copy_tenant UUID := '00000000-0000-0000-0000-000000000000';
    biz_term_type_id UUID;
    sem_term_type_id UUID;
    synonym_type_id UUID;
    biz_obj_type_id UUID;
    has_synonym_type_id UUID;
    alias_of_type_id UUID;
    maps_to_sem_type_id UUID;
    feeds_into_type_id UUID;
    _tenant UUID;
    syn JSONB;
    syn_text TEXT;
    bt_id UUID;
    syn_node_id UUID;
    alias_rec RECORD;
    bt_key TEXT;
BEGIN
    -- ============================================================
    -- 1. Resolve required node type IDs
    -- ============================================================
    SELECT id INTO biz_term_type_id
      FROM catalog_node_type
     WHERE catalog_type_name = 'business_term'
     LIMIT 1;

    SELECT id INTO sem_term_type_id
      FROM catalog_node_type
     WHERE catalog_type_name = 'semantic_term'
     LIMIT 1;

    SELECT id INTO biz_obj_type_id
      FROM catalog_node_type
     WHERE catalog_type_name = 'business_object'
     LIMIT 1;

    -- Create 'synonym' node type if it does not exist (for alias nodes)
    SELECT id INTO synonym_type_id
      FROM catalog_node_type
     WHERE catalog_type_name = 'synonym'
       AND tenant_id = gold_copy_tenant
     LIMIT 1;

    IF synonym_type_id IS NULL THEN
        INSERT INTO catalog_node_type (
            tenant_id, catalog_type_name, description,
            is_active, config, created_at, updated_at
        ) VALUES (
            gold_copy_tenant, 'synonym',
            'Synonym / alias node for enterprise vocabulary resolution',
            true, '{}'::jsonb, now(), now()
        )
        RETURNING id INTO synonym_type_id;
        RAISE NOTICE 'Created synonym node type: %', synonym_type_id;
    ELSE
        RAISE NOTICE 'synonym node type already exists: %', synonym_type_id;
    END IF;

    RAISE NOTICE 'Node types — business_term=%, semantic_term=%, synonym=%, business_object=%',
        biz_term_type_id, sem_term_type_id, synonym_type_id, biz_obj_type_id;

    -- ============================================================
    -- 2. Seed Gold-Copy edge types (idempotent)
    -- ============================================================

    -- HAS_SYNONYM: business_term → business_term
    INSERT INTO catalog_edge_types (
        tenant_id, edge_type_name, description,
        source_node_type_id, target_node_type_id,
        is_active, config, created_at, updated_at
    )
    SELECT
        gold_copy_tenant,
        'HAS_SYNONYM',
        'Indicates two business terms are synonyms (e.g. "Order" ≈ "Trade ticket")',
        biz_term_type_id, biz_term_type_id,
        true, '{"cardinality":"N:N","directed":false}'::jsonb, now(), now()
    WHERE NOT EXISTS (
        SELECT 1 FROM catalog_edge_types
         WHERE edge_type_name = 'HAS_SYNONYM'
           AND tenant_id = gold_copy_tenant
    )
    RETURNING id INTO has_synonym_type_id;

    IF has_synonym_type_id IS NOT NULL THEN
        RAISE NOTICE 'Seeded HAS_SYNONYM edge type: %', has_synonym_type_id;
    ELSE
        SELECT id INTO has_synonym_type_id
          FROM catalog_edge_types
         WHERE edge_type_name = 'HAS_SYNONYM'
           AND tenant_id = gold_copy_tenant
         LIMIT 1;
    END IF;

    -- ALIAS_OF: synonym/alias node → business_term
    INSERT INTO catalog_edge_types (
        tenant_id, edge_type_name, description,
        source_node_type_id, target_node_type_id,
        is_active, config, created_at, updated_at
    )
    SELECT
        gold_copy_tenant,
        'ALIAS_OF',
        'Indicates an alias/synonym node resolves to a canonical business term',
        synonym_type_id, biz_term_type_id,
        true, '{"cardinality":"N:1","directed":true}'::jsonb, now(), now()
    WHERE NOT EXISTS (
        SELECT 1 FROM catalog_edge_types
         WHERE edge_type_name = 'ALIAS_OF'
           AND tenant_id = gold_copy_tenant
    )
    RETURNING id INTO alias_of_type_id;

    IF alias_of_type_id IS NOT NULL THEN
        RAISE NOTICE 'Seeded ALIAS_OF edge type: %', alias_of_type_id;
    ELSE
        SELECT id INTO alias_of_type_id
          FROM catalog_edge_types
         WHERE edge_type_name = 'ALIAS_OF'
           AND tenant_id = gold_copy_tenant
         LIMIT 1;
    END IF;

    -- MAPS_TO_SEMANTIC_TERM: business_term → semantic_term
    INSERT INTO catalog_edge_types (
        tenant_id, edge_type_name, description,
        source_node_type_id, target_node_type_id,
        is_active, config, created_at, updated_at
    )
    SELECT
        gold_copy_tenant,
        'MAPS_TO_SEMANTIC_TERM',
        'Indicates a business term maps to a semantic term (canonical definition)',
        biz_term_type_id, sem_term_type_id,
        true, '{"cardinality":"N:1","directed":true}'::jsonb, now(), now()
    WHERE NOT EXISTS (
        SELECT 1 FROM catalog_edge_types
         WHERE edge_type_name = 'MAPS_TO_SEMANTIC_TERM'
           AND tenant_id = gold_copy_tenant
    )
    RETURNING id INTO maps_to_sem_type_id;

    IF maps_to_sem_type_id IS NOT NULL THEN
        RAISE NOTICE 'Seeded MAPS_TO_SEMANTIC_TERM edge type: %', maps_to_sem_type_id;
    ELSE
        SELECT id INTO maps_to_sem_type_id
          FROM catalog_edge_types
         WHERE edge_type_name = 'MAPS_TO_SEMANTIC_TERM'
           AND tenant_id = gold_copy_tenant
         LIMIT 1;
    END IF;

    -- FEEDS_INTO: business_object → business_object (cross-domain lineage)
    INSERT INTO catalog_edge_types (
        tenant_id, edge_type_name, description,
        source_node_type_id, target_node_type_id,
        is_active, config, created_at, updated_at
    )
    SELECT
        gold_copy_tenant,
        'FEEDS_INTO',
        'Indicates a business object feeds data into another (cross-domain lineage)',
        biz_obj_type_id, biz_obj_type_id,
        true, '{"cardinality":"N:N","directed":true}'::jsonb, now(), now()
    WHERE NOT EXISTS (
        SELECT 1 FROM catalog_edge_types
         WHERE edge_type_name = 'FEEDS_INTO'
           AND tenant_id = gold_copy_tenant
    )
    RETURNING id INTO feeds_into_type_id;

    IF feeds_into_type_id IS NOT NULL THEN
        RAISE NOTICE 'Seeded FEEDS_INTO edge type: %', feeds_into_type_id;
    ELSE
        SELECT id INTO feeds_into_type_id
          FROM catalog_edge_types
         WHERE edge_type_name = 'FEEDS_INTO'
           AND tenant_id = gold_copy_tenant
         LIMIT 1;
    END IF;

    RAISE NOTICE 'Edge types seeded — HAS_SYNONYM=%, ALIAS_OF=%, MAPS_TO_SEMANTIC_TERM=%, FEEDS_INTO=%',
        has_synonym_type_id, alias_of_type_id, maps_to_sem_type_id, feeds_into_type_id;

    -- ============================================================
    -- 3. Backfill synonym nodes from business_terms.synonyms JSONB
    --    For each business_term, iterate its synonyms array and
    --    create a catalog_node (synonym type) + HAS_SYNONYM edge.
    -- ============================================================
    IF has_synonym_type_id IS NOT NULL AND synonym_type_id IS NOT NULL THEN
        FOR bt_id, syn, _tenant IN
            SELECT bt.id, bt.synonyms, bt.tenant_id
              FROM business_terms bt
             WHERE bt.synonyms IS NOT NULL
               AND jsonb_typeof(bt.synonyms) = 'array'
               AND jsonb_array_length(bt.synonyms) > 0
        LOOP
            -- Iterate each synonym text in the JSONB array
            FOR syn_text IN
                SELECT jsonb_array_elements_text(bt.synonyms)
            LOOP
                -- Upsert synonym catalog_node (one per tenant+synonym text, type=synonym)
                INSERT INTO catalog_node (
                    tenant_id, tenant_datasource_id, node_type_id,
                    node_name, qualified_path, description,
                    properties, created_at, updated_at
                )
                VALUES (
                    _tenant,
                    '00000000-0000-0000-0000-000000000000'::uuid,  -- placeholder datasource
                    synonym_type_id,
                    lower(trim(syn_text)),
                    'synonym/' || lower(trim(syn_text)),
                    'Synonym node auto-promoted from business_terms.synonyms JSONB',
                    ('{"source":"business_terms_jsonb","canonical_term_id":"' || bt_id || '"}')::jsonb,
                    now(), now()
                )
                ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path)
                DO UPDATE SET
                    properties = EXCLUDED.properties,
                    updated_at = now()
                RETURNING id INTO syn_node_id;

                -- Upsert HAS_SYNONYM edge (synonym_node → business_term node)
                -- We traverse: synonym_node (source) → business_term (target)
                -- Note: the edge connects the synonym node to the business_term
                -- that owns the synonym. Direction: synonym → canonical term.
                INSERT INTO catalog_edge (
                    tenant_datasource_id, source_node_id, target_node_id,
                    relationship_type, edge_type_id, properties,
                    created_at, updated_at, tenant_id
                )
                VALUES (
                    '00000000-0000-0000-0000-000000000000'::uuid,
                    syn_node_id,
                    bt_id,
                    'HAS_SYNONYM',
                    has_synonym_type_id,
                    ('{"direction":"synonym_to_term","source":"business_terms_jsonb"}')::jsonb,
                    now(), now(),
                    _tenant
                )
                ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
                DO NOTHING;

            END LOOP;
        END LOOP;
        RAISE NOTICE 'Backfill complete: synonym nodes + HAS_SYNONYM edges from business_terms.synonyms JSONB';
    ELSE
        RAISE NOTICE 'Skipping synonym backfill: edge type or node type missing';
    END IF;

    -- ============================================================
    -- 4. Backfill ALIAS_OF edges from catalog_aliases
    --    Each row in catalog_aliases becomes a synonym catalog_node
    --    + ALIAS_OF edge to the canonical business_term.
    -- ============================================================
    IF alias_of_type_id IS NOT NULL AND synonym_type_id IS NOT NULL THEN
        FOR alias_rec IN
            SELECT ca.alias, ca.canonical_key, ca.tenant_id
              FROM catalog_aliases ca
        LOOP
            -- Find the canonical business_term node by canonical_key
            -- The canonical_key is the bo_key or term key
            SELECT cn.id INTO bt_id
              FROM catalog_node cn
              JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
             WHERE cnt.catalog_type_name = 'business_term'
               AND cn.node_name = alias_rec.canonical_key
               AND cn.tenant_id = alias_rec.tenant_id
             LIMIT 1;

            CONTINUE WHEN bt_id IS NULL;

            -- Upsert synonym/alias node
            INSERT INTO catalog_node (
                tenant_id, tenant_datasource_id, node_type_id,
                node_name, qualified_path, description,
                properties, created_at, updated_at
            )
            VALUES (
                alias_rec.tenant_id,
                '00000000-0000-0000-0000-000000000000'::uuid,
                synonym_type_id,
                lower(trim(alias_rec.alias)),
                'synonym/' || lower(trim(alias_rec.alias)),
                'Alias node promoted from catalog_aliases',
                ('{"source":"catalog_aliases","canonical_key":"' || alias_rec.canonical_key || '"}')::jsonb,
                now(), now()
            )
            ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path)
            DO UPDATE SET
                properties = EXCLUDED.properties,
                updated_at = now()
            RETURNING id INTO syn_node_id;

            -- Upsert ALIAS_OF edge (alias_node → canonical business_term)
            INSERT INTO catalog_edge (
                tenant_datasource_id, source_node_id, target_node_id,
                relationship_type, edge_type_id, properties,
                created_at, updated_at, tenant_id
            )
            VALUES (
                '00000000-0000-0000-0000-000000000000'::uuid,
                syn_node_id,
                bt_id,
                'ALIAS_OF',
                alias_of_type_id,
                ('{"source":"catalog_aliases","alias":"' || alias_rec.alias || '"}')::jsonb,
                now(), now(),
                alias_rec.tenant_id
            )
            ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
            DO NOTHING;
        END LOOP;
        RAISE NOTICE 'Backfill complete: synonym nodes + ALIAS_OF edges from catalog_aliases';
    ELSE
        RAISE NOTICE 'Skipping alias backfill: edge type or node type missing';
    END IF;

    -- ============================================================
    -- 5. Backfill MAPS_TO_SEMANTIC_TERM edges
    --    Walk business_terms that have a canonical_key that matches
    --    an existing semantic_term node name.
    -- ============================================================
    IF maps_to_sem_type_id IS NOT NULL THEN
        INSERT INTO catalog_edge (
            tenant_datasource_id, source_node_id, target_node_id,
            relationship_type, edge_type_id, properties,
            created_at, updated_at, tenant_id
        )
        SELECT
            '00000000-0000-0000-0000-000000000000'::uuid,
            bt.id,
            st.id,
            'MAPS_TO_SEMANTIC_TERM',
            maps_to_sem_type_id,
            ('{"source":"business_terms.canonical_key","confidence":1.0}')::jsonb,
            now(), now(),
            bt.tenant_id
        FROM business_terms bt
        JOIN catalog_node bt_node ON bt_node.node_name = bt.term
                                 AND bt_node.tenant_id = bt.tenant_id
                                 AND bt_node.node_type_id = biz_term_type_id
        JOIN catalog_node st ON st.node_name = bt.canonical_key
                            AND st.tenant_id = bt.tenant_id
                            AND st.node_type_id = sem_term_type_id
        WHERE bt.canonical_key IS NOT NULL
          AND bt.canonical_key != ''
        ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
        DO NOTHING;
        RAISE NOTICE 'Backfill complete: MAPS_TO_SEMANTIC_TERM edges from business_terms canonical_key';
    ELSE
        RAISE NOTICE 'Skipping MAPS_TO_SEMANTIC_TERM backfill: edge type missing';
    END IF;

    RAISE NOTICE 'Phase 0 vocabulary graph migration completed successfully';

END $$;

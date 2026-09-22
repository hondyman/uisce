-- 20261022_001_seed_mdm_tier1_business_objects.up.sql
--
-- Seeds tier-1 MDM business objects (9 root + 5 child), their bindings, fields
-- (from the semantic terms already mapped to each column) and relationships.
--
-- Conventions (matching the existing core BOs):
--   * Authored in the gold-copy tenant with is_core = true, so regular tenants
--     inherit read-only. Nothing here writes to the existing core BOs.
--   * business_objects.model_id = id.
--   * Every node id is resolved from catalog_node by qualified_path at run
--     time, so the seed is environment-independent.
--   * Idempotent: ON CONFLICT DO NOTHING everywhere, no wipe. Re-running
--     preserves manual edits made after the first run.
--   * Skips cleanly (NOTICE) when the gold-copy tenant or the /mdm catalog
--     nodes are absent, so CI and fresh environments are unaffected.
--
-- Deferred (not tier 1): polymorphic references (entity_id + entity_type),
-- BOs for orm.issuer / orm.benchmark (none exist yet), and the EAV layer.

DO $$
DECLARE
    v_tenant   uuid;
    v_ds       uuid;
    v_bo       record;
    v_rel      record;
    v_bo_id    uuid;
    v_table    uuid;
    v_from     uuid;
    v_to       uuid;
    v_basis    uuid;
    v_edge_type uuid;
    v_n        int;
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN
        RAISE NOTICE 'mdm tier-1 seed: no gold-copy tenant, skipping';
        RETURN;
    END IF;

    SELECT tenant_datasource_id INTO v_ds
    FROM public.catalog_node
    WHERE tenant_id = v_tenant AND qualified_path = '/mdm/party'
    LIMIT 1;
    IF v_ds IS NULL THEN
        RAISE NOTICE 'mdm tier-1 seed: /mdm/party not cataloged, skipping';
        RETURN;
    END IF;

    -- business_object_binding.backend_id references physical_backend. Existing
    -- datasources get an auto-registered row; register this one if missing.
    INSERT INTO public.physical_backend
        (backend_id, backend_name, description, storage_tier, dialect_name, driver_class, is_system)
    SELECT tpd.id, tpd.source_name, 'Auto-registered backend', 'oltp', 'postgres', '*sql.DB', false
    FROM public.tenant_product_datasource tpd
    WHERE tpd.id = v_ds
    ON CONFLICT (backend_id) DO NOTHING;

    -- ------------------------------------------------------------------
    -- Business objects: (bo_key, name, type, table, business key column)
    -- ------------------------------------------------------------------
    CREATE TEMP TABLE _mdm_bo (
        bo_key text, bo_name text, bo_type text, tbl text, bk_col text, descr text,
        temporal_mode text, valid_from_col text, valid_to_col text
    );

    INSERT INTO _mdm_bo VALUES
      ('party',                'Party',                'ENTITY',    'party',                'party_cd',     'Master party (legal entity or person).',                              'NONE', NULL, NULL),
      ('portfolio',            'Portfolio',            'ENTITY',    'portfolio',            'portfolio_cd', 'Investment portfolio master.',                                        'NONE', NULL, NULL),
      ('mandate',              'Mandate',              'ENTITY',    'mandate',              'mandate_cd',   'Investment mandate governing a portfolio.',                           'NONE', NULL, NULL),
      ('portfolio_composite',  'Portfolio Composite',  'ENTITY',    'portfolio_composite',  'composite_cd', 'Composite grouping of portfolios.',                                   'NONE', NULL, NULL),
      ('source_system',        'Source System',        'REFERENCE', 'source_system',        'source_cd',    'Registered source and target systems for master data.',               'NONE', NULL, NULL),
      ('steward',              'Steward',              'ENTITY',    'steward',              'email',        'Data steward.',                                                       'NONE', NULL, NULL),
      ('issuer_steward',       'Issuer Steward',       'ENTITY',    'issuer_steward',       'id',           'Issuer data steward with scoped permissions.',                        'NONE', NULL, NULL),
      ('hierarchy',            'Hierarchy',            'ENTITY',    'hierarchy',            'name',         'Named hierarchy definition.',                                         'NONE', NULL, NULL),
      ('issuer_golden_record', 'Issuer Golden Record', 'ENTITY',    'issuer_golden_record', 'id',           'Versioned golden record for an issuer.',                              'NONE', NULL, NULL),
      ('portfolio_account',    'Portfolio Account',    'BRIDGE',    'portfolio_account',    'id',           'Assignment of an account to a portfolio.',                            'VALID_TIME', 'effective_from', 'effective_to'),
      ('mandate_restriction',  'Mandate Restriction',  'ENTITY',    'mandate_restriction',  'id',           'Restriction attached to a mandate.',                                  'VALID_TIME', 'effective_from', 'effective_to'),
      ('portfolio_target',     'Portfolio Target',     'ENTITY',    'portfolio_target',     'id',           'Target allocation for a mandate by asset class.',                     'VALID_TIME', 'effective_from', 'effective_to'),
      ('hierarchy_closure',    'Hierarchy Closure',    'BRIDGE',    'hierarchy_closure',    'id',           'Ancestor/descendant closure rows for a hierarchy.',                   'NONE', NULL, NULL),
      ('issuer_golden_field',  'Issuer Golden Field',  'ENTITY',    'issuer_golden_field',  'id',           'Surviving field value on an issuer golden record.',                   'NONE', NULL, NULL);

    FOR v_bo IN SELECT * FROM _mdm_bo LOOP
        SELECT id INTO v_table FROM public.catalog_node
        WHERE tenant_id = v_tenant AND qualified_path = '/mdm/' || v_bo.tbl;
        IF v_table IS NULL THEN
            RAISE NOTICE 'mdm tier-1 seed: table node /mdm/% missing, skipping %', v_bo.tbl, v_bo.bo_key;
            CONTINUE;
        END IF;

        INSERT INTO public.business_objects
            (id, tenant_id, model_id, bo_key, bo_name, description, bo_type,
             classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
             is_active, is_core, driver_table_id, driver_table_name)
        VALUES (
            md5('mdm-bo:' || v_bo.bo_key)::uuid, v_tenant, md5('mdm-bo:' || v_bo.bo_key)::uuid,
            v_bo.bo_key, v_bo.bo_name, v_bo.descr, v_bo.bo_type,
            v_table,
            coalesce((SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
                      AND qualified_path = '/mdm/' || v_bo.tbl || '/' || v_bo.bk_col), v_table),
            coalesce((SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
                      AND qualified_path = '/mdm/' || v_bo.tbl || '/id'), v_table),
            v_table,
            true, true, v_table, '/mdm/' || v_bo.tbl)
        ON CONFLICT DO NOTHING;

        SELECT id INTO v_bo_id FROM public.business_objects
        WHERE tenant_id = v_tenant AND bo_key = v_bo.bo_key;

        -- Binding: one per BO on the CRIMS datasource, driven by the table node.
        INSERT INTO public.business_object_binding
            (tenant_id, bo_id, backend_id, driving_node_id, binding_name, is_core, is_active,
             temporal_mode, valid_from_column_node_id, valid_to_column_node_id)
        VALUES (
            v_tenant, v_bo_id, v_ds, v_table, v_bo.bo_name || ' Binding', true, true,
            v_bo.temporal_mode,
            (SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
              AND qualified_path = '/mdm/' || v_bo.tbl || '/' || v_bo.valid_from_col),
            (SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant
              AND qualified_path = '/mdm/' || v_bo.tbl || '/' || v_bo.valid_to_col))
        ON CONFLICT DO NOTHING;

        -- Fields: one per column, named by the column's single MAPS_TO semantic term.
        INSERT INTO public.business_object_fields
            (tenant_id, bo_id, term_node_id, field_name, field_role, aggregation_type,
             binding_requirement, eligibility_source, subtype_scope, is_exposed, inherits_defaults,
             display_name, technical_name, data_type, is_required, is_system, display_order)
        SELECT
            v_tenant, v_bo_id, term.id, term.node_name,
            CASE
                WHEN split_part(col.qualified_path, '/', 4) IN ('id', v_bo.bk_col) THEN 'KEY'
                WHEN col.properties->>'data_type' ~* 'date|time'                    THEN 'TIME_DIMENSION'
                WHEN col.properties->>'data_type' ~* 'numeric|decimal'              THEN 'MEASURE'
                WHEN col.properties->>'data_type' ~* 'json'                         THEN 'ATTRIBUTE'
                ELSE 'DIMENSION'
            END,
            'NONE',
            CASE WHEN (col.properties->>'is_nullable') = 'false' THEN 'REQUIRED' ELSE 'OPTIONAL' END,
            'DIRECT', 'ALL',
            split_part(col.qualified_path, '/', 4) NOT IN ('tenant_id', 'custom_attributes'),
            true,
            coalesce(col.properties->>'title', term.node_name),
            split_part(col.qualified_path, '/', 4),
            col.properties->>'data_type',
            (col.properties->>'is_nullable') = 'false',
            split_part(col.qualified_path, '/', 4) IN ('tenant_id', 'created_at', 'updated_at'),
            coalesce((col.properties->>'ordinal_position')::int, 0)
        FROM public.catalog_node col
        JOIN public.catalog_node_type cnt ON cnt.id = col.node_type_id
             AND cnt.catalog_type_name IN ('column', 'database_column')
        JOIN public.catalog_edge e ON e.target_node_id = col.id AND e.tenant_id = v_tenant
        JOIN public.catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
        JOIN public.catalog_node term ON term.id = e.source_node_id
        JOIN public.catalog_node_type tnt ON tnt.id = term.node_type_id
             AND tnt.catalog_type_name = 'semantic_term'
        WHERE col.tenant_id = v_tenant
          AND col.qualified_path LIKE '/mdm/' || v_bo.tbl || '/%'
        ON CONFLICT DO NOTHING;

        GET DIAGNOSTICS v_n = ROW_COUNT;
        RAISE NOTICE 'mdm tier-1 seed: % -> % fields inserted', v_bo.bo_key, v_n;
    END LOOP;

    -- ------------------------------------------------------------------
    -- Relationships derived from the declared FKs (column FK metadata).
    -- (parent, child, child FK column, parent key column, parent rel_key, child rel_key)
    -- ------------------------------------------------------------------
    SELECT id INTO v_edge_type FROM public.catalog_edge_type
    -- catalog_edge_type.tenant_id is varchar (all rows are valid UUIDs); compare
    -- as uuid rather than casting the tenant down to text.
    WHERE tenant_id::uuid = v_tenant AND edge_type_name = 'BO_RELATIONSHIP' LIMIT 1;

    FOR v_rel IN
        SELECT * FROM (VALUES
          ('portfolio',            'portfolio_account',   'portfolio_id',     'portfolio_accounts', 'portfolio'),
          ('portfolio',            'mandate',             'portfolio_id',     'mandates',           'portfolio'),
          ('mandate',              'mandate_restriction', 'mandate_id',       'restrictions',       'mandate'),
          ('mandate',              'portfolio_target',    'mandate_id',       'targets',            'mandate'),
          ('hierarchy',            'hierarchy_closure',   'hierarchy_id',     'closure_rows',       'hierarchy'),
          ('issuer_golden_record', 'issuer_golden_field', 'golden_record_id', 'fields',             'golden_record'),
          ('source_system',        'issuer_golden_field', 'source_system_id', 'golden_fields',      'source_system')
        ) AS r(parent_bo, child_bo, fk_col, parent_key, child_key)
    LOOP
        SELECT id INTO v_from FROM public.business_objects WHERE tenant_id = v_tenant AND bo_key = v_rel.parent_bo;
        SELECT id INTO v_to   FROM public.business_objects WHERE tenant_id = v_tenant AND bo_key = v_rel.child_bo;
        IF v_from IS NULL OR v_to IS NULL THEN CONTINUE; END IF;

        SELECT id INTO v_basis FROM public.catalog_node
        WHERE tenant_id = v_tenant AND qualified_path = '/mdm/' || v_rel.child_bo || '/' || v_rel.fk_col;

        INSERT INTO public.business_object_relationships
            (tenant_id, from_bo_id, to_bo_id, rel_key, rel_name, cardinality, join_type, relationship_basis_node_id, is_active)
        VALUES
            (v_tenant, v_from, v_to, v_rel.parent_key, initcap(replace(v_rel.parent_key, '_', ' ')), '1:M', 'LEFT', v_basis, true),
            (v_tenant, v_to, v_from, v_rel.child_key,  initcap(replace(v_rel.child_key,  '_', ' ')), 'M:1', 'LEFT', v_basis, true)
        ON CONFLICT DO NOTHING;

        -- Join columns live on the BO_RELATIONSHIP edge (child table -> parent table).
        IF v_edge_type IS NOT NULL AND NOT EXISTS (
            SELECT 1 FROM public.catalog_edge ce
            WHERE ce.tenant_id = v_tenant AND ce.edge_type_id = v_edge_type
              AND ce.properties->>'source_bo_id' = v_to::text AND ce.properties->>'source_column' = v_rel.fk_col
        ) THEN
            INSERT INTO public.catalog_edge
                (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
            SELECT gen_random_uuid(), v_tenant, v_ds::text, ct.id, pt.id,
                   jsonb_build_object('source_bo_id', v_to, 'target_bo_id', v_from,
                                      'source_column', v_rel.fk_col, 'target_column', 'id',
                                      'relationship_type', 'belongs_to'),
                   v_edge_type, NOW(), NOW()
            FROM public.catalog_node ct, public.catalog_node pt
            WHERE ct.tenant_id = v_tenant AND ct.qualified_path = '/mdm/' || v_rel.child_bo
              AND pt.tenant_id = v_tenant AND pt.qualified_path = '/mdm/' || v_rel.parent_bo;
        END IF;
    END LOOP;

    -- portfolio_account -> orm account (existing core BO). One direction only, so
    -- the core Account BO's own relationship surface is left untouched.
    SELECT id INTO v_from FROM public.business_objects WHERE tenant_id = v_tenant AND bo_key = 'portfolio_account';
    SELECT id INTO v_to   FROM public.business_objects WHERE tenant_id = v_tenant AND bo_key = 'account';
    IF v_from IS NOT NULL AND v_to IS NOT NULL THEN
        INSERT INTO public.business_object_relationships
            (tenant_id, from_bo_id, to_bo_id, rel_key, rel_name, cardinality, join_type, relationship_basis_node_id, is_active)
        VALUES (v_tenant, v_from, v_to, 'account', 'Account', 'M:1', 'LEFT',
                (SELECT id FROM public.catalog_node WHERE tenant_id = v_tenant AND qualified_path = '/mdm/portfolio_account/account_id'),
                true)
        ON CONFLICT DO NOTHING;

        IF v_edge_type IS NOT NULL AND NOT EXISTS (
            SELECT 1 FROM public.catalog_edge ce
            WHERE ce.tenant_id = v_tenant AND ce.edge_type_id = v_edge_type
              AND ce.properties->>'source_bo_id' = v_from::text AND ce.properties->>'source_column' = 'account_id'
        ) THEN
            INSERT INTO public.catalog_edge
                (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
            SELECT gen_random_uuid(), v_tenant, v_ds::text, ct.id, pt.id,
                   jsonb_build_object('source_bo_id', v_from, 'target_bo_id', v_to,
                                      'source_column', 'account_id', 'target_column', 'id',
                                      'relationship_type', 'belongs_to'),
                   v_edge_type, NOW(), NOW()
            FROM public.catalog_node ct, public.catalog_node pt
            WHERE ct.tenant_id = v_tenant AND ct.qualified_path = '/mdm/portfolio_account'
              AND pt.tenant_id = v_tenant AND pt.qualified_path = '/orm/account';
        END IF;
    END IF;

    -- The runner reuses one pooled connection, so do not leave temp tables behind.
    DROP TABLE IF EXISTS _mdm_bo;
END
$$;

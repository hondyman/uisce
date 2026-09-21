-- 20261023_002_seed_mdm_tier2_business_objects.up.sql
--
-- Tier 2 MDM business objects: 7 rule/config tables, 3 source-mapping tables and 7
-- stewardship workflow queues (17 BOs), with bindings, fields and relationships.
--
-- Depends on 20261022_001 (tier 1) and 20261023_001 (issuer, benchmark). Same conventions:
-- gold-copy tenant, is_core, model_id = id, ids resolved from catalog_node by qualified_path,
-- idempotent (ON CONFLICT DO NOTHING), skips with a NOTICE when the gold-copy tenant or the
-- /mdm catalog nodes are absent, and relationships whose BOs are missing are skipped.
--
-- Business keys: the *_cd / name column where the table has one (rule_cd, rule_name,
-- attribute_name, vendor_field, id_type, vendor_type_cd, request_ref); id for the queues.
-- Two tables key on expression indexes (issuer_survivorship_rule, issuer_type_mapping);
-- their business key is the leading natural column. validity ranges: issuer_field_mapping
-- (valid_from/valid_to) and issuer_identifier_authority (effective_from/effective_to).
--
-- Deferred (tier 3): xref, entity_relationship, attribute_value (polymorphic
-- entity_type + entity_id). Not BOs: the append-only logs and outputs.

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
    v_child_path  text;
    v_parent_path text;
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN
        RAISE NOTICE 'mdm tier-2 seed: no gold-copy tenant, skipping';
        RETURN;
    END IF;

    SELECT tenant_datasource_id INTO v_ds
    FROM public.catalog_node
    WHERE tenant_id = v_tenant AND qualified_path = '/mdm/match_rule'
    LIMIT 1;
    IF v_ds IS NULL THEN
        RAISE NOTICE 'mdm tier-2 seed: /mdm/match_rule not cataloged, skipping';
        RETURN;
    END IF;

    INSERT INTO public.physical_backend
        (backend_id, backend_name, description, storage_tier, dialect_name, driver_class, is_system)
    SELECT tpd.id, tpd.source_name, 'Auto-registered backend', 'oltp', 'postgres', '*sql.DB', false
    FROM public.tenant_product_datasource tpd
    WHERE tpd.id = v_ds
    ON CONFLICT (backend_id) DO NOTHING;

    CREATE TEMP TABLE _mdm_bo (
        bo_key text, bo_name text, bo_type text, tbl text, bk_col text, descr text,
        temporal_mode text, valid_from_col text, valid_to_col text
    );

    INSERT INTO _mdm_bo VALUES
      ('match_rule', 'Match Rule', 'REFERENCE', 'match_rule', 'rule_name', 'Configuration of an entity match rule (algorithm, fields, thresholds).', 'NONE', NULL, NULL),
      ('survivorship_rule', 'Survivorship Rule', 'REFERENCE', 'survivorship_rule', 'attribute_name', 'Per-attribute survivorship strategy and source priority.', 'NONE', NULL, NULL),
      ('dq_rule', 'DQ Rule', 'REFERENCE', 'dq_rule', 'rule_name', 'Data-quality check definition.', 'NONE', NULL, NULL),
      ('issuer_match_rule', 'Issuer Match Rule', 'REFERENCE', 'issuer_match_rule', 'rule_cd', 'Issuer matching rule with deterministic and fuzzy keys and thresholds.', 'NONE', NULL, NULL),
      ('issuer_survivorship_rule', 'Issuer Survivorship Rule', 'REFERENCE', 'issuer_survivorship_rule', 'field_name', 'Issuer field survivorship strategy by issuer type and field group.', 'NONE', NULL, NULL),
      ('issuer_hierarchy_rule', 'Issuer Hierarchy Rule', 'REFERENCE', 'issuer_hierarchy_rule', 'rule_cd', 'Validation rule for issuer hierarchies.', 'NONE', NULL, NULL),
      ('attribute_def', 'Attribute Definition', 'REFERENCE', 'attribute_def', 'attribute_name', 'Definition of a custom (EAV) attribute for an entity type.', 'NONE', NULL, NULL),
      ('issuer_field_mapping', 'Issuer Field Mapping', 'REFERENCE', 'issuer_field_mapping', 'vendor_field', 'Mapping of a vendor issuer field to an internal field.', 'VALID_TIME', 'valid_from', 'valid_to'),
      ('issuer_identifier_authority', 'Issuer Identifier Authority', 'REFERENCE', 'issuer_identifier_authority', 'id_type', 'Which source is authoritative for an issuer identifier type.', 'VALID_TIME', 'effective_from', 'effective_to'),
      ('issuer_type_mapping', 'Issuer Type Mapping', 'REFERENCE', 'issuer_type_mapping', 'vendor_type_cd', 'Mapping of a vendor issuer type to the internal issuer type.', 'NONE', NULL, NULL),
      ('match_candidate', 'Match Candidate', 'ENTITY', 'match_candidate', 'id', 'Candidate duplicate pair awaiting steward review.', 'NONE', NULL, NULL),
      ('issuer_match_candidate', 'Issuer Match Candidate', 'ENTITY', 'issuer_match_candidate', 'id', 'Candidate duplicate issuer pair awaiting review.', 'NONE', NULL, NULL),
      ('change_request', 'Change Request', 'ENTITY', 'change_request', 'id', 'Proposed change to a master record awaiting approval.', 'NONE', NULL, NULL),
      ('issuer_change_request', 'Issuer Change Request', 'ENTITY', 'issuer_change_request', 'request_ref', 'Proposed change to an issuer awaiting approval.', 'NONE', NULL, NULL),
      ('dq_issue', 'DQ Issue', 'ENTITY', 'dq_issue', 'id', 'Data-quality issue raised against a record.', 'NONE', NULL, NULL),
      ('issuer_exception', 'Issuer Exception', 'ENTITY', 'issuer_exception', 'id', 'Exception detected on an issuer or identifier.', 'NONE', NULL, NULL),
      ('issuer_hierarchy_review', 'Issuer Hierarchy Review', 'ENTITY', 'issuer_hierarchy_review', 'id', 'Proposed issuer hierarchy change awaiting review.', 'NONE', NULL, NULL);

    FOR v_bo IN SELECT * FROM _mdm_bo LOOP
        SELECT id INTO v_table FROM public.catalog_node
        WHERE tenant_id = v_tenant AND qualified_path = '/mdm/' || v_bo.tbl;
        IF v_table IS NULL THEN
            RAISE NOTICE 'mdm tier-2 seed: table node /mdm/% missing, skipping %', v_bo.tbl, v_bo.bo_key;
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
        RAISE NOTICE 'mdm tier-2 seed: % -> % fields inserted', v_bo.bo_key, v_n;
    END LOOP;

    -- ------------------------------------------------------------------
    -- Relationships (declared FKs). Table paths come from each BO's driver_table_name so the
    -- parent may live in another schema (orm.issuer). A parent with two FKs from one child
    -- (issuer_match_candidate a/b, issuer_hierarchy_review parent/child) gets one pair each.
    -- (parent, child, child FK column, parent rel_key, child rel_key)
    -- ------------------------------------------------------------------
    SELECT id INTO v_edge_type FROM public.catalog_edge_type
    WHERE tenant_id::uuid = v_tenant AND edge_type_name = 'BO_RELATIONSHIP' LIMIT 1;

    FOR v_rel IN
        SELECT * FROM (VALUES
          ('match_rule',        'match_candidate',         'match_rule_id',      'candidates',            'match_rule'),
          ('issuer_match_rule', 'issuer_match_candidate',  'match_rule_id',      'candidates',            'match_rule'),
          ('dq_rule',           'dq_issue',                'rule_id',            'issues',                'rule'),
          ('source_system',     'issuer_field_mapping',    'source_system_id',   'field_mappings',        'source_system'),
          ('source_system',     'issuer_identifier_authority', 'source_system_id', 'identifier_authorities', 'source_system'),
          ('source_system',     'issuer_type_mapping',     'source_system_id',   'type_mappings',         'source_system'),
          ('source_system',     'issuer_exception',        'source_system_id',   'exceptions',            'source_system'),
          ('issuer',            'issuer_change_request',   'issuer_id',          'change_requests',       'issuer'),
          ('issuer',            'issuer_exception',        'issuer_id',          'exceptions',            'issuer'),
          ('issuer',            'issuer_match_candidate',  'issuer_id_a',        'candidates_as_a',       'issuer_a'),
          ('issuer',            'issuer_match_candidate',  'issuer_id_b',        'candidates_as_b',       'issuer_b'),
          ('issuer',            'issuer_hierarchy_review', 'parent_issuer_id',   'reviews_as_parent',     'parent_issuer'),
          ('issuer',            'issuer_hierarchy_review', 'child_issuer_id',    'reviews_as_child',      'child_issuer')
        ) AS r(parent_bo, child_bo, fk_col, parent_key, child_key)
    LOOP
        SELECT id, driver_table_name INTO v_from, v_parent_path FROM public.business_objects
         WHERE tenant_id = v_tenant AND bo_key = v_rel.parent_bo;
        SELECT id, driver_table_name INTO v_to, v_child_path FROM public.business_objects
         WHERE tenant_id = v_tenant AND bo_key = v_rel.child_bo;
        IF v_from IS NULL OR v_to IS NULL THEN CONTINUE; END IF;

        SELECT id INTO v_basis FROM public.catalog_node
        WHERE tenant_id = v_tenant AND qualified_path = v_child_path || '/' || v_rel.fk_col;

        INSERT INTO public.business_object_relationships
            (tenant_id, from_bo_id, to_bo_id, rel_key, rel_name, cardinality, join_type, relationship_basis_node_id, is_active)
        VALUES
            (v_tenant, v_from, v_to, v_rel.parent_key, initcap(replace(v_rel.parent_key, '_', ' ')), '1:M', 'LEFT', v_basis, true),
            (v_tenant, v_to, v_from, v_rel.child_key,  initcap(replace(v_rel.child_key,  '_', ' ')), 'M:1', 'LEFT', v_basis, true)
        ON CONFLICT DO NOTHING;

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
            WHERE ct.tenant_id = v_tenant AND ct.qualified_path = v_child_path
              AND pt.tenant_id = v_tenant AND pt.qualified_path = v_parent_path;
        END IF;
    END LOOP;

    DROP TABLE IF EXISTS _mdm_bo;
END
$$;

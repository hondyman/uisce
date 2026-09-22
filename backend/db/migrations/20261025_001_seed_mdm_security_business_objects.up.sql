-- 20261025_001_seed_mdm_security_business_objects.up.sql
--
-- Security MDM business objects: 30 BOs over the mdm.security_* tables and the shared vocabularies
-- (rating scale, day-count / business-day conventions, payment frequency, classification schemes), each with a
-- binding on the CRIMS datasource, fields named by the column's semantic term, and relationships.
--
-- Depends on the security MDM tables (backend/db/crims/0001_mdm_security.sql), a catalog scan of CRIMS, and the
-- semantic terms for those columns. Same conventions as the tier 1-3 seeds: gold-copy tenant, is_core,
-- model_id = id, ids resolved from catalog_node by qualified_path, idempotent (ON CONFLICT DO NOTHING), skips
-- with a NOTICE when the gold-copy tenant or the /mdm catalog nodes are absent, and relationships whose BOs are
-- missing are skipped.
--
-- The security master itself is the existing `security` BO (orm.security); the golden record for it is
-- security_golden_record. Relationships from `security` to the security_id columns are declared by convention:
-- security_id has no foreign key (as with the issuer golden-record tables).
--
-- Not BOs: append-only logs, run records and outputs (security_feed_health, security_survivorship_log,
-- security_merge_log, security_split_log, security_golden_publication, security_golden_distribution,
-- security_term_extraction_job, security_reconciliation).

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
        RAISE NOTICE 'mdm security seed: no gold-copy tenant, skipping';
        RETURN;
    END IF;

    SELECT tenant_datasource_id INTO v_ds
    FROM public.catalog_node
    WHERE tenant_id = v_tenant AND qualified_path = '/mdm/security_golden_record'
    LIMIT 1;
    IF v_ds IS NULL THEN
        RAISE NOTICE 'mdm security seed: /mdm/security_golden_record not cataloged, skipping';
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
      ('security_asset_class', 'Security Asset Class', 'REFERENCE', 'security_asset_class', 'asset_class_cd', 'Asset class of a security (equity, fixed income, derivative, fund, ...).', 'NONE', NULL, NULL),
      ('security_type', 'Security Type', 'REFERENCE', 'security_type', 'sec_typ_cd', 'Security type and sub-type, with the detail table and required attributes for each.', 'NONE', NULL, NULL),
      ('security_type_mapping', 'Security Type Mapping', 'REFERENCE', 'security_type_mapping', 'vendor_type_cd', 'Mapping of a vendor security type to the internal asset class, type and sub-type.', 'NONE', NULL, NULL),
      ('classification_scheme', 'Classification Scheme', 'REFERENCE', 'classification_scheme', 'scheme_cd', 'Industry classification scheme (GICS, ICB, NAICS, ...) and its depth.', 'NONE', NULL, NULL),
      ('classification_scheme_map', 'Classification Scheme Map', 'REFERENCE', 'classification_scheme_map', 'id', 'Mapping of a code in one classification scheme to another.', 'NONE', NULL, NULL),
      ('rating_scale', 'Rating Scale', 'REFERENCE', 'rating_scale', 'rating_value', 'Agency rating value with its numeric equivalent and investment-grade flag.', 'NONE', NULL, NULL),
      ('day_count_convention', 'Day Count Convention', 'REFERENCE', 'day_count_convention', 'convention_cd', 'Day-count convention and its formula.', 'NONE', NULL, NULL),
      ('business_day_convention', 'Business Day Convention', 'REFERENCE', 'business_day_convention', 'convention_cd', 'Business-day adjustment convention.', 'NONE', NULL, NULL),
      ('payment_frequency', 'Payment Frequency', 'REFERENCE', 'payment_frequency', 'frequency_cd', 'Payment frequency and periods per year.', 'NONE', NULL, NULL),
      ('security_source_priority', 'Security Source Priority', 'REFERENCE', 'security_source_priority', 'field_group', 'Which source wins for which field group, per asset class and sub-type.', 'NONE', NULL, NULL),
      ('security_feed_schedule', 'Security Feed Schedule', 'REFERENCE', 'security_feed_schedule', 'feed_name', 'Expected delivery cadence and SLA of a security data feed.', 'NONE', NULL, NULL),
      ('security_asset_class_routing', 'Security Asset Class Routing', 'REFERENCE', 'security_asset_class_routing', 'sec_typ_cd', 'Which operational detail and satellite tables hold each security type.', 'NONE', NULL, NULL),
      ('security_identifier_authority', 'Security Identifier Authority', 'REFERENCE', 'security_identifier_authority', 'id_type', 'Which source is authoritative for a security identifier type.', 'VALID_TIME', 'effective_from', 'effective_to'),
      ('security_field_mapping', 'Security Field Mapping', 'REFERENCE', 'security_field_mapping', 'vendor_field', 'Mapping of a vendor security field to an internal table and field.', 'VALID_TIME', 'valid_from', 'valid_to'),
      ('security_field_transform', 'Security Field Transform', 'REFERENCE', 'security_field_transform', 'transform_cd', 'Reusable transformation applied when mapping vendor fields.', 'NONE', NULL, NULL),
      ('security_survivorship_rule', 'Security Survivorship Rule', 'REFERENCE', 'security_survivorship_rule', 'field_name', 'Per-field survivorship strategy and source priority for securities.', 'NONE', NULL, NULL),
      ('security_match_rule', 'Security Match Rule', 'REFERENCE', 'security_match_rule', 'rule_cd', 'Security matching rule with deterministic and fuzzy keys and thresholds.', 'NONE', NULL, NULL),
      ('security_status_authority', 'Security Status Authority', 'REFERENCE', 'security_status_authority', 'status_field', 'Which source is authoritative for a security status field.', 'NONE', NULL, NULL),
      ('security_identifier_issuance', 'Security Identifier Issuance', 'ENTITY', 'security_identifier_issuance', 'id_value', 'An identifier issued for a security, with the source and validation.', 'NONE', NULL, NULL),
      ('security_identifier_conflict', 'Security Identifier Conflict', 'ENTITY', 'security_identifier_conflict', 'id', 'Two sources disagree on an identifier; awaiting resolution.', 'NONE', NULL, NULL),
      ('security_match_candidate', 'Security Match Candidate', 'ENTITY', 'security_match_candidate', 'id', 'Candidate duplicate security pair awaiting steward review.', 'NONE', NULL, NULL),
      ('security_golden_record', 'Security Golden Record', 'ENTITY', 'security_golden_record', 'security_id', 'Versioned golden record of a security with its winning sources.', 'NONE', NULL, NULL),
      ('security_golden_field', 'Security Golden Field', 'ENTITY', 'security_golden_field', 'field_name', 'One field of a security golden record and where its value came from.', 'NONE', NULL, NULL),
      ('security_term_sheet', 'Security Term Sheet', 'ENTITY', 'security_term_sheet', 'document_name', 'Term sheet or prospectus document for a security and its extraction status.', 'NONE', NULL, NULL),
      ('security_term_extraction_field', 'Security Term Extraction Field', 'ENTITY', 'security_term_extraction_field', 'field_name', 'A value extracted from a term sheet, awaiting validation.', 'NONE', NULL, NULL),
      ('security_reconciliation_result', 'Security Reconciliation Result', 'ENTITY', 'security_reconciliation_result', 'id', 'A field difference between two sources found by reconciliation.', 'NONE', NULL, NULL),
      ('security_exception', 'Security Exception', 'ENTITY', 'security_exception', 'id', 'Exception detected on a security or identifier.', 'NONE', NULL, NULL),
      ('security_steward', 'Security Steward', 'ENTITY', 'security_steward', 'user_id', 'A steward and the asset classes and rights they hold.', 'NONE', NULL, NULL),
      ('security_change_request', 'Security Change Request', 'ENTITY', 'security_change_request', 'request_ref', 'Proposed change to a security awaiting approval.', 'NONE', NULL, NULL),
      ('security_ca_linkage', 'Security Corporate Action Linkage', 'ENTITY', 'security_ca_linkage', 'id', 'Link between a security and a corporate action.', 'NONE', NULL, NULL);

    FOR v_bo IN SELECT * FROM _mdm_bo LOOP
        SELECT id INTO v_table FROM public.catalog_node
        WHERE tenant_id = v_tenant AND qualified_path = '/mdm/' || v_bo.tbl;
        IF v_table IS NULL THEN
            RAISE NOTICE 'mdm security seed: table node /mdm/% missing, skipping %', v_bo.tbl, v_bo.bo_key;
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
            (tenant_id, bo_id, backend_id, driving_node_id, binding_name, is_core, is_active, is_default,
             temporal_mode, valid_from_column_node_id, valid_to_column_node_id)
        VALUES (
            v_tenant, v_bo_id, v_ds, v_table, v_bo.bo_name || ' Binding', true, true, true,
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
        RAISE NOTICE 'mdm security seed: % -> % fields inserted', v_bo.bo_key, v_n;
    END LOOP;

    -- ------------------------------------------------------------------
    -- Relationships (declared FKs). Table paths come from each BO's driver_table_name so the
    -- parent may live in another schema (orm.issuer). A parent with two FKs from one child
    -- (issuer_match_candidate a/b, issuer_hierarchy_review parent/child) gets one pair each.
    -- (parent, child, child FK column, parent rel_key, child rel_key). Links to `security` are by convention.
    -- ------------------------------------------------------------------
    SELECT id INTO v_edge_type FROM public.catalog_edge_type
    WHERE tenant_id::uuid = v_tenant AND edge_type_name = 'BO_RELATIONSHIP' LIMIT 1;

    FOR v_rel IN
        SELECT * FROM (VALUES
          ('source_system', 'security_type_mapping', 'source_system_id', 'security_type_mappings', 'source_system'),
          ('source_system', 'security_source_priority', 'source_system_id', 'security_source_priorities', 'source_system'),
          ('source_system', 'security_feed_schedule', 'source_system_id', 'security_feed_schedules', 'source_system'),
          ('source_system', 'security_identifier_authority', 'source_system_id', 'security_identifier_authorities', 'source_system'),
          ('source_system', 'security_identifier_issuance', 'source_system_id', 'security_identifier_issuances', 'source_system'),
          ('source_system', 'security_identifier_conflict', 'source_system_id_a', 'security_identifier_conflicts_as_a', 'source_system_a'),
          ('source_system', 'security_identifier_conflict', 'source_system_id_b', 'security_identifier_conflicts_as_b', 'source_system_b'),
          ('source_system', 'security_field_mapping', 'source_system_id', 'security_field_mappings', 'source_system'),
          ('source_system', 'security_golden_field', 'source_system_id', 'security_golden_fields', 'source_system'),
          ('source_system', 'security_status_authority', 'source_system_id', 'security_status_authorities', 'source_system'),
          ('source_system', 'security_exception', 'source_system_id', 'security_exceptions', 'source_system'),
          ('security_match_rule', 'security_match_candidate', 'match_rule_id', 'candidates', 'match_rule'),
          ('security_golden_record', 'security_golden_field', 'golden_record_id', 'fields', 'golden_record'),
          ('security', 'security_identifier_issuance', 'security_id', 'security_identifier_issuances', 'security'),
          ('security', 'security_identifier_conflict', 'security_id_a', 'security_identifier_conflicts_as_a', 'security_a'),
          ('security', 'security_identifier_conflict', 'security_id_b', 'security_identifier_conflicts_as_b', 'security_b'),
          ('security', 'security_match_candidate', 'security_id_a', 'security_match_candidates_as_a', 'security_a'),
          ('security', 'security_match_candidate', 'security_id_b', 'security_match_candidates_as_b', 'security_b'),
          ('security', 'security_golden_record', 'security_id', 'security_golden_records', 'security'),
          ('security', 'security_term_sheet', 'security_id', 'security_term_sheets', 'security'),
          ('security', 'security_exception', 'security_id', 'security_exceptions', 'security'),
          ('security', 'security_change_request', 'security_id', 'security_change_requests', 'security'),
          ('security', 'security_ca_linkage', 'security_id', 'security_ca_linkages', 'security'),
          ('security', 'security_reconciliation_result', 'security_id', 'security_reconciliation_results', 'security')
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

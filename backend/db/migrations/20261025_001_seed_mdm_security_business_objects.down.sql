-- Reverses 20261025_001_seed_mdm_security_business_objects.up.sql.
-- Removes only the 30 security MDM BOs (by bo_key) in the gold-copy tenant, their bindings, fields and
-- relationships, and the BO_RELATIONSHIP edges that reference them.

DO $$
DECLARE
    v_tenant uuid;
    v_ids    uuid[];
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN RETURN; END IF;

    SELECT array_agg(id) INTO v_ids FROM public.business_objects
    WHERE tenant_id = v_tenant AND bo_key IN (
        'security_asset_class',
        'security_type',
        'security_type_mapping',
        'classification_scheme',
        'classification_scheme_map',
        'rating_scale',
        'day_count_convention',
        'business_day_convention',
        'payment_frequency',
        'security_source_priority',
        'security_feed_schedule',
        'security_asset_class_routing',
        'security_identifier_authority',
        'security_field_mapping',
        'security_field_transform',
        'security_survivorship_rule',
        'security_match_rule',
        'security_status_authority',
        'security_identifier_issuance',
        'security_identifier_conflict',
        'security_match_candidate',
        'security_golden_record',
        'security_golden_field',
        'security_term_sheet',
        'security_term_extraction_field',
        'security_reconciliation_result',
        'security_exception',
        'security_steward',
        'security_change_request',
        'security_ca_linkage') AND is_core = true;
    IF v_ids IS NULL THEN RETURN; END IF;

    DELETE FROM public.catalog_edge ce
    USING public.catalog_edge_type et
    WHERE et.id = ce.edge_type_id AND et.edge_type_name = 'BO_RELATIONSHIP'
      AND ce.tenant_id = v_tenant
      AND ((ce.properties->>'source_bo_id')::uuid = ANY (v_ids)
           OR (ce.properties->>'target_bo_id')::uuid = ANY (v_ids));

    DELETE FROM public.business_object_binding WHERE tenant_id = v_tenant AND bo_id = ANY (v_ids);
    DELETE FROM public.business_object_relationships WHERE tenant_id = v_tenant AND (from_bo_id = ANY (v_ids) OR to_bo_id = ANY (v_ids));
    DELETE FROM public.business_object_fields WHERE tenant_id = v_tenant AND bo_id = ANY (v_ids);
    DELETE FROM public.business_objects WHERE tenant_id = v_tenant AND id = ANY (v_ids);
END
$$;

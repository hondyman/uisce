-- Reverses 20261023_002_seed_mdm_tier2_business_objects.up.sql.
-- Removes only the 17 tier-2 MDM BOs (by bo_key) in the gold-copy tenant, and the
-- BO_RELATIONSHIP edges that reference them.

DO $$
DECLARE
    v_tenant uuid;
    v_ids    uuid[];
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN RETURN; END IF;

    SELECT array_agg(id) INTO v_ids FROM public.business_objects
    WHERE tenant_id = v_tenant AND bo_key IN (
        'match_rule',
        'survivorship_rule',
        'dq_rule',
        'issuer_match_rule',
        'issuer_survivorship_rule',
        'issuer_hierarchy_rule',
        'attribute_def',
        'issuer_field_mapping',
        'issuer_identifier_authority',
        'issuer_type_mapping',
        'match_candidate',
        'issuer_match_candidate',
        'change_request',
        'issuer_change_request',
        'dq_issue',
        'issuer_exception',
        'issuer_hierarchy_review') AND is_core = true;
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

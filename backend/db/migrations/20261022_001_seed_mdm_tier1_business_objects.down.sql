-- Reverses 20261022_001_seed_mdm_tier1_business_objects.up.sql.
-- Removes only the 14 tier-1 MDM BOs (by bo_key) in the gold-copy tenant, and
-- the BO_RELATIONSHIP edges that reference them. The physical_backend row for
-- the CRIMS datasource is left in place (shared infrastructure registry).

DO $$
DECLARE
    v_tenant uuid;
    v_ids    uuid[];
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN RETURN; END IF;

    SELECT array_agg(id) INTO v_ids FROM public.business_objects
    WHERE tenant_id = v_tenant AND bo_key IN (
        'party', 'portfolio', 'mandate', 'portfolio_composite', 'source_system', 'steward',
        'issuer_steward', 'hierarchy', 'issuer_golden_record', 'portfolio_account',
        'mandate_restriction', 'portfolio_target', 'hierarchy_closure', 'issuer_golden_field')
      AND is_core = true;
    IF v_ids IS NULL THEN RETURN; END IF;

    DELETE FROM public.catalog_edge ce
    USING public.catalog_edge_type et
    WHERE et.id = ce.edge_type_id AND et.edge_type_name = 'BO_RELATIONSHIP'
      AND ce.tenant_id = v_tenant
      AND (ce.properties->>'source_bo_id')::uuid = ANY (v_ids);

    DELETE FROM public.business_object_binding WHERE tenant_id = v_tenant AND bo_id = ANY (v_ids);
    DELETE FROM public.business_object_relationships WHERE tenant_id = v_tenant AND (from_bo_id = ANY (v_ids) OR to_bo_id = ANY (v_ids));
    DELETE FROM public.business_object_fields WHERE tenant_id = v_tenant AND bo_id = ANY (v_ids);
    DELETE FROM public.business_objects WHERE tenant_id = v_tenant AND id = ANY (v_ids);
END
$$;

-- Reverses 20261023_001_seed_orm_issuer_benchmark_business_objects.up.sql.
-- Removes only the issuer and benchmark BOs (by bo_key) in the gold-copy tenant, and the
-- BO_RELATIONSHIP edges that reference them.

DO $$
DECLARE
    v_tenant uuid;
    v_ids    uuid[];
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN RETURN; END IF;

    SELECT array_agg(id) INTO v_ids FROM public.business_objects
    WHERE tenant_id = v_tenant AND bo_key IN ('issuer', 'benchmark') AND is_core = true;
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

-- Reverses 20261101_001_price_business_object.up.sql: the Price BO, its binding and fields, the MAPS_TO
-- edges onto mdm.price_golden_record and the price terms it added (reused terms stay), messages 9400-37..40.
DO $$
DECLARE
    v_tenant uuid;
    v_bo_id  uuid := md5('mdm-bo:price')::uuid;
BEGIN
    SELECT id INTO v_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_tenant IS NULL THEN RETURN; END IF;
    DELETE FROM public.business_object_fields WHERE bo_id = v_bo_id;
    DELETE FROM public.business_object_binding WHERE bo_id = v_bo_id;
    DELETE FROM public.business_objects WHERE id = v_bo_id;
    DELETE FROM public.catalog_edge e
     USING public.catalog_node col
     WHERE e.target_node_id = col.id AND col.tenant_id = v_tenant
       AND col.qualified_path LIKE '/mdm/price_golden_record/%'
       AND e.edge_type_id IN (SELECT id FROM public.catalog_edge_type WHERE edge_type_name = 'MAPS_TO');
    DELETE FROM public.catalog_node
     WHERE tenant_id = v_tenant AND id IN (SELECT md5('semantic-term:' || t)::uuid FROM unnest(ARRAY[
        'PriceEntityId', 'PriceEntityType', 'PriceTypeCd', 'Price', 'PriceTime', 'IsOfficialPrice', 'IsStalePrice',
        'SourceCount', 'VariancePct']) AS t);
END
$$;
DELETE FROM public.message_catalog WHERE set_nbr = 9400 AND message_nbr BETWEEN 37 AND 40;

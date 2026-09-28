DO $down$
DECLARE
    _tenant uuid;
BEGIN
    SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy = true LIMIT 1;
    IF _tenant IS NOT NULL THEN
        PERFORM set_config('app.current_tenant', _tenant::text, true);
        DELETE FROM public.catalog_edge e
        USING public.catalog_node n
        WHERE (e.source_node_id=n.id OR e.target_node_id=n.id)
          AND n.properties->>'source' IN ('security_catalog_graph','security_seed')
          AND n.properties->>'entity_type'='SECURITY';
        DELETE FROM public.catalog_node
        WHERE properties->>'source' IN ('security_catalog_graph','security_seed')
          AND properties->>'entity_type'='SECURITY';
        DELETE FROM public.catalog_node
        WHERE qualified_path IN ('business_object/security')
          AND properties->>'source'='security_catalog_graph';
    END IF;
END
$down$;

DELETE FROM public.semantic_survivorship_rules WHERE entity_type='SECURITY';
DELETE FROM public.attribute_def WHERE entity_type='SECURITY';

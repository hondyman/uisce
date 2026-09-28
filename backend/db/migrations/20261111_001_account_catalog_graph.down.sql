-- Remove Account catalog graph edges and nodes created by the up migration.
-- Leaves semantic/account/* terms (shared with survivorship) in place.

DO $down$
DECLARE
    _tenant uuid;
BEGIN
    SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        RETURN;
    END IF;
    PERFORM set_config('app.current_tenant', _tenant::text, true);
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);

    DELETE FROM public.catalog_edge e
    USING public.catalog_node n
    WHERE e.target_node_id = n.id
      AND n.properties->>'source' = 'account_catalog_graph';

    DELETE FROM public.catalog_edge e
    USING public.catalog_node n
    WHERE e.source_node_id = n.id
      AND n.properties->>'source' = 'account_catalog_graph';

    DELETE FROM public.catalog_node
    WHERE properties->>'source' = 'account_catalog_graph';
END
$down$;

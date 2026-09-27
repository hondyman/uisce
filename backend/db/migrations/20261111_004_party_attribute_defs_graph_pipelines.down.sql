DO $down$
DECLARE _tenant uuid;
BEGIN
  SELECT t.id INTO _tenant FROM public.tenants t WHERE t.gold_copy LIMIT 1;
  IF _tenant IS NOT NULL THEN
    PERFORM set_config('app.current_tenant', _tenant::text, true);
    DELETE FROM public.catalog_edge e USING public.catalog_node n
    WHERE (e.source_node_id=n.id OR e.target_node_id=n.id)
      AND n.properties->>'entity_type'='PARTY'
      AND n.properties->>'source' IN ('party_catalog_graph','party_seed');
    DELETE FROM public.catalog_node WHERE properties->>'entity_type'='PARTY'
      AND properties->>'source' IN ('party_catalog_graph','party_seed');
    DELETE FROM public.catalog_node WHERE qualified_path='business_object/party'
      AND properties->>'source'='party_catalog_graph';
  END IF;
END $down$;
DELETE FROM public.data_pipeline_definitions WHERE created_by='seed:party-mdm';
DELETE FROM public.staging_bindings sb USING public.business_objects bo
 WHERE sb.bo_id=bo.id AND bo.bo_key='party' AND sb.staging_table='staging.party_data';
DELETE FROM public.semantic_survivorship_rules WHERE entity_type='PARTY';
DELETE FROM public.attribute_def WHERE entity_type='PARTY';

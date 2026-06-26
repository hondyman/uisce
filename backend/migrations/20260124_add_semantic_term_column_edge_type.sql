-- Add edge type for connecting semantic terms to database columns  
-- This allows semantic terms to provide context to database columns

DO $do$
DECLARE
  _tenant uuid;
  _src uuid;
  _tgt uuid;
BEGIN
  SELECT id INTO _tenant FROM tenants LIMIT 1;
  SELECT id INTO _src FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
  SELECT id INTO _tgt FROM catalog_node_type WHERE catalog_type_name IN ('database_column','column') LIMIT 1;

  IF _tenant IS NOT NULL AND _src IS NOT NULL AND _tgt IS NOT NULL THEN
    INSERT INTO catalog_edge_types (id, tenant_id, edge_type_name, description, source_node_type_id, target_node_type_id, is_active)
    VALUES (
      gen_random_uuid(),
      _tenant,
      'provides_context_to',
      'Semantic term provides context to database column',
      _src,
      _tgt,
      true
    )
    ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
  ELSE
    RAISE NOTICE 'Skipping seed for catalog_edge_types.provides_context_to: tenant=% src=% tgt=%', _tenant, _src, _tgt;
  END IF;
END
$do$;

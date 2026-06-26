-- Ensure has_semantic edge type only maps business_term to semantic_model
-- Remove any incorrect has_semantic entries and recreate with correct node types

DO $do$
DECLARE
  _tenant UUID := (SELECT id FROM tenants LIMIT 1);
  _biz UUID := (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_term' LIMIT 1);
  _sem UUID := (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_model' LIMIT 1);
BEGIN
  IF _tenant IS NULL OR _biz IS NULL OR _sem IS NULL THEN
    RAISE NOTICE 'Skipping has_semantic fix: tenant=% biz=% sem=%', _tenant, _biz, _sem;
    RETURN;
  END IF;

  DELETE FROM catalog_edge_types
  WHERE tenant_id = _tenant
    AND edge_type_name = 'has_semantic'
    AND (source_node_type_id != _biz OR target_node_type_id != _sem);

  INSERT INTO catalog_edge_types (id, tenant_id, edge_type_name, description, source_node_type_id, target_node_type_id, is_active)
  VALUES (
    gen_random_uuid(),
    _tenant,
    'has_semantic',
    'Has Semantic Mapping',
    _biz,
    _sem,
    true
  )
  ON CONFLICT (tenant_id, edge_type_name) DO UPDATE
  SET source_node_type_id = EXCLUDED.source_node_type_id,
      target_node_type_id = EXCLUDED.target_node_type_id,
      is_active = true
  WHERE catalog_edge_types.source_node_type_id != EXCLUDED.source_node_type_id
     OR catalog_edge_types.target_node_type_id != EXCLUDED.target_node_type_id;
END
$do$;

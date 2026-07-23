-- Migration: 20260722_sync_bo_to_catalog.sql
-- Purpose: Sync Business Objects, BO Fields, and BO Relationships into catalog_node and catalog_edge for Impact Analysis and Glossary documentation.

BEGIN;

-- 1. Ensure catalog_node_type entries exist for business_object and bo_field
INSERT INTO catalog_node_type (id, catalog_type_name, tenant_id, description, is_active, created_at, updated_at)
VALUES 
  ('06bb774c-8666-4ab1-84eb-4f4d439ac84c', 'business_object', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'Business Object entity node', true, NOW(), NOW()),
  ('e7c7e5a8-5e43-4c91-a12d-887711223344', 'bo_field', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'Business Object field node', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET 
  catalog_type_name = EXCLUDED.catalog_type_name,
  updated_at = NOW();

-- 2. Ensure catalog_edge_type entries exist for HAS_FIELD, BACKED_BY_TERM, USES_SEMANTIC_TERM, BO_RELATIONSHIP
INSERT INTO catalog_edge_type (id, edge_type_name, source_node_type_id, target_node_type_id, tenant_id, description, is_active, created_at, updated_at)
VALUES
  ('88888888-1111-4444-8888-000000000001', 'HAS_FIELD', '06bb774c-8666-4ab1-84eb-4f4d439ac84c', 'e7c7e5a8-5e43-4c91-a12d-887711223344', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'Business Object owns Field', true, NOW(), NOW()),
  ('88888888-1111-4444-8888-000000000002', 'BACKED_BY_TERM', 'e7c7e5a8-5e43-4c91-a12d-887711223344', '820b942a-9c9e-4abc-acdc-84616db33098', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'BO Field backed by Semantic Term', true, NOW(), NOW()),
  ('88888888-1111-4444-8888-000000000003', 'USES_SEMANTIC_TERM', '06bb774c-8666-4ab1-84eb-4f4d439ac84c', '820b942a-9c9e-4abc-acdc-84616db33098', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'Business Object uses Semantic Term', true, NOW(), NOW()),
  ('d5fd8908-96ad-4ac5-b2e0-f86bc666f6bd', 'BO_RELATIONSHIP', '06bb774c-8666-4ab1-84eb-4f4d439ac84c', '06bb774c-8666-4ab1-84eb-4f4d439ac84c', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'Relationship between Business Objects', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET 
  edge_type_name = EXCLUDED.edge_type_name,
  updated_at = NOW();

-- 3. Function to sync Business Object inserts/updates/deletes to catalog_node
CREATE OR REPLACE FUNCTION fn_sync_business_object_catalog_node()
RETURNS TRIGGER AS $$
BEGIN
  IF (TG_OP = 'DELETE') THEN
    DELETE FROM catalog_edge WHERE source_node_id = OLD.id OR target_node_id = OLD.id;
    DELETE FROM catalog_node WHERE id = OLD.id;
    RETURN OLD;
  ELSE
    INSERT INTO catalog_node (
      id,
      node_type_id,
      node_name,
      description,
      qualified_path,
      tenant_id,
      tenant_datasource_id,
      properties,
      is_active,
      created_at,
      updated_at
    ) VALUES (
      NEW.id,
      '06bb774c-8666-4ab1-84eb-4f4d439ac84c'::uuid,
      COALESCE(NEW.display_name, NEW.name, NEW.key),
      NEW.description,
      'bo:' || NEW.key,
      NEW.tenant_id,
      NEW.tenant_datasource_id,
      jsonb_build_object(
        'key', NEW.key,
        'technical_name', NEW.technical_name,
        'is_core', NEW.is_core,
        'category', NEW.category
      ),
      COALESCE(NEW.is_active, true),
      COALESCE(NEW.created_at, NOW()),
      NOW()
    )
    ON CONFLICT (id) DO UPDATE SET
      node_name = EXCLUDED.node_name,
      description = EXCLUDED.description,
      qualified_path = EXCLUDED.qualified_path,
      tenant_id = EXCLUDED.tenant_id,
      tenant_datasource_id = EXCLUDED.tenant_datasource_id,
      properties = EXCLUDED.properties,
      is_active = EXCLUDED.is_active,
      updated_at = NOW();
    RETURN NEW;
  END IF;
END;
$$ LANGUAGE plpgsql;

-- 4. Function to sync Business Object Field inserts/updates/deletes to catalog_node & catalog_edge
CREATE OR REPLACE FUNCTION fn_sync_bo_fields_catalog_edge()
RETURNS TRIGGER AS $$
DECLARE
  v_tenant_id uuid;
  v_term_name text;
  v_bo_key text;
BEGIN
  IF (TG_OP = 'DELETE') THEN
    DELETE FROM catalog_edge WHERE source_node_id = OLD.id OR target_node_id = OLD.id;
    IF OLD.semantic_term_node_id IS NOT NULL THEN
      DELETE FROM catalog_edge WHERE source_node_id = OLD.bo_id AND target_node_id = OLD.semantic_term_node_id AND relationship_type = 'USES_SEMANTIC_TERM';
    END IF;
    DELETE FROM catalog_node WHERE id = OLD.id;
    RETURN OLD;
  ELSE
    SELECT tenant_id, key INTO v_tenant_id, v_bo_key FROM business_objects WHERE id = NEW.bo_id;
    IF NEW.semantic_term_node_id IS NOT NULL THEN
      SELECT node_name INTO v_term_name FROM catalog_node WHERE id = NEW.semantic_term_node_id;
    END IF;

    -- Upsert bo_field node in catalog_node
    INSERT INTO catalog_node (
      id,
      node_type_id,
      parent_id,
      node_name,
      qualified_path,
      tenant_id,
      properties,
      is_active,
      created_at,
      updated_at
    ) VALUES (
      NEW.id,
      'e7c7e5a8-5e43-4c91-a12d-887711223344'::uuid,
      NEW.bo_id,
      COALESCE(v_term_name, NEW.field_role, 'field'),
      'bo_field:' || COALESCE(v_bo_key, NEW.bo_id::text) || ':' || NEW.id::text,
      v_tenant_id,
      jsonb_build_object(
        'field_role', NEW.field_role,
        'binding_requirement', NEW.binding_requirement,
        'is_exposed', NEW.is_exposed,
        'semantic_term_node_id', NEW.semantic_term_node_id
      ),
      COALESCE(NEW.is_exposed, true),
      NOW(),
      NOW()
    )
    ON CONFLICT (id) DO UPDATE SET
      parent_id = EXCLUDED.parent_id,
      node_name = EXCLUDED.node_name,
      qualified_path = EXCLUDED.qualified_path,
      tenant_id = EXCLUDED.tenant_id,
      properties = EXCLUDED.properties,
      is_active = EXCLUDED.is_active,
      updated_at = NOW();

    -- Upsert edge BO -> BO Field (HAS_FIELD)
    INSERT INTO catalog_edge (
      id,
      source_node_id,
      target_node_id,
      edge_type_id,
      relationship_type,
      tenant_id,
      created_at,
      updated_at
    ) VALUES (
      pg_catalog.gen_random_uuid(),
      NEW.bo_id,
      NEW.id,
      '88888888-1111-4444-8888-000000000001'::uuid,
      'HAS_FIELD',
      v_tenant_id,
      NOW(),
      NOW()
    )
    ON CONFLICT DO NOTHING;

    -- Upsert edge BO Field -> Semantic Term (BACKED_BY_TERM) and direct BO -> Semantic Term (USES_SEMANTIC_TERM)
    IF NEW.semantic_term_node_id IS NOT NULL THEN
      INSERT INTO catalog_edge (
        id,
        source_node_id,
        target_node_id,
        edge_type_id,
        relationship_type,
        tenant_id,
        created_at,
        updated_at
      ) VALUES (
        pg_catalog.gen_random_uuid(),
        NEW.id,
        NEW.semantic_term_node_id,
        '88888888-1111-4444-8888-000000000002'::uuid,
        'BACKED_BY_TERM',
        v_tenant_id,
        NOW(),
        NOW()
      )
      ON CONFLICT DO NOTHING;

      INSERT INTO catalog_edge (
        id,
        source_node_id,
        target_node_id,
        edge_type_id,
        relationship_type,
        tenant_id,
        created_at,
        updated_at
      ) VALUES (
        pg_catalog.gen_random_uuid(),
        NEW.bo_id,
        NEW.semantic_term_node_id,
        '88888888-1111-4444-8888-000000000003'::uuid,
        'USES_SEMANTIC_TERM',
        v_tenant_id,
        NOW(),
        NOW()
      )
      ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
  END IF;
END;
$$ LANGUAGE plpgsql;

-- 5. Function to sync Business Object Relationships to catalog_edge
CREATE OR REPLACE FUNCTION fn_sync_bo_relationships_catalog_edge()
RETURNS TRIGGER AS $$
BEGIN
  IF (TG_OP = 'DELETE') THEN
    DELETE FROM catalog_edge WHERE id = OLD.id;
    RETURN OLD;
  ELSE
    INSERT INTO catalog_edge (
      id,
      source_node_id,
      target_node_id,
      edge_type_id,
      relationship_type,
      tenant_id,
      properties,
      is_active,
      created_at,
      updated_at
    ) VALUES (
      NEW.id,
      NEW.source_object_id,
      NEW.target_object_id,
      'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd'::uuid,
      'BO_RELATIONSHIP',
      NEW.tenant_id,
      jsonb_build_object(
        'cardinality', NEW.cardinality,
        'relationship_type', NEW.relationship_type,
        'description', NEW.description
      ),
      true,
      COALESCE(NEW.created_at, NOW()),
      NOW()
    )
    ON CONFLICT DO NOTHING;
    RETURN NEW;
  END IF;
END;
$$ LANGUAGE plpgsql;

-- 6. Attach Triggers
DROP TRIGGER IF EXISTS trg_sync_business_objects ON business_objects;
CREATE TRIGGER trg_sync_business_objects
  AFTER INSERT OR UPDATE OR DELETE ON business_objects
  FOR EACH ROW EXECUTE FUNCTION fn_sync_business_object_catalog_node();

DROP TRIGGER IF EXISTS trg_sync_business_object_fields ON business_object_fields;
CREATE TRIGGER trg_sync_business_object_fields
  AFTER INSERT OR UPDATE OR DELETE ON business_object_fields
  FOR EACH ROW EXECUTE FUNCTION fn_sync_bo_fields_catalog_edge();

DROP TRIGGER IF EXISTS trg_sync_business_object_relationships ON business_object_relationships;
CREATE TRIGGER trg_sync_business_object_relationships
  AFTER INSERT OR UPDATE OR DELETE ON business_object_relationships
  FOR EACH ROW EXECUTE FUNCTION fn_sync_bo_relationships_catalog_edge();

-- 7. Perform Initial Bulk Seed / Sync for existing Business Objects
INSERT INTO catalog_node (
  id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, properties, is_active, created_at, updated_at
)
SELECT 
  bo.id,
  '06bb774c-8666-4ab1-84eb-4f4d439ac84c'::uuid,
  COALESCE(bo.display_name, bo.name, bo.key),
  bo.description,
  'bo:' || bo.key,
  bo.tenant_id,
  bo.tenant_datasource_id,
  jsonb_build_object('key', bo.key, 'technical_name', bo.technical_name, 'is_core', bo.is_core, 'category', bo.category),
  COALESCE(bo.is_active, true),
  COALESCE(bo.created_at, NOW()),
  NOW()
FROM business_objects bo
ON CONFLICT (id) DO UPDATE SET
  node_name = EXCLUDED.node_name,
  description = EXCLUDED.description,
  qualified_path = EXCLUDED.qualified_path,
  tenant_id = EXCLUDED.tenant_id,
  properties = EXCLUDED.properties,
  updated_at = NOW();

-- Seed BO Fields & Edges
INSERT INTO catalog_node (
  id, node_type_id, parent_id, node_name, qualified_path, tenant_id, properties, is_active, created_at, updated_at
)
SELECT 
  bof.id,
  'e7c7e5a8-5e43-4c91-a12d-887711223344'::uuid,
  bof.bo_id,
  COALESCE(st.node_name, bof.field_role, 'field'),
  'bo_field:' || COALESCE(bo.key, bof.bo_id::text) || ':' || bof.id::text,
  bo.tenant_id,
  jsonb_build_object('field_role', bof.field_role, 'binding_requirement', bof.binding_requirement, 'is_exposed', bof.is_exposed, 'semantic_term_node_id', bof.semantic_term_node_id),
  COALESCE(bof.is_exposed, true),
  NOW(),
  NOW()
FROM business_object_fields bof
JOIN business_objects bo ON bo.id = bof.bo_id
LEFT JOIN catalog_node st ON st.id = bof.semantic_term_node_id
ON CONFLICT (id) DO UPDATE SET
  parent_id = EXCLUDED.parent_id,
  node_name = EXCLUDED.node_name,
  qualified_path = EXCLUDED.qualified_path,
  tenant_id = EXCLUDED.tenant_id,
  properties = EXCLUDED.properties,
  updated_at = NOW();

-- Seed Edges: BO -> BO Field (HAS_FIELD)
INSERT INTO catalog_edge (
  id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, created_at, updated_at
)
SELECT 
  pg_catalog.gen_random_uuid(),
  bof.bo_id,
  bof.id,
  '88888888-1111-4444-8888-000000000001'::uuid,
  'HAS_FIELD',
  bo.tenant_id,
  NOW(),
  NOW()
FROM business_object_fields bof
JOIN business_objects bo ON bo.id = bof.bo_id
ON CONFLICT DO NOTHING;

-- Seed Edges: BO Field -> Semantic Term (BACKED_BY_TERM)
INSERT INTO catalog_edge (
  id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, created_at, updated_at
)
SELECT 
  pg_catalog.gen_random_uuid(),
  bof.id,
  bof.semantic_term_node_id,
  '88888888-1111-4444-8888-000000000002'::uuid,
  'BACKED_BY_TERM',
  bo.tenant_id,
  NOW(),
  NOW()
FROM business_object_fields bof
JOIN business_objects bo ON bo.id = bof.bo_id
WHERE bof.semantic_term_node_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- Seed Edges: BO -> Semantic Term (USES_SEMANTIC_TERM)
INSERT INTO catalog_edge (
  id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, created_at, updated_at
)
SELECT 
  pg_catalog.gen_random_uuid(),
  bof.bo_id,
  bof.semantic_term_node_id,
  '88888888-1111-4444-8888-000000000003'::uuid,
  'USES_SEMANTIC_TERM',
  bo.tenant_id,
  NOW(),
  NOW()
FROM business_object_fields bof
JOIN business_objects bo ON bo.id = bof.bo_id
WHERE bof.semantic_term_node_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- Seed BO Relationships
INSERT INTO catalog_edge (
  id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, properties, is_active, created_at, updated_at
)
SELECT 
  bor.id,
  bor.source_object_id,
  bor.target_object_id,
  'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd'::uuid,
  'BO_RELATIONSHIP',
  bor.tenant_id,
  jsonb_build_object('cardinality', bor.cardinality, 'relationship_type', bor.relationship_type, 'description', bor.description),
  true,
  COALESCE(bor.created_at, NOW()),
  NOW()
FROM business_object_relationships bor
ON CONFLICT DO NOTHING;

COMMIT;

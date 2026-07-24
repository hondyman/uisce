-- NOTE: This migration creates a compatibility view named `catalog_edge_type`
-- which maps the canonical `catalog_edge_types` to the legacy schema expected
-- by older code paths. This view is intentionally non-destructive and helps
-- Phase A compatibility. It can be removed in Phase B after verifying no
-- remaining consumers rely on the legacy name.
-- 002010: Create read-only view `catalog_edge_type` that maps to `catalog_edge_types` to preserve legacy reads
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.views WHERE table_schema='public' AND table_name='catalog_edge_type'
  ) THEN
    EXECUTE 'CREATE VIEW public.catalog_edge_type AS SELECT id::text AS id, tenant_id, edge_type_name, description, is_active, source_node_type_id::text AS subject_node_type_id, target_node_type_id::text AS object_node_type_id, config AS properties, created_at, updated_at FROM public.catalog_edge_types';
  END IF;
EXCEPTION WHEN others THEN
  -- If view creation fails, raise for visibility
  RAISE;
END $$;

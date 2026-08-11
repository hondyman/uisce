-- Migration: Backfill parent_id from JSON properties in catalog_node table
-- 1. Backfill parent_id from properties->>'table_id' or properties->>'parent_id' where parent_id IS NULL
UPDATE public.catalog_node
SET parent_id = COALESCE(
    (properties->>'table_id')::uuid,
    (properties->>'parent_id')::uuid
)
WHERE parent_id IS NULL
  AND (
      (properties->>'table_id' IS NOT NULL AND properties->>'table_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
      OR
      (properties->>'parent_id' IS NOT NULL AND properties->>'parent_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
  );

-- 2. Clean up JSON properties by removing 'table_id' and 'parent_id' keys from properties column
UPDATE public.catalog_node
SET properties = properties - 'table_id' - 'parent_id'
WHERE properties ? 'table_id' OR properties ? 'parent_id';

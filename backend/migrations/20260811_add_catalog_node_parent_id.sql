-- Migration: Add parent_id column to catalog_node and backfill existing data
-- Parent-child relationships between catalog nodes should ALWAYS use the parent_id column

BEGIN;

-- 1. Add parent_id column to catalog_node
ALTER TABLE public.catalog_node
ADD COLUMN IF NOT EXISTS parent_id uuid REFERENCES public.catalog_node(id) ON DELETE SET NULL;

-- 2. Backfill parent_id from JSON properties for existing rows
-- This handles legacy data where parent IDs were incorrectly stored in JSON
UPDATE public.catalog_node
SET parent_id = COALESCE(
    (properties->>'table_id')::uuid,
    (properties->>'parent_id')::uuid
)
WHERE parent_id IS NULL
  AND (
    (properties->>'table_id' IS NOT NULL AND (properties->>'table_id') ~* '^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$')
    OR
    (properties->>'parent_id' IS NOT NULL AND (properties->>'parent_id') ~* '^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$')
  );

-- 3. Clean JSON properties - remove table_id and parent_id keys
-- Parent relationships should ONLY be stored in the parent_id column
UPDATE public.catalog_node
SET properties = properties - 'table_id' - 'parent_id'
WHERE properties ? 'table_id' OR properties ? 'parent_id';

COMMIT;

-- Verification queries (run separately):
--
-- SELECT COUNT(*) FROM catalog_node WHERE node_type_id = 'a64c1011-16e8-4ddf-b447-363bf8e15c9a' AND parent_id IS NOT NULL;
-- SELECT COUNT(*) FROM catalog_node WHERE properties ? 'table_id' OR properties ? 'parent_id';

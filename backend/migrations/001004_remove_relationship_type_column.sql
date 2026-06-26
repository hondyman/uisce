-- Migration: Remove redundant relationship_type column from catalog_edge
-- The edge_type_id should be the single source of truth for the relationship type

-- Drop the relationship_type column
ALTER TABLE catalog_edge
DROP COLUMN IF EXISTS relationship_type;

-- Ensure edge_type_id is NOT NULL (it should always have a value)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_edge' AND column_name='edge_type_id') THEN
    -- First, delete any edges with NULL edge_type_id
    DELETE FROM catalog_edge WHERE edge_type_id IS NULL;

    -- Now make it NOT NULL
    ALTER TABLE catalog_edge
    ALTER COLUMN edge_type_id SET NOT NULL;
  ELSE
    RAISE NOTICE 'Skipping edge_type_id NOT NULL enforcement because column does not exist';
  END IF;
END;
$$;

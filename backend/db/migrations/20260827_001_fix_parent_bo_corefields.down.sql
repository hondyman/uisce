-- backend/db/migrations/20260827_001_fix_parent_bo_corefields.down.sql
-- Reverses 20260827_001_fix_parent_bo_corefields.up.sql

BEGIN;

-- Remove the CoreFields we added for parent BOs
DELETE FROM business_object_fields
WHERE bo_id IN (
    SELECT id FROM business_objects
    WHERE active_subtype_filter IS NOT NULL
      AND bo_key = active_subtype_filter
);

-- Remove the added columns
ALTER TABLE business_objects
  DROP COLUMN IF EXISTS driver_table_id,
  DROP COLUMN IF EXISTS driver_table_name;

COMMIT;
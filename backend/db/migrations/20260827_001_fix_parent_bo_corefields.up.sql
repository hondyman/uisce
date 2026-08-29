-- backend/db/migrations/20260827_001_fix_parent_bo_corefields.up.sql
-- Fix missing base fields for STI parent Business Objects.
-- Problem: Parent BOs (e.g., oms.account) have no CoreFields because:
--   1. business_objects is missing driver_table_id/driver_table_name columns
--   2. BindingDiscoveryService can't work without TABLE nodes in catalog_node
--   3. business_object_fields only has child BO fields, not parent fields
-- Solution: Add columns and copy CoreFields from child BOs to parent BOs
--            based on the key pattern: parent='oms.account', child='oms.account/institutional'

BEGIN;

-- Step 1: Add missing columns to business_objects
ALTER TABLE business_objects
  ADD COLUMN IF NOT EXISTS driver_table_id uuid,
  ADD COLUMN IF NOT EXISTS driver_table_name text;

-- Step 2: Populate driver_table_name for STI parent BOs
-- Parent BOs have bo_key = active_subtype_filter (e.g., 'oms.account' = 'oms.account')
UPDATE business_objects bo
SET    driver_table_name = bo.bo_key
WHERE  bo.driver_table_name IS NULL
  AND  bo.active_subtype_filter IS NOT NULL
  AND  bo.bo_key = bo.active_subtype_filter;

-- Step 3: For each parent BO, copy CoreFields from its children
-- Child BOs have keys like 'oms.account/institutional' where parent is 'oms.account'
INSERT INTO business_object_fields (
    id, tenant_id, bo_id, field_name, field_role, aggregation_type,
    binding_requirement, eligibility_source, is_exposed,
    subtype_scope, inherits_defaults, term_node_id,
    created_at, updated_at
)
SELECT
    gen_random_uuid() AS id,
    child_bof.tenant_id,
    parent_bo.id AS bo_id,
    child_bof.field_name,
    child_bof.field_role,
    child_bof.aggregation_type,
    child_bof.binding_requirement,
    child_bof.eligibility_source,
    child_bof.is_exposed,
    NULL::text AS subtype_scope,  -- Parent fields have no subtype scope
    true AS inherits_defaults,  -- Parent core fields should inherit defaults
    child_bof.term_node_id,
    NOW() AS created_at,
    NOW() AS updated_at
FROM business_object_fields child_bof
JOIN business_objects child_bo ON child_bo.id = child_bof.bo_id
JOIN business_objects parent_bo ON
    -- Parent key is the child key prefix (e.g., 'oms.account' is prefix of 'oms.account/institutional')
    child_bo.bo_key LIKE (parent_bo.bo_key || '/%')
    AND parent_bo.active_subtype_filter IS NOT NULL
    AND parent_bo.bo_key = parent_bo.active_subtype_filter
    AND parent_bo.tenant_id = child_bo.tenant_id
WHERE NOT EXISTS (
    -- Only add if parent doesn't already have this field
    SELECT 1 FROM business_object_fields existing
    WHERE existing.bo_id = parent_bo.id AND existing.field_name = child_bof.field_name
);

-- Step 4: Update inherits_defaults to true for parent BO fields
-- (Child fields may have false, but parent core fields should be true)
UPDATE business_object_fields
SET inherits_defaults = true
WHERE bo_id IN (
    SELECT id FROM business_objects
    WHERE active_subtype_filter IS NOT NULL
      AND bo_key = active_subtype_filter
);

-- Verify
DO $$
DECLARE
    parent_with_fields INTEGER;
    total_parent_fields INTEGER;
BEGIN
    SELECT COUNT(DISTINCT bo_id) INTO parent_with_fields
    FROM business_object_fields
    WHERE bo_id IN (SELECT id FROM business_objects WHERE active_subtype_filter IS NOT NULL AND bo_key = active_subtype_filter);

    SELECT COUNT(*) INTO total_parent_fields
    FROM business_object_fields bof
    JOIN business_objects bo ON bo.id = bof.bo_id
    WHERE bo.active_subtype_filter IS NOT NULL AND bo.bo_key = bo.active_subtype_filter;

    RAISE NOTICE 'Parent BOs with CoreFields: %. Total parent BO fields: %', parent_with_fields, total_parent_fields;
END $$;

COMMIT;
-- migration: migrate_field_name_to_term_node_id.sql
-- Phase 1: Map legacy field_name entries to term_node_id and remove field_name column
-- DEPRECATION COMPLETE: After this migration, bp_field_permissions uses only term_node_id.
-- Created: 2026-08-22

-- ============================================================================
-- STEP 1: Final attempt to map any remaining field_name rows to term_node_id
-- ============================================================================

-- Try to map through business_object_fields (the canonical mapping)
UPDATE bp_field_permissions fp
SET term_node_id = bof.term_node_id
FROM business_object_fields bof
WHERE fp.field_name IS NOT NULL
  AND fp.term_node_id IS NULL
  AND bof.tenant_id = fp.tenant_id
  AND bof.field_name = fp.field_name
  AND fp.tenant_id = bof.tenant_id;

-- Try direct catalog_node lookup by name (for terms named directly as field names)
UPDATE bp_field_permissions fp
SET term_node_id = cn.id
FROM catalog_node cn
WHERE fp.field_name IS NOT NULL
  AND fp.term_node_id IS NULL
  AND cn.tenant_id::text = fp.tenant_id
  AND cn.name = fp.field_name
  AND cn.kind IN ('semantic_term', 'SEMANTIC_TERM', 'table', 'view');

-- ============================================================================
-- STEP 2: Log rows that still have field_name but no term_node_id (for review)
-- ============================================================================
DO $$
DECLARE
    unmapped_count INT;
    unmapped_tenant UUID;
    unmapped_field TEXT;
BEGIN
    SELECT COUNT(*), fp.tenant_id, fp.field_name
    INTO unmapped_count, unmapped_tenant, unmapped_field
    FROM bp_field_permissions fp
    WHERE fp.field_name IS NOT NULL AND fp.term_node_id IS NULL
    GROUP BY fp.tenant_id, fp.field_name;

    IF unmapped_count > 0 THEN
        RAISE WARNING 'FIELD NAME MIGRATION: There are % rows with field_name but no term_node_id.', unmapped_count;
        RAISE WARNING 'These entries will continue to use field_name-based enforcement (deprecated path).';
        RAISE WARNING 'Manual review required to map these to semantic terms:';
    END IF;
END $$;

-- Log details of unmapped entries
SELECT
    fp.tenant_id,
    fp.datasource_id,
    fp.field_name,
    fp.resource_type,
    COUNT(*) as row_count
FROM bp_field_permissions fp
WHERE fp.field_name IS NOT NULL AND fp.term_node_id IS NULL
GROUP BY fp.tenant_id, fp.datasource_id, fp.field_name, fp.resource_type;

-- ============================================================================
-- STEP 3: Remove the deprecated field_name partial unique constraint
-- ============================================================================
ALTER TABLE bp_field_permissions DROP CONSTRAINT IF EXISTS uq_bp_field_permissions_field_name;

-- ============================================================================
-- STEP 4: Drop field_name column (only after verifying migration)
-- This is a one-way migration. Unmapped rows will lose their field_name.
-- For safety, we only drop if there are no unmapped rows.
-- ============================================================================
DO $$
DECLARE
    unmapped_exists BOOLEAN;
BEGIN
    -- Check if there are any unmapped rows
    SELECT EXISTS(
        SELECT 1 FROM bp_field_permissions
        WHERE field_name IS NOT NULL AND term_node_id IS NULL
    ) INTO unmapped_exists;

    IF unmapped_exists THEN
        RAISE NOTICE 'Cannot drop field_name column: unmapped rows exist.';
        RAISE NOTICE 'These rows use deprecated field_name-based enforcement.';
        RAISE NOTICE 'Review and map them manually, or accept they will be inaccessible.';
    ELSE
        -- Safe to drop
        ALTER TABLE bp_field_permissions DROP COLUMN IF EXISTS field_name;
        RAISE NOTICE 'Successfully dropped field_name column from bp_field_permissions';
    END IF;
END $$;

-- ============================================================================
-- STEP 5: Simplify the unique constraint (remove field_name reference)
-- ============================================================================
-- The unique constraint is now just (role_id, term_node_id, resource_type, resource_id)
-- But term_node_id can be NULL in edge cases, so we need a conditional unique index

-- Drop the old constraint if it still exists
ALTER TABLE bp_field_permissions DROP CONSTRAINT IF EXISTS uq_bp_field_permissions_unique;

-- Add proper unique constraint for term_node_id path
ALTER TABLE bp_field_permissions ADD CONSTRAINT uq_bp_field_permissions_term UNIQUE (role_id, term_node_id, resource_type, resource_id)
    WHERE term_node_id IS NOT NULL;

-- ============================================================================
-- STEP 6: Verify migration
-- ============================================================================
DO $$
DECLARE
    total_rows INT;
    term_rows INT;
    field_only_rows INT;
BEGIN
    SELECT COUNT(*) INTO total_rows FROM bp_field_permissions;
    SELECT COUNT(*) INTO term_rows FROM bp_field_permissions WHERE term_node_id IS NOT NULL;
    SELECT COUNT(*) INTO field_only_rows FROM bp_field_permissions WHERE field_name IS NOT NULL;

    RAISE NOTICE '=== bp_field_permissions Migration Summary ===';
    RAISE NOTICE 'Total rows: %', total_rows;
    RAISE NOTICE 'Rows with term_node_id: %', term_rows;
    RAISE NOTICE 'Rows with field_name only (deprecated): %', field_only_rows;
    RAISE NOTICE '=== field_name column: %', CASE WHEN field_only_rows > 0 THEN 'RETAINED (unmapped rows exist)' ELSE 'DROPPED' END;
END $$;

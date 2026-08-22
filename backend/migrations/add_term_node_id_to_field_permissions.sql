-- migration: add_term_node_id_to_field_permissions.sql
-- Adds semantic term-level RBAC to existing bp_field_permissions table

-- 1. Add term_node_id column (nullable initially)
ALTER TABLE bp_field_permissions ADD COLUMN IF NOT EXISTS term_node_id UUID;

-- 2. Make resource_type nullable (semantic permissions are global by default)
ALTER TABLE bp_field_permissions ALTER COLUMN resource_type DROP NOT NULL;

-- 3. Backfill term_node_id from existing field_name entries via business_object_fields
-- This maps physical field names to their semantic term counterparts
UPDATE bp_field_permissions fp
SET term_node_id = bof.term_node_id
FROM business_object_fields bof
WHERE fp.field_name IS NOT NULL
  AND bof.tenant_id = fp.tenant_id
  AND bof.field_name = fp.field_name
  AND fp.term_node_id IS NULL;

-- 4. For field_names that couldn't be mapped, try direct catalog_node lookup by name
-- (Some semantic terms may be named directly)
UPDATE bp_field_permissions fp
SET term_node_id = cn.id
FROM catalog_node cn
WHERE fp.field_name IS NOT NULL
  AND fp.term_node_id IS NULL
  AND cn.tenant_id = fp.tenant_id::text
  AND cn.name = fp.field_name
  AND cn.kind = 'semantic_term';

-- 5. Create index for efficient term_node_id lookups
CREATE INDEX IF NOT EXISTS idx_bp_field_permissions_term ON bp_field_permissions(term_node_id);

-- 6. Update unique constraint to prefer term_node_id
ALTER TABLE bp_field_permissions DROP CONSTRAINT IF EXISTS uq_bp_field_permissions_unique;
ALTER TABLE bp_field_permissions ADD CONSTRAINT uq_bp_field_permissions_unique
    UNIQUE (role_id, term_node_id, resource_type, resource_id) WHERE term_node_id IS NOT NULL;

-- 7. Keep field_name unique constraint for backward compatibility (deprecated path)
ALTER TABLE bp_field_permissions DROP CONSTRAINT IF EXISTS uq_bp_field_permissions_field_name;
ALTER TABLE bp_field_permissions ADD CONSTRAINT uq_bp_field_permissions_field_name
    UNIQUE (role_id, field_name, resource_type, resource_id) WHERE field_name IS NOT NULL AND term_node_id IS NULL;

-- 8. Add check constraint: at least one targeting method
ALTER TABLE bp_field_permissions ADD CONSTRAINT chk_bp_field_permissions_target
    CHECK (term_node_id IS NOT NULL OR field_name IS NOT NULL);

-- 9. Mark unmapped rows for review (those with field_name but no term_node_id)
-- These will use legacy field_name-based enforcement
DO $$
BEGIN
    RAISE NOTICE 'Field permissions without semantic term mapping:';
    RAISE NOTICE 'These use deprecated field_name-based enforcement.';
END $$;

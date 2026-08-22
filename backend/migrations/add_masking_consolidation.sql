-- migration: add_masking_consolidation.sql
-- Phase 1: Migrate bp_field_masking_rules into bp_field_permissions as permission_level='mask'
-- This consolidates two separate masking mechanisms into the unified bp_field_permissions system.
-- DEPRECATION: bp_field_masking_rules is deprecated; use bp_field_permissions.permission_level='mask' instead.
-- Created: 2026-08-22

-- ============================================================================
-- STEP 1: Add masking_pattern column to bp_field_permissions if not exists
-- ============================================================================
ALTER TABLE bp_field_permissions ADD COLUMN IF NOT EXISTS masking_pattern VARCHAR(200);

-- ============================================================================
-- STEP 2: Add backward-compat note about unmasked_roles semantics
-- In bp_field_masking_rules, unmasked_roles are roles that BYPASS masking.
-- In bp_field_permissions, we model this as explicit permissions:
--   - Roles NOT in unmasked_roles: permission_level = 'mask'
--   - Roles IN unmasked_roles: permission_level = 'read' (can see unmasked)
-- ============================================================================

-- For each masking rule, find all roles in the tenant/datasource and create permissions
-- Roles NOT in unmasked_roles get 'mask', roles in unmasked_roles get 'read'

DO $$
DECLARE
    masking_rule RECORD;
    role_id_iter UUID;
    has_unmasked BOOLEAN;
BEGIN
    FOR masking_rule IN
        SELECT mr.id, mr.tenant_id, mr.datasource_id, mr.resource_type, mr.field_name,
               mr.masking_type, mr.masking_pattern, mr.unmasked_roles
        FROM bp_field_masking_rules mr
        WHERE mr.is_active = true
    LOOP
        has_unmasked := (masking_rule.unmasked_roles IS NOT NULL AND array_length(masking_rule.unmasked_roles, 1) > 0);

        -- Find term_node_id from business_object_fields matching field_name
        -- Try to map through business_object_fields first
        UPDATE bp_field_permissions fp
        SET masking_pattern = masking_rule.masking_pattern
        FROM business_object_fields bof
        WHERE fp.tenant_id = masking_rule.tenant_id
          AND fp.datasource_id = masking_rule.datasource_id
          AND bof.tenant_id = masking_rule.tenant_id
          AND bof.field_name = masking_rule.field_name
          AND fp.term_node_id = bof.term_node_id
          AND fp.resource_type = masking_rule.resource_type
          AND masking_rule.masking_pattern IS NOT NULL;

        -- For roles NOT in unmasked_roles: create/update permission to 'mask'
        -- Insert 'mask' permission for all roles in the tenant that are NOT in unmasked_roles
        IF has_unmasked THEN
            -- Get term_node_id for this field (if exists via business_object_fields)
            INSERT INTO bp_field_permissions (
                tenant_id, datasource_id, role_id, term_node_id, resource_type, resource_id,
                field_name, permission_level, masking_pattern, created_at, updated_at
            )
            SELECT
                r.tenant_id,
                masking_rule.datasource_id,
                r.id AS role_id,
                bof.term_node_id,
                masking_rule.resource_type,
                NULL,
                masking_rule.field_name,
                'mask',
                masking_rule.masking_pattern,
                NOW(),
                NOW()
            FROM bp_roles r
            JOIN bp_user_roles ur ON r.id = ur.role_id
            LEFT JOIN business_object_fields bof
                ON bof.tenant_id = r.tenant_id AND bof.field_name = masking_rule.field_name
            WHERE r.tenant_id = masking_rule.tenant_id
              AND r.id != ALL(masking_rule.unmasked_roles)  -- exclude unmasked roles
              AND ur.is_active = true
            ON CONFLICT (role_id, term_node_id, resource_type, resource_id)
            DO UPDATE SET
                permission_level = 'mask',
                masking_pattern = EXCLUDED.masking_pattern,
                updated_at = NOW()
            WHERE bp_field_permissions.permission_level != 'mask' OR bp_field_permissions.masking_pattern IS DISTINCT FROM EXCLUDED.masking_pattern;

            -- Get roles in unmasked_roles: give them 'read' permission (they bypass masking)
            INSERT INTO bp_field_permissions (
                tenant_id, datasource_id, role_id, term_node_id, resource_type, resource_id,
                field_name, permission_level, created_at, updated_at
            )
            SELECT
                r.tenant_id,
                masking_rule.datasource_id,
                r.id AS role_id,
                bof.term_node_id,
                masking_rule.resource_type,
                NULL,
                masking_rule.field_name,
                'read',
                NOW(),
                NOW()
            FROM bp_roles r
            JOIN bp_user_roles ur ON r.id = ur.role_id
            LEFT JOIN business_object_fields bof
                ON bof.tenant_id = r.tenant_id AND bof.field_name = masking_rule.field_name
            WHERE r.tenant_id = masking_rule.tenant_id
              AND r.id = ANY(masking_rule.unmasked_roles)
              AND ur.is_active = true
            ON CONFLICT (role_id, term_node_id, resource_type, resource_id)
            DO UPDATE SET
                permission_level = 'read',
                updated_at = NOW()
            WHERE bp_field_permissions.permission_level = 'mask';  -- Only upgrade from mask to read
        ELSE
            -- No unmasked_roles means ALL roles get 'mask'
            INSERT INTO bp_field_permissions (
                tenant_id, datasource_id, role_id, term_node_id, resource_type, resource_id,
                field_name, permission_level, masking_pattern, created_at, updated_at
            )
            SELECT
                r.tenant_id,
                masking_rule.datasource_id,
                r.id AS role_id,
                bof.term_node_id,
                masking_rule.resource_type,
                NULL,
                masking_rule.field_name,
                'mask',
                masking_rule.masking_pattern,
                NOW(),
                NOW()
            FROM bp_roles r
            JOIN bp_user_roles ur ON r.id = ur.role_id
            LEFT JOIN business_object_fields bof
                ON bof.tenant_id = r.tenant_id AND bof.field_name = masking_rule.field_name
            WHERE r.tenant_id = masking_rule.tenant_id
              AND ur.is_active = true
            ON CONFLICT (role_id, term_node_id, resource_type, resource_id)
            DO UPDATE SET
                permission_level = 'mask',
                masking_pattern = EXCLUDED.masking_pattern,
                updated_at = NOW()
            WHERE bp_field_permissions.permission_level != 'mask' OR bp_field_permissions.masking_pattern IS DISTINCT FROM EXCLUDED.masking_pattern;
        END IF;

        RAISE NOTICE 'Migrated masking rule for field % in tenant %', masking_rule.field_name, masking_rule.tenant_id;
    END LOOP;
END $$;

-- ============================================================================
-- STEP 3: Mark bp_field_masking_rules as deprecated
-- ============================================================================
COMMENT ON TABLE bp_field_masking_rules IS
    'DEPRECATED: Use bp_field_permissions with permission_level = ''mask'' instead. '
    'This table will be removed in v3.0. Data has been migrated to bp_field_permissions.';

-- ============================================================================
-- STEP 4: Log summary
-- ============================================================================
DO $$
DECLARE
    migrated_count INT;
    remaining_count INT;
BEGIN
    SELECT COUNT(*) INTO migrated_count
    FROM bp_field_permissions
    WHERE masking_pattern IS NOT NULL;

    SELECT COUNT(*) INTO remaining_count
    FROM bp_field_masking_rules
    WHERE is_active = true;

    RAISE NOTICE 'Masking consolidation complete:';
    RAISE NOTICE '  - bp_field_permissions rows with masking_pattern: %', migrated_count;
    RAISE NOTICE '  - Remaining active bp_field_masking_rules entries: %', remaining_count;
    RAISE NOTICE '  - bp_field_masking_rules is now DEPRECATED - use bp_field_permissions instead';
END $$;

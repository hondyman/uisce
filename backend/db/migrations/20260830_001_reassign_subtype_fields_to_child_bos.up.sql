-- 20260830_001_reassign_subtype_fields_to_child_bos.up.sql
-- Corrective backfill: yesterday's manual migration (20260825_001_*) put all subtype
-- fields onto the PARENT business object with subtype_scope='ALL'.  This migration
-- moves each field to the correct child (subtype) BO with the proper subtype_scope
-- value, matching what the catalog-admin sync pipeline should have done.
--
-- IMPORTANT: oms.subtype_registry uses tenant_id='00000000-...' (system shared)
-- while business_objects use their own tenant IDs.  We match BOs by bo_key, not tenant.
--
-- Symptom fixed: "fields not assigned to subtypes" — parent BOs had all fields
-- (e.g. oms.account=23) while every child BO had 0.

BEGIN;

-- ---------------------------------------------------------------------------
-- Phase 1: For each (root_object, subtype_code) in the registry, find the
-- child BO and move the allowlist fields from the parent to the child.
-- Uses ON CONFLICT DO NOTHING to stay idempotent across duplicate registry rows.
-- ---------------------------------------------------------------------------
DO $$
DECLARE
    rec            RECORD;
    parent_row     RECORD;
    child_bo_id   UUID;
    parent_bo_id  UUID;
    parent_tenant UUID;
    f             TEXT;
    moved_count   INT := 0;
BEGIN
    FOR rec IN
        -- DISTINCT ON avoids processing the same subtype twice when the registry
        -- has duplicate rows (confirmed in alpha DB: each subtype appears twice).
        -- ORDER BY ensures deterministic pick.
        SELECT DISTINCT ON (root_object, subtype_code)
               sr.root_object,
               sr.subtype_code,
               sr.field_allowlist
        FROM   oms.subtype_registry sr
        WHERE  sr.is_active = true
          AND  jsonb_array_length(sr.field_allowlist) > 0
        ORDER BY root_object, subtype_code
    LOOP
        -- Find the parent BO by bo_key (tenant-agnostic — registry tenant differs from BO tenant)
        SELECT id, tenant_id INTO parent_row
        FROM   business_objects
        WHERE  bo_key = 'oms.' || rec.root_object
        LIMIT 1;

        IF parent_row.id IS NULL THEN
            RAISE WARNING 'Parent BO not found for root_object=%. Skipping.', rec.root_object;
            CONTINUE;
        END IF;
        parent_bo_id  := parent_row.id;
        parent_tenant := parent_row.tenant_id;

        -- Find the child BO by bo_key on the SAME tenant as the parent
        SELECT id INTO child_bo_id
        FROM   business_objects
        WHERE  tenant_id = parent_tenant
          AND  bo_key   = 'oms.' || rec.root_object || '/' || rec.subtype_code
        LIMIT 1;

        IF child_bo_id IS NULL THEN
            RAISE WARNING 'Child BO not found for %. Skipping.',
                'oms.' || rec.root_object || '/' || rec.subtype_code;
            CONTINUE;
        END IF;

        -- Move each field from the parent row to the child row.
        -- Use parent_tenant (the tenant that actually owns the BOs), not the registry tenant.
        -- Rows that were already moved (now pointing at child_bo_id) are no-ops
        -- because the UPDATE ... WHERE bo_id=parent_bo_id will match 0 rows.
        FOR f IN
            SELECT jsonb_array_elements_text(rec.field_allowlist)
        LOOP
            UPDATE business_object_fields
            SET    bo_id            = child_bo_id,
                   subtype_scope    = UPPER(rec.subtype_code),
                   inherits_defaults = FALSE,
                   updated_at      = NOW()
            WHERE  tenant_id       = parent_tenant
              AND  bo_id          = parent_bo_id
              AND  field_name     = f;

            GET DIAGNOSTICS moved_count = ROW_COUNT;
            IF moved_count > 0 THEN
                RAISE DEBUG 'Moved field % to child BO % (scope=%)',
                    f, child_bo_id, UPPER(rec.subtype_code);
            END IF;
        END LOOP;
    END LOOP;
END;
$$;

-- ---------------------------------------------------------------------------
-- Phase 2: Mark all remaining parent-level fields with inherits_defaults=true
-- so the UI correctly shows them as "core/inherited" on the parent.
-- Already-moved fields are on the child with inherits_defaults=false.
-- ---------------------------------------------------------------------------
UPDATE business_object_fields
SET    inherits_defaults = TRUE
WHERE  bo_id IN (
    SELECT id FROM business_objects
    WHERE bo_key ~ '^oms\.[^/]+$'   -- top-level parent BOs only
)
  AND inherits_defaults IS DISTINCT FROM TRUE;

COMMIT;

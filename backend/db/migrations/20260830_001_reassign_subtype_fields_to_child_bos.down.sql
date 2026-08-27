-- 20260830_001_reassign_subtype_fields_to_child_bos.down.sql
-- Rolls back the corrective backfill by moving subtype-scoped fields
-- BACK to their parent BOs with subtype_scope='ALL' and inherits_defaults=true.
-- This mirrors the broken state left by 20260825_001_* so running this and then
-- re-running the forward migration reproduces the same end state as a clean sync.

BEGIN;

DO $$
DECLARE
    rec            RECORD;
    parent_row     RECORD;
    child_bo_id   UUID;
    parent_bo_id   UUID;
    parent_tenant UUID;
BEGIN
    FOR rec IN
        SELECT DISTINCT ON (root_object, subtype_code)
               sr.root_object,
               sr.subtype_code
        FROM   oms.subtype_registry sr
        WHERE  sr.is_active = true
          AND  jsonb_array_length(sr.field_allowlist) > 0
        ORDER BY root_object, subtype_code
    LOOP
        -- Find the parent BO by bo_key
        SELECT id, tenant_id INTO parent_row
        FROM   business_objects
        WHERE  bo_key = 'oms.' || rec.root_object
        LIMIT 1;

        IF parent_row.id IS NULL THEN
            CONTINUE;
        END IF;
        parent_bo_id  := parent_row.id;
        parent_tenant := parent_row.tenant_id;

        -- Find the child BO on the same tenant
        SELECT id INTO child_bo_id
        FROM   business_objects
        WHERE  tenant_id = parent_tenant
          AND  bo_key   = 'oms.' || rec.root_object || '/' || rec.subtype_code
        LIMIT 1;

        IF child_bo_id IS NULL THEN
            CONTINUE;
        END IF;

        -- Move fields back to parent with scope ALL
        UPDATE business_object_fields
        SET    bo_id            = parent_bo_id,
               subtype_scope    = 'ALL',
               inherits_defaults = TRUE,
               updated_at      = NOW()
        WHERE  tenant_id       = parent_tenant
          AND  bo_id          = child_bo_id
          AND  subtype_scope  = UPPER(rec.subtype_code);
    END LOOP;
END;
$$;

COMMIT;

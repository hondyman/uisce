-- Phase 5: Warning trigger for inline calc fields in business_objects.fields
-- Logs a NOTICE when an INSERT or UPDATE attempts to add a calculated field
-- inline in the business_objects.fields JSONB column.
-- This is a soft enforcement — it warns but does not block.

DO $$
BEGIN
    CREATE OR REPLACE FUNCTION warn_inline_calc_fields()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    AS $$
    DECLARE
        has_inline_calc boolean;
        field_entry jsonb;
        field_names text[];
    BEGIN
        IF NEW.fields IS NULL OR jsonb_typeof(NEW.fields) != 'array' THEN
            RETURN NEW;
        END IF;

        has_inline_calc := false;
        field_names := ARRAY[]::text[];

        FOR field_entry IN
            SELECT elem FROM jsonb_array_elements(NEW.fields) AS elem
        LOOP
            IF (field_entry->>'is_calculated')::boolean = true
               OR field_entry->>'type' = 'calculated' THEN
                has_inline_calc := true;
                field_names := array_append(field_names, COALESCE(field_entry->>'name', '<unnamed>'));
            END IF;
        END LOOP;

        IF has_inline_calc THEN
            RAISE NOTICE
                '[business_objects:%] inline calc field detected — move to public.calc_fields catalog. Fields: %',
                NEW.id, array_to_string(field_names, ', ');
        END IF;

        RETURN NEW;
    END;
    $$;

    DROP TRIGGER IF EXISTS trg_warn_inline_calc_fields ON public.business_objects;

    CREATE TRIGGER trg_warn_inline_calc_fields
    BEFORE INSERT OR UPDATE OF fields ON public.business_objects
    FOR EACH ROW
    EXECUTE FUNCTION warn_inline_calc_fields();

    RAISE NOTICE 'Trigger trg_warn_inline_calc_fields created on business_objects.fields';
END $$;

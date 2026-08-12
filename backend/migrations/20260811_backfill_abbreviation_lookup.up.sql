-- Backfill sml.abbreviation_lookup from public.abbreviations for the gold_copy tenant
-- This migration populates the sml.abbreviation_lookup table with the existing abbreviations
-- data, associating them with the gold_copy tenant so they appear in the abbreviations UI.

DO $$
DECLARE
    gold_copy_tid VARCHAR(255);
BEGIN
    -- Get the gold copy tenant ID
    SELECT id INTO gold_copy_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;

    IF gold_copy_tid IS NULL THEN
        RAISE NOTICE 'No gold_copy tenant found, skipping backfill';
        RETURN;
    END IF;

    RAISE NOTICE 'Backfilling abbreviations for gold_copy tenant: %', gold_copy_tid;

    -- Insert abbreviations from public.abbreviations into sml.abbreviation_lookup
    -- Using ON CONFLICT to handle any duplicates gracefully
    INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id, created_at, updated_at)
    SELECT
        a.abbreviation,
        a.full_word,
        COALESCE(a.notes, ''),
        gold_copy_tid,
        COALESCE(a.created_at, NOW()),
        COALESCE(a.updated_at, NOW())
    FROM public.abbreviations a
    WHERE NOT EXISTS (
        SELECT 1 FROM sml.abbreviation_lookup l
        WHERE l.tenant_id = gold_copy_tid
          AND UPPER(l.abbreviation) = UPPER(a.abbreviation)
    );

    RAISE NOTICE 'Backfill complete. Verifying row count...';

END $$;

-- Verify the backfill
DO $$
DECLARE
    row_count INTEGER;
    gold_copy_tid VARCHAR(255);
BEGIN
    SELECT id INTO gold_copy_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;
    SELECT COUNT(*) INTO row_count FROM sml.abbreviation_lookup WHERE tenant_id = gold_copy_tid;
    RAISE NOTICE 'Gold copy tenant (%) now has % abbreviations', gold_copy_tid, row_count;
END $$;

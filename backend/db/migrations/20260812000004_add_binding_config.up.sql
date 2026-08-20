-- Phase 3b: Add config JSONB column to business_object_bindings
-- Enables carrying calc field implementation details (semantic term refs,
-- bi-temporal column mappings) inside the binding.

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'business_object_bindings'
          AND column_name = 'config'
    ) THEN
        ALTER TABLE public.business_object_bindings
            ADD COLUMN config JSONB DEFAULT '{}'::jsonb;
        RAISE NOTICE 'Added config JSONB column to business_object_bindings';
    ELSE
        RAISE NOTICE 'config column already exists in business_object_bindings';
    END IF;
END $$;

COMMIT;

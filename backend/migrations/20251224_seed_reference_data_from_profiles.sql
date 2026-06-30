-- Migration: 000024_seed_reference_data_from_profiles.sql
-- Idempotent seed: upsert reference data from sml.column_profiles



-- Ensure reference_data table exists with minimal columns we need
CREATE TABLE IF NOT EXISTS public.reference_data (
    tenant_datasource_id uuid,
    semantic_term_id uuid,
    properties jsonb,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Make sure expected columns exist (safe to run if table pre-exists)
ALTER TABLE public.reference_data ADD COLUMN IF NOT EXISTS tenant_datasource_id uuid;
ALTER TABLE public.reference_data ADD COLUMN IF NOT EXISTS semantic_term_id uuid;
ALTER TABLE public.reference_data ADD COLUMN IF NOT EXISTS properties jsonb;
ALTER TABLE public.reference_data ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();
ALTER TABLE public.reference_data ADD COLUMN IF NOT EXISTS updated_at timestamptz DEFAULT now();

-- Unique index so we can upsert by tenant_datasource_id + semantic_term_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'i' AND c.relname = 'idx_reference_term_tenant_ds_semantic'
    ) THEN
        BEGIN
            CREATE UNIQUE INDEX IF NOT EXISTS idx_reference_term_tenant_ds_semantic ON public.reference_data (tenant_datasource_id, semantic_term_id);
        EXCEPTION WHEN others THEN
            -- If index creation fails for any reason (existing incompatible index), ignore to keep migration idempotent
            RAISE NOTICE 'Could not CREATE index IF NOT EXISTS idx_reference_term_tenant_ds_semantic: %', SQLERRM;
        END;
    END IF;
END$$;

-- Upsert properties from profiler table. Merge JSON objects so the new properties override or extend existing ones.
INSERT INTO public.reference_data (tenant_datasource_id, semantic_term_id, properties, created_at, updated_at)
SELECT
    tenant_datasource_id,
    id AS semantic_term_id,
    COALESCE(properties, '{}'::jsonb) AS properties,
    now() as created_at,
    now() as updated_at
FROM sml.column_profiles
WHERE tenant_datasource_id IS NOT NULL AND id IS NOT NULL
ON CONFLICT (tenant_datasource_id, semantic_term_id) DO UPDATE SET
    properties = COALESCE(public.reference_data.properties, '{}'::jsonb) || COALESCE(EXCLUDED.properties, '{}'::jsonb),
    updated_at = now();



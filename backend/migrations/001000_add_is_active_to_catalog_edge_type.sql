-- NOTE: This migration intentionally references the legacy singular table
-- `catalog_edge_type` for backward compatibility. It is conditional and will
-- only run if that legacy table exists in the target database. Keep this
-- migration unchanged unless you are performing an explicit Phase B
-- canonicalization (rename/drop) and have verified backups.
-- Idempotent migration to ensure catalog_edge_type (singular) has is_active column and index
-- Safe to run multiple times. Adds column if missing, updates existing rows to true,
-- and creates the index if missing.


-- Only run if the table exists
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_type') THEN
    ALTER TABLE public.catalog_edge_type
      ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

    -- Backfill any NULLs to true for safety
    UPDATE public.catalog_edge_type
      SET is_active = true
      WHERE is_active IS NULL;

    -- Create index if not exists
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_catalog_edge_type_active ON public.catalog_edge_type(is_active)';
  END IF;
END$$;



-- Idempotent migration to ensure catalog_edge_types has is_active column and index
-- Safe to run multiple times. Adds column if missing, updates existing rows to true,
-- and creates the index if missing.
-- Idempotent migration to ensure catalog_edge_types has is_active column and index
-- Safe to run multiple times. Adds column if missing, updates existing rows to true,
-- and creates the index only if the table exists.


DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'catalog_edge_types'
  ) THEN
    ALTER TABLE public.catalog_edge_types
      ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

    -- Backfill any NULLs to true for safety
    UPDATE public.catalog_edge_types
      SET is_active = true
      WHERE is_active IS NULL;

    -- Create index if not exists
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_active ON public.catalog_edge_types(is_active)';
  END IF;
END$$;



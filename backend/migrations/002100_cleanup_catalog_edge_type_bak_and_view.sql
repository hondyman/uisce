-- NOTE: This cleanup migration intentionally drops legacy artifacts (the
-- compatibility view `catalog_edge_type` and backup table
-- `catalog_edge_type_bak`). This is a destructive Phase B step and should
-- only be applied once you have validated Phase A compatibility and have a
-- verified DB backup for the target environment.
-- 002100: Cleanup legacy artifacts for catalog_edge_type
-- WARNING: Destructive. Drops the compatibility view and the backup table.
-- Idempotent: safe to run multiple times.

DO $$
BEGIN
  -- Drop compatibility view if present
  IF EXISTS (
    SELECT 1 FROM information_schema.views WHERE table_schema='public' AND table_name='catalog_edge_type'
  ) THEN
    EXECUTE 'DROP VIEW IF EXISTS public.catalog_edge_type CASCADE';
  END IF;

  -- Drop legacy backup table if present
  IF EXISTS (
    SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_edge_type_bak'
  ) THEN
    EXECUTE 'DROP TABLE IF EXISTS public.catalog_edge_type_bak CASCADE';
  END IF;
EXCEPTION WHEN others THEN
  -- Bubble up any unexpected errors so the migration runner notices
  RAISE;
END $$;

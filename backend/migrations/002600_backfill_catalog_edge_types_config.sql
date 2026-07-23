-- 002600: Backfill and expose `config` jsonb on catalog_edge_types for compatibility
-- Idempotent: safe to run multiple times in development and production.

DO $$
BEGIN
  -- 1) Add column if missing
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'catalog_edge_types' AND column_name = 'config'
  ) THEN
    EXECUTE 'ALTER TABLE public.catalog_edge_types ADD COLUMN config jsonb';
  END IF;

  -- 2) Backfill from properties where config is null (avoid overwriting existing data)
  -- Handle multiple shapes: catalog_edge_types may or may not have a properties column.
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='catalog_edge_types' AND column_name='properties') THEN
    UPDATE public.catalog_edge_types
    SET config = jsonb_build_object('properties', COALESCE(properties, '[]'::jsonb))
    WHERE config IS NULL;
  END IF;

  -- 3) Ensure GIN index exists for query performance on jsonb config
  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND c.relname = 'idx_catalog_edge_types_config'
  ) THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_config ON public.catalog_edge_types USING GIN(config)';
  END IF;
END$$;

-- NOTE: This migration is intentionally small and reversible: it only adds the
-- optional `config` column and backfills it from the existing `properties` JSONB.
-- It is safe to run on databases that already include `config`.

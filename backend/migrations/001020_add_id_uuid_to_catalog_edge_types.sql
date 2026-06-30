-- 001020: Add id_uuid to catalog_edge_types and backfill from existing id when possible
-- Idempotent: safe to run multiple times

DO $$
BEGIN
  -- add column if missing
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'catalog_edge_types' AND column_name = 'id_uuid'
  ) THEN
    ALTER TABLE public.catalog_edge_types ADD COLUMN id_uuid UUID;
  END IF;

  -- backfill id_uuid from id when id looks like a UUID
  UPDATE public.catalog_edge_types
  SET id_uuid = CASE
    WHEN char_length(id::text) = 36 AND id::text ~ '^[0-9a-fA-F0-9-]{36}$' THEN id::uuid
    ELSE gen_random_uuid()
  END
  WHERE id_uuid IS NULL;

  -- create a unique index if it doesn't already exist
  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND c.relname = 'idx_catalog_edge_types_id_uuid'
  ) THEN
    CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_edge_types_id_uuid ON public.catalog_edge_types(id_uuid);
  END IF;
EXCEPTION WHEN others THEN
  -- If something goes wrong, raise a clear error so migration tooling can surface it
  RAISE;
END $$;

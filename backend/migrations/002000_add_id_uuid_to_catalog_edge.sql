-- 002000: Add id_uuid to catalog_edge (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='catalog_edge' AND column_name='id_uuid'
  ) THEN
    ALTER TABLE public.catalog_edge ADD COLUMN id_uuid UUID;
  END IF;

  UPDATE public.catalog_edge
  SET id_uuid = CASE
    WHEN char_length(id::text) = 36 AND id::text ~ '^[0-9a-fA-F0-9-]{36}$' THEN id::uuid
    ELSE gen_random_uuid()
  END
  WHERE id_uuid IS NULL;

  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND c.relname = 'idx_catalog_edge_id_uuid'
  ) THEN
    CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_edge_id_uuid ON public.catalog_edge(id_uuid);
  END IF;
END $$;

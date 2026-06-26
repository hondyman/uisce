-- 001002: Add id_uuid to catalog_node_type and backfill from existing id values where possible
-- Idempotent and safe to run multiple times


DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_node_type') THEN

    -- add new UUID column if missing
    ALTER TABLE public.catalog_node_type
      ADD COLUMN IF NOT EXISTS id_uuid UUID;

    -- backfill id_uuid where possible (safe cast when looks like UUID)
    UPDATE public.catalog_node_type
    SET id_uuid = CASE
      WHEN char_length(id::text) = 36 AND id::text ~ '^[0-9a-fA-F0-9-]{36}$' THEN id::uuid
      ELSE gen_random_uuid()
    END
    WHERE id_uuid IS NULL;

    -- ensure uniqueness/index on id_uuid
    CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_node_type_id_uuid ON public.catalog_node_type(id_uuid);

  END IF;
END$$;



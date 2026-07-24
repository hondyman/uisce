-- NOTE: This migration renames the legacy `catalog_edge_type` to
-- `catalog_edge_type_bak` as a non-destructive safety step during migration.
-- It is intentionally conditional and should remain as a historical step
-- supporting rollbacks and audits. Do not remove or rename the referenced
-- legacy table identifier unless performing Phase B canonicalization with
-- appropriate backups.
-- 001005: Non-destructive rename of singular `catalog_edge_type` to a backup name
-- This keeps the original table for safety; the application should now use `catalog_edge_types` (plural)


DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_edge_type') THEN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='catalog_edge_type_bak') THEN
      ALTER TABLE public.catalog_edge_type RENAME TO catalog_edge_type_bak;
    END IF;
  END IF;
END$$;



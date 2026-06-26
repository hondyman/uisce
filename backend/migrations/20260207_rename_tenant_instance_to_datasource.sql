-- Migration: Rename datasource_id -> datasource_id (DRAFT)
-- NOTE: Review carefully before running in production. This migration renames columns and updates constraints where necessary.

BEGIN;

-- tenant_product
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenant_product' AND column_name='datasource_id') THEN
        ALTER TABLE public.tenant_product RENAME COLUMN datasource_id TO datasource_id;
    END IF;
END$$;

-- tenant_product_datasource
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenant_product_datasource' AND column_name='datasource_id') THEN
        ALTER TABLE public.tenant_product_datasource RENAME COLUMN datasource_id TO datasource_id;
    END IF;
END$$;

-- Add further tables as needed
-- TODO: Add renames for other tables referencing datasource_id and update/rename constraints and indexes accordingly.

COMMIT;

-- Rollback guidance:
-- This migration is destructive for schema names; to rollback, ensure you have a DB snapshot and/or write a reverse migration renaming datasource_id back to datasource_id and restoring constraint names.

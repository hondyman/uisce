-- Rollback: remove allowed_regions column and index
DROP INDEX IF EXISTS idx_tenants_allowed_regions;
ALTER TABLE public.tenants DROP COLUMN IF EXISTS allowed_regions;
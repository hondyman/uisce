-- Remove Starlark script content column
ALTER TABLE public.catalog_validation_rules DROP COLUMN IF EXISTS script_content;

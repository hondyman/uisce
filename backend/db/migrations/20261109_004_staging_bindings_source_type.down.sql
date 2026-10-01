ALTER TABLE public.staging_bindings DROP CONSTRAINT IF EXISTS staging_bindings_source_type_check;
ALTER TABLE public.staging_bindings DROP COLUMN IF EXISTS source_type;

ALTER TABLE public.message_catalog_changes
    DROP COLUMN IF EXISTS requested_by_name,
    DROP COLUMN IF EXISTS reviewed_by_name;

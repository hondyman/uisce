-- Leaves message_sets / message_catalog / tenant_message_catalog in place:
-- they predate this migration on existing environments.
DROP TABLE IF EXISTS public.message_catalog_changes;
DROP INDEX IF EXISTS public.idx_tenant_message_catalog_key;
ALTER TABLE public.tenant_message_catalog
    DROP COLUMN IF EXISTS user_action,
    DROP COLUMN IF EXISTS updated_by;
ALTER TABLE public.message_catalog
    DROP COLUMN IF EXISTS user_action,
    DROP COLUMN IF EXISTS updated_by;

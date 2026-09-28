-- Control plane (alpha): classify staging binding targets as COLUMN or JSON_PATH.
-- JSON_PATH field values encode the target expression in fields JSONB, e.g.
--   "trustee_ids": "custom_attributes->>'trustee_ids'"
-- COLUMN values remain plain staging column names, e.g. "account_cd": "account_cd"

ALTER TABLE public.staging_bindings
    ADD COLUMN IF NOT EXISTS source_type varchar(20) NOT NULL DEFAULT 'COLUMN';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'staging_bindings_source_type_check'
    ) THEN
        ALTER TABLE public.staging_bindings
            ADD CONSTRAINT staging_bindings_source_type_check
            CHECK (source_type IN ('COLUMN', 'JSON_PATH'));
    END IF;
END $$;

COMMENT ON COLUMN public.staging_bindings.source_type IS
    'COLUMN = fields map to staging columns; JSON_PATH = fields map to JSONB path expressions on a staging column (typically custom_attributes->>key)';

ALTER TABLE public.api_keys
    ADD COLUMN IF NOT EXISTS user_id text,
    ADD COLUMN IF NOT EXISTS roles text[] DEFAULT '{}'::text[] NOT NULL,
    ADD COLUMN IF NOT EXISTS tenant_ids text[] DEFAULT '{}'::text[] NOT NULL;

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON public.api_keys (user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_roles_gin ON public.api_keys USING gin (roles);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_ids_gin ON public.api_keys USING gin (tenant_ids);


-- Create base table if it doesn't exist
CREATE TABLE IF NOT EXISTS public.tenant_datasources (
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    PRIMARY KEY (tenant_id, datasource_id)
);



ALTER TABLE public.tenant_datasources
    ADD COLUMN IF NOT EXISTS resource_group TEXT,
    ADD COLUMN IF NOT EXISTS schema_override_repo TEXT,
    ADD COLUMN IF NOT EXISTS schema_override_branch TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS provisioning_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS last_provisioned_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_provision_error TEXT;

CREATE TABLE IF NOT EXISTS public.tenant_provision_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    triggered_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tenant_provision_jobs_scope_fk FOREIGN KEY (tenant_id, datasource_id)
        REFERENCES public.tenant_datasources(tenant_id, datasource_id) ON DELETE CASCADE,
    CONSTRAINT tenant_provision_jobs_unique UNIQUE (tenant_id, datasource_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_datasources_resource_group
    ON public.tenant_datasources(resource_group);

CREATE INDEX IF NOT EXISTS idx_tenant_provision_jobs_status
    ON public.tenant_provision_jobs(status);

CREATE INDEX IF NOT EXISTS idx_tenant_provision_jobs_updated_at
    ON public.tenant_provision_jobs(updated_at);



-- Create public.bp_process_definition table to store compiled Flow Builder DAGs and their versioned DynamicBP steps
CREATE TABLE IF NOT EXISTS public.bp_process_definition (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    process_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    name TEXT NOT NULL,
    description TEXT,
    steps_json JSONB NOT NULL,
    graph_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT bp_process_definition_tenant_version_uniq UNIQUE (process_id, tenant_id, version)
);

CREATE INDEX IF NOT EXISTS idx_bp_process_definition_lookup 
ON public.bp_process_definition (process_id, tenant_id, version DESC);

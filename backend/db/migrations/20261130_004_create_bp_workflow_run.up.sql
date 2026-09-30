-- 20261130_004_create_bp_workflow_run.up.sql
-- Table to store business-facing workflow execution run history
SET search_path = public;

CREATE TABLE IF NOT EXISTS public.bp_workflow_run (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    run_id VARCHAR(255) NOT NULL UNIQUE,
    tenant_id VARCHAR(255) NOT NULL,
    process_id VARCHAR(255) NOT NULL,
    process_name VARCHAR(255),
    trigger_type VARCHAR(50) NOT NULL,
    trigger_name VARCHAR(255),
    entity VARCHAR(100),
    entity_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
    input_payload JSONB,
    output_payload JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bp_workflow_run_tenant_entity 
    ON public.bp_workflow_run (tenant_id, entity, entity_id);

CREATE INDEX IF NOT EXISTS idx_bp_workflow_run_tenant_status 
    ON public.bp_workflow_run (tenant_id, status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_bp_workflow_run_tenant_process 
    ON public.bp_workflow_run (tenant_id, process_id, started_at DESC);

GRANT ALL PRIVILEGES ON TABLE public.bp_workflow_run TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.bp_workflow_run TO app_user;

-- Multi-Tenant Isolation & Global Scoping for Scheduler
-- Phase 10 of Scheduler Intelligence Layer

-- Update scheduled_jobs
ALTER TABLE scheduled_jobs ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE scheduled_jobs ADD COLUMN scope VARCHAR(16) NOT NULL DEFAULT 'TENANT';
ALTER TABLE scheduled_jobs ADD COLUMN parent_job_id UUID REFERENCES scheduled_jobs(id);
CREATE INDEX idx_scheduled_jobs_scope ON scheduled_jobs(scope);
CREATE INDEX idx_scheduled_jobs_parent ON scheduled_jobs(parent_job_id);

-- Add constraint to ensure scope/tenant_id consistency
ALTER TABLE scheduled_jobs ADD CONSTRAINT check_job_scope_tenant 
    CHECK ((scope = 'GLOBAL' AND tenant_id IS NULL) OR (scope = 'TENANT' AND tenant_id IS NOT NULL));

-- Ensure global job names are unique
ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS name VARCHAR(255) DEFAULT 'unnamed';
CREATE UNIQUE INDEX idx_scheduled_jobs_global_name_unique ON scheduled_jobs(name) WHERE scope = 'GLOBAL';

-- Update scheduled_dags
ALTER TABLE scheduled_dags ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE scheduled_dags ADD COLUMN scope VARCHAR(16) NOT NULL DEFAULT 'TENANT';
ALTER TABLE scheduled_dags ADD COLUMN parent_dag_id UUID REFERENCES scheduled_dags(id);
CREATE INDEX idx_scheduled_dags_scope ON scheduled_dags(scope);
CREATE INDEX idx_scheduled_dags_parent ON scheduled_dags(parent_dag_id);

-- Add constraint to ensure scope/tenant_id consistency
ALTER TABLE scheduled_dags ADD CONSTRAINT check_dag_scope_tenant 
    CHECK ((scope = 'GLOBAL' AND tenant_id IS NULL) OR (scope = 'TENANT' AND tenant_id IS NOT NULL));

-- Ensure global DAG names are unique
ALTER TABLE scheduled_dags ADD COLUMN IF NOT EXISTS name VARCHAR(255) DEFAULT 'unnamed';
CREATE UNIQUE INDEX idx_scheduled_dags_global_name_unique ON scheduled_dags(name) WHERE scope = 'GLOBAL';

-- Add tenant_id to job_runs (ensure it's indexed correctly for filtering)
-- It already exists, but let's ensure it's used correctly in context
-- The spec didn't mention adding scope to runs, as runs are always instance-specific.

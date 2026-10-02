-- Create public.bp_approval_task table to track human-in-the-loop task assignments and decisions
CREATE TABLE IF NOT EXISTS public.bp_approval_task (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    step_id TEXT NOT NULL,
    step_name TEXT,
    signal_name TEXT NOT NULL,
    assignee_role TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT,
    comment TEXT
);

CREATE INDEX IF NOT EXISTS idx_bp_approval_task_tenant_status 
ON public.bp_approval_task (tenant_id, status, assignee_role);

CREATE INDEX IF NOT EXISTS idx_bp_approval_task_wf 
ON public.bp_approval_task (workflow_id, run_id);

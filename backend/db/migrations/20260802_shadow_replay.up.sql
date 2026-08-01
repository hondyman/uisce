CREATE TYPE shadow_job_status_enum AS ENUM ('RUNNING', 'COMPLETED', 'CANCELLED', 'PROMOTED');

CREATE TABLE IF NOT EXISTS public.shadow_replay_jobs (
    job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    draft_rule_id UUID NOT NULL,
    rule_name VARCHAR(150) NOT NULL,
    rule_node_payload JSONB NOT NULL,
    status shadow_job_status_enum NOT NULL DEFAULT 'RUNNING',
    total_evaluated INT DEFAULT 0,
    prod_passed_count INT DEFAULT 0,
    shadow_passed_count INT DEFAULT 0,
    hard_block_count INT DEFAULT 0,
    soft_warn_count INT DEFAULT 0,
    discrepancy_count INT DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    completed_at TIMESTAMPTZ,
    created_by VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS public.shadow_replay_diffs (
    diff_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    job_id UUID NOT NULL REFERENCES public.shadow_replay_jobs(job_id) ON DELETE CASCADE,
    external_trade_id VARCHAR(255) NOT NULL,
    prod_result BOOLEAN NOT NULL,
    shadow_result BOOLEAN NOT NULL,
    evaluated_val NUMERIC,
    threshold_limit NUMERIC,
    breach_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_shadow_jobs_lookup ON public.shadow_replay_jobs(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_shadow_diffs_job ON public.shadow_replay_diffs(job_id);

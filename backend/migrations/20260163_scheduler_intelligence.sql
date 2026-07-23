-- Scheduler Intelligence Layer Schema
-- Epic 31: Autonomous Scheduler & Orchestration Engine

-- ============================================================================
-- Scheduled Jobs
-- ============================================================================

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL, -- report, workflow, integration, ai, preagg, compliance, data_quality, migration
    job_type VARCHAR(100) NOT NULL,
    parameters JSONB DEFAULT '{}',
    semantic_bindings JSONB DEFAULT '[]', -- [{type: 'business_object', id: '...'}, {type: 'api', id: '...'}]
    
    -- Scheduling
    schedule_type VARCHAR(20) NOT NULL, -- cron, event, predictive, manual
    cron_expression VARCHAR(100),
    event_trigger JSONB, -- {event_type: 'data_arrival', conditions: {...}}
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Calendars & Constraints
    calendar_ids UUID[] DEFAULT '{}',
    blackout_windows JSONB DEFAULT '[]', -- [{start: '...', end: '...', reason: '...'}]
    constraints JSONB DEFAULT '{}',
    
    -- Execution Config
    retry_policy JSONB DEFAULT '{"max_attempts": 3, "initial_interval_seconds": 60, "backoff_coefficient": 2}',
    timeout_seconds INT DEFAULT 3600,
    priority INT DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    
    -- Risk & Compliance
    risk_score INT DEFAULT 0 CHECK (risk_score >= 0 AND risk_score <= 100),
    slo_critical BOOLEAN DEFAULT FALSE,
    compliance_tags TEXT[] DEFAULT '{}',
    pii_exposure_level VARCHAR(20) DEFAULT 'none', -- none, low, medium, high
    residency_rules JSONB DEFAULT '{}',
    
    -- Governance
    changeset_id UUID,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    
    -- Audit
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT scheduled_jobs_tenant_name_unique UNIQUE (tenant_id, name)
);

-- ============================================================================
-- Scheduled DAGs (Directed Acyclic Graphs)
-- ============================================================================

CREATE TABLE IF NOT EXISTS scheduled_dags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    
    -- Graph Structure
    nodes JSONB NOT NULL DEFAULT '[]', -- [{id, job_id, conditions, position}]
    edges JSONB NOT NULL DEFAULT '[]', -- [{from_node_id, to_node_id, type, conditions}]
    
    -- Scheduling (inherited by jobs if not overridden)
    schedule_type VARCHAR(20),
    cron_expression VARCHAR(100),
    calendar_ids UUID[] DEFAULT '{}',
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Execution Config
    max_parallel_jobs INT DEFAULT 5,
    fail_fast BOOLEAN DEFAULT FALSE, -- Stop entire DAG on first failure
    timeout_seconds INT DEFAULT 7200,
    
    -- Risk & Governance
    risk_score INT DEFAULT 0,
    slo_critical BOOLEAN DEFAULT FALSE,
    changeset_id UUID,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    
    -- Audit
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT scheduled_dags_tenant_name_unique UNIQUE (tenant_id, name)
);

-- ============================================================================
-- DAG Runs
-- ============================================================================

CREATE TABLE IF NOT EXISTS dag_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dag_id UUID NOT NULL REFERENCES scheduled_dags(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    
    -- Temporal Integration
    temporal_workflow_id VARCHAR(255),
    temporal_run_id VARCHAR(255),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled, paused
    trigger_type VARCHAR(20) NOT NULL, -- scheduled, manual, event, api
    triggered_by UUID,
    
    -- Timing
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INT,
    
    -- Results
    completed_jobs INT DEFAULT 0,
    failed_jobs INT DEFAULT 0,
    skipped_jobs INT DEFAULT 0,
    error_message TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- Job Runs
-- ============================================================================

CREATE TABLE IF NOT EXISTS job_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE,
    dag_run_id UUID REFERENCES dag_runs(id) ON DELETE SET NULL,
    tenant_id UUID NOT NULL,
    
    -- Temporal Integration
    temporal_workflow_id VARCHAR(255),
    temporal_run_id VARCHAR(255),
    task_queue VARCHAR(100),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled, retrying
    attempt_number INT DEFAULT 1,
    trigger_type VARCHAR(20) NOT NULL,
    triggered_by UUID,
    
    -- Timing
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INT,
    
    -- Results
    result JSONB,
    error_message TEXT,
    error_details JSONB,
    
    -- SLO Tracking
    slo_target_ms INT,
    slo_breached BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    input_parameters JSONB,
    output_artifacts JSONB DEFAULT '[]',
    logs_url TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- Job Dependencies (for DAG-less dependencies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS job_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE,
    depends_on_job_id UUID NOT NULL REFERENCES scheduled_jobs(id) ON DELETE CASCADE,
    dependency_type VARCHAR(20) DEFAULT 'success', -- success, completion, any
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT job_dependencies_unique UNIQUE (job_id, depends_on_job_id),
    CONSTRAINT job_dependencies_no_self CHECK (job_id != depends_on_job_id)
);

-- ============================================================================
-- AI Suggestions
-- ============================================================================

CREATE TABLE IF NOT EXISTS scheduler_ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    suggestion_type VARCHAR(50) NOT NULL, -- schedule_optimization, dag_optimization, new_job, risk_alert
    target_type VARCHAR(20), -- job, dag, calendar
    target_id UUID,
    
    -- Suggestion Content
    title VARCHAR(255) NOT NULL,
    description TEXT,
    impact_summary TEXT,
    risk_level VARCHAR(20) DEFAULT 'low',
    affected_tenants UUID[] DEFAULT '{}',
    
    -- Proposed Changes
    proposed_changes JSONB NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, dismissed, snoozed
    dismissed_reason TEXT,
    snoozed_until TIMESTAMPTZ,
    
    -- If accepted, the resulting changeset
    changeset_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- Indexes
-- ============================================================================

-- Ensure status & scheduling columns exist before creating partial indexes (idempotent)
ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE scheduled_dags ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ;
ALTER TABLE scheduled_dags ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ;
ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS category VARCHAR(50) DEFAULT 'general';

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_tenant ON scheduled_jobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_active ON scheduled_jobs(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run ON scheduled_jobs(next_run_at) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_category ON scheduled_jobs(tenant_id, category);

CREATE INDEX IF NOT EXISTS idx_scheduled_dags_tenant ON scheduled_dags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_dags_active ON scheduled_dags(is_active) WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_dag_runs_dag ON dag_runs(dag_id);
CREATE INDEX IF NOT EXISTS idx_dag_runs_tenant ON dag_runs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dag_runs_status ON dag_runs(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_job_runs_job ON job_runs(job_id);
CREATE INDEX IF NOT EXISTS idx_job_runs_dag_run ON job_runs(dag_run_id);
CREATE INDEX IF NOT EXISTS idx_job_runs_tenant ON job_runs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_runs_temporal ON job_runs(temporal_workflow_id);

CREATE INDEX IF NOT EXISTS idx_scheduler_ai_suggestions_tenant ON scheduler_ai_suggestions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scheduler_ai_suggestions_status ON scheduler_ai_suggestions(status) WHERE status = 'pending';

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_scheduler_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tr_scheduled_jobs_updated ON scheduled_jobs;
CREATE TRIGGER tr_scheduled_jobs_updated
    BEFORE UPDATE ON scheduled_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_scheduler_updated_at();

DROP TRIGGER IF EXISTS tr_scheduled_dags_updated ON scheduled_dags;
CREATE TRIGGER tr_scheduled_dags_updated
    BEFORE UPDATE ON scheduled_dags
    FOR EACH ROW
    EXECUTE FUNCTION update_scheduler_updated_at();

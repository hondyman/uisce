-- Premium Cube Worker Configuration Schema
-- Extends cube_worker_schema.sql with production-ready features

-- Worker pool tier definitions
INSERT INTO cube_worker_pools (id, name, display_name, description, tier, min_workers, max_workers, 
    memory_limit_mb, cpu_limit_cores, concurrent_jobs, auto_scale_enabled, 
    scale_up_threshold, scale_down_threshold, scale_cooldown_seconds, status)
VALUES
    -- Standard tier - shared worker pool
    ('10000000-0000-0000-0000-000000000001'::uuid, 'standard_pool', 'Standard Workers', 
     'Shared worker pool for standard tier tenants', 'standard', 
     1, 4, 2048, 2.0, 2, true, 0.75, 0.25, 300, 'active'),
    
    -- Professional tier - dedicated capacity
    ('10000000-0000-0000-0000-000000000002'::uuid, 'professional_pool', 'Professional Workers',
     'Dedicated worker pool for professional tier tenants', 'professional',
     2, 8, 4096, 4.0, 4, true, 0.80, 0.30, 180, 'active'),
    
    -- Enterprise tier - isolated high-performance
    ('10000000-0000-0000-0000-000000000003'::uuid, 'enterprise_pool', 'Enterprise Workers',
     'Isolated high-performance workers for enterprise tenants', 'enterprise',
     4, 16, 8192, 8.0, 8, true, 0.85, 0.35, 120, 'active'),
    
    -- Priority tier - low-latency dedicated
    ('10000000-0000-0000-0000-000000000004'::uuid, 'priority_pool', 'Priority Workers',
     'Low-latency workers for critical pre-aggregations', 'priority',
     2, 12, 16384, 16.0, 16, true, 0.70, 0.20, 60, 'active')
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    min_workers = EXCLUDED.min_workers,
    max_workers = EXCLUDED.max_workers,
    memory_limit_mb = EXCLUDED.memory_limit_mb,
    cpu_limit_cores = EXCLUDED.cpu_limit_cores;

-- Pre-aggregation job priority rules
CREATE TABLE IF NOT EXISTS cube_preagg_priority_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    tenant_tier VARCHAR(50), -- Match tenant tier
    cube_pattern VARCHAR(255), -- Regex pattern for cube name matching
    preagg_pattern VARCHAR(255), -- Regex pattern for pre-agg name matching
    priority INT NOT NULL DEFAULT 50, -- 0-100, higher = more priority
    worker_pool_id UUID REFERENCES cube_worker_pools(id),
    max_execution_time_mins INT DEFAULT 60,
    retry_backoff_multiplier NUMERIC(3,1) DEFAULT 2.0,
    max_partitions_per_job INT DEFAULT 10,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default priority rules
INSERT INTO cube_preagg_priority_rules (name, description, tenant_tier, cube_pattern, priority, max_execution_time_mins)
VALUES
    ('enterprise_critical', 'Critical enterprise pre-aggregations', 'enterprise', '.*_critical$', 100, 120),
    ('enterprise_default', 'Default enterprise pre-aggregations', 'enterprise', '.*', 80, 90),
    ('professional_critical', 'Critical professional pre-aggregations', 'professional', '.*_critical$', 75, 60),
    ('professional_default', 'Default professional pre-aggregations', 'professional', '.*', 60, 45),
    ('standard_default', 'Default standard pre-aggregations', 'standard', '.*', 40, 30),
    ('background_rebuild', 'Low-priority full rebuild jobs', NULL, '.*_rebuild$', 10, 180)
ON CONFLICT (name) DO NOTHING;

-- Pre-aggregation refresh schedules
CREATE TABLE IF NOT EXISTS cube_preagg_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    preagg_definition_id UUID REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    schedule_type VARCHAR(50) NOT NULL DEFAULT 'cron', -- cron, interval, event
    cron_expression VARCHAR(100), -- For cron type
    interval_minutes INT, -- For interval type
    event_trigger VARCHAR(255), -- For event type (e.g., webhook, queue message)
    timezone VARCHAR(50) DEFAULT 'UTC',
    enabled BOOLEAN DEFAULT true,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    last_run_status VARCHAR(50),
    last_run_duration_ms BIGINT,
    consecutive_failures INT DEFAULT 0,
    max_consecutive_failures INT DEFAULT 3,
    pause_on_failure BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_preagg_schedules_next_run ON cube_preagg_schedules(next_run_at) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_preagg_schedules_tenant ON cube_preagg_schedules(tenant_id, datasource_id);

-- Pre-aggregation dependencies for DAG-based builds
CREATE TABLE IF NOT EXISTS cube_preagg_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preagg_id UUID NOT NULL REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    depends_on_preagg_id UUID NOT NULL REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    dependency_type VARCHAR(50) DEFAULT 'data', -- data, schema, soft
    wait_for_completion BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_preagg_dependency UNIQUE (preagg_id, depends_on_preagg_id)
);

-- Worker performance baselines for autoscaling
CREATE TABLE IF NOT EXISTS cube_worker_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_pool_id UUID NOT NULL REFERENCES cube_worker_pools(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL, -- throughput, latency, memory, cpu
    time_bucket VARCHAR(20) NOT NULL, -- hour, day, week
    bucket_key VARCHAR(50) NOT NULL, -- e.g., '14' for 2pm, 'monday', 'week_1'
    avg_value NUMERIC(15,4),
    p50_value NUMERIC(15,4),
    p95_value NUMERIC(15,4),
    p99_value NUMERIC(15,4),
    sample_count INT DEFAULT 0,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_worker_baseline UNIQUE (worker_pool_id, metric_type, time_bucket, bucket_key)
);

-- Autoscaling decisions log
CREATE TABLE IF NOT EXISTS cube_autoscale_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_pool_id UUID NOT NULL REFERENCES cube_worker_pools(id),
    decision_type VARCHAR(50) NOT NULL, -- scale_up, scale_down, no_change
    trigger_reason VARCHAR(255), -- queue_depth, cpu_usage, memory_usage, scheduled
    current_workers INT,
    target_workers INT,
    metrics_snapshot JSONB, -- Snapshot of metrics at decision time
    executed BOOLEAN DEFAULT false,
    execution_result VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_autoscale_decisions_pool ON cube_autoscale_decisions(worker_pool_id, created_at DESC);

-- Pre-aggregation build statistics (for analytics)
CREATE TABLE IF NOT EXISTS cube_preagg_build_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preagg_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    job_id UUID REFERENCES cube_preagg_jobs(id),
    partition_key VARCHAR(100),
    build_type VARCHAR(50), -- full, incremental, partition
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    rows_processed BIGINT DEFAULT 0,
    bytes_written BIGINT DEFAULT 0,
    source_query_count INT DEFAULT 0,
    source_query_time_ms BIGINT DEFAULT 0,
    storage_engine VARCHAR(50),
    worker_pool_id UUID,
    worker_id UUID,
    success BOOLEAN,
    error_message TEXT,
    build_metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_build_stats_preagg ON cube_preagg_build_stats(preagg_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_build_stats_tenant ON cube_preagg_build_stats(tenant_id, datasource_id, started_at DESC);

-- Function to calculate next run time based on cron expression
CREATE OR REPLACE FUNCTION calculate_next_cron_run(
    p_cron_expression VARCHAR,
    p_timezone VARCHAR DEFAULT 'UTC',
    p_after TIMESTAMPTZ DEFAULT NOW()
) RETURNS TIMESTAMPTZ AS $$
DECLARE
    v_next_run TIMESTAMPTZ;
BEGIN
    -- Simple implementation - in production use pg_cron or external scheduler
    -- For now, just add 1 hour as placeholder
    v_next_run := p_after + INTERVAL '1 hour';
    RETURN v_next_run AT TIME ZONE p_timezone;
END;
$$ LANGUAGE plpgsql;

-- Function to assign job to optimal worker pool based on priority rules
CREATE OR REPLACE FUNCTION assign_job_worker_pool(
    p_tenant_id UUID,
    p_tenant_tier VARCHAR,
    p_cube_name VARCHAR,
    p_preagg_name VARCHAR
) RETURNS TABLE (
    worker_pool_id UUID,
    priority INT,
    max_execution_time_mins INT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(r.worker_pool_id, 
            CASE p_tenant_tier
                WHEN 'enterprise' THEN '10000000-0000-0000-0000-000000000003'::uuid
                WHEN 'professional' THEN '10000000-0000-0000-0000-000000000002'::uuid
                ELSE '10000000-0000-0000-0000-000000000001'::uuid
            END
        ) as worker_pool_id,
        COALESCE(r.priority, 50) as priority,
        COALESCE(r.max_execution_time_mins, 60) as max_execution_time_mins
    FROM cube_preagg_priority_rules r
    WHERE r.enabled = true
      AND (r.tenant_tier IS NULL OR r.tenant_tier = p_tenant_tier)
      AND (r.cube_pattern IS NULL OR p_cube_name ~ r.cube_pattern)
      AND (r.preagg_pattern IS NULL OR p_preagg_name ~ r.preagg_pattern)
    ORDER BY r.priority DESC
    LIMIT 1;
    
    -- Return default if no rules match
    IF NOT FOUND THEN
        RETURN QUERY
        SELECT 
            '10000000-0000-0000-0000-000000000001'::uuid as worker_pool_id,
            50 as priority,
            60 as max_execution_time_mins;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update schedule next_run_at when enabled
CREATE OR REPLACE FUNCTION update_schedule_next_run() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.enabled = true AND NEW.schedule_type = 'cron' AND NEW.cron_expression IS NOT NULL THEN
        NEW.next_run_at := calculate_next_cron_run(NEW.cron_expression, NEW.timezone, NOW());
    ELSIF NEW.enabled = true AND NEW.schedule_type = 'interval' AND NEW.interval_minutes IS NOT NULL THEN
        NEW.next_run_at := NOW() + (NEW.interval_minutes * INTERVAL '1 minute');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_schedule_next_run
    BEFORE INSERT OR UPDATE OF enabled, cron_expression, interval_minutes ON cube_preagg_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_schedule_next_run();

-- View for active job queue with priority
CREATE OR REPLACE VIEW cube_active_job_queue AS
SELECT 
    j.id,
    j.preagg_id,
    j.tenant_id,
    j.datasource_id,
    d.cube_name,
    d.preagg_name,
    j.job_type,
    j.partition_key,
    j.priority,
    j.status,
    j.progress_percent,
    j.scheduled_at,
    j.queued_at,
    j.started_at,
    j.retry_count,
    j.max_retries,
    wp.name as worker_pool_name,
    wp.tier as worker_pool_tier,
    wi.hostname as assigned_worker,
    CASE 
        WHEN j.status = 'running' THEN EXTRACT(EPOCH FROM (NOW() - j.started_at))
        ELSE NULL
    END as running_seconds
FROM cube_preagg_jobs j
LEFT JOIN cube_preagg_definitions d ON j.preagg_id = d.id
LEFT JOIN cube_worker_pools wp ON j.worker_pool_id = wp.id
LEFT JOIN cube_worker_instances wi ON j.assigned_worker_id = wi.id
WHERE j.status IN ('pending', 'queued', 'running')
ORDER BY j.priority DESC, j.scheduled_at ASC;

-- View for worker pool health summary
CREATE OR REPLACE VIEW cube_worker_pool_health AS
SELECT 
    wp.id,
    wp.name,
    wp.display_name,
    wp.tier,
    wp.status,
    wp.current_workers,
    wp.target_workers,
    wp.min_workers,
    wp.max_workers,
    COUNT(wi.id) FILTER (WHERE wi.status = 'idle') as idle_workers,
    COUNT(wi.id) FILTER (WHERE wi.status = 'busy') as busy_workers,
    COUNT(wi.id) FILTER (WHERE wi.status = 'unhealthy') as unhealthy_workers,
    AVG(wi.cpu_used_percent) as avg_cpu_percent,
    AVG(wi.memory_used_mb) as avg_memory_mb,
    SUM(wi.jobs_completed) as total_jobs_completed,
    SUM(wi.jobs_failed) as total_jobs_failed,
    COUNT(j.id) FILTER (WHERE j.status = 'pending') as pending_jobs,
    COUNT(j.id) FILTER (WHERE j.status = 'running') as running_jobs,
    wp.auto_scale_enabled,
    wp.last_scale_at,
    wp.health_check_at
FROM cube_worker_pools wp
LEFT JOIN cube_worker_instances wi ON wp.id = wi.pool_id
LEFT JOIN cube_preagg_jobs j ON wp.id = j.worker_pool_id AND j.status IN ('pending', 'running')
GROUP BY wp.id;

COMMENT ON VIEW cube_active_job_queue IS 'Active pre-aggregation job queue with priority ordering';
COMMENT ON VIEW cube_worker_pool_health IS 'Worker pool health summary with metrics';
COMMENT ON TABLE cube_preagg_priority_rules IS 'Priority rules for pre-aggregation job scheduling';
COMMENT ON TABLE cube_preagg_schedules IS 'Refresh schedules for pre-aggregation definitions';
COMMENT ON TABLE cube_worker_baselines IS 'Historical performance baselines for autoscaling decisions';

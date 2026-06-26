-- Cube Worker & Pre-aggregation Premium Features Schema
-- Extends cube_admin_schema.sql with worker management tables

-- Worker Pools (Enterprise feature: dedicated workers per tenant tier)
CREATE TABLE IF NOT EXISTS cube_worker_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    tier VARCHAR(50) NOT NULL DEFAULT 'standard', -- enterprise, standard, starter
    
    -- Capacity settings
    min_workers INT DEFAULT 1,
    max_workers INT DEFAULT 4,
    current_workers INT DEFAULT 1,
    target_workers INT DEFAULT 1,
    
    -- Resource limits
    memory_limit_mb INT DEFAULT 2048,
    cpu_limit_cores DECIMAL(4,2) DEFAULT 2.0,
    concurrent_jobs INT DEFAULT 4,
    queue_size INT DEFAULT 100,
    
    -- Auto-scaling settings
    auto_scale_enabled BOOLEAN DEFAULT FALSE,
    scale_up_threshold DECIMAL(5,2) DEFAULT 80.0, -- Queue utilization %
    scale_down_threshold DECIMAL(5,2) DEFAULT 20.0,
    scale_cooldown_seconds INT DEFAULT 300,
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, draining, stopped, error
    last_scale_at TIMESTAMPTZ,
    health_check_at TIMESTAMPTZ,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_worker_pools_tier ON cube_worker_pools(tier);
CREATE INDEX idx_worker_pools_status ON cube_worker_pools(status);

-- Worker Instances
CREATE TABLE IF NOT EXISTS cube_worker_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES cube_worker_pools(id) ON DELETE CASCADE,
    instance_id VARCHAR(255) NOT NULL, -- Container/pod ID
    hostname VARCHAR(255),
    ip_address INET,
    
    -- Status
    status VARCHAR(50) DEFAULT 'starting', -- starting, running, busy, idle, draining, stopped, error
    current_job_id UUID,
    jobs_completed INT DEFAULT 0,
    jobs_failed INT DEFAULT 0,
    
    -- Resources
    memory_used_mb INT DEFAULT 0,
    cpu_used_percent DECIMAL(5,2) DEFAULT 0,
    
    -- Timing
    started_at TIMESTAMPTZ DEFAULT NOW(),
    last_heartbeat_at TIMESTAMPTZ DEFAULT NOW(),
    last_job_at TIMESTAMPTZ,
    
    metadata JSONB DEFAULT '{}',
    
    UNIQUE(pool_id, instance_id)
);

CREATE INDEX idx_worker_instances_pool ON cube_worker_instances(pool_id, status);
CREATE INDEX idx_worker_instances_heartbeat ON cube_worker_instances(last_heartbeat_at);

-- Pre-aggregation Definitions (enhanced)
CREATE TABLE IF NOT EXISTS cube_preagg_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    cube_name VARCHAR(255) NOT NULL,
    preagg_name VARCHAR(255) NOT NULL,
    
    -- Definition
    measures TEXT[] DEFAULT '{}',
    dimensions TEXT[] DEFAULT '{}',
    time_dimension VARCHAR(255),
    granularity VARCHAR(50),
    partition_granularity VARCHAR(50),
    
    -- Schedule
    refresh_key JSONB,
    scheduled_refresh BOOLEAN DEFAULT TRUE,
    refresh_cron VARCHAR(100),
    refresh_interval_minutes INT,
    refresh_timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Storage
    external_storage BOOLEAN DEFAULT TRUE,
    storage_engine VARCHAR(50) DEFAULT 'starrocks', -- starrocks, cube_store, redis
    table_name VARCHAR(255),
    
    -- Indexes and optimization
    indexes JSONB DEFAULT '[]',
    build_range_start TIMESTAMPTZ,
    build_range_end TIMESTAMPTZ,
    
    -- Priority
    priority INT DEFAULT 5, -- 1-10, higher = more important
    worker_pool_id UUID REFERENCES cube_worker_pools(id),
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, paused, building, error, deprecated
    last_build_at TIMESTAMPTZ,
    last_build_duration_ms BIGINT,
    last_build_rows BIGINT,
    last_error TEXT,
    
    -- Metadata
    yaml_definition TEXT,
    metadata JSONB DEFAULT '{}',
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tenant_id, datasource_id, cube_name, preagg_name)
);

CREATE INDEX idx_preagg_def_tenant ON cube_preagg_definitions(tenant_id, datasource_id);
CREATE INDEX idx_preagg_def_cube ON cube_preagg_definitions(cube_name);
CREATE INDEX idx_preagg_def_status ON cube_preagg_definitions(status);
CREATE INDEX idx_preagg_def_schedule ON cube_preagg_definitions(scheduled_refresh, status) WHERE scheduled_refresh = TRUE;

-- Pre-aggregation Jobs (build queue)
CREATE TABLE IF NOT EXISTS cube_preagg_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preagg_id UUID NOT NULL REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Job type
    job_type VARCHAR(50) NOT NULL DEFAULT 'refresh', -- refresh, build, rebuild, partition, cleanup
    partition_key VARCHAR(255), -- For incremental partition builds
    
    -- Priority & assignment
    priority INT DEFAULT 5,
    worker_pool_id UUID REFERENCES cube_worker_pools(id),
    assigned_worker_id UUID REFERENCES cube_worker_instances(id),
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, queued, running, completed, failed, cancelled, timeout
    progress_percent INT DEFAULT 0,
    current_step VARCHAR(255),
    
    -- Timing
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    queued_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    timeout_at TIMESTAMPTZ,
    
    -- Results
    rows_processed BIGINT DEFAULT 0,
    bytes_written BIGINT DEFAULT 0,
    duration_ms BIGINT,
    
    -- Errors
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    error_message TEXT,
    error_stack TEXT,
    
    -- Metadata
    build_options JSONB DEFAULT '{}',
    result_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_preagg_jobs_preagg ON cube_preagg_jobs(preagg_id, status);
CREATE INDEX idx_preagg_jobs_tenant ON cube_preagg_jobs(tenant_id, status);
CREATE INDEX idx_preagg_jobs_queue ON cube_preagg_jobs(status, priority DESC, scheduled_at) WHERE status IN ('pending', 'queued');
CREATE INDEX idx_preagg_jobs_worker ON cube_preagg_jobs(assigned_worker_id, status) WHERE status = 'running';
CREATE INDEX idx_preagg_jobs_time ON cube_preagg_jobs(created_at DESC);

-- Pre-aggregation Partitions (for incremental refresh)
CREATE TABLE IF NOT EXISTS cube_preagg_partitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preagg_id UUID NOT NULL REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    partition_key VARCHAR(255) NOT NULL, -- e.g., "2024-01", "2024-01-15"
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, building, ready, stale, error
    
    -- Storage
    table_name VARCHAR(255),
    row_count BIGINT DEFAULT 0,
    size_bytes BIGINT DEFAULT 0,
    
    -- Freshness
    data_from TIMESTAMPTZ,
    data_to TIMESTAMPTZ,
    built_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    refresh_key_value VARCHAR(255),
    
    -- Build info
    build_duration_ms BIGINT,
    last_error TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(preagg_id, partition_key)
);

CREATE INDEX idx_preagg_parts_preagg ON cube_preagg_partitions(preagg_id, status);
CREATE INDEX idx_preagg_parts_stale ON cube_preagg_partitions(status, expires_at) WHERE status = 'ready';

-- Worker Metrics (time-series telemetry)
CREATE TABLE IF NOT EXISTS cube_worker_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id UUID NOT NULL REFERENCES cube_worker_instances(id) ON DELETE CASCADE,
    pool_id UUID NOT NULL REFERENCES cube_worker_pools(id) ON DELETE CASCADE,
    
    -- Metrics
    jobs_active INT DEFAULT 0,
    jobs_queued INT DEFAULT 0,
    memory_used_mb INT DEFAULT 0,
    cpu_percent DECIMAL(5,2) DEFAULT 0,
    disk_io_mb_s DECIMAL(10,2) DEFAULT 0,
    network_io_mb_s DECIMAL(10,2) DEFAULT 0,
    
    -- Job metrics
    jobs_completed_1m INT DEFAULT 0,
    jobs_failed_1m INT DEFAULT 0,
    avg_job_duration_ms BIGINT DEFAULT 0,
    
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_worker_metrics_worker ON cube_worker_metrics(worker_id, recorded_at DESC);
CREATE INDEX idx_worker_metrics_pool ON cube_worker_metrics(pool_id, recorded_at DESC);
-- Partition by time for efficient cleanup
-- PARTITION BY RANGE (recorded_at) -- Enable in production

-- Refresh Key Cache (optimization for incremental refresh)
CREATE TABLE IF NOT EXISTS cube_refresh_key_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preagg_id UUID NOT NULL REFERENCES cube_preagg_definitions(id) ON DELETE CASCADE,
    partition_key VARCHAR(255),
    
    refresh_key_sql TEXT,
    refresh_key_value VARCHAR(500),
    computed_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(preagg_id, partition_key)
);

CREATE INDEX idx_refresh_cache_preagg ON cube_refresh_key_cache(preagg_id);

-- Insert default worker pools
INSERT INTO cube_worker_pools (id, name, display_name, tier, min_workers, max_workers, memory_limit_mb, cpu_limit_cores, concurrent_jobs)
VALUES 
    ('00000000-0000-0000-0001-000000000001', 'enterprise', 'Enterprise Pool', 'enterprise', 2, 8, 4096, 4.0, 8),
    ('00000000-0000-0000-0001-000000000002', 'standard', 'Standard Pool', 'standard', 1, 4, 2048, 2.0, 4),
    ('00000000-0000-0000-0001-000000000003', 'starter', 'Starter Pool', 'starter', 1, 2, 1024, 1.0, 2)
ON CONFLICT (name) DO NOTHING;

-- Views for monitoring

-- Active workers view
CREATE OR REPLACE VIEW v_cube_active_workers AS
SELECT 
    p.name AS pool_name,
    p.tier,
    p.current_workers,
    p.target_workers,
    COUNT(wi.id) AS registered_workers,
    COUNT(wi.id) FILTER (WHERE wi.status = 'running') AS healthy_workers,
    COUNT(wi.id) FILTER (WHERE wi.status = 'busy') AS busy_workers,
    SUM(wi.jobs_completed) AS total_jobs_completed,
    SUM(wi.jobs_failed) AS total_jobs_failed,
    AVG(wi.memory_used_mb) AS avg_memory_mb,
    AVG(wi.cpu_used_percent) AS avg_cpu_percent
FROM cube_worker_pools p
LEFT JOIN cube_worker_instances wi ON wi.pool_id = p.id
GROUP BY p.id, p.name, p.tier, p.current_workers, p.target_workers;

-- Job queue summary view
CREATE OR REPLACE VIEW v_cube_job_queue AS
SELECT 
    j.tenant_id,
    j.status,
    COUNT(*) AS job_count,
    AVG(j.priority) AS avg_priority,
    MIN(j.scheduled_at) AS oldest_job,
    AVG(EXTRACT(EPOCH FROM (NOW() - j.scheduled_at))) AS avg_wait_seconds
FROM cube_preagg_jobs j
WHERE j.status IN ('pending', 'queued', 'running')
GROUP BY j.tenant_id, j.status;

-- Pre-aggregation health view
CREATE OR REPLACE VIEW v_cube_preagg_health AS
SELECT 
    pd.tenant_id,
    pd.datasource_id,
    pd.cube_name,
    pd.preagg_name,
    pd.status,
    pd.scheduled_refresh,
    pd.last_build_at,
    pd.last_build_duration_ms,
    pd.last_build_rows,
    CASE 
        WHEN pd.last_build_at IS NULL THEN 'never_built'
        WHEN pd.status = 'error' THEN 'error'
        WHEN pd.last_build_at < NOW() - INTERVAL '1 day' THEN 'stale'
        ELSE 'healthy'
    END AS health_status,
    COUNT(pp.id) AS partition_count,
    COUNT(pp.id) FILTER (WHERE pp.status = 'ready') AS ready_partitions,
    SUM(pp.row_count) AS total_rows,
    SUM(pp.size_bytes) AS total_bytes
FROM cube_preagg_definitions pd
LEFT JOIN cube_preagg_partitions pp ON pp.preagg_id = pd.id
GROUP BY pd.id;

-- Comments
COMMENT ON TABLE cube_worker_pools IS 'Worker pool configurations for different tenant tiers';
COMMENT ON TABLE cube_worker_instances IS 'Active worker instances and their status';
COMMENT ON TABLE cube_preagg_definitions IS 'Pre-aggregation definitions with scheduling config';
COMMENT ON TABLE cube_preagg_jobs IS 'Pre-aggregation build job queue';
COMMENT ON TABLE cube_preagg_partitions IS 'Partition-level tracking for incremental refresh';
COMMENT ON TABLE cube_worker_metrics IS 'Worker telemetry time-series data';
COMMENT ON TABLE cube_refresh_key_cache IS 'Cached refresh key values for optimization';

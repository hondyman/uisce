-- Migration 009: Advanced Async Features - Exports and Scheduling
-- Date: February 21, 2026
-- Purpose: Add export and scheduling infrastructure for async jobs

-- ============================================================================
-- EXPORTS SCHEMA
-- ============================================================================

-- Export tracking table
CREATE TABLE IF NOT EXISTS edm.job_exports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  export_format VARCHAR(20) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  
  -- File storage information
  file_location TEXT,
  file_size BIGINT DEFAULT 0,
  record_count INT DEFAULT 0,
  
  -- Download/access information
  presigned_url TEXT,
  presigned_url_expires TIMESTAMP,
  download_count INT DEFAULT 0,
  
  -- Filter and configuration
  filter_criteria JSONB,
  include_errors BOOLEAN DEFAULT false,
  
  -- Audit trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  expires_at TIMESTAMP,
  
  -- Error handling
  error_message TEXT,
  
  CONSTRAINT fk_exports_job FOREIGN KEY (job_id) 
    REFERENCES edm.async_jobs(id) ON DELETE CASCADE,
  CONSTRAINT valid_export_format CHECK (export_format IN ('csv', 'json', 'parquet')),
  CONSTRAINT valid_export_status CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'expired'))
);

-- Performance indexes for exports
CREATE INDEX IF NOT EXISTS idx_exports_job_status
  ON edm.job_exports(job_id, status)
  WHERE status != 'completed';

CREATE INDEX IF NOT EXISTS idx_exports_tenant
  ON edm.job_exports(tenant_id)
  WHERE status = 'completed';

CREATE INDEX IF NOT EXISTS idx_exports_expires
  ON edm.job_exports(expires_at)
  WHERE status != 'expired';

-- Enable RLS on job_exports
ALTER TABLE edm.job_exports ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own exports
CREATE POLICY job_exports_tenant_isolation ON edm.job_exports
  USING (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID))
  WITH CHECK (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID));

-- ============================================================================
-- SCHEDULING SCHEMA
-- ============================================================================

-- Scheduled jobs table
CREATE TABLE IF NOT EXISTS edm.scheduled_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  
  -- Job template (what to run)
  job_template JSONB NOT NULL,
  
  -- Schedule configuration
  schedule_type VARCHAR(20) NOT NULL,  -- once, daily, weekly, monthly, cron
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP,
  cron_expression VARCHAR(255),
  timezone VARCHAR(50) DEFAULT 'UTC',
  
  -- Execution settings
  max_run_duration INT,  -- Max seconds allowed
  retry_on_failure BOOLEAN DEFAULT true,
  max_retries INT DEFAULT 3,
  
  -- Status
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  is_active BOOLEAN DEFAULT true,
  
  -- Execution tracking
  last_run_at TIMESTAMP,
  last_run_status VARCHAR(20),
  last_run_error TEXT,
  next_run_at TIMESTAMP,
  run_count INT DEFAULT 0,
  success_count INT DEFAULT 0,
  failure_count INT DEFAULT 0,
  
  -- Metadata
  name VARCHAR(500),
  description TEXT,
  priority INT DEFAULT 10,
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  CONSTRAINT valid_schedule_type CHECK (schedule_type IN ('once', 'daily', 'weekly', 'monthly', 'cron')),
  CONSTRAINT valid_sched_status CHECK (status IN ('active', 'paused', 'completed', 'failed', 'disabled')),
  CONSTRAINT valid_operation_type CHECK (operation_type IN ('bulk-create', 'bulk-publish', 'bulk-promote'))
);

-- Scheduled job run history
CREATE TABLE IF NOT EXISTS edm.scheduled_job_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  schedule_id UUID NOT NULL,
  job_id UUID,
  tenant_id UUID NOT NULL,
  
  -- Timing information
  scheduled_time TIMESTAMP NOT NULL,
  actual_start_time TIMESTAMP,
  actual_end_time TIMESTAMP,
  
  -- Status and results
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  error_message TEXT,
  result_summary JSONB,
  
  -- Created
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  CONSTRAINT fk_runs_schedule FOREIGN KEY (schedule_id) 
    REFERENCES edm.scheduled_jobs(id) ON DELETE CASCADE,
  CONSTRAINT fk_runs_job FOREIGN KEY (job_id) 
    REFERENCES edm.async_jobs(id) ON DELETE SET NULL,
  CONSTRAINT valid_run_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped'))
);

-- Performance indexes for scheduling
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run
  ON edm.scheduled_jobs(next_run_at)
  WHERE status = 'active' AND is_active = true;

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_tenant
  ON edm.scheduled_jobs(tenant_id)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_created
  ON edm.scheduled_jobs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_job_runs_schedule
  ON edm.scheduled_job_runs(schedule_id);

CREATE INDEX IF NOT EXISTS idx_job_runs_scheduled_time
  ON edm.scheduled_job_runs(scheduled_time DESC);

CREATE INDEX IF NOT EXISTS idx_job_runs_status
  ON edm.scheduled_job_runs(status)
  WHERE status IN ('pending', 'running');

-- Enable RLS on scheduled jobs
ALTER TABLE edm.scheduled_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE edm.scheduled_job_runs ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own scheduled jobs
CREATE POLICY scheduled_jobs_tenant_isolation ON edm.scheduled_jobs
  USING (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID))
  WITH CHECK (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID));

-- RLS Policy: Tenants can only see their own job runs
CREATE POLICY job_runs_tenant_isolation ON edm.scheduled_job_runs
  USING (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID))
  WITH CHECK (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID));

-- ============================================================================
-- VIEWS AND UTILITY
-- ============================================================================

-- View: Next scheduled jobs due to run
DROP VIEW IF EXISTS edm.next_scheduled_jobs;
CREATE VIEW edm.next_scheduled_jobs AS
SELECT 
  j.id,
  j.tenant_id,
  j.operation_type,
  j.next_run_at,
  j.schedule_type,
  j.cron_expression,
  j.timezone,
  j.priority,
  j.job_template,
  (EXTRACT(EPOCH FROM (j.next_run_at - NOW())) :: INT) as seconds_until_run,
  j.run_count,
  j.success_count,
  j.failure_count
FROM edm.scheduled_jobs j
WHERE j.status = 'active'
  AND j.is_active = true
  AND j.next_run_at <= NOW() + INTERVAL '1 minute'
ORDER BY j.priority DESC, j.next_run_at ASC;

-- View: Recent exports
DROP VIEW IF EXISTS edm.recent_exports;
CREATE VIEW edm.recent_exports AS
SELECT 
  e.id,
  e.job_id,
  e.tenant_id,
  e.export_format,
  e.status,
  e.file_size,
  e.record_count,
  e.created_at,
  e.completed_at,
  (EXTRACT(EPOCH FROM (e.completed_at - e.created_at)) :: INT) as duration_seconds,
  CASE WHEN e.expires_at > NOW() THEN true ELSE false END as is_available
FROM edm.job_exports e
WHERE e.status IN ('completed', 'processing')
ORDER BY e.created_at DESC;

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update scheduled job's next run time
CREATE OR REPLACE FUNCTION edm.update_next_run_time(
  p_schedule_id UUID,
  p_current_run_time TIMESTAMP
) RETURNS TIMESTAMP AS $$
DECLARE
  v_schedule RECORD;
  v_next_run TIMESTAMP;
  v_tz TEXT;
BEGIN
  -- Get schedule details
  SELECT * INTO v_schedule
  FROM edm.scheduled_jobs
  WHERE id = p_schedule_id;
  
  IF v_schedule IS NULL THEN
    RETURN NULL;
  END IF;
  
  v_tz := COALESCE(v_schedule.timezone, 'UTC');
  
  -- Calculate next run based on schedule type
  CASE v_schedule.schedule_type
    WHEN 'once' THEN
      -- One-time execution, mark as completed after running
      RETURN NULL;
      
    WHEN 'daily' THEN
      -- Add 1 day to current run time
      v_next_run := (p_current_run_time AT TIME ZONE v_tz) + INTERVAL '1 day';
      
    WHEN 'weekly' THEN
      -- Add 7 days
      v_next_run := (p_current_run_time AT TIME ZONE v_tz) + INTERVAL '7 days';
      
    WHEN 'monthly' THEN
      -- Add 1 month
      v_next_run := (p_current_run_time AT TIME ZONE v_tz) + INTERVAL '1 month';
      
    WHEN 'cron' THEN
      -- Cron expression handling - simplified (use cron library in Go)
      v_next_run := (p_current_run_time AT TIME ZONE v_tz) + INTERVAL '1 hour';
      
    ELSE
      RETURN NULL;
  END CASE;
  
  RETURN v_next_run AT TIME ZONE 'UTC';
END;
$$ LANGUAGE plpgsql;

-- Function to record a scheduled job run
CREATE OR REPLACE FUNCTION edm.record_scheduled_run(
  p_schedule_id UUID,
  p_job_id UUID,
  p_status VARCHAR,
  p_error_message TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
  v_run_id UUID;
  v_next_run TIMESTAMP;
BEGIN
  -- Create run record
  INSERT INTO edm.scheduled_job_runs (
    schedule_id,
    job_id,
    tenant_id,
    scheduled_time,
    status,
    error_message,
    created_at
  ) VALUES (
    p_schedule_id,
    p_job_id,
    (SELECT tenant_id FROM edm.scheduled_jobs WHERE id = p_schedule_id),
    NOW(),
    p_status,
    p_error_message,
    NOW()
  ) RETURNING id INTO v_run_id;
  
  -- Update scheduled job counters
  UPDATE edm.scheduled_jobs
  SET 
    last_run_at = NOW(),
    last_run_status = p_status,
    last_run_error = p_error_message,
    run_count = run_count + 1,
    success_count = CASE WHEN p_status = 'completed' THEN success_count + 1 ELSE success_count END,
    failure_count = CASE WHEN p_status = 'failed' THEN failure_count + 1 ELSE failure_count END,
    next_run_at = edm.update_next_run_time(p_schedule_id, NOW())
  WHERE id = p_schedule_id;
  
  RETURN v_run_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON edm.job_exports TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON edm.scheduled_jobs TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON edm.scheduled_job_runs TO postgres;
GRANT SELECT ON edm.next_scheduled_jobs TO postgres;
GRANT SELECT ON edm.recent_exports TO postgres;
GRANT EXECUTE ON FUNCTION edm.update_next_run_time TO postgres;
GRANT EXECUTE ON FUNCTION edm.record_scheduled_run TO postgres;

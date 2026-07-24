-- Migration 008: Async Jobs Infrastructure
-- Date: February 21, 2026
-- Purpose: Add support for asynchronous bulk operations with job tracking

-- Create async jobs table
CREATE TABLE IF NOT EXISTS edm.async_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  
  -- Job metadata
  total_items INT NOT NULL DEFAULT 0,
  processed_items INT NOT NULL DEFAULT 0,
  succeeded_items INT NOT NULL DEFAULT 0,
  failed_items INT NOT NULL DEFAULT 0,
  
  -- Payload and results
  payload JSONB NOT NULL,
  result_ids UUID[] DEFAULT '{}',
  error_details JSONB,
  
  -- Webhook callback
  webhook_url TEXT,
  webhook_sent BOOLEAN DEFAULT FALSE,
  webhook_attempts INT DEFAULT 0,
  
  -- Audit trail
  created_by UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  
  -- Retry logic
  priority INT DEFAULT 0,
  retry_count INT DEFAULT 0,
  max_retries INT DEFAULT 3,
  
  CONSTRAINT valid_status CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
  CONSTRAINT valid_operation CHECK (operation_type IN ('bulk-create', 'bulk-publish', 'bulk-promote')),
  CONSTRAINT valid_priority CHECK (priority >= 0 AND priority <= 100)
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_async_jobs_tenant_status
  ON edm.async_jobs(tenant_id, status)
  WHERE status IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS idx_async_jobs_created
  ON edm.async_jobs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_async_jobs_queue
  ON edm.async_jobs(priority DESC, created_at)
  WHERE status = 'queued';

-- Enable RLS on async_jobs
ALTER TABLE edm.async_jobs ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenants can only see their own jobs
CREATE POLICY async_jobs_tenant_isolation ON edm.async_jobs
  USING (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID))
  WITH CHECK (tenant_id = CAST(current_setting('app.current_tenant_id') AS UUID));

-- Create job items table for tracking individual items in batch
CREATE TABLE IF NOT EXISTS edm.job_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id UUID NOT NULL,
  item_index INT NOT NULL,
  item_name VARCHAR(500),
  item_data JSONB NOT NULL,
  
  -- Processing status
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  error_message TEXT,
  result_id UUID,
  
  -- Audit trail
  processed_at TIMESTAMP,
  
  CONSTRAINT fk_job_items_job FOREIGN KEY (job_id) 
    REFERENCES edm.async_jobs(id) ON DELETE CASCADE,
  CONSTRAINT valid_item_status CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'skipped')),
  CONSTRAINT unique_job_item UNIQUE(job_id, item_index)
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_job_items_job_status
  ON edm.job_items(job_id, status)
  WHERE status != 'succeeded';

CREATE INDEX IF NOT EXISTS idx_job_items_job
  ON edm.job_items(job_id);

-- Job processing progress view
CREATE OR REPLACE VIEW edm.job_progress_summary AS
SELECT 
  j.id,
  j.tenant_id,
  j.operation_type,
  j.status,
  j.total_items,
  j.processed_items,
  j.succeeded_items,
  j.failed_items,
  COUNT(CASE WHEN ji.status = 'pending' THEN 1 END) AS pending_items,
  COUNT(CASE WHEN ji.status = 'processing' THEN 1 END) AS processing_items,
  COUNT(CASE WHEN ji.status = 'failed' THEN 1 END) AS item_errors,
  ROUND(100.0 * j.processed_items / NULLIF(j.total_items, 0))::INT AS progress_percent,
  j.created_at,
  j.started_at,
  j.completed_at,
  EXTRACT(EPOCH FROM (COALESCE(j.completed_at, NOW()) - j.created_at))::INT AS duration_seconds
FROM edm.async_jobs j
LEFT JOIN edm.job_items ji ON j.id = ji.job_id
GROUP BY j.id, j.tenant_id, j.operation_type, j.status, j.total_items, 
         j.processed_items, j.succeeded_items, j.failed_items, j.created_at, 
         j.started_at, j.completed_at;

-- Grant permissions
GRANT SELECT, INSERT, UPDATE ON edm.async_jobs TO postgres;
GRANT SELECT, INSERT, UPDATE ON edm.job_items TO postgres;
GRANT SELECT ON edm.job_progress_summary TO postgres;

-- Function to update job progress
CREATE OR REPLACE FUNCTION edm.update_job_progress(
  p_job_id UUID,
  p_processed INT DEFAULT NULL,
  p_succeeded INT DEFAULT NULL,
  p_failed INT DEFAULT NULL
) RETURNS void AS $$
BEGIN
  UPDATE edm.async_jobs
  SET 
    processed_items = COALESCE(p_processed, processed_items),
    succeeded_items = COALESCE(p_succeeded, succeeded_items),
    failed_items = COALESCE(p_failed, failed_items),
    status = CASE 
      WHEN (COALESCE(p_processed, processed_items) >= total_items) 
        AND status = 'running' THEN 'completed'
      ELSE status
    END,
    completed_at = CASE 
      WHEN (COALESCE(p_processed, processed_items) >= total_items) 
        AND status = 'running' THEN NOW()
      ELSE completed_at
    END
  WHERE id = p_job_id;
END;
$$ LANGUAGE plpgsql;

-- Function to mark job as started
CREATE OR REPLACE FUNCTION edm.mark_job_started(p_job_id UUID)
RETURNS void AS $$
BEGIN
  UPDATE edm.async_jobs
  SET 
    status = 'running',
    started_at = NOW()
  WHERE id = p_job_id AND status = 'queued';
END;
$$ LANGUAGE plpgsql;

-- Function to fail job
CREATE OR REPLACE FUNCTION edm.fail_job(
  p_job_id UUID,
  p_error_details JSONB DEFAULT NULL
) RETURNS void AS $$
BEGIN
  UPDATE edm.async_jobs
  SET 
    status = 'failed',
    error_details = p_error_details,
    completed_at = NOW()
  WHERE id = p_job_id AND status IN ('queued', 'running');
END;
$$ LANGUAGE plpgsql;

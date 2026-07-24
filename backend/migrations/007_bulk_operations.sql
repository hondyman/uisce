-- Migration 007: Bulk Operations Tracking
-- Purpose: Track bulk template and rule operations for auditing and status monitoring
-- Timeline: 2026-02-20

-- Ensure edm schema exists
CREATE SCHEMA IF NOT EXISTS edm;

-- ========== BULK OPERATIONS TRACKING TABLE ==========
CREATE TABLE IF NOT EXISTS edm.bulk_operations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  request_count INT NOT NULL DEFAULT 0,
  success_count INT NOT NULL DEFAULT 0,
  failure_count INT NOT NULL DEFAULT 0,
  payload_size INT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_by UUID NOT NULL,
  completed_at TIMESTAMP,
  error_summary TEXT,
  
  -- Constraints
  CONSTRAINT ck_operation_type CHECK (operation_type IN ('bulk-create', 'bulk-publish', 'bulk-promote')),
  CONSTRAINT ck_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'partial'))
);

-- ========== INDEXES FOR PERFORMANCE ==========
CREATE INDEX IF NOT EXISTS idx_bulk_ops_tenant
  ON edm.bulk_operations(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_bulk_ops_status
  ON edm.bulk_operations(status) WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS idx_bulk_ops_operation_type
  ON edm.bulk_operations(operation_type, created_at DESC);

-- ========== VERIFICATION QUERY ==========
-- Run this to verify migration success
/*
SELECT 
  COUNT(*) as bulk_operations_table_exists,
  (SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'bulk_operations') as index_count
FROM pg_tables 
WHERE schemaname = 'edm' AND tablename = 'bulk_operations';

-- Expected output:
-- bulk_operations_table_exists: 1 (indicating table exists)
-- index_count: 3 (three indexes created)
*/

-- ========== DOCUMENTATION ==========
-- 
-- Table: edm.bulk_operations
-- Purpose: Audit trail for bulk template and rule operations
-- 
-- Fields:
--   id: Unique batch ID for this bulk operation
--   tenant_id: Which tenant performed the operation
--   operation_type: Type of operation (bulk-create, bulk-publish, bulk-promote)
--   status: Current status (pending/running/completed/failed/partial)
--   request_count: Total items in request
--   success_count: Items successfully processed
--   failure_count: Items that failed
--   payload_size: Size of request in bytes
--   created_at: When operation started
--   created_by: User ID who initiated operation
--   completed_at: When operation finished
--   error_summary: Summary of errors if operation failed
-- 
-- Example Query:
-- Find all bulk operations from last 7 days for a tenant:
--   SELECT * FROM edm.bulk_operations 
--   WHERE tenant_id = '...' 
--   AND created_at > NOW() - INTERVAL '7 days'
--   ORDER BY created_at DESC;
-- 
-- Find failed bulk operations:
--   SELECT * FROM edm.bulk_operations 
--   WHERE status = 'failed'
--   ORDER BY created_at DESC;

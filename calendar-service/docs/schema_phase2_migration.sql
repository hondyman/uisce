-- ============================================================================
-- PHASE 2: Schema Updates for Global Distribution & Priority Routing
-- ============================================================================
-- Epic 31 Calendar Service - Production Deployment
-- Run after initial schema.sql setup
-- 2026-02-17

BEGIN;

-- ============================================================================
-- 1. ADD PRIORITY & REGION ROUTING TO JOBS TABLE
-- ============================================================================
ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 5 
  CHECK (priority BETWEEN 1 AND 10);

ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS region VARCHAR(50) NOT NULL DEFAULT 'us-east-1'
  CHECK (region IN ('us-east-1', 'eu-west-1', 'ap-southeast-1', 'us-west-2', 'eu-central-1'));

ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS resource_profile VARCHAR(50) DEFAULT 'standard'
  CHECK (resource_profile IN ('minimal', 'standard', 'high-memory', 'cpu-intensive'));

ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS sla_deadline TIMESTAMPTZ;

-- ============================================================================
-- 2. CREATE INDEXES FOR EFFICIENT ROUTING
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_jobs_priority_region_status 
ON jobs(priority DESC, region, status) 
WHERE status IN ('pending', 'active');

CREATE INDEX IF NOT EXISTS idx_jobs_region_tenant 
ON jobs(region, tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_sla_deadline 
ON jobs(sla_deadline, priority) 
WHERE sla_deadline IS NOT NULL AND status = 'pending';

-- ============================================================================
-- 3. CREATE REGION-AUTHORIZED PROFILES TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS tenant_region_authorizations (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    region VARCHAR(50) NOT NULL CHECK (region IN ('us-east-1', 'eu-west-1', 'ap-southeast-1', 'us-west-2', 'eu-central-1')),
    authorized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    PRIMARY KEY (tenant_id, region)
);

CREATE INDEX IF NOT EXISTS idx_tenant_region_authorizations 
ON tenant_region_authorizations(tenant_id, region);

-- ============================================================================
-- 4. SEED INITIAL REGION AUTHORIZATIONS
-- ============================================================================
INSERT INTO tenant_region_authorizations (tenant_id, region, created_by)
SELECT t.id, r.region, 'system'
FROM tenants t
CROSS JOIN (
    VALUES 
        ('us-east-1'::VARCHAR),
        ('eu-west-1'::VARCHAR),
        ('ap-southeast-1'::VARCHAR),
        ('us-west-2'::VARCHAR),
        ('eu-central-1'::VARCHAR)
) AS r(region)
ON CONFLICT (tenant_id, region) DO NOTHING;

-- ============================================================================
-- 5. ADD COMMENTS FOR DOCUMENTATION
-- ============================================================================
COMMENT ON COLUMN jobs.priority IS 
    'Job priority: 1-2 (critical, scaled workers), 3-7 (standard), 8-10 (bulk). Lower = higher priority.';

COMMENT ON COLUMN jobs.region IS 
    'Target region for job execution. Must be in tenant_region_authorizations. Enforced at API layer.';

COMMENT ON COLUMN jobs.resource_profile IS 
    'Resource profile hint: minimal=t2.micro, standard=t3.medium, high-memory=r5.large, cpu-intensive=c5.xlarge';

COMMENT ON COLUMN jobs.sla_deadline IS 
    'Target completion time. Used by priority queue to schedule critical jobs near deadline.';

COMMENT ON TABLE tenant_region_authorizations IS 
    'Data residency control. Each (tenant, region) pair must exist for tenant to submit jobs to that region.';

-- ============================================================================
-- 6. VERIFICATION QUERIES
-- ============================================================================
-- Run these to verify schema updates applied correctly:
/*
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns 
WHERE table_name='jobs' AND column_name IN ('priority', 'region', 'resource_profile', 'sla_deadline')
ORDER BY ordinal_position;

SELECT indexname, indexdef
FROM pg_indexes 
WHERE tablename='jobs' AND indexname LIKE 'idx_jobs%'
ORDER BY indexname;

SELECT COUNT(*) as total_authorizations, COUNT(DISTINCT tenant_id) as authorized_tenants
FROM tenant_region_authorizations;
*/

COMMIT;

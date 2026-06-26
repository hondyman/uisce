#!/usr/bin/env bash
# docs/SCHEMA_UPDATES.sql
# Critical schema additions for production deployment
# Run AFTER initial schema.sql setup

set -e

echo "🔧 Applying critical schema updates for Epic 31 v2..."

# ============================================================================
# 1. ADD PRIORITY & REGION ROUTING TO JOBS TABLE
# ============================================================================
echo "📝 Adding priority and region fields to jobs table..."

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
-- Add priority field (1-10, lower = higher priority)
ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 5 
  CHECK (priority BETWEEN 1 AND 10);

-- Add region for global distribution
ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS region VARCHAR(50) NOT NULL DEFAULT 'us-east-1'
  CHECK (region IN ('us-east-1', 'eu-west-1', 'ap-southeast-1', 'us-west-2', 'eu-central-1'));

-- Add resource profile for cost optimization
ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS resource_profile VARCHAR(50) DEFAULT 'standard'
  CHECK (resource_profile IN ('minimal', 'standard', 'high-memory', 'cpu-intensive'));

-- Add SLA deadline for priority queue management
ALTER TABLE IF EXISTS jobs
ADD COLUMN IF NOT EXISTS sla_deadline TIMESTAMPTZ;

-- ============================================================================
-- 2. CREATE INDEXES FOR EFFICIENT ROUTING
-- ============================================================================
-- Primary routing index: (priority DESC, region, status) for queue ordering
CREATE INDEX IF NOT EXISTS idx_jobs_priority_region_status 
ON jobs(priority DESC, region, status) 
WHERE status IN ('pending', 'active');

-- Region scoping for data residency checks
CREATE INDEX IF NOT EXISTS idx_jobs_region_tenant 
ON jobs(region, tenant_id, created_at DESC);

-- SLA deadline for deadline-aware scheduling
CREATE INDEX IF NOT EXISTS idx_jobs_sla_deadline 
ON jobs(sla_deadline, priority) 
WHERE sla_deadline IS NOT NULL AND status = 'pending';

-- ============================================================================
-- 3. CREATE REGION-AUTHORIZED PROFILES TABLE
-- ============================================================================
-- Track which regions each tenant is allowed to use (for data residency)
CREATE TABLE IF NOT EXISTS tenant_region_authorizations (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    region VARCHAR(50) NOT NULL CHECK (region IN ('us-east-1', 'eu-west-1', 'ap-southeast-1', 'us-west-2', 'eu-central-1')),
    authorized_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, region)
);

CREATE INDEX IF NOT EXISTS idx_tenant_region_authorizations 
ON tenant_region_authorizations(tenant_id, region);

-- ============================================================================
-- 4. SEED INITIAL REGION AUTHORIZATIONS
-- ============================================================================
-- By default, tenants can use all regions (override in config if needed)
INSERT INTO tenant_region_authorizations (tenant_id, region)
SELECT t.id, r.region
FROM tenants t
CROSS JOIN (
    VALUES 
        ('us-east-1'::VARCHAR),
        ('eu-west-1'::VARCHAR),
        ('ap-southeast-1'::VARCHAR),
        ('us-west-2'::VARCHAR),
        ('eu-central-1'::VARCHAR)
) AS r(region)
ON CONFLICT DO NOTHING;

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
    'Data residency control. Each (tenant, region) pair must exist for tenant to submit jobs to that region. Checked at /api/v1/check-availability and /api/v1/jobs endpoints.';

EOF

echo "✅ Schema updates applied successfully!"
echo ""
echo "📊 Verify updates:"
echo "  SELECT * FROM information_schema.columns WHERE table_name='jobs' AND column_name IN ('priority', 'region', 'resource_profile', 'sla_deadline');"
echo "  SELECT * FROM information_schema.indexes WHERE table_name='jobs' AND indexname LIKE 'idx_jobs%';"

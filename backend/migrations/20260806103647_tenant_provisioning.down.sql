DROP INDEX IF EXISTS idx_provisioning_jobs_tenant_id;
DROP INDEX IF EXISTS idx_provisioning_jobs_status;
DROP TABLE IF EXISTS provisioning_jobs;
DROP INDEX IF EXISTS idx_tenants_gold_copy;
ALTER TABLE tenants DROP COLUMN IF EXISTS database_name;

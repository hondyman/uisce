-- 0011_staging_load_run_crims.down.sql
BEGIN;
DROP POLICY IF EXISTS _me_tenant_write ON staging._mapping_error;
DROP POLICY IF EXISTS _me_tenant_read  ON staging._mapping_error;
ALTER TABLE IF EXISTS staging._mapping_error DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS _lr_tenant_write ON staging._load_run;
DROP POLICY IF EXISTS _lr_tenant_read  ON staging._load_run;
ALTER TABLE IF EXISTS staging._load_run DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS staging._mapping_error CASCADE;
DROP TABLE IF EXISTS staging._load_run CASCADE;
COMMIT;

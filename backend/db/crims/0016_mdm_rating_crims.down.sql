-- 0016_mdm_rating_crims.down.sql
BEGIN;
DROP POLICY IF EXISTS rating_tenant_write ON mdm.rating;
DROP POLICY IF EXISTS rating_tenant_read  ON mdm.rating;
ALTER TABLE IF EXISTS mdm.rating DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS mdm.rating CASCADE;
COMMIT;

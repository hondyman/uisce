-- 0017_mdm_rating_action_crims.down.sql
BEGIN;
DROP POLICY IF EXISTS rating_action_tenant_write ON mdm.rating_action;
DROP POLICY IF EXISTS rating_action_tenant_read  ON mdm.rating_action;
ALTER TABLE IF EXISTS mdm.rating_action DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS mdm.rating_action CASCADE;
COMMIT;

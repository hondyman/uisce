-- 0014_mdm_party_crims.down.sql
BEGIN;
DROP POLICY IF EXISTS party_tenant_write ON mdm.party;
DROP POLICY IF EXISTS party_tenant_read  ON mdm.party;
ALTER TABLE IF EXISTS mdm.party DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS mdm.party CASCADE;
COMMIT;

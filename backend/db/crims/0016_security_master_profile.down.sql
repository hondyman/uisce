-- 0016_security_master_profile.down.sql
-- Reverses 0016's configuration. Keeps the one vendor registry (foreign keys stay on
-- mdm.source_systems) and mdm.security_master.master_id (dropping it would orphan golden
-- records and identifiers written since); drop it by hand only for a crims rebuild.

\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
DELETE FROM mdm.mastering_policy WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'SECURITY';
DELETE FROM mdm.security_source_priority WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND asset_class_cd = '*';
DELETE FROM mdm.security_match_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND rule_cd IN ('SEC_ID_EXACT', 'SEC_NAME_REVIEW');
DELETE FROM mdm.mastering_entity WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'SECURITY';
DROP TABLE IF EXISTS staging.security_incoming;
DROP TRIGGER IF EXISTS security_master_carry_master_id ON mdm.security_master;
DROP FUNCTION IF EXISTS mdm.security_master_carry_master_id();
COMMIT;

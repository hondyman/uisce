-- 0011_mastering_foundation.down.sql
-- Reverses 0011. Drops mastering state (xref, runs): only for a crims rebuild.

\set ON_ERROR_STOP on
BEGIN;

SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
DELETE FROM mdm.product_match_rule
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
   AND rule_cd IN ('PRODUCT_ID_EXACT', 'PRODUCT_NAME_FUZZY');

DROP TABLE IF EXISTS mdm.mastering_run;
DROP TABLE IF EXISTS mdm.entity_xref;
DROP TABLE IF EXISTS mdm.mastering_entity;

COMMIT;

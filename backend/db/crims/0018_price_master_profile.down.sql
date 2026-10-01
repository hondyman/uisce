-- 0018_price_master_profile.down.sql
-- Reverses 0018's configuration and staging. Keeps the one vendor registry (price foreign keys stay on
-- mdm.source_systems, price_source.source_system_id stays) and mastering_entity.kind; golden prices and
-- observations written since are left in place.

\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
DELETE FROM mdm.mastering_policy WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRICE';
DELETE FROM mdm.survivorship_rule WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_type = 'PRICE';
DELETE FROM mdm.price_variance_threshold WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
   AND asset_class_cd IN ('*', 'Equity', 'FixedIncome') AND sec_sub_typ_cd IS NULL AND price_type_cd IS NULL;
DELETE FROM mdm.price_source_priority WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
   AND price_entity_type = 'SECURITY' AND price_type_cd IS NULL;
DELETE FROM mdm.mastering_entity WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRICE';
DROP TABLE IF EXISTS staging.ice_price;
DROP TABLE IF EXISTS staging.rdp_price;
DROP TABLE IF EXISTS staging.bbg_price;
DROP INDEX IF EXISTS mdm.idx_pex_open;
DROP INDEX IF EXISTS mdm.idx_price_key;
DROP INDEX IF EXISTS mdm.idx_pgr_key_version;
DROP INDEX IF EXISTS mdm.uq_pgr_current;
COMMIT;

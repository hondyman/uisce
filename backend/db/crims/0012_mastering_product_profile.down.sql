-- 0012_mastering_product_profile.down.sql
\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);
UPDATE mdm.mastering_entity SET settings = '{"source_table": "mdm.source_systems"}'::jsonb, updated_at = now()
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRODUCT';
DELETE FROM mdm.product_type_mapping
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
   AND mdm_source_system_id = 'a2222222-2222-2222-2222-222222222222' AND vendor_type_cd = 'ETF';
COMMIT;

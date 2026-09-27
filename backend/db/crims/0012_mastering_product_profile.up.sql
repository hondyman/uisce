-- 0012_mastering_product_profile.up.sql
-- Product mastering configuration (gold copy, inherited by tenants). Run against crims.
--
--   * product_type_id is a reference: a source's vendor type (FactSet "Mutual Fund") maps to the
--     internal type (MUTUAL_FUND) through mdm.product_type_mapping; product_type_cd is what survives.
--     Required: a record whose type has no mapping is an exception for a steward, not a guess.
--   * status_id is a reference with no vendor map; a new product starts LIVE.
--   * product_golden_record.product_type_cd is filled from the surviving type.
--   * FactSet sends "ETF" (not "Exchange Traded Fund"): mapped. SICAV and UCITS are legal forms, not
--     product types, and stay unmapped until a steward decides.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0012_mastering_product_profile.up.sql

\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('app.current_tenant', '99e99e99-99e9-49e9-89e9-99e99e99e999', true);

UPDATE mdm.mastering_entity
   SET settings = '{
         "source_table": "mdm.source_systems",
         "name_attribute": "name",
         "defaults": {"status_cd": "LIVE"},
         "record_columns": {"product_type_cd": "product_type_cd"},
         "references": [
           {"column": "product_type_id", "attribute": "product_type_cd", "required": true,
            "ref_table": "mdm.product_type", "ref_code_column": "type_cd",
            "map_table": "mdm.product_type_mapping", "map_source_column": "mdm_source_system_id",
            "map_vendor_column": "vendor_type_cd", "map_code_column": "internal_type_cd"},
           {"column": "status_id", "attribute": "status_cd", "required": true,
            "ref_table": "mdm.product_status", "ref_code_column": "status_cd"}
         ]
       }'::jsonb,
       updated_at = now()
 WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND entity_cd = 'PRODUCT';

INSERT INTO mdm.product_type_mapping (tenant_id, mdm_source_system_id, vendor_type_cd, internal_type_cd)
SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999', 'a2222222-2222-2222-2222-222222222222', 'ETF', 'ETF'
WHERE NOT EXISTS (SELECT 1 FROM mdm.product_type_mapping
                   WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
                     AND mdm_source_system_id = 'a2222222-2222-2222-2222-222222222222' AND vendor_type_cd = 'ETF');

COMMIT;

-- 20261118_004_mdm_scoring_expansion.down.sql
-- Revert MDM Vendor Scoring Expansion

DROP TABLE IF EXISTS mdm_eval.vendor_contract_rights CASCADE;
DROP TABLE IF EXISTS mdm_eval.vendor_operational_friction CASCADE;
DROP TABLE IF EXISTS mdm_eval.vendor_revision_log CASCADE;
DROP TABLE IF EXISTS mdm_eval.vendor_feed_log CASCADE;
DROP TABLE IF EXISTS mdm_eval.scoring_weight_profiles CASCADE;
DROP TABLE IF EXISTS mdm_eval.scoring_settings CASCADE;

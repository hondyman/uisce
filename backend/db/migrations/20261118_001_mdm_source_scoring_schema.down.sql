-- 20261118_001_mdm_source_scoring_schema.down.sql
DROP TABLE IF EXISTS mdm_eval.value_override;
DROP TABLE IF EXISTS mdm_eval.golden_value;
DROP TABLE IF EXISTS mdm_eval.certified_reference_set;
DROP TABLE IF EXISTS mdm_eval.attribute_tolerance;
DROP SCHEMA IF EXISTS mdm_eval CASCADE;

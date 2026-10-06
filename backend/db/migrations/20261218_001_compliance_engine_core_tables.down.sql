-- 20261218_001_compliance_engine_core_tables.down.sql
--
-- Rollback Compliance Engine Core Schema & Storage Lifecycle Tables

DROP TABLE IF EXISTS compliance.compliance_orphan_object CASCADE;
DROP TABLE IF EXISTS compliance.compliance_watermark_checkpoint CASCADE;
DROP TABLE IF EXISTS compliance.compliance_archive_manifest CASCADE;
DROP TABLE IF EXISTS compliance.compliance_evaluation_event CASCADE;
DROP TABLE IF EXISTS compliance.compliance_rule CASCADE;
DROP FUNCTION IF EXISTS compliance.prevent_evaluation_mutation CASCADE;
DROP SCHEMA IF EXISTS compliance CASCADE;

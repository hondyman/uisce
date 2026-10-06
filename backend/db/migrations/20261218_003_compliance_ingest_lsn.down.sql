-- Rollback Migration 20261218_003: Drop ingest_lsn column and index

DROP INDEX IF EXISTS compliance.idx_compliance_eval_ingest_lsn;
ALTER TABLE compliance.compliance_evaluation_event DROP COLUMN IF EXISTS ingest_lsn;

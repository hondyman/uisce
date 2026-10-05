-- Migration 20261218_003: Add ingest_lsn to compliance_evaluation_event for deterministic LSN watermarking
-- Captures monotonic PostgreSQL WAL sequence number for certified Hot -> Warm -> Cold tiering

ALTER TABLE compliance.compliance_evaluation_event
ADD COLUMN IF NOT EXISTS ingest_lsn BIGINT NOT NULL DEFAULT ((pg_current_wal_lsn() - '0/0'::pg_lsn)::bigint);

CREATE INDEX IF NOT EXISTS idx_compliance_eval_ingest_lsn ON compliance.compliance_evaluation_event (ingest_lsn);

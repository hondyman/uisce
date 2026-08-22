-- 20260831_mdm_streaming_and_batch_ingestion.sql
-- Ingestion Feed Registries, Batch Execution Ledgers, and Dead-Letter Queues (DLQ)

CREATE SCHEMA IF NOT EXISTS mdm_ingest;

-- 1. Ingestion Feed Configuration (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS mdm_ingest.feed_configurations (
    feed_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL REFERENCES mdm.master_domain_registry(domain_key),
    vendor_name VARCHAR(50) NOT NULL,              -- BLOOMBERG, REFINITIV, IDC, DTCC, CRIMS
    ingestion_mode VARCHAR(30) NOT NULL,           -- REALTIME_STREAM, FILE_BATCH, API_POLLER
    stream_topic VARCHAR(150),                     -- e.g. "mdm.vendor.prices.v1"
    file_path_pattern TEXT,                        -- e.g. "s3://tenant-{id}-mdm-inbound/bloomberg/*.csv"
    payload_format VARCHAR(30) NOT NULL,           -- JSON, CSV, FIX_35_D, PARQUET
    batch_flush_size INT DEFAULT 1000,
    batch_flush_interval_ms INT DEFAULT 250,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_vendor_feed UNIQUE (tenant_id, domain_key, vendor_name)
);

-- 2. Ingestion Batch Execution Ledger (Rule 6: Semantic/OLTP Boundary)
CREATE TABLE IF NOT EXISTS mdm_ingest.batch_execution_ledger (
    batch_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    feed_id UUID NOT NULL REFERENCES mdm_ingest.feed_configurations(feed_id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL,
    vendor_name VARCHAR(50) NOT NULL,
    batch_status VARCHAR(30) NOT NULL,             -- RUNNING, COMPLETED, PARTIAL_ERRORS, FAILED
    total_records_ingested INT DEFAULT 0,
    successful_records INT DEFAULT 0,
    anomalies_flagged INT DEFAULT 0,
    failed_records INT DEFAULT 0,
    throughput_records_per_sec NUMERIC(10, 2) DEFAULT 0.0,
    execution_duration_ms INT DEFAULT 0,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ingest_ledger_search 
ON mdm_ingest.batch_execution_ledger (tenant_id, domain_key, started_at DESC);

-- 3. Ingestion Dead-Letter Queue (DLQ)
CREATE TABLE IF NOT EXISTS mdm_ingest.dead_letter_queue (
    dlq_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id UUID REFERENCES mdm_ingest.batch_execution_ledger(batch_id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL,
    vendor_name VARCHAR(50) NOT NULL,
    raw_payload TEXT NOT NULL,
    error_reason TEXT NOT NULL,
    retry_count INT DEFAULT 0,
    is_resolved BOOLEAN DEFAULT FALSE,
    captured_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_dlq_unresolved 
ON mdm_ingest.dead_letter_queue (tenant_id, is_resolved, captured_at DESC);

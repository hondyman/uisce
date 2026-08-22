-- 20260823_telemetry_holiday_feed_and_distribution.sql

-- 1. Extend Burst Batch Ledger for DLQ & Telemetry
ALTER TABLE public.report_burst_batches
    ADD COLUMN IF NOT EXISTS retry_batch_id UUID REFERENCES public.report_burst_batches(id),
    ADD COLUMN IF NOT EXISTS render_throughput_per_sec NUMERIC(8, 2) DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS p50_latency_ms INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS p95_latency_ms INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS p99_latency_ms INT DEFAULT 0;

ALTER TABLE public.report_burst_artifacts
    ADD COLUMN IF NOT EXISTS error_reason TEXT,
    ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS distribution_status VARCHAR(50) DEFAULT 'PENDING'; -- PENDING, DISPATCHED, FAILED, SKIPPED

-- 2. External Holiday Ingestion Feed Configurations
CREATE TABLE IF NOT EXISTS public.exchange_holiday_sync_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES public.tenant_exchange_calendars(id) ON DELETE CASCADE,
    source_provider VARCHAR(50) NOT NULL, -- SIFMA, FINRA, ECB_TARGET2, MANUAL
    feed_url TEXT NOT NULL,
    sync_frequency VARCHAR(50) DEFAULT 'DAILY', -- DAILY, WEEKLY
    last_synced_at TIMESTAMPTZ,
    last_sync_status VARCHAR(50) DEFAULT 'NEVER_RUN',
    last_sync_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_calendar_sync UNIQUE (tenant_id, calendar_id, source_provider)
);

-- 3. Client Distribution Channel & Delivery Audit Ledger
CREATE TABLE IF NOT EXISTS public.client_distribution_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    client_id VARCHAR(100) NOT NULL,
    channel_type VARCHAR(50) NOT NULL, -- EMAIL, SFTP, S3_PRESIGNED, WEBHOOK
    destination_target TEXT NOT NULL,  -- email address, sftp://host:port/path, webhook URL
    encryption_type VARCHAR(50) DEFAULT 'NONE', -- NONE, PGP_ENCRYPTED, AES_ZIP_PASSWORD
    pgp_public_key TEXT,
    auth_secret_arn TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_client_channel UNIQUE (tenant_id, client_id, channel_type)
);

CREATE TABLE IF NOT EXISTS public.report_burst_distribution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES public.report_burst_batches(id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES public.report_burst_artifacts(id) ON DELETE CASCADE,
    client_id VARCHAR(100) NOT NULL,
    channel_type VARCHAR(50) NOT NULL,
    destination_target TEXT NOT NULL,
    delivery_status VARCHAR(50) NOT NULL, -- DELIVERED, FAILED, RETRYING
    error_message TEXT,
    dispatched_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_distribution_logs_lookup 
ON public.report_burst_distribution_logs (tenant_id, batch_id, delivery_status);

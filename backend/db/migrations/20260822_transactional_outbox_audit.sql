-- 20260822_transactional_outbox_audit.sql
-- Implements atomic outbox staging and tamper-evident governance ledger

CREATE TABLE IF NOT EXISTS public.catalog_outbox_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL, -- BUSINESS_OBJECT, SEMANTIC_MAPPING, TAXONOMY, DRIFT_PATCH
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,    -- BO_PUBLISHED, FIELD_OVERRIDE_SAVED, MAPPING_REJECTED, REBASE_APPLIED
    actor_id TEXT NOT NULL,
    actor_role TEXT NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    chain_seal VARCHAR(64),
    retry_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    last_error TEXT,
    CONSTRAINT uq_outbox_idempotency UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending_drain 
ON public.catalog_outbox_events (created_at ASC) 
WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_audit_search 
ON public.catalog_outbox_events (tenant_id, aggregate_type, aggregate_id, created_at DESC);

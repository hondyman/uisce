-- Migration: Cryptographic Tamper-Evident Audit Ledger Schema
-- Date: 2026-07-31
-- Purpose: Store immutable, SHA-256 hash-chained audit trails for enterprise data governance and SEC/FINRA compliance.

CREATE TABLE IF NOT EXISTS public.cryptographic_audit_ledger (
    sequence_id BIGSERIAL PRIMARY KEY,
    audit_id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    entity_type VARCHAR(100) NOT NULL, -- e.g., 'BusinessObject', 'AccountRecord', 'SchemaMigration'
    entity_id VARCHAR(255) NOT NULL,
    action_type VARCHAR(50) NOT NULL,  -- 'INSERT', 'UPDATE', 'DELETE', 'SCHEMA_CHANGE'
    payload_snapshot JSONB NOT NULL,
    performed_by VARCHAR(255) NOT NULL DEFAULT 'system',
    previous_hash VARCHAR(64) NOT NULL DEFAULT 'GENESIS_BLOCK',
    current_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_audit_ledger_tenant ON public.cryptographic_audit_ledger(tenant_id, entity_type, entity_id);

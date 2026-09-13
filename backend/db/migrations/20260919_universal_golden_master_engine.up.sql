-- 20260919_universal_golden_master_engine.up.sql
-- Universal Golden Record, Graph-RAG Cross-Reference & Neural Survivorship Ledger

CREATE SCHEMA IF NOT EXISTS catalog_mdm;
CREATE SCHEMA IF NOT EXISTS catalog_mdm_ai;

-- 1. Universal Exception Queue (Tolerance breaks & vendor divergences)
CREATE TABLE IF NOT EXISTS catalog_mdm.universal_exception_queue (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    golden_id UUID,
    master_entity_sid VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    competing_values JSONB NOT NULL,
    deviation_pct NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN', -- OPEN, RESOLVED_AUTO, RESOLVED_MANUAL, DISMISSED
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mdm_exception_queue 
ON catalog_mdm.universal_exception_queue (tenant_id, status, domain_key);

-- 2. Vendor Dynamic Trust & Neural Weight Parameters
CREATE TABLE IF NOT EXISTS catalog_mdm_ai.vendor_dynamic_trust_weights (
    weight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,
    asset_class VARCHAR(50) NOT NULL DEFAULT 'ALL',
    base_trust_score NUMERIC(5, 2) NOT NULL DEFAULT 85.00,
    historical_accuracy_pct NUMERIC(5, 2) NOT NULL DEFAULT 99.20,
    staleness_decay_half_life_sec INT NOT NULL DEFAULT 3600,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_vendor_domain_master UNIQUE (tenant_id, domain_key, vendor_source, asset_class)
);

-- 3. Immutable Golden Records Ledger (SEC Rule 17a-4 WORM with Merkle roots)
CREATE TABLE IF NOT EXISTS catalog_mdm.golden_records_ledger (
    ledger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_id UUID NOT NULL,
    domain_key VARCHAR(50) NOT NULL,
    master_entity_sid VARCHAR(100) NOT NULL,
    effective_date DATE NOT NULL,
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    golden_attributes JSONB NOT NULL,
    vendor_attributions JSONB NOT NULL,
    merkle_root_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mdm_golden_ledger_lookup 
ON catalog_mdm.golden_records_ledger (tenant_id, golden_id, knowledge_time DESC);

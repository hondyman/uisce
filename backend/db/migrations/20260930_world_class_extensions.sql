-- 20260930_world_class_extensions.sql
-- OpenDataContract Registry, SLA Policies, and Two-Stage AI Feedback Mesh

CREATE SCHEMA IF NOT EXISTS datacontract;
CREATE SCHEMA IF NOT EXISTS ai_telemetry;

-- 1. Declarative Data Product & Contract Registry
CREATE TABLE IF NOT EXISTS datacontract.data_product_contracts (
    contract_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    contract_key VARCHAR(100) NOT NULL,
    bo_id UUID NOT NULL REFERENCES public.business_object(id) ON DELETE CASCADE,
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    owner_team VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, DEPRECATED, RETIRED
    sla_freshness_sec INT NOT NULL DEFAULT 900,  -- 15m Freshness target
    sla_max_latency_ms INT NOT NULL DEFAULT 250, -- 250ms p95 target
    sla_availability_pct NUMERIC(5, 2) DEFAULT 99.95,
    contract_yaml TEXT NOT NULL,
    generated_schema_json JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_contract_version UNIQUE (tenant_id, contract_key, version)
);

CREATE INDEX IF NOT EXISTS idx_contracts_bo 
ON datacontract.data_product_contracts (tenant_id, bo_id, status);

-- 2. Two-Stage Closed-Loop AI Feedback Ledger
CREATE TABLE IF NOT EXISTS ai_telemetry.explicit_feedback_ledger (
    feedback_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    interaction_id VARCHAR(64) NOT NULL,          -- Hash of prompt + response
    user_id VARCHAR(100) NOT NULL,
    is_positive BOOLEAN NOT NULL,                 -- Stage 1: Thumbs Up (true) / Down (false)
    error_category VARCHAR(50),                   -- Stage 2: WRONG_TABLE, INCORRECT_FORMULA, BAD_JOIN, HALLUCINATION
    user_notes TEXT,
    resolved_bo_id UUID REFERENCES public.business_object(id) ON DELETE SET NULL,
    applied_weight_delta NUMERIC(4, 3) DEFAULT 0.000,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_feedback_lookup 
ON ai_telemetry.explicit_feedback_ledger (tenant_id, error_category, created_at DESC);

-- 3. Dynamic Bitemporal Snapshot Ledger
CREATE TABLE IF NOT EXISTS public.bitemporal_state_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_key VARCHAR(100) NOT NULL,
    effective_date DATE NOT NULL,                 -- Te
    knowledge_timestamp TIMESTAMPTZ NOT NULL,     -- Tk
    merkle_root_hash VARCHAR(64) NOT NULL,
    record_count BIGINT NOT NULL,
    storage_tier VARCHAR(20) NOT NULL,           -- STARROCKS_HOT, ICEBERG_COLD
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_bitemporal_snapshot UNIQUE (tenant_id, entity_key, effective_date, knowledge_timestamp)
);

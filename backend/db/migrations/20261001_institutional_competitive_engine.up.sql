-- backend/db/migrations/20261001_institutional_competitive_engine.up.sql
-- Zero-Hallucination CFG Rules, Dynamic Neural Survivorship, Bitemporal Snapshots & Merkle Sealing

CREATE SCHEMA IF NOT EXISTS catalog_governance;
CREATE SCHEMA IF NOT EXISTS catalog_mdm_neural;
CREATE SCHEMA IF NOT EXISTS catalog_calc;

-- 1. CFG (Context-Free Grammar) Permitted Lexical Rules (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_governance.cfg_semantic_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    allowed_dimensions JSONB NOT NULL DEFAULT '[]'::jsonb,   -- ["sector", "region", "asset_class"]
    allowed_measures JSONB NOT NULL DEFAULT '[]'::jsonb,     -- ["sum_market_value", "net_pnl"]
    non_additive_metrics JSONB NOT NULL DEFAULT '[]'::jsonb, -- ["xirr", "weighted_yield"]
    max_join_depth INT NOT NULL DEFAULT 3,
    complexity_ceiling INT NOT NULL DEFAULT 85,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_bo_cfg UNIQUE (tenant_id, bo_id)
);

-- 2. Neural Survivorship Weightings with Exponential Decay Half-Life
CREATE TABLE IF NOT EXISTS catalog_mdm_neural.source_decay_profiles (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    vendor_code VARCHAR(50) NOT NULL, -- 'BLOOMBERG', 'REFINITIV', 'IDC', 'CRIMS'
    domain_key VARCHAR(50) NOT NULL,  -- 'SECURITY_PRICE', 'CREDIT_RATING'
    base_trust_score NUMERIC(5, 4) NOT NULL DEFAULT 0.9500,
    decay_lambda NUMERIC(8, 6) NOT NULL DEFAULT 0.000120, -- Exponential decay factor per second
    historical_accuracy_pct NUMERIC(5, 2) NOT NULL DEFAULT 99.80,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_vendor_domain UNIQUE (tenant_id, vendor_code, domain_key)
);

-- 3. Cryptographic SEC Rule 17a-4 / FINRA Merkle Provenance Ledger
CREATE TABLE IF NOT EXISTS catalog_governance.cryptographic_merkle_ledger (
    leaf_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,   -- 'GOLDEN_RECORD', 'COMPLIANCE_OVERRIDE', 'DATA_CONTRACT'
    entity_id UUID NOT NULL,
    leaf_hash VARCHAR(64) NOT NULL,     -- SHA256(canonical_payload + previous_leaf_hash)
    merkle_root_hash VARCHAR(64) NOT NULL,
    tree_depth INT NOT NULL,
    sealed_by VARCHAR(100) NOT NULL,
    sealed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_merkle_tenant_search 
ON catalog_governance.cryptographic_merkle_ledger (tenant_id, entity_type, entity_id, sealed_at DESC);

-- 4. Bitemporal State Index & Copy-on-Write Sandbox Definitions
CREATE TABLE IF NOT EXISTS catalog_calc.cow_simulation_sandboxes (
    sandbox_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    sandbox_name VARCHAR(100) NOT NULL,
    base_knowledge_time TIMESTAMPTZ NOT NULL,   -- Tk (System transaction time)
    base_effective_time TIMESTAMPTZ NOT NULL,   -- Te (Valid economic event time)
    delta_patch_payload JSONB NOT NULL DEFAULT '{}'::jsonb, -- Hypothecated order and tax-lot changes
    is_ephemeral BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cow_sandboxes_eval 
ON catalog_calc.cow_simulation_sandboxes (tenant_id, user_id, expires_at);

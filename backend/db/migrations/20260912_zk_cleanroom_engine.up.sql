-- 20260912_zk_cleanroom_engine.up.sql
-- Zero-Knowledge Private Credit & Syndicate Clean Room Engine

CREATE SCHEMA IF NOT EXISTS cleanroom_zk;

-- 1. Master ZK Covenant & Verification Circuit Registry
CREATE TABLE IF NOT EXISTS cleanroom_zk.circuit_definitions (
    circuit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    circuit_key VARCHAR(100) NOT NULL UNIQUE,
    circuit_name VARCHAR(150) NOT NULL,
    curve_type VARCHAR(30) NOT NULL DEFAULT 'BN254',
    proving_system VARCHAR(30) NOT NULL DEFAULT 'GROTH16',
    verifying_key_payload BYTEA NOT NULL,
    constraint_count INT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Confidential Borrower Covenant Snapshots
CREATE TABLE IF NOT EXISTS cleanroom_zk.loan_covenant_statements (
    statement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    facility_node_id UUID NOT NULL,
    borrower_node_id UUID NOT NULL,
    period_end_date DATE NOT NULL,
    
    min_dscr_threshold NUMERIC(8, 4) NOT NULL DEFAULT 1.3500,
    max_leverage_threshold NUMERIC(8, 4) NOT NULL DEFAULT 4.2000,
    min_liquidity_usd NUMERIC(28, 4) NOT NULL DEFAULT 5000000.0000,
    
    encrypted_financial_witness BYTEA NOT NULL,
    witness_sha256 VARCHAR(64) NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_PROOF',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_facility_period UNIQUE (tenant_id, facility_node_id, period_end_date)
);

-- 3. Attested Zero-Knowledge Proof Receipts
CREATE TABLE IF NOT EXISTS cleanroom_zk.zk_attestation_proofs (
    proof_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    statement_id UUID NOT NULL REFERENCES cleanroom_zk.loan_covenant_statements(statement_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    circuit_id UUID NOT NULL REFERENCES cleanroom_zk.circuit_definitions(circuit_id) ON DELETE RESTRICT,
    
    proof_payload_bytes BYTEA NOT NULL,
    public_inputs_json JSONB NOT NULL,
    verification_passed BOOLEAN NOT NULL DEFAULT FALSE,
    prover_latency_ms NUMERIC(8, 3) NOT NULL,
    verifier_latency_ms NUMERIC(8, 3) NOT NULL,
    
    merkle_attestation_root VARCHAR(64) NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Differential Privacy Consumption Budget Ledger
CREATE TABLE IF NOT EXISTS cleanroom_zk.differential_privacy_budgets (
    budget_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    syndicate_cohort_key VARCHAR(100) NOT NULL,
    total_epsilon_budget NUMERIC(8, 4) NOT NULL DEFAULT 10.0000,
    consumed_epsilon NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    delta_parameter NUMERIC(14, 12) NOT NULL DEFAULT 0.000000000001,
    last_query_at TIMESTAMPTZ,
    CONSTRAINT uq_tenant_cohort_budget UNIQUE (tenant_id, syndicate_cohort_key)
);

CREATE INDEX IF NOT EXISTS idx_covenant_status 
ON cleanroom_zk.loan_covenant_statements (tenant_id, period_end_date, status);

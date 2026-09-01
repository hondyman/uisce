-- 20260903_multi_agent_swarm_passports.up.sql
-- Multi-Agent Domain Swarm & Cryptographic Merkle Passport Ledger

CREATE SCHEMA IF NOT EXISTS catalog_agent;

CREATE TABLE IF NOT EXISTS catalog_agent.swarm_execution_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    orchestrator_session_id UUID NOT NULL,
    requesting_user_id VARCHAR(100) NOT NULL,
    intent_description TEXT NOT NULL,
    participating_agents TEXT[] NOT NULL,
    execution_status VARCHAR(30) NOT NULL DEFAULT 'RUNNING', -- RUNNING, COMPLETED, FAILED, ESCALATED
    merkle_passport VARCHAR(64) NOT NULL,
    total_latency_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS catalog_agent.agent_task_receipts (
    receipt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES catalog_agent.swarm_execution_runs(run_id) ON DELETE CASCADE,
    agent_type VARCHAR(50) NOT NULL, -- ALLOCATION, RECON_BREAK, RISK_SHOCK, REGULATORY_PACK
    okf_concept_key VARCHAR(150) NOT NULL,
    input_payload JSONB NOT NULL,
    output_result JSONB NOT NULL,
    validation_status VARCHAR(30) NOT NULL, -- VERIFIED, ANOMALY_DETECTED, BLOCKED
    latency_ms INT NOT NULL,
    merkle_leaf_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

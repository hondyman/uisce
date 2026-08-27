-- 20260916_impact_simulator_and_drift_healer.up.sql
-- Lineage Graph Blast-Radius Simulator & Autonomous Schema Drift Auto-Healer

CREATE SCHEMA IF NOT EXISTS catalog_drift;

-- 1. Metadata Mutation Change Proposals & Impact Simulations
CREATE TABLE IF NOT EXISTS catalog_drift.mutation_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    target_node_id UUID NOT NULL,
    mutation_type VARCHAR(50) NOT NULL,
    proposed_payload JSONB NOT NULL,
    
    total_impacted_nodes INT NOT NULL DEFAULT 0,
    impact_severity VARCHAR(20) NOT NULL DEFAULT 'GREEN',
    blast_radius_manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_SIMULATION',
    simulated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Physical Schema Drift Crawl Log
CREATE TABLE IF NOT EXISTS catalog_drift.detected_schema_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    backend_id UUID NOT NULL,
    schema_name VARCHAR(100) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    drift_type VARCHAR(50) NOT NULL,
    old_definition JSONB,
    new_definition JSONB,
    is_healed BOOLEAN DEFAULT FALSE,
    detected_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Autonomous AI Auto-Healing Candidate Proposals
CREATE TABLE IF NOT EXISTS catalog_drift.auto_healing_proposals (
    remediation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_id UUID NOT NULL REFERENCES catalog_drift.detected_schema_events(event_id) ON DELETE CASCADE,
    broken_bo_field_id UUID NOT NULL,
    broken_column_node_id UUID NOT NULL,
    
    candidate_column_node_id UUID NOT NULL,
    vector_cosine_similarity NUMERIC(6, 4) NOT NULL,
    synthetic_query_ast_pass BOOLEAN DEFAULT FALSE,
    remediation_sql TEXT NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_STEWARD_APPROVAL',
    applied_by VARCHAR(100),
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drift_remediation_status 
ON catalog_drift.auto_healing_proposals (tenant_id, status, vector_cosine_similarity DESC);

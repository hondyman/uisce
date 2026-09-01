-- 20260829_self_healing_schema_drift.up.sql
-- Autonomous Schema Drift, Matcher Proposals, and Maker-Checker Governance

CREATE SCHEMA IF NOT EXISTS catalog_drift;

-- 1. Raw Schema Drift Events
CREATE TABLE IF NOT EXISTS catalog_drift.schema_drift_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    backend_id UUID NOT NULL,
    table_node_id UUID NOT NULL,
    change_type VARCHAR(50) NOT NULL, -- COLUMN_DROPPED, COLUMN_RENAMED, TYPE_MUTATED, TABLE_DROPPED
    column_name VARCHAR(100) NOT NULL,
    old_data_type VARCHAR(50),
    new_data_type VARCHAR(50),
    detected_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drift_events_tenant 
    ON catalog_drift.schema_drift_events (tenant_id, table_node_id, detected_at DESC);

-- 2. Staged Drift Remediation Proposals
CREATE TABLE IF NOT EXISTS catalog_drift.schema_drift_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_id UUID REFERENCES catalog_drift.schema_drift_events(event_id) ON DELETE SET NULL,
    field_binding_id UUID NOT NULL,
    orphaned_node_id UUID NOT NULL,
    proposed_candidate_node_id UUID NOT NULL,
    confidence_score NUMERIC(5, 2) NOT NULL, -- 0.00 to 100.00%
    matching_strategy VARCHAR(50) NOT NULL, -- FINANCIAL_SYNONYM, PGVECTOR_COSINE, SUBSTRING
    remediation_rationale TEXT NOT NULL,
    blast_radius_affected_reports INT DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_APPROVAL', -- PENDING_APPROVAL, APPLIED, REJECTED
    maker_user_id VARCHAR(100) NOT NULL DEFAULT 'SYSTEM_AUTONOMOUS_DRIFT_DAEMON',
    checker_user_id VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_maker_checker_drift CHECK (
        status != 'APPLIED' OR (checker_user_id IS NOT NULL AND checker_user_id <> maker_user_id)
    )
);

CREATE INDEX IF NOT EXISTS idx_drift_proposals_status 
    ON catalog_drift.schema_drift_proposals (tenant_id, status, confidence_score DESC);

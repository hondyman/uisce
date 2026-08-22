-- backend/migrations/20260829_self_healing_schema_drift.sql
-- Autonomous Schema Drift Detection, Remediation Proposals & Hot-Swap Ledger

CREATE SCHEMA IF NOT EXISTS catalog_drift;

-- 1. Raw Schema Drift Change Events
CREATE TABLE IF NOT EXISTS catalog_drift.schema_drift_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    backend_id UUID NOT NULL,
    table_node_id UUID NOT NULL REFERENCES public.catalog_node(node_id) ON DELETE CASCADE,
    change_type VARCHAR(50) NOT NULL, -- COLUMN_DROPPED, COLUMN_RENAMED, TYPE_MUTATED, TABLE_DROPPED, NEW_COLUMN
    column_name VARCHAR(100) NOT NULL,
    old_data_type VARCHAR(50),
    new_data_type VARCHAR(50),
    detected_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drift_events_tenant 
ON catalog_drift.schema_drift_events (tenant_id, table_node_id, detected_at DESC);

-- 2. Staged Drift Remediation Proposals & Impact Assessment
CREATE TABLE IF NOT EXISTS catalog_drift.schema_drift_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES catalog_drift.schema_drift_events(event_id) ON DELETE CASCADE,
    bo_id UUID NOT NULL,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    field_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    current_source_node_id UUID,
    proposed_source_node_id UUID NOT NULL REFERENCES public.catalog_node(node_id) ON DELETE CASCADE,
    proposed_column_name VARCHAR(100) NOT NULL,
    confidence_score NUMERIC(5, 4) NOT NULL, -- 0.0000 to 1.0000
    matching_strategy VARCHAR(50) NOT NULL,  -- SUBSTRING_SIMILARITY, FINANCIAL_SYNONYM_DICTIONARY, EXACT_PREFIX
    affected_reports_count INT DEFAULT 0,
    status VARCHAR(30) DEFAULT 'PENDING',    -- PENDING, APPLIED, REJECTED, AUTO_PATCHED
    remediation_rationale TEXT NOT NULL,
    applied_by_user_id VARCHAR(100),
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_drift_bo_field_binding UNIQUE (tenant_id, bo_id, binding_id, field_id, status)
);

CREATE INDEX IF NOT EXISTS idx_drift_proposals_status 
ON catalog_drift.schema_drift_proposals (tenant_id, status, confidence_score DESC);

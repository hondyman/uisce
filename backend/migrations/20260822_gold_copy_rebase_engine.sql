-- 20260822_gold_copy_rebase_engine.sql
-- Tracks base lineage versions and rebase conflict states across tenants

ALTER TABLE public.catalog_node
    ADD COLUMN IF NOT EXISTS version_id INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS derived_from_version_id INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS base_snapshot_properties JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS last_rebased_at TIMESTAMPTZ;

ALTER TABLE public.catalog_edge
    ADD COLUMN IF NOT EXISTS version_id INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS derived_from_version_id INT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS base_snapshot_properties JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS last_rebased_at TIMESTAMPTZ;

ALTER TABLE public.catalog_edge_rejection_store
    ADD COLUMN IF NOT EXISTS gold_copy_rationale_snapshot TEXT,
    ADD COLUMN IF NOT EXISTS rebase_status VARCHAR(50) DEFAULT 'PRESERVED', -- PRESERVED, REVIEW_REQUIRED, RE_EVALUATED
    ADD COLUMN IF NOT EXISTS review_notes TEXT;

CREATE TABLE IF NOT EXISTS public.catalog_rebase_conflict_ledger (
    conflict_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- CATALOG_NODE, CATALOG_EDGE, BO_FIELD
    entity_id UUID NOT NULL,
    gold_copy_node_id UUID NOT NULL,
    base_v1_version INT NOT NULL,
    base_v2_version INT NOT NULL,
    base_v1_payload JSONB NOT NULL,
    base_v2_payload JSONB NOT NULL,
    tenant_custom_payload JSONB NOT NULL,
    conflicting_keys JSONB NOT NULL,
    resolution_status VARCHAR(50) DEFAULT 'PENDING_REVIEW', -- PENDING_REVIEW, AUTO_RESOLVED, RESOLVED_TENANT_OVERRIDE, RESOLVED_GOLD_COPY_ADOPTED
    resolved_by TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rebase_conflicts_lookup 
ON public.catalog_rebase_conflict_ledger (tenant_id, resolution_status);

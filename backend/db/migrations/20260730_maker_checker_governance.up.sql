-- Migration: Maker-Checker Configuration Governance (Four-Eyes Principle)
-- Date: 2026-07-30
-- Description: Adds status branching & change request table to enforce multi-user approval before activating catalog nodes/edges.

BEGIN;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'governance_status_enum') THEN
        CREATE TYPE governance_status_enum AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'ACTIVE', 'REJECTED');
    END IF;
END $$;

ALTER TABLE public.catalog_node 
ADD COLUMN IF NOT EXISTS governance_status governance_status_enum DEFAULT 'ACTIVE',
ADD COLUMN IF NOT EXISTS branch_id UUID;

ALTER TABLE public.catalog_edge 
ADD COLUMN IF NOT EXISTS governance_status governance_status_enum DEFAULT 'ACTIVE',
ADD COLUMN IF NOT EXISTS branch_id UUID;

CREATE TABLE IF NOT EXISTS public.catalog_change_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    branch_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    maker_user_id UUID NOT NULL,
    checker_user_id UUID,
    status governance_status_enum NOT NULL DEFAULT 'PENDING_APPROVAL',
    proposed_diff_json JSONB NOT NULL,
    ast_diff_summary TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_change_requests_tenant ON public.catalog_change_requests(tenant_id, status);

COMMIT;

-- Migration: Agentic Workflow Orchestration & Maker-Checker Compliance Engine
-- Date: 2026-07-31
-- Purpose: Schema for AI-generated operational proposals, pre-trade compliance gates, and human Checker dual-authorization approvals.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'approval_status_enum') THEN
        CREATE TYPE approval_status_enum AS ENUM ('PENDING_CHECKER', 'APPROVED', 'REJECTED', 'COMPLIANCE_FAILED');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.agent_approval_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    agent_id VARCHAR(150) NOT NULL, -- e.g. 'RebalanceAgent-v2'
    target_bo_id VARCHAR(128) NOT NULL,
    action_type VARCHAR(50) NOT NULL, -- 'BULK_REBALANCE', 'SCHEMA_EXTENSION', 'STATUS_OVERRIDE'
    proposed_payload JSONB NOT NULL,
    compliance_validation_results JSONB NOT NULL,
    status approval_status_enum NOT NULL DEFAULT 'PENDING_CHECKER',
    created_by_ai BOOLEAN DEFAULT true,
    checked_by VARCHAR(255), -- User ID of the human reviewer
    checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_agent_tickets_status ON public.agent_approval_tickets(tenant_id, status);

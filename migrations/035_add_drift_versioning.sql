-- Migration: 035_add_drift_versioning.sql
-- Adds version tracking columns for drift analysis and creates governance auditing log table.

DO $$ 
BEGIN
    -- 1. Add version tracking to business_object
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='business_object' AND column_name='version_id') THEN
        ALTER TABLE public.business_object ADD COLUMN version_id VARCHAR(50) DEFAULT '1.0.0';
    END IF;

    -- 2. Add version tracking to business_object_field (singular matching the real table)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='business_object_field' AND column_name='version_id') THEN
        ALTER TABLE public.business_object_field ADD COLUMN version_id VARCHAR(50) DEFAULT '1.0.0';
        ALTER TABLE public.business_object_field ADD COLUMN derived_from_version_id VARCHAR(50);
    END IF;

    -- 3. Governance Audit Log Table
    CREATE TABLE IF NOT EXISTS public.bo_governance_events (
        event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        tenant_id UUID NOT NULL,
        bo_id UUID NOT NULL,
        event_type VARCHAR(50) NOT NULL, -- e.g., 'DRIFT_DETECTED', 'REBASE_COMPLETED'
        details JSONB NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now()
    );
END $$;

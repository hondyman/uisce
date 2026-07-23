-- Scheduler Governance Schema
-- Phase 11 of Scheduler Intelligence Layer

-- 1. Scheduler ChangeSets
CREATE TABLE IF NOT EXISTS scheduler_changesets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- NULL for GLOBAL scope
    scope VARCHAR(16) NOT NULL CHECK (scope IN ('GLOBAL', 'TENANT')),
    type VARCHAR(64) NOT NULL, -- scheduler.job.create, scheduler.dag.update, etc.
    title VARCHAR(255) NOT NULL,
    description TEXT,
    author TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review' CHECK (status IN (
        'draft', 'pending_review', 'approved', 'rejected', 'applied', 'rolled_back'
    )),
    
    -- Target Object
    target_type VARCHAR(32) NOT NULL CHECK (target_type IN ('JOB', 'DAG', 'CALENDAR')),
    target_id UUID, -- May be NULL for CREATE operations until applied
    
    -- Change Definition
    diff JSONB NOT NULL, -- {old: {...}, new: {...}}
    
    -- Analysis & AI
    impact_analysis JSONB DEFAULT '{}', -- Blast radius, affected tenants, SLO
    ai_review JSONB DEFAULT '{}', -- Summary, risk score, PII flags
    risk_score FLOAT DEFAULT 0.0,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Scheduler ChangeSet Approvals
CREATE TABLE IF NOT EXISTS scheduler_changeset_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    changeset_id UUID NOT NULL REFERENCES scheduler_changesets(id) ON DELETE CASCADE,
    approver_id TEXT NOT NULL,
    approver_role VARCHAR(64) NOT NULL,
    decision VARCHAR(32) NOT NULL CHECK (decision IN ('approved', 'rejected', 'needs_info')),
    comment TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indices
CREATE INDEX idx_scheduler_changesets_tenant ON scheduler_changesets(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_scheduler_changesets_status ON scheduler_changesets(status);
CREATE INDEX idx_scheduler_changesets_scope ON scheduler_changesets(scope);
CREATE INDEX idx_scheduler_changeset_approvals_cs ON scheduler_changeset_approvals(changeset_id);

-- Update calendar table to support scopes if not already existing
DO $$ 
BEGIN 
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='business_calendars') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='business_calendars' AND column_name='scope') THEN
            ALTER TABLE business_calendars ADD COLUMN scope VARCHAR(16) DEFAULT 'TENANT';
            UPDATE business_calendars SET scope = 'GLOBAL' WHERE is_global = TRUE;
        END IF;
    ELSE
        RAISE NOTICE 'business_calendars table does not exist; skipping calendar scope migration';
    END IF;
END $$;

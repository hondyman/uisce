-- Migration: Graph-Based Compliance Governance
-- Date: 2026-02-01
-- Description: Adds semantic_compliance_events and extends scheduler_jobs. 
--             Business Terms and Semantic Terms leverage existing catalog_node and catalog_edge tables.

-- 0. Expand Catalog Node Constraints
-- We need to drop the existing check constraint and add a new one that includes governance types.
DO $$
BEGIN
    -- Attempt to drop the constraint if it exists (name might vary, so we look it up or use standard naming)
    -- Assuming standard naming 'catalog_node_kind_check' from 000004_create_catalog_schema.sql
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'catalog_node_kind_check') THEN
        ALTER TABLE catalog_node DROP CONSTRAINT catalog_node_kind_check;
    END IF;

    -- Re-add with expanded values
    ALTER TABLE catalog_node ADD CONSTRAINT catalog_node_kind_check 
    CHECK (kind IN ('table', 'view', 'bo', 'BUSINESS_TERM', 'SEMANTIC_TERM'));
END $$;

-- 1. Semantic Compliance Events (Event Sourcing)
CREATE TABLE IF NOT EXISTS semantic_compliance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semantic_term_id TEXT, -- Can be null if event is for Business Term (referred to by catalog_node.id)
    business_term_id TEXT, -- Can be null if event is strictly technical (referred to by catalog_node.id)
    event_type TEXT NOT NULL, -- 'DRIFT', 'UPDATE', 'VIOLATION', 'BUSINESS_TERM_UPDATE'
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_compliance_events_type ON semantic_compliance_events(event_type);
CREATE INDEX idx_compliance_events_created ON semantic_compliance_events(created_at DESC);

-- 2. Extend Scheduler Jobs with Compliance Metadata
-- Note: 'scheduler_jobs' table is assumed to exist from previous migrations.
-- We check if column exists first to be safe.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'scheduler_jobs') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'scheduler_jobs' AND column_name = 'compliance') THEN
            ALTER TABLE scheduler_jobs ADD COLUMN compliance JSONB NOT NULL DEFAULT '{}'::jsonb;
            CREATE INDEX idx_scheduler_jobs_compliance ON scheduler_jobs USING GIN (compliance);
        END IF;
    ELSE
        RAISE NOTICE 'Table scheduler_jobs does not exist, skipping column addition.';
    END IF;
END $$;

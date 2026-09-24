-- 20260905_ambient_tribal_knowledge_pipeline.up.sql
-- Ambient Tribal Knowledge Ingestion, Proposals & Sanity Check Diagnostics

CREATE SCHEMA IF NOT EXISTS catalog_ambient;

-- 1. Raw Ingestion Ingress Stream
CREATE TABLE IF NOT EXISTS catalog_ambient.ingestion_stream (
    stream_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_channel VARCHAR(50) NOT NULL, -- SLACK, MS_TEAMS, EMAIL, REST_CAPTURE
    originator_id VARCHAR(100) NOT NULL,
    raw_text_payload TEXT NOT NULL,
    payload_sha256 VARCHAR(64) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'RECEIVED', -- RECEIVED, PARSED, FAILED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Staged Knowledge Proposals & Sanity Check Diagnostics
CREATE TABLE IF NOT EXISTS catalog_ambient.knowledge_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    stream_id UUID REFERENCES catalog_ambient.ingestion_stream(stream_id) ON DELETE SET NULL,
    target_bo_id UUID,
    proposed_okf_key VARCHAR(150) NOT NULL,
    okf_yaml_frontmatter JSONB NOT NULL,
    okf_markdown_body TEXT NOT NULL,
    generated_base_sql TEXT,
    
    -- Automated Sanity Verification Metrics
    sanity_pass BOOLEAN DEFAULT FALSE,
    graph_resolved BOOLEAN DEFAULT FALSE,
    has_contradiction BOOLEAN DEFAULT FALSE,
    sanity_report JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Dual-Routing Workflow State
    destination_scope VARCHAR(30) NOT NULL DEFAULT 'TENANT_LOCAL', -- TENANT_LOCAL, NOMINATED_FOR_CORE, PROMOTED_TO_CORE
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_REVIEW', -- PENDING_REVIEW, APPROVED_LOCAL, REJECTED, SUBMITTED_TO_UISCE, ACCEPTED_CORE
    local_reviewed_by VARCHAR(100),
    local_reviewed_at TIMESTAMPTZ,
    core_reviewed_by VARCHAR(100),
    core_reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ambient_proposals_status 
ON catalog_ambient.knowledge_proposals (tenant_id, status, destination_scope);

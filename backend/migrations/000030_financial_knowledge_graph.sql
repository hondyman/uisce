-- Financial Knowledge Graph Schema
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- CREATE EXTENSION IF NOT EXISTS vector;

-- Migration: 000030_financial_knowledge_graph.sql
-- Created: 2025-11-24
-- Description: Financial Knowledge Graph (FKG) with entity relationships, 
--              ownership chains, and hierarchical taxonomies

-- ===========================================
-- EXTENSIONS
-- ===========================================

-- Enable ltree for static hierarchy traversal (GICS codes, org charts)
CREATE EXTENSION IF NOT EXISTS ltree;

-- pg_trgm already enabled for entity resolution
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ===========================================
-- FINANCIAL ENTITIES (Node Table)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.financial_entities (
    entity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Canonical identifiers
    canonical_id TEXT UNIQUE, -- LEI, CIK, or internal ID
    lei TEXT, -- Legal Entity Identifier (20-char alphanumeric)
    cik TEXT, -- SEC Central Index Key
    ein TEXT, -- Employer Identification Number
    
    -- Entity classification
    entity_type TEXT NOT NULL CHECK (entity_type IN (
        'CORP', 'INDIVIDUAL', 'TRUST', 'GOV', 'FUND', 
        'PARTNERSHIP', 'LLC', 'NONPROFIT', 'ESTATE', 'CUSTODIAN'
    )),
    entity_subtype TEXT, -- e.g., 'hedge_fund', 'family_office', 'bank'
    
    -- Core attributes
    name TEXT NOT NULL,
    legal_name TEXT, -- Full legal name for matching
    trade_name TEXT, -- DBA name
    
    -- Jurisdiction & domicile
    jurisdiction_code CHAR(2), -- ISO 3166-1 alpha-2
    state_province TEXT,
    formation_date DATE,
    dissolution_date DATE,
    
    -- Classification & taxonomy
    sector_path ltree, -- e.g., 'Financials.Banks.Regional'
    industry_code TEXT, -- GICS code
    sic_code TEXT, -- Standard Industrial Classification
    naics_code TEXT, -- North American Industry Classification
    
    -- Risk & compliance
    risk_score DECIMAL(5, 2) CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB DEFAULT '[]'::jsonb,
    pep_status BOOLEAN DEFAULT FALSE, -- Politically Exposed Person
    sanctions_status BOOLEAN DEFAULT FALSE,
    last_kyc_date TIMESTAMP WITH TIME ZONE,
    kyc_status TEXT CHECK (kyc_status IN ('pending', 'approved', 'rejected', 'expired')),
    
    -- Financial data
    market_cap DECIMAL(20, 2),
    total_assets DECIMAL(20, 2),
    total_revenue DECIMAL(20, 2),
    fiscal_year_end TEXT, -- e.g., '12-31'
    
    -- Flexible attributes for regulatory variance
    properties JSONB DEFAULT '{}'::jsonb,
    
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    -- Data lineage
    source_system TEXT, -- e.g., 'SEC_EDGAR', 'GLEIF', 'manual'
    source_id TEXT,
    last_verified_at TIMESTAMP WITH TIME ZONE
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_entity_tenant ON public.financial_entities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_entity_canonical ON public.financial_entities(canonical_id);
CREATE INDEX IF NOT EXISTS idx_entity_lei ON public.financial_entities(lei);
CREATE INDEX IF NOT EXISTS idx_entity_cik ON public.financial_entities(cik);
CREATE INDEX IF NOT EXISTS idx_entity_type ON public.financial_entities(tenant_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_entity_jurisdiction ON public.financial_entities(jurisdiction_code);

-- GIN index for JSONB property queries
CREATE INDEX IF NOT EXISTS idx_entity_properties ON public.financial_entities USING GIN (properties);
CREATE INDEX IF NOT EXISTS idx_entity_risk_factors ON public.financial_entities USING GIN (risk_factors);

-- Trigram index for fuzzy name matching (entity resolution)
CREATE INDEX IF NOT EXISTS idx_entity_name_trgm ON public.financial_entities USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_entity_legal_name_trgm ON public.financial_entities USING GIN (legal_name gin_trgm_ops);

-- ltree index for sector hierarchy traversal
CREATE INDEX IF NOT EXISTS idx_entity_sector_path ON public.financial_entities USING GIST (sector_path);

-- ===========================================
-- OWNERSHIP RELATIONSHIPS (Edge Table)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.ownership_relationships (
    relationship_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Directed edge: source owns/controls target
    source_entity_id UUID NOT NULL REFERENCES public.financial_entities(entity_id) ON DELETE CASCADE,
    target_entity_id UUID NOT NULL REFERENCES public.financial_entities(entity_id) ON DELETE CASCADE,
    
    -- Relationship classification
    relationship_type TEXT NOT NULL CHECK (relationship_type IN (
        'SHAREHOLDER', 'BENEFICIAL_OWNER', 'DIRECTOR', 'OFFICER',
        'SUBSIDIARY', 'PARENT', 'AFFILIATE', 'CONTROL_PERSON',
        'CUSTODIAN', 'TRUSTEE', 'BENEFICIARY', 'NOMINEE',
        'JOINT_VENTURE', 'PARTNERSHIP', 'ADVISOR'
    )),
    
    -- Ownership metrics
    percentage_ownership DECIMAL(7, 4) CHECK (percentage_ownership >= 0 AND percentage_ownership <= 100),
    shares_held DECIMAL(20, 4),
    share_class TEXT, -- 'Common', 'Preferred A', etc.
    voting_power DECIMAL(7, 4), -- May differ from ownership %
    
    -- Control attributes
    has_voting_rights BOOLEAN DEFAULT FALSE,
    has_board_seat BOOLEAN DEFAULT FALSE,
    is_controlling_interest BOOLEAN DEFAULT FALSE,
    
    -- Temporal validity
    effective_date DATE,
    termination_date DATE, -- NULL implies currently active
    
    -- Provenance
    metadata JSONB DEFAULT '{}'::jsonb, -- e.g., {"source": "SEC Filing 10-K, 2023"}
    filing_reference TEXT, -- SEC filing number, etc.
    verified_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT check_no_self_loop CHECK (source_entity_id != target_entity_id),
    CONSTRAINT unique_relationship UNIQUE (tenant_id, source_entity_id, target_entity_id, relationship_type, effective_date)
);

-- Critical indexes for graph traversal (bidirectional)
CREATE INDEX IF NOT EXISTS idx_rel_source ON public.ownership_relationships(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_rel_target ON public.ownership_relationships(target_entity_id);
CREATE INDEX IF NOT EXISTS idx_rel_source_target ON public.ownership_relationships(source_entity_id, target_entity_id);
CREATE INDEX IF NOT EXISTS idx_rel_target_source ON public.ownership_relationships(target_entity_id, source_entity_id);
CREATE INDEX IF NOT EXISTS idx_rel_tenant_type ON public.ownership_relationships(tenant_id, relationship_type);
CREATE INDEX IF NOT EXISTS idx_rel_active ON public.ownership_relationships(tenant_id) WHERE termination_date IS NULL;
CREATE INDEX IF NOT EXISTS idx_rel_metadata ON public.ownership_relationships USING GIN (metadata);

-- ===========================================
-- SECTOR HIERARCHY (Static Tree using ltree)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.sector_hierarchy (
    sector_id SERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    path ltree NOT NULL,
    
    -- GICS mapping
    gics_code TEXT,
    gics_level INT CHECK (gics_level BETWEEN 1 AND 4), -- Sector, Industry Group, Industry, Sub-Industry
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sector_path ON public.sector_hierarchy USING GIST (path);
CREATE INDEX IF NOT EXISTS idx_sector_tenant ON public.sector_hierarchy(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sector_gics ON public.sector_hierarchy(gics_code);

-- ===========================================
-- ENTITY RESOLUTION CANDIDATES
-- ===========================================

CREATE TABLE IF NOT EXISTS public.entity_resolution_queue (
    resolution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Input data
    input_name TEXT NOT NULL,
    input_jurisdiction TEXT,
    input_identifiers JSONB DEFAULT '{}'::jsonb,
    
    -- Matching results
    candidate_entity_id UUID REFERENCES public.financial_entities(entity_id),
    similarity_score DECIMAL(5, 4),
    match_type TEXT CHECK (match_type IN ('exact', 'high_confidence', 'needs_review', 'no_match')),
    
    -- Resolution status
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'auto_matched', 'manual_review', 'resolved', 'new_entity')),
    resolved_by UUID,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resolution_status ON public.entity_resolution_queue(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_resolution_candidate ON public.entity_resolution_queue(candidate_entity_id);

-- ===========================================
-- DOCUMENT CHUNKS (MMU Pipeline Output)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.document_chunks (
    chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Link to entity
    entity_id UUID REFERENCES public.financial_entities(entity_id) ON DELETE SET NULL,
    
    -- Document metadata
    document_id UUID NOT NULL,
    document_type TEXT, -- '10-K', '10-Q', 'proxy', 'prospectus'
    document_date DATE,
    filing_number TEXT,
    
    -- Chunk data
    page_number INT,
    chunk_index INT, -- Order within document
    content TEXT NOT NULL,
    
    -- Vector embedding for semantic search (1536 for OpenAI, 768 for Google)
    -- embedding vector(1536),
    
    -- Extracted structured data
    extracted_data JSONB DEFAULT '{}'::jsonb,
    extraction_model TEXT, -- 'gpt-4o', 'gemini-1.5-pro', etc.
    extraction_confidence DECIMAL(5, 4),
    
    -- Full-text search
    content_tsv tsvector GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- HNSW index for fast approximate nearest neighbor search
-- CREATE INDEX IF NOT EXISTS idx_chunk_embedding_hnsw ON public.document_chunks 
--     USING hnsw (embedding vector_cosine_ops) 
--     WITH (m = 16, ef_construction = 64);

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_chunk_content_tsv ON public.document_chunks USING GIN (content_tsv);

-- Standard indexes
CREATE INDEX IF NOT EXISTS idx_chunk_tenant ON public.document_chunks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_chunk_entity ON public.document_chunks(entity_id);
CREATE INDEX IF NOT EXISTS idx_chunk_document ON public.document_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_chunk_extracted ON public.document_chunks USING GIN (extracted_data);

-- ===========================================
-- ENTITY MONITORS (Proactive Intelligence)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.entity_monitors (
    monitor_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES public.financial_entities(entity_id) ON DELETE CASCADE,
    
    -- Monitor configuration
    monitor_type TEXT NOT NULL CHECK (monitor_type IN (
        'price_alert', 'news_sentiment', 'filing_watch', 
        'risk_threshold', 'ownership_change', 'covenant_breach'
    )),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Temporal workflow reference
    workflow_id TEXT UNIQUE, -- Temporal workflow ID
    workflow_run_id TEXT,
    
    -- State tracking
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'stopped', 'error')),
    last_check_at TIMESTAMP WITH TIME ZONE,
    last_alert_at TIMESTAMP WITH TIME ZONE,
    check_count INT DEFAULT 0,
    alert_count INT DEFAULT 0,
    
    -- Error handling
    last_error TEXT,
    error_count INT DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_monitor_tenant ON public.entity_monitors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_monitor_entity ON public.entity_monitors(entity_id);
CREATE INDEX IF NOT EXISTS idx_monitor_workflow ON public.entity_monitors(workflow_id);
CREATE INDEX IF NOT EXISTS idx_monitor_status ON public.entity_monitors(tenant_id, status);

-- ===========================================
-- RISK EVENTS (Alert History)
-- ===========================================

CREATE TABLE IF NOT EXISTS public.risk_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES public.financial_entities(entity_id) ON DELETE CASCADE,
    monitor_id UUID REFERENCES public.entity_monitors(monitor_id) ON DELETE SET NULL,
    
    -- Event details
    event_type TEXT NOT NULL,
    severity TEXT CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    title TEXT NOT NULL,
    description TEXT,
    
    -- Event data
    event_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_url TEXT,
    
    -- Processing status
    status TEXT DEFAULT 'new' CHECK (status IN ('new', 'acknowledged', 'investigating', 'resolved', 'false_positive')),
    acknowledged_by UUID,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_tenant ON public.risk_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_risk_entity ON public.risk_events(entity_id);
CREATE INDEX IF NOT EXISTS idx_risk_status ON public.risk_events(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_risk_severity ON public.risk_events(tenant_id, severity);
CREATE INDEX IF NOT EXISTS idx_risk_created ON public.risk_events(tenant_id, created_at DESC);

-- ===========================================
-- FUNCTIONS: Ultimate Beneficial Ownership
-- ===========================================

-- Function to calculate effective ownership through chains
CREATE OR REPLACE FUNCTION calculate_effective_ownership(
    p_tenant_id UUID,
    p_target_entity_id UUID,
    p_max_depth INT DEFAULT 20,
    p_min_ownership DECIMAL DEFAULT 0.01
)
RETURNS TABLE (
    owner_entity_id UUID,
    owner_name TEXT,
    direct_ownership DECIMAL,
    effective_ownership DECIMAL,
    depth INT,
    path UUID[],
    is_cycle BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE ownership_chain AS (
        -- Anchor: Direct owners of target entity
        SELECT 
            r.source_entity_id,
            r.target_entity_id,
            r.percentage_ownership / 100.0 AS direct_own,
            r.percentage_ownership / 100.0 AS effective_own,
            1 AS chain_depth,
            ARRAY[r.source_entity_id] AS chain_path,
            FALSE AS cycle_detected
        FROM ownership_relationships r
        WHERE r.target_entity_id = p_target_entity_id
          AND r.tenant_id = p_tenant_id
          AND r.termination_date IS NULL
          AND r.percentage_ownership > 0
        
        UNION ALL
        
        -- Recursive: Owners of owners
        SELECT 
            r.source_entity_id,
            r.target_entity_id,
            r.percentage_ownership / 100.0,
            (oc.effective_own * r.percentage_ownership / 100.0)::DECIMAL,
            oc.chain_depth + 1,
            oc.chain_path || r.source_entity_id,
            r.source_entity_id = ANY(oc.chain_path) -- Cycle detection
        FROM ownership_relationships r
        JOIN ownership_chain oc ON r.target_entity_id = oc.source_entity_id
        WHERE r.tenant_id = p_tenant_id
          AND r.termination_date IS NULL
          AND oc.chain_depth < p_max_depth
          AND NOT oc.cycle_detected
          AND r.percentage_ownership > 0
    )
    SELECT 
        oc.source_entity_id,
        e.name,
        oc.direct_own * 100,
        oc.effective_own * 100,
        oc.chain_depth,
        oc.chain_path,
        oc.cycle_detected
    FROM ownership_chain oc
    JOIN financial_entities e ON oc.source_entity_id = e.entity_id
    WHERE oc.effective_own >= p_min_ownership
    ORDER BY oc.effective_own DESC;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- FUNCTIONS: JSONB Drift Detection
-- ===========================================

CREATE OR REPLACE FUNCTION jsonb_diff(old_val JSONB, new_val JSONB)
RETURNS JSONB AS $$
DECLARE
    result JSONB := '{}'::jsonb;
    key TEXT;
    old_value JSONB;
    new_value JSONB;
BEGIN
    -- Find changed and added keys
    FOR key IN SELECT jsonb_object_keys(new_val)
    LOOP
        old_value := old_val -> key;
        new_value := new_val -> key;
        
        IF old_value IS NULL THEN
            -- New key added
            result := result || jsonb_build_object(key, jsonb_build_object('added', new_value));
        ELSIF old_value != new_value THEN
            -- Value changed
            result := result || jsonb_build_object(key, jsonb_build_object('old', old_value, 'new', new_value));
        END IF;
    END LOOP;
    
    -- Find deleted keys
    FOR key IN SELECT jsonb_object_keys(old_val)
    LOOP
        IF NOT new_val ? key THEN
            result := result || jsonb_build_object(key, jsonb_build_object('deleted', old_val -> key));
        END IF;
    END LOOP;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ===========================================
-- FUNCTIONS: Hybrid Search (RRF) - Commented out due to vector dependency
-- ===========================================

/*
CREATE OR REPLACE FUNCTION hybrid_search(
    p_tenant_id UUID,
    p_query_embedding vector(1536),
    p_keyword_query TEXT,
    p_entity_id UUID DEFAULT NULL,
    p_limit INT DEFAULT 20,
    p_rrf_k INT DEFAULT 60
)
RETURNS TABLE (
    chunk_id UUID,
    content TEXT,
    document_type TEXT,
    rrf_score DECIMAL,
    vector_rank INT,
    keyword_rank INT
) AS $$
BEGIN
    RETURN QUERY
    WITH semantic_search AS (
        SELECT 
            c.chunk_id,
            c.content,
            c.document_type,
            ROW_NUMBER() OVER (ORDER BY c.embedding <=> p_query_embedding) AS vec_rank
        FROM document_chunks c
        WHERE c.tenant_id = p_tenant_id
          AND (p_entity_id IS NULL OR c.entity_id = p_entity_id)
        ORDER BY c.embedding <=> p_query_embedding
        LIMIT 100
    ),
    keyword_search AS (
        SELECT 
            c.chunk_id,
            c.content,
            c.document_type,
            ROW_NUMBER() OVER (ORDER BY ts_rank_cd(c.content_tsv, plainto_tsquery('english', p_keyword_query)) DESC) AS text_rank
        FROM document_chunks c
        WHERE c.tenant_id = p_tenant_id
          AND (p_entity_id IS NULL OR c.entity_id = p_entity_id)
          AND c.content_tsv @@ plainto_tsquery('english', p_keyword_query)
        LIMIT 100
    )
    SELECT 
        COALESCE(s.chunk_id, k.chunk_id) AS chunk_id,
        COALESCE(s.content, k.content) AS content,
        COALESCE(s.document_type, k.document_type) AS document_type,
        (
            (1.0 / (p_rrf_k + COALESCE(s.vec_rank, 100))) + 
            (1.0 / (p_rrf_k + COALESCE(k.text_rank, 100)))
        )::DECIMAL AS rrf_score,
        COALESCE(s.vec_rank, 0)::INT AS vector_rank,
        COALESCE(k.text_rank, 0)::INT AS keyword_rank
    FROM semantic_search s
    FULL OUTER JOIN keyword_search k ON s.chunk_id = k.chunk_id
    ORDER BY rrf_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
*/

-- ===========================================
-- FUNCTIONS: Entity Resolution
-- ===========================================

CREATE OR REPLACE FUNCTION find_entity_matches(
    p_tenant_id UUID,
    p_name TEXT,
    p_jurisdiction CHAR(2) DEFAULT NULL,
    p_threshold DECIMAL DEFAULT 0.7,
    p_limit INT DEFAULT 10
)
RETURNS TABLE (
    entity_id UUID,
    name TEXT,
    legal_name TEXT,
    jurisdiction_code CHAR(2),
    entity_type TEXT,
    similarity_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.entity_id,
        e.name,
        e.legal_name,
        e.jurisdiction_code,
        e.entity_type,
        GREATEST(
            similarity(e.name, p_name),
            COALESCE(similarity(e.legal_name, p_name), 0),
            COALESCE(similarity(e.trade_name, p_name), 0)
        ) AS sim_score
    FROM financial_entities e
    WHERE e.tenant_id = p_tenant_id
      AND (
          e.name % p_name 
          OR e.legal_name % p_name 
          OR e.trade_name % p_name
      )
      AND (p_jurisdiction IS NULL OR e.jurisdiction_code = p_jurisdiction)
    ORDER BY sim_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- ===========================================
-- TRIGGERS: Auto-update timestamps
-- ===========================================

CREATE OR REPLACE FUNCTION update_fkg_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_financial_entities_updated ON public.financial_entities;
CREATE TRIGGER trg_financial_entities_updated
    BEFORE UPDATE ON public.financial_entities
    FOR EACH ROW
    EXECUTE FUNCTION update_fkg_timestamp();

DROP TRIGGER IF EXISTS trg_ownership_relationships_updated ON public.ownership_relationships;
CREATE TRIGGER trg_ownership_relationships_updated
    BEFORE UPDATE ON public.ownership_relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_fkg_timestamp();

DROP TRIGGER IF EXISTS trg_entity_monitors_updated ON public.entity_monitors;
CREATE TRIGGER trg_entity_monitors_updated
    BEFORE UPDATE ON public.entity_monitors
    FOR EACH ROW
    EXECUTE FUNCTION update_fkg_timestamp();

-- ===========================================
-- SEED: GICS Sector Hierarchy
-- ===========================================

INSERT INTO public.sector_hierarchy (name, display_name, path, gics_code, gics_level) VALUES
-- Level 1: Sectors
('Energy', 'Energy', 'Energy', '10', 1),
('Materials', 'Materials', 'Materials', '15', 1),
('Industrials', 'Industrials', 'Industrials', '20', 1),
('ConsumerDiscretionary', 'Consumer Discretionary', 'ConsumerDiscretionary', '25', 1),
('ConsumerStaples', 'Consumer Staples', 'ConsumerStaples', '30', 1),
('HealthCare', 'Health Care', 'HealthCare', '35', 1),
('Financials', 'Financials', 'Financials', '40', 1),
('InformationTechnology', 'Information Technology', 'InformationTechnology', '45', 1),
('CommunicationServices', 'Communication Services', 'CommunicationServices', '50', 1),
('Utilities', 'Utilities', 'Utilities', '55', 1),
('RealEstate', 'Real Estate', 'RealEstate', '60', 1),

-- Level 2: Industry Groups (sample)
('Banks', 'Banks', 'Financials.Banks', '4010', 2),
('DiversifiedFinancials', 'Diversified Financials', 'Financials.DiversifiedFinancials', '4020', 2),
('Insurance', 'Insurance', 'Financials.Insurance', '4030', 2),
('Software', 'Software & Services', 'InformationTechnology.Software', '4510', 2),
('Hardware', 'Technology Hardware', 'InformationTechnology.Hardware', '4520', 2),
('Semiconductors', 'Semiconductors', 'InformationTechnology.Semiconductors', '4530', 2),

-- Level 3: Industries (sample)
('RegionalBanks', 'Regional Banks', 'Financials.Banks.Regional', '401010', 3),
('DiversifiedBanks', 'Diversified Banks', 'Financials.Banks.Diversified', '401020', 3),
('AssetManagement', 'Asset Management', 'Financials.DiversifiedFinancials.AssetManagement', '402010', 3),
('ApplicationSoftware', 'Application Software', 'InformationTechnology.Software.Application', '451030', 3),
('SystemsSoftware', 'Systems Software', 'InformationTechnology.Software.Systems', '451020', 3)
ON CONFLICT DO NOTHING;

-- ===========================================
-- COMMENTS
-- ===========================================

COMMENT ON TABLE public.financial_entities IS 'Master registry of financial entities (corporations, individuals, trusts, funds)';
COMMENT ON TABLE public.ownership_relationships IS 'Directed edges representing ownership and control relationships between entities';
COMMENT ON TABLE public.sector_hierarchy IS 'GICS-compatible sector taxonomy using ltree for fast hierarchy traversal';
COMMENT ON TABLE public.document_chunks IS 'Vectorized document chunks for semantic search (MMU pipeline output)';
COMMENT ON TABLE public.entity_monitors IS 'Proactive intelligence monitors linked to Temporal workflows';
COMMENT ON TABLE public.risk_events IS 'Risk event history generated by entity monitors';

COMMENT ON FUNCTION calculate_effective_ownership IS 'Recursive CTE to compute ultimate beneficial ownership through ownership chains';
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'hybrid_search') THEN
    COMMENT ON FUNCTION hybrid_search IS 'Reciprocal Rank Fusion combining pgvector semantic search with full-text keyword search';
  END IF;
END;
$$;
COMMENT ON FUNCTION find_entity_matches IS 'Entity resolution using pg_trgm for fuzzy name matching';
COMMENT ON FUNCTION jsonb_diff IS 'Compute drift between two JSONB objects for change detection';

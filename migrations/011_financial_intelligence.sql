-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "ltree";
CREATE EXTENSION IF NOT EXISTS "vector";

-- 1. Financial Entities (Nodes)
CREATE TABLE financial_entities (
    entity_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canonical_id TEXT UNIQUE, -- LEI, CIK, or internal ID
    entity_type TEXT NOT NULL CHECK (entity_type IN ('CORP', 'INDIVIDUAL', 'TRUST', 'GOV')),
    name TEXT NOT NULL,
    jurisdiction_code CHAR(2), -- ISO 3166-1 alpha-2
    risk_score DECIMAL(5, 2),
    properties JSONB DEFAULT '{}'::jsonb, -- Flexible attributes
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Entity Lookup
CREATE INDEX idx_entity_canonical ON financial_entities (canonical_id);
CREATE INDEX idx_entity_properties ON financial_entities USING GIN (properties);
CREATE INDEX idx_entity_name_trgm ON financial_entities USING GIN (name gin_trgm_ops);

-- 2. Ownership Relationships (Edges)
CREATE TABLE ownership_relationships (
    relationship_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_entity_id UUID NOT NULL REFERENCES financial_entities(entity_id),
    target_entity_id UUID NOT NULL REFERENCES financial_entities(entity_id),
    relationship_type TEXT NOT NULL, -- 'SHAREHOLDER', 'DIRECTOR', 'UBO', 'SUBSIDIARY'
    percentage_ownership DECIMAL(5, 4), -- 0.0000 to 1.0000
    voting_rights BOOLEAN DEFAULT FALSE,
    start_date DATE,
    end_date DATE, -- NULL implies active
    metadata JSONB DEFAULT '{}'::jsonb,
    CONSTRAINT fk_source FOREIGN KEY (source_entity_id) REFERENCES financial_entities(entity_id),
    CONSTRAINT fk_target FOREIGN KEY (target_entity_id) REFERENCES financial_entities(entity_id),
    CONSTRAINT check_self_loop CHECK (source_entity_id != target_entity_id)
);

-- Indexes for Graph Traversal
CREATE INDEX idx_rel_source_target ON ownership_relationships (source_entity_id, target_entity_id);
CREATE INDEX idx_rel_target_source ON ownership_relationships (target_entity_id, source_entity_id);

-- 3. Document Chunks (Vectors)
CREATE TABLE document_chunks (
    chunk_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID REFERENCES financial_entities(entity_id),
    content TEXT,
    embedding vector(1536), -- OpenAI embedding dimension
    metadata JSONB
);

-- HNSW Index for Vector Search
CREATE INDEX idx_vector_hnsw ON document_chunks 
USING hnsw (embedding vector_cosine_ops) 
WITH (m = 16, ef_construction = 64);

-- 4. Sector Hierarchy (Taxonomy)
CREATE TABLE sector_hierarchy (
    sector_id SERIAL PRIMARY KEY,
    name TEXT,
    path ltree
);

CREATE INDEX idx_sector_path ON sector_hierarchy USING GIST (path);

-- Enable pgvector extension for semantic similarity search
DO $do$
BEGIN
  BEGIN
    CREATE EXTENSION IF NOT EXISTS vector;
  EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pgvector extension not available or cannot be created: %', SQLERRM;
  END;
END
$do$;

-- Titan Knowledge Base for RAG-powered migration (created only if pgvector is available)
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
    CREATE TABLE IF NOT EXISTS titan_knowledge_base (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        category VARCHAR(50) NOT NULL, -- 'component_schema', 'data_object', 'example', 'rego_policy'
        name VARCHAR(255) NOT NULL,
        description TEXT,
        content JSONB NOT NULL,
        embedding VECTOR(768), -- Gemini/OpenAI embedding dimension
        tags TEXT[], -- For filtering (e.g., ['branch', 'approval', 'financial'])
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX IF NOT EXISTS idx_tkb_category ON titan_knowledge_base(category);
    CREATE INDEX IF NOT EXISTS idx_tkb_embedding ON titan_knowledge_base USING ivfflat (embedding vector_cosine_ops);

    -- Seed initial Knowledge Base with Titan component schemas
    -- Seed handled conditionally above; no-op here

  ELSE
    RAISE NOTICE 'pgvector not available; skipping titan_knowledge_base creation and seed.';
  END IF;
END
$do$;

-- Migration Jobs Table (tracks Code-to-Config migrations)
CREATE TABLE IF NOT EXISTS migration_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, ANALYZING, EXTRACTED, GENERATING, REVIEW, APPROVED, REJECTED
    
    -- Stage 1: Code-to-Intent
    source_code TEXT NOT NULL,
    source_language VARCHAR(50), -- 'java', 'csharp', 'cobol', 'sql', 'python'
    ast_json JSONB,
    extracted_intent JSONB, -- BusinessRuleIntent JSON
    
    -- Stage 2: Intent-to-Config
    generated_dag JSONB,
    generated_rego TEXT,
    rag_context JSONB, -- Retrieved knowledge base items used
    
    -- HITL
    reviewer_id VARCHAR(255),
    review_notes TEXT,
    approved_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_migration_jobs_status ON migration_jobs(status);
CREATE INDEX idx_migration_jobs_tenant ON migration_jobs(tenant_id);

-- Seed initial Knowledge Base with Titan component schemas (handled above when pgvector present)





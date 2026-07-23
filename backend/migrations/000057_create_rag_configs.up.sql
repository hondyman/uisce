-- Migration: Create RAG configuration tables
-- Description: Stores metadata-driven configurations for RAG and document processing

CREATE TABLE IF NOT EXISTS rag_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    embedding_model JSONB NOT NULL DEFAULT '{
        "provider": "openai",
        "model": "text-embedding-ada-002",
        "dimensions": 1536
    }',
    retrieval_config JSONB NOT NULL DEFAULT '{
        "top_k": 10,
        "similarity_threshold": 0.7,
        "rerank": false
    }',
    hybrid_search JSONB NOT NULL DEFAULT '{
        "enabled": true,
        "semantic_weight": 0.7,
        "keyword_weight": 0.3
    }',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE TABLE IF NOT EXISTS document_type_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    type_code VARCHAR(50) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    chunking_strategy JSONB NOT NULL DEFAULT '{
        "method": "semantic",
        "max_chunk_size": 512,
        "overlap_tokens": 50
    }',
    extraction_rules JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, type_code)
);

CREATE INDEX IF NOT EXISTS idx_rag_configs_tenant ON rag_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_doc_type_configs_tenant ON document_type_configs(tenant_id);

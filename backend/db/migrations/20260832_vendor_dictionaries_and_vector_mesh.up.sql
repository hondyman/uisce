-- 20260832_vendor_dictionaries_and_vector_mesh.up.sql
-- Vendor Data Dictionaries & Multi-Vendor Mesh Schema

CREATE SCHEMA IF NOT EXISTS catalog_vendor;

-- 1. Master Vendor Data Dictionary
CREATE TABLE IF NOT EXISTS catalog_vendor.vendor_data_dictionary (
    vendor_field_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_name VARCHAR(50) NOT NULL, -- BLOOMBERG, REFINITIV, FACTSET, SP_CAPITAL_IQ
    field_mnemonic VARCHAR(100) NOT NULL, -- PX_LAST, ID_ISIN, YLD_YTM_MID
    field_name VARCHAR(150) NOT NULL,
    category VARCHAR(100) NOT NULL, -- Pricing, Symbology, Fixed Income, Fundamentals
    feed_type VARCHAR(50) NOT NULL, -- Data License, B-PIPE, Real-Time Stream, Per Security
    data_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    aliases TEXT[] DEFAULT '{}',
    standards_mapping JSONB DEFAULT '{}'::jsonb, -- FIBO / ISO references
    catalog_node_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_vendor_mnemonic UNIQUE (vendor_name, field_mnemonic)
);

-- 2. Semantic Term to Vendor Licensing Alignment (catalog_edge bridge)
CREATE TABLE IF NOT EXISTS catalog_vendor.term_vendor_alignment (
    alignment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    semantic_term_node_id UUID NOT NULL,
    vendor_field_id UUID NOT NULL REFERENCES catalog_vendor.vendor_data_dictionary(vendor_field_id) ON DELETE CASCADE,
    is_primary_vendor_mapping BOOLEAN DEFAULT TRUE,
    transformation_rule TEXT, -- e.g. "DIVIDE_BY_100" for percentages
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_term_vendor_alignment UNIQUE (tenant_id, semantic_term_node_id, vendor_field_id)
);

-- 3. High-Density Vector Embeddings for Fuzzy Vendor Resolution (pgvector)
CREATE TABLE IF NOT EXISTS catalog_vendor.vendor_field_embeddings (
    embedding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_field_id UUID NOT NULL REFERENCES catalog_vendor.vendor_data_dictionary(vendor_field_id) ON DELETE CASCADE,
    embedding_vector vector(1536) NOT NULL,
    indexed_text_payload TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vendor_field_vector_hnsw 
ON catalog_vendor.vendor_field_embeddings USING hnsw (embedding_vector vector_cosine_ops);

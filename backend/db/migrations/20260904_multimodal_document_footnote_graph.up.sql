-- 20260904_multimodal_document_footnote_graph.up.sql
-- Multimodal Financial Document Footnote & AST Graph Topology

CREATE SCHEMA IF NOT EXISTS catalog_doc;

-- 1. Source Document Manifest
CREATE TABLE IF NOT EXISTS catalog_doc.document_manifest (
    document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    document_key VARCHAR(150) NOT NULL,
    document_type VARCHAR(50) NOT NULL, -- AUDITED_FINANCIAL_STMT, CAPITAL_CALL_NOTICE, LPA_AGREEMENT, SEC_10K
    file_name VARCHAR(255) NOT NULL,
    object_store_uri VARCHAR(500) NOT NULL,
    sha256_checksum VARCHAR(64) NOT NULL,
    effective_date DATE,
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_pages INT NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_doc UNIQUE (tenant_id, document_key)
);

-- 2. Hierarchical Document Grid Cells & Bounding Boxes
CREATE TABLE IF NOT EXISTS catalog_doc.statement_table_cells (
    cell_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES catalog_doc.document_manifest(document_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    page_number INT NOT NULL,
    table_index INT NOT NULL DEFAULT 0,
    row_index INT NOT NULL,
    col_index INT NOT NULL,
    row_header VARCHAR(255),
    col_header VARCHAR(255),
    raw_text TEXT NOT NULL,
    numeric_value NUMERIC(28, 6),
    currency VARCHAR(10) DEFAULT 'USD',
    bbox_coordinates JSONB NOT NULL,
    associated_semantic_term_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Cell-to-Footnote & Discrepancy Graph Edges
CREATE TABLE IF NOT EXISTS catalog_doc.cell_footnote_bindings (
    binding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    cell_id UUID NOT NULL REFERENCES catalog_doc.statement_table_cells(cell_id) ON DELETE CASCADE,
    footnote_number VARCHAR(20) NOT NULL, -- e.g., "Note 4", "Clause 3.2(b)"
    footnote_text TEXT NOT NULL,
    footnote_bbox JSONB,
    accounting_standard VARCHAR(50) DEFAULT 'US_GAAP',
    valuation_hierarchy VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_statement_cell_lookup 
ON catalog_doc.statement_table_cells (document_id, page_number, row_index, col_index);

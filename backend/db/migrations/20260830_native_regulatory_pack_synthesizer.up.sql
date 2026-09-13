-- 20260830_native_regulatory_pack_synthesizer.up.sql
-- Native Regulatory Pack Synthesizer Schema

CREATE SCHEMA IF NOT EXISTS catalog_regulatory;

-- 1. Regulatory Filing Templates
CREATE TABLE IF NOT EXISTS catalog_regulatory.regulatory_filing_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key VARCHAR(100) NOT NULL UNIQUE, -- SEC_13F_HR, SEC_FORM_PF, MIFID_II_RTS28
    regulatory_body VARCHAR(50) NOT NULL, -- SEC, FINRA, ESMA, CFTC
    schema_version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    output_format VARCHAR(20) NOT NULL, -- XML_EDGAR, XML_ISO20022, JSON, CSV
    xsd_validation_schema TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Declarative Tag Mapping Matrix (Semantic Term -> Regulatory Field)
CREATE TABLE IF NOT EXISTS catalog_regulatory.regulatory_field_mappings (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES catalog_regulatory.regulatory_filing_templates(template_id) ON DELETE CASCADE,
    target_tag_path VARCHAR(255) NOT NULL,
    semantic_term_node_id UUID NOT NULL,
    is_mandatory BOOLEAN DEFAULT TRUE,
    formatting_rule VARCHAR(100),
    transformation_expr TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Immutable Filing Run & Merkle Attestation Outbox
CREATE TABLE IF NOT EXISTS catalog_regulatory.regulatory_filing_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    template_id UUID NOT NULL REFERENCES catalog_regulatory.regulatory_filing_templates(template_id),
    reporting_period_end DATE NOT NULL,
    knowledge_cutoff_time TIMESTAMPTZ NOT NULL,
    total_records_processed INT NOT NULL,
    raw_payload_size_bytes BIGINT NOT NULL,
    generated_payload TEXT NOT NULL,
    merkle_filing_passport VARCHAR(64) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    certified_by VARCHAR(100),
    certified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reg_runs_lookup 
    ON catalog_regulatory.regulatory_filing_runs (tenant_id, template_id, reporting_period_end DESC);

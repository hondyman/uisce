-- 20260915_report_books.sql
-- Report Book Definitions, Chapter Sections, and Booklet Artifact Vault

CREATE TABLE IF NOT EXISTS public.report_book_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    book_code VARCHAR(50) NOT NULL,
    book_name TEXT NOT NULL,
    description TEXT,
    cover_template_id UUID,
    include_toc BOOLEAN DEFAULT TRUE,
    toc_title VARCHAR(100) DEFAULT 'Table of Contents',
    page_numbering_style VARCHAR(30) DEFAULT 'CONTINUOUS', -- CONTINUOUS, PER_SECTION, FRONT_MATTER_ROMAN
    page_format VARCHAR(30) DEFAULT 'A4_PORTRAIT',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_book_code UNIQUE (tenant_id, book_code)
);

CREATE TABLE IF NOT EXISTS public.report_book_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    book_id UUID NOT NULL REFERENCES public.report_book_definitions(id) ON DELETE CASCADE,
    report_definition_id UUID NOT NULL,
    section_order INT NOT NULL DEFAULT 1,
    chapter_title VARCHAR(150) NOT NULL,
    insert_divider_tab BOOLEAN DEFAULT FALSE,
    divider_subtitle TEXT,
    condition_expression TEXT,                          -- Dynamic AST rule (omit section if false)
    override_parameters JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_book_section_order UNIQUE (book_id, section_order)
);

CREATE TABLE IF NOT EXISTS public.report_book_burst_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    book_id UUID NOT NULL REFERENCES public.report_book_definitions(id) ON DELETE CASCADE,
    effective_date DATE NOT NULL,
    status VARCHAR(50) DEFAULT 'RUNNING',              -- RUNNING, COMPLETED, FAILED, PARTIAL
    total_booklets INT DEFAULT 0,
    successful_renders INT DEFAULT 0,
    failed_renders INT DEFAULT 0,
    total_pages_generated INT DEFAULT 0,
    total_cost_usd NUMERIC(10, 6) DEFAULT 0.000000,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.report_book_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    batch_id UUID NOT NULL REFERENCES public.report_book_burst_batches(id) ON DELETE CASCADE,
    client_id VARCHAR(100) NOT NULL,
    file_format VARCHAR(20) NOT NULL,                  -- PDF, EXCEL, FLIPBOOK_ZIP
    storage_path TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    sha256_checksum VARCHAR(64) NOT NULL,
    merkle_passport_hash VARCHAR(64) NOT NULL,
    total_pages INT NOT NULL DEFAULT 1,
    render_duration_ms INT NOT NULL,
    status VARCHAR(50) DEFAULT 'READY',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_book_artifacts_lookup 
ON public.report_book_artifacts (tenant_id, client_id, created_at DESC);

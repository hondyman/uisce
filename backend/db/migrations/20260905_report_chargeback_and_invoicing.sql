-- 20260905_report_chargeback_and_invoicing.sql
-- Tiered Charge Rates, Report Rendering Ledger, and Monthly Invoicing Schema

CREATE SCHEMA IF NOT EXISTS finops;
CREATE SCHEMA IF NOT EXISTS audit;

-- 1. Tiered Unit Charge Rates (Rule 1: Config-Before-Code & Gold Copy Inheritance)
CREATE TABLE IF NOT EXISTS finops.bo_charge_rates (
    rate_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    backend_id UUID,
    backend_type VARCHAR(50) NOT NULL DEFAULT 'DEFAULT',    -- STARROCKS, ICEBERG, POSTGRES, VECTOR
    complexity_rate_per_unit NUMERIC(10, 6) NOT NULL DEFAULT 0.000500, -- Rate per AST complexity point
    volume_rate_per_gb NUMERIC(10, 6) NOT NULL DEFAULT 0.020000,       -- Rate per GB scanned
    cpu_second_rate NUMERIC(10, 6) NOT NULL DEFAULT 0.005000,          -- Rate per CPU second
    pdf_base_artifact_rate NUMERIC(10, 6) NOT NULL DEFAULT 0.010000,   -- Base cost per PDF document
    pdf_page_rate NUMERIC(10, 6) NOT NULL DEFAULT 0.002000,            -- Cost per compiled page
    excel_base_artifact_rate NUMERIC(10, 6) NOT NULL DEFAULT 0.005000,
    storage_rate_per_gb_month NUMERIC(10, 6) NOT NULL DEFAULT 0.023000,
    backend_weight_multiplier NUMERIC(4, 2) NOT NULL DEFAULT 1.00,     -- e.g., 5.0x for Iceberg Lakehouse
    active_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_backend_rate UNIQUE (tenant_id, backend_type, active_from)
);

CREATE INDEX IF NOT EXISTS idx_charge_rates_lookup 
ON finops.bo_charge_rates (tenant_id, backend_type, is_active);

-- 2. Itemized Report Rendering Chargeback Ledger
CREATE TABLE IF NOT EXISTS finops.report_render_chargeback_ledger (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    batch_id UUID,
    schedule_id UUID,
    report_definition_id UUID NOT NULL,
    client_slice_id VARCHAR(100) NOT NULL,                  -- Account or Client ID
    export_format VARCHAR(20) NOT NULL,                     -- PDF, EXCEL
    page_count INT NOT NULL DEFAULT 1,
    render_duration_ms INT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    scanned_bytes BIGINT NOT NULL DEFAULT 0,
    ast_complexity_score NUMERIC(6, 2) NOT NULL DEFAULT 0.00,
    
    -- Cost Attribution Breakdown (USD)
    query_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    vector_math_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    render_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    storage_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    total_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    
    billing_period VARCHAR(7) NOT NULL,                     -- YYYY-MM
    is_invoiced BOOLEAN DEFAULT FALSE,
    invoice_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_render_chargeback_tenant_period 
ON finops.report_render_chargeback_ledger (tenant_id, billing_period, is_invoiced);

-- 3. Monthly Consolidated Client Invoices
CREATE TABLE IF NOT EXISTS finops.tenant_monthly_invoices (
    invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    billing_period VARCHAR(7) NOT NULL,                     -- YYYY-MM
    total_query_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    total_render_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    total_storage_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    total_invoice_usd NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    total_reports_rendered INT NOT NULL DEFAULT 0,
    total_pages_generated INT NOT NULL DEFAULT 0,
    total_gigabytes_scanned NUMERIC(10, 3) NOT NULL DEFAULT 0.000,
    payment_status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',    -- DRAFT, ISSUED, PAID, OVERDUE
    stripe_invoice_id VARCHAR(100),
    issued_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_invoice_period UNIQUE (tenant_id, billing_period)
);

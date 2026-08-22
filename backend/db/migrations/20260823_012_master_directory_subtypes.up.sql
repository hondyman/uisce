-- 20260823_012_master_directory_subtypes.up.sql
-- STI Tables for Master Catalogs and Directories

CREATE TABLE IF NOT EXISTS master.customer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_name TEXT NOT NULL,
    subtype_code TEXT NOT NULL,
    lei_code VARCHAR(20),
    kyc_status TEXT NOT NULL DEFAULT 'PENDING',
    suitability_profile TEXT,
    relationship_tier TEXT,
    parent_group_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    CONSTRAINT ck_customer_subtype CHECK (
        subtype_code IN ('institutional_client', 'private_wealth', 'broker_dealer', 'corporate_treasury')
    )
);

CREATE INDEX IF NOT EXISTS idx_customer_inst ON master.customer (tenant_id) WHERE subtype_code = 'institutional_client';
CREATE INDEX IF NOT EXISTS idx_customer_wealth ON master.customer (tenant_id) WHERE subtype_code = 'private_wealth';
CREATE INDEX IF NOT EXISTS idx_customer_bd ON master.customer (tenant_id) WHERE subtype_code = 'broker_dealer';
CREATE INDEX IF NOT EXISTS idx_customer_corp ON master.customer (tenant_id) WHERE subtype_code = 'corporate_treasury';

CREATE TABLE IF NOT EXISTS master.vendor (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    vendor_name TEXT NOT NULL,
    subtype_code TEXT NOT NULL,
    vendor_category TEXT,
    sla_tier TEXT,
    soc2_certification_date DATE,
    soc1_type2_on_file BOOLEAN,
    billing_cycle TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    CONSTRAINT ck_vendor_subtype CHECK (
        subtype_code IN ('custodian_prime_broker', 'market_data', 'fund_admin', 'cloud_tech')
    )
);

CREATE INDEX IF NOT EXISTS idx_vendor_custodian ON master.vendor (tenant_id) WHERE subtype_code = 'custodian_prime_broker';
CREATE INDEX IF NOT EXISTS idx_vendor_marketdata ON master.vendor (tenant_id) WHERE subtype_code = 'market_data';
CREATE INDEX IF NOT EXISTS idx_vendor_fundmin ON master.vendor (tenant_id) WHERE subtype_code = 'fund_admin';
CREATE INDEX IF NOT EXISTS idx_vendor_cloud ON master.vendor (tenant_id) WHERE subtype_code = 'cloud_tech';

CREATE TABLE IF NOT EXISTS master.personnel (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL,
    subtype_code TEXT NOT NULL,
    crd_number TEXT,
    series_licenses_held TEXT[],
    supervisory_id UUID,
    discretionary_authority_limit NUMERIC(18,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    CONSTRAINT ck_personnel_subtype CHECK (
        subtype_code IN ('portfolio_manager', 'trade_execution', 'compliance_officer', 'client_advisor')
    )
);

CREATE INDEX IF NOT EXISTS idx_personnel_pm ON master.personnel (tenant_id) WHERE subtype_code = 'portfolio_manager';
CREATE INDEX IF NOT EXISTS idx_personnel_compliance ON master.personnel (tenant_id) WHERE subtype_code = 'compliance_officer';
CREATE INDEX IF NOT EXISTS idx_personnel_adv ON master.personnel (tenant_id) WHERE subtype_code = 'client_advisor';

CREATE TABLE IF NOT EXISTS master.sales_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_number TEXT NOT NULL,
    client_id UUID NOT NULL,
    subtype_code TEXT NOT NULL,
    billing_period_end DATE NOT NULL,
    aum_basis_amount NUMERIC(18,2),
    effective_fee_bps NUMERIC(8,4),
    hwm_benchmark_nav NUMERIC(18,2),
    invoice_status TEXT NOT NULL DEFAULT 'ISSUED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    CONSTRAINT ck_sales_ledger_subtype CHECK (
        subtype_code IN ('aum_management_fee', 'trading_commission', 'performance_fee', 'platform_subscription')
    )
);

CREATE INDEX IF NOT EXISTS idx_sales_aum ON master.sales_ledger (tenant_id) WHERE subtype_code = 'aum_management_fee';
CREATE INDEX IF NOT EXISTS idx_sales_commission ON master.sales_ledger (tenant_id) WHERE subtype_code = 'trading_commission';
CREATE INDEX IF NOT EXISTS idx_sales_perf ON master.sales_ledger (tenant_id) WHERE subtype_code = 'performance_fee';
CREATE INDEX IF NOT EXISTS idx_sales_platform ON master.sales_ledger (tenant_id) WHERE subtype_code = 'platform_subscription';

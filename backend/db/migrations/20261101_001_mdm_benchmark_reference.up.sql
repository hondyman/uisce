-- 20261101_001_mdm_benchmark_reference.up.sql
-- Reference/lookup tables for the benchmark MDM layer.

-- ── benchmark_provider ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_provider (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    provider_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    short_name varchar(50),
    provider_type varchar(30) NOT NULL,
    domicile varchar(2),
    parent_provider_id uuid,
    is_administrator bool DEFAULT false NOT NULL,
    administrator_status varchar(20),
    is_iosco_compliant bool DEFAULT false NOT NULL,
    is_esma_registered bool DEFAULT false NOT NULL,
    is_uk_fca_registered bool DEFAULT false NOT NULL,
    is_sec_registered bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    website varchar(250),
    lei varchar(20),
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_provider_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_provider_cd_key UNIQUE (tenant_id, provider_cd),
    CONSTRAINT chk_bp_type CHECK (provider_type IN (
        'INDEX_PROVIDER','DATA_VENDOR','PEER_GROUP_PROVIDER',
        'REGULATORY','INTERNAL','CALCULATION_AGENT')),
    CONSTRAINT fk_bp_parent FOREIGN KEY (parent_provider_id) REFERENCES mdm.benchmark_provider(id)
);
CREATE INDEX IF NOT EXISTS idx_bp_type   ON mdm.benchmark_provider (provider_type);
CREATE INDEX IF NOT EXISTS idx_bp_active ON mdm.benchmark_provider (is_active);
CREATE INDEX IF NOT EXISTS idx_bp_tenant ON mdm.benchmark_provider (tenant_id);

-- ── benchmark_type ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    is_investable bool DEFAULT false NOT NULL,
    is_publicly_available bool DEFAULT false NOT NULL,
    requires_methodology bool DEFAULT true NOT NULL,
    requires_regulatory_registration bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_type_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_type_cd_key UNIQUE (tenant_id, type_cd),
    CONSTRAINT chk_bt_category CHECK (category IN (
        'MARKET_INDEX','COMPOSITE','PEER_GROUP','POLICY','CUSTOM',
        'REGULATORY','STRATEGY','HYBRID','CASH','ZERO'))
);
CREATE INDEX IF NOT EXISTS idx_bt_tenant ON mdm.benchmark_type (tenant_id);

-- ── benchmark_return_variant ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_return_variant (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    variant_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    variant_type varchar(30) NOT NULL,
    dividend_treatment varchar(30),
    withholding_tax_applied bool DEFAULT false NOT NULL,
    withholding_tax_basis varchar(30),
    description text,
    is_default bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_return_variant_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_return_variant_cd_key UNIQUE (tenant_id, variant_cd),
    CONSTRAINT chk_brv_type CHECK (variant_type IN (
        'PRICE_RETURN','GROSS_TOTAL_RETURN','NET_TOTAL_RETURN',
        'EXCESS_RETURN','TOTAL_RETURN'))
);
CREATE INDEX IF NOT EXISTS idx_brv_tenant ON mdm.benchmark_return_variant (tenant_id);

-- ── benchmark_currency_variant ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_currency_variant (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    variant_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    base_currency varchar(3) NOT NULL,
    is_hedged bool DEFAULT false NOT NULL,
    hedge_currency varchar(3),
    hedging_frequency varchar(20),
    hedging_method varchar(30),
    hedge_ratio_pct numeric(7,4),
    description text,
    is_default bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_currency_variant_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_currency_variant_cd_key UNIQUE (tenant_id, variant_cd)
);
CREATE INDEX IF NOT EXISTS idx_bcv_tenant ON mdm.benchmark_currency_variant (tenant_id);

-- ── benchmark_weighting_method ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_weighting_method (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    method_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    requires_rebalance bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_weighting_method_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_weighting_method_cd_key UNIQUE (tenant_id, method_cd)
);
CREATE INDEX IF NOT EXISTS idx_bwm_tenant ON mdm.benchmark_weighting_method (tenant_id);

-- ── benchmark_rebalance_frequency ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_rebalance_frequency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    frequency_cd varchar(30) NOT NULL,
    name varchar(100) NOT NULL,
    periods_per_year numeric(6,2),
    typical_rebalance_month varchar(15),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_rebalance_frequency_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_rebalance_frequency_cd_key UNIQUE (tenant_id, frequency_cd)
);
CREATE INDEX IF NOT EXISTS idx_brf_tenant ON mdm.benchmark_rebalance_frequency (tenant_id);

-- ── benchmark_regulatory_regime ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_regulatory_regime (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    regime_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    jurisdiction varchar(10) NOT NULL,
    regulator_cd varchar(50),
    regulation_name varchar(250),
    regulation_reference varchar(100),
    effective_from date NOT NULL,
    effective_to date,
    requires_registration bool DEFAULT true NOT NULL,
    requires_authorisation bool DEFAULT false NOT NULL,
    requires_esg_disclosure bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_regulatory_regime_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_regulatory_regime_cd_key UNIQUE (tenant_id, regime_cd)
);
CREATE INDEX IF NOT EXISTS idx_brr_tenant ON mdm.benchmark_regulatory_regime (tenant_id);

-- ── benchmark_use_case ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_use_case (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    use_case_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_regulated bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_use_case_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_use_case_cd_key UNIQUE (tenant_id, use_case_cd)
);
CREATE INDEX IF NOT EXISTS idx_buc_tenant ON mdm.benchmark_use_case (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'benchmark_provider','benchmark_type','benchmark_return_variant',
        'benchmark_currency_variant','benchmark_weighting_method',
        'benchmark_rebalance_frequency','benchmark_regulatory_regime',
        'benchmark_use_case'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$,
            t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$,
            t || '_tenant_write', t);
    END LOOP;
END
$rls$;

-- 20261103_001_mdm_price_reference.up.sql

-- ── price_source ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_source (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    short_name varchar(50),
    source_type varchar(30) NOT NULL,
    vendor_type varchar(30),
    is_consolidated bool DEFAULT false NOT NULL,
    is_primary_source bool DEFAULT false NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    delivery_method varchar(30),
    cutoff_time time,
    timezone varchar(50),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    parent_source_id uuid,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_source_pkey PRIMARY KEY (id),
    CONSTRAINT price_source_cd_key UNIQUE (tenant_id, source_cd),
    CONSTRAINT chk_ps_type CHECK (source_type IN (
        'EXCHANGE','DEALER','BROKER','EVALUATOR','MODEL','INTERNAL',
        'ADMINISTRATOR','PRICING_SERVICE','CONSENSUS','TRADE_CAPTURE')),
    CONSTRAINT chk_ps_status CHECK (status IN (
        'ACTIVE','DORMANT','DISCONTINUED','SUSPENDED')),
    CONSTRAINT fk_ps_parent FOREIGN KEY (parent_source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_psrc_type   ON mdm.price_source (source_type);
CREATE INDEX IF NOT EXISTS idx_psrc_active ON mdm.price_source (is_active);
CREATE INDEX IF NOT EXISTS idx_psrc_tenant ON mdm.price_source (tenant_id);

-- ── price_source_alias ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_source_alias (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_id uuid NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    alias_cd varchar(50) NOT NULL,
    alias_name varchar(250),
    is_primary bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_source_alias_pkey PRIMARY KEY (id),
    CONSTRAINT fk_psa_source FOREIGN KEY (source_id)            REFERENCES mdm.price_source(id),
    CONSTRAINT fk_psa_mdm    FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_psa_source ON mdm.price_source_alias (source_id);
CREATE INDEX IF NOT EXISTS idx_psa_tenant ON mdm.price_source_alias (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_psa_key
    ON mdm.price_source_alias (tenant_id, mdm_source_system_id, alias_cd);

-- ── price_type ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    price_side varchar(10),
    is_executable bool DEFAULT false NOT NULL,
    is_official bool DEFAULT false NOT NULL,
    requires_model bool DEFAULT false NOT NULL,
    requires_source bool DEFAULT true NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_type_pkey PRIMARY KEY (id),
    CONSTRAINT price_type_cd_key UNIQUE (tenant_id, price_type_cd),
    CONSTRAINT chk_pt_cat CHECK (category IN (
        'QUOTE','TRADE','EVALUATED','MODEL','DERIVED','INDICATIVE','OFFICIAL')),
    CONSTRAINT chk_pt_side CHECK (price_side IS NULL OR price_side IN (
        'BID','ASK','MID','LAST','NONE'))
);
CREATE INDEX IF NOT EXISTS idx_ptype_tenant ON mdm.price_type (tenant_id);

-- ── price_quality_tier ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_quality_tier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tier_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    tier_level int4 NOT NULL,
    description text,
    typical_sources jsonb,
    is_executable bool DEFAULT false NOT NULL,
    is_observable bool DEFAULT true NOT NULL,
    is_level_1 bool DEFAULT false NOT NULL,
    is_level_2 bool DEFAULT false NOT NULL,
    is_level_3 bool DEFAULT false NOT NULL,
    max_staleness_minutes int4,
    requires_review bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_quality_tier_pkey PRIMARY KEY (id),
    CONSTRAINT price_quality_tier_cd_key UNIQUE (tenant_id, tier_cd)
);
CREATE INDEX IF NOT EXISTS idx_pqt_tenant ON mdm.price_quality_tier (tenant_id);

-- ── fair_value_level ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fair_value_level (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    level_cd varchar(20) NOT NULL,
    name varchar(100) NOT NULL,
    level_number int4 NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT fair_value_level_pkey PRIMARY KEY (id),
    CONSTRAINT fair_value_level_cd_key UNIQUE (tenant_id, level_cd)
);
CREATE INDEX IF NOT EXISTS idx_fvl_tenant ON mdm.fair_value_level (tenant_id);

-- ── price_observation_type ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_observation_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    observation_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_close bool DEFAULT false NOT NULL,
    is_intraday bool DEFAULT false NOT NULL,
    is_snapshot bool DEFAULT false NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_observation_type_pkey PRIMARY KEY (id),
    CONSTRAINT price_observation_type_cd_key UNIQUE (tenant_id, observation_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_pot_tenant ON mdm.price_observation_type (tenant_id);

-- ── curve_category ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_category (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    curve_class varchar(30) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_category_pkey PRIMARY KEY (id),
    CONSTRAINT curve_category_cd_key UNIQUE (tenant_id, category_cd),
    CONSTRAINT chk_cc_class CHECK (curve_class IN (
        'RATE','CREDIT','INFLATION','FX','VOLATILITY','COMMODITY',
        'EQUITY','MUNICIPAL','FUNDING','COLLATERAL'))
);
CREATE INDEX IF NOT EXISTS idx_cc_tenant ON mdm.curve_category (tenant_id);

-- ── curve_interpolation_method ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_interpolation_method (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    method_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_interpolation_method_pkey PRIMARY KEY (id),
    CONSTRAINT curve_interpolation_method_cd_key UNIQUE (tenant_id, method_cd)
);
CREATE INDEX IF NOT EXISTS idx_cim_tenant ON mdm.curve_interpolation_method (tenant_id);

-- ── curve_compounding_frequency ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_compounding_frequency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    compounding_cd varchar(20) NOT NULL,
    name varchar(100) NOT NULL,
    periods_per_year numeric(6,2),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_compounding_frequency_pkey PRIMARY KEY (id),
    CONSTRAINT curve_compounding_frequency_cd_key UNIQUE (tenant_id, compounding_cd)
);
CREATE INDEX IF NOT EXISTS idx_ccf_tenant ON mdm.curve_compounding_frequency (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'price_source','price_source_alias','price_type','price_quality_tier',
        'fair_value_level','price_observation_type','curve_category',
        'curve_interpolation_method','curve_compounding_frequency'
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

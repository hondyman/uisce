-- 20261104_001_mdm_calendar_reference.up.sql
-- Reference/lookup tables for calendar MDM.

-- ── calendar_type ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    description text,
    applies_to_entity_type varchar(30),
    required_for_settlement bool DEFAULT false NOT NULL,
    required_for_valuation bool DEFAULT false NOT NULL,
    required_for_dealing bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cal_type_cd UNIQUE (tenant_id, calendar_type_cd),
    CONSTRAINT chk_cal_type_category CHECK (category IN (
        'MARKET','SETTLEMENT','PAYMENT','FUND','BENCHMARK','REGULATORY',
        'CENTRAL_BANK','TAX','FISCAL','CORPORATE','OPERATIONAL'))
);
CREATE INDEX IF NOT EXISTS idx_cal_type_cat    ON mdm.calendar_type (category);
CREATE INDEX IF NOT EXISTS idx_cal_type_tenant ON mdm.calendar_type (tenant_id);

-- ── time_zone ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.time_zone (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tz_name varchar(100) NOT NULL,
    display_name varchar(150),
    utc_offset_standard_minutes int4,
    utc_offset_dst_minutes int4,
    observes_dst bool DEFAULT false NOT NULL,
    tz_database_version varchar(30),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT time_zone_pkey PRIMARY KEY (id),
    CONSTRAINT uq_tz_name UNIQUE (tenant_id, tz_name)
);
CREATE INDEX IF NOT EXISTS idx_tz_name   ON mdm.time_zone (tz_name);
CREATE INDEX IF NOT EXISTS idx_tz_tenant ON mdm.time_zone (tenant_id);

-- ── calendar_source ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_source (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    short_name varchar(50),
    source_type varchar(30) NOT NULL,
    vendor_type varchar(30),
    authority_level varchar(20),
    coverage_scope varchar(30),
    delivery_method varchar(30),
    update_frequency varchar(20),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    parent_source_id uuid,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_source_pkey PRIMARY KEY (id),
    CONSTRAINT uq_calsrc_cd UNIQUE (tenant_id, source_cd),
    CONSTRAINT chk_calsrc_type CHECK (source_type IN (
        'EXCHANGE','REGULATOR','CENTRAL_BANK','CSD','CUSTODIAN','ADMINISTRATOR',
        'INDEX_PROVIDER','DATA_VENDOR','INTERNAL','ISO','IANA','COMMERCIAL')),
    CONSTRAINT fk_calsrc_parent FOREIGN KEY (parent_source_id) REFERENCES mdm.calendar_source(id)
);
CREATE INDEX IF NOT EXISTS idx_calsrc_type   ON mdm.calendar_source (source_type);
CREATE INDEX IF NOT EXISTS idx_calsrc_active ON mdm.calendar_source (is_active);
CREATE INDEX IF NOT EXISTS idx_calsrc_tenant ON mdm.calendar_source (tenant_id);

-- ── holiday_type ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.holiday_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    holiday_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    is_market_closure bool DEFAULT true NOT NULL,
    is_bank_closure bool DEFAULT true NOT NULL,
    is_settlement_closure bool DEFAULT true NOT NULL,
    is_observed_when_weekend bool DEFAULT true NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT holiday_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ht_cd UNIQUE (tenant_id, holiday_type_cd),
    CONSTRAINT chk_ht_cat CHECK (category IN (
        'RELIGIOUS','NATIONAL','REGIONAL','CULTURAL','COMMERCIAL',
        'REGULATORY','EMERGENCY','DISCRETIONARY','SUBSTITUTE'))
);
CREATE INDEX IF NOT EXISTS idx_ht_tenant ON mdm.holiday_type (tenant_id);

-- ── holiday_rule_type ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.holiday_rule_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    requires_params bool DEFAULT true NOT NULL,
    requires_manual_dates bool DEFAULT false NOT NULL,
    is_deterministic bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT holiday_rule_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_hrt_cd UNIQUE (tenant_id, rule_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_hrt_tenant ON mdm.holiday_rule_type (tenant_id);

-- ── rolling_convention ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rolling_convention (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    convention_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    direction varchar(20),
    is_modified bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rolling_convention_pkey PRIMARY KEY (id),
    CONSTRAINT uq_rc_cd UNIQUE (tenant_id, convention_cd),
    CONSTRAINT chk_rc_dir CHECK (direction IS NULL OR direction IN ('FORWARD','BACKWARD','NONE'))
);
CREATE INDEX IF NOT EXISTS idx_rc_tenant ON mdm.rolling_convention (tenant_id);

-- ── business_day_definition ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.business_day_definition (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    definition_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_settlement_day bool DEFAULT false NOT NULL,
    is_trading_day bool DEFAULT false NOT NULL,
    is_payment_day bool DEFAULT false NOT NULL,
    is_dealing_day bool DEFAULT false NOT NULL,
    is_valuation_day bool DEFAULT false NOT NULL,
    is_notification_day bool DEFAULT false NOT NULL,
    is_reporting_day bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT business_day_definition_pkey PRIMARY KEY (id),
    CONSTRAINT uq_bdd_cd UNIQUE (tenant_id, definition_cd)
);
CREATE INDEX IF NOT EXISTS idx_bdd_tenant ON mdm.business_day_definition (tenant_id);

-- ── calendar_hierarchy_type ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_hierarchy_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    hierarchy_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_inherited bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_hierarchy_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cht_cd UNIQUE (tenant_id, hierarchy_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_cht_tenant ON mdm.calendar_hierarchy_type (tenant_id);

-- ── calendar_regulatory_regime ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_regulatory_regime (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    regime_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    jurisdiction varchar(10) NOT NULL,
    regulator_cd varchar(50),
    regulation_name varchar(250),
    requires_specific_calendar bool DEFAULT false NOT NULL,
    required_calendar_type_cd varchar(30),
    effective_from date NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_regulatory_regime_pkey PRIMARY KEY (id),
    CONSTRAINT uq_crr_cd UNIQUE (tenant_id, regime_cd)
);
CREATE INDEX IF NOT EXISTS idx_crr_tenant ON mdm.calendar_regulatory_regime (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_type','time_zone','calendar_source','holiday_type',
        'holiday_rule_type','rolling_convention','business_day_definition',
        'calendar_hierarchy_type','calendar_regulatory_regime'
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

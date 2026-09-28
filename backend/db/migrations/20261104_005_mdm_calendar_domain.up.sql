-- 20261104_005_mdm_calendar_domain.up.sql

-- ── settlement_calendar ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.settlement_calendar (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    settlement_calendar_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    calendar_id uuid NOT NULL,
    secondary_calendar_ids uuid[],
    currency_cd varchar(3),
    instrument_type varchar(30),
    market_segment varchar(30),
    regulatory_regime_cd varchar(50),
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT settlement_calendar_pkey PRIMARY KEY (id),
    CONSTRAINT uq_settcal_cd UNIQUE (tenant_id, settlement_calendar_cd),
    CONSTRAINT fk_settcal_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_settcal_calendar ON mdm.settlement_calendar (calendar_id);
CREATE INDEX IF NOT EXISTS idx_settcal_currency ON mdm.settlement_calendar (currency_cd);
CREATE INDEX IF NOT EXISTS idx_settcal_tenant   ON mdm.settlement_calendar (tenant_id);

-- ── settlement_rule ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.settlement_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    settlement_calendar_id uuid NOT NULL,
    instrument_type varchar(30) NOT NULL,
    market_segment varchar(30),
    trade_type varchar(30),
    settlement_days int4 NOT NULL,
    settlement_basis varchar(20),
    rolling_convention_id uuid,
    applies_to_leg int4,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    regulatory_reference varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT settlement_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_sr_cd UNIQUE (tenant_id, rule_cd),
    CONSTRAINT chk_sr_basis CHECK (settlement_basis IS NULL OR settlement_basis IN (
        'BUSINESS_DAYS','CALENDAR_DAYS','WEEKLY','MONTHLY')),
    CONSTRAINT fk_sr_calendar FOREIGN KEY (settlement_calendar_id) REFERENCES mdm.settlement_calendar(id),
    CONSTRAINT fk_sr_rolling  FOREIGN KEY (rolling_convention_id)  REFERENCES mdm.rolling_convention(id)
);
CREATE INDEX IF NOT EXISTS idx_sr_calendar ON mdm.settlement_rule (settlement_calendar_id);
CREATE INDEX IF NOT EXISTS idx_sr_type     ON mdm.settlement_rule (instrument_type, market_segment);
CREATE INDEX IF NOT EXISTS idx_sr_tenant   ON mdm.settlement_rule (tenant_id);

-- ── settlement_rule_calendar ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.settlement_rule_calendar (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    settlement_rule_id uuid NOT NULL,
    calendar_id uuid NOT NULL,
    calendar_role varchar(30) NOT NULL,
    composition_rule varchar(20) NOT NULL,
    sequence int4 DEFAULT 1 NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT settlement_rule_calendar_pkey PRIMARY KEY (id),
    CONSTRAINT chk_src_role CHECK (calendar_role IN (
        'PRIMARY','HOMEMARKET','CURRENCY','COUNTERPARTY','COLLATERAL',
        'CROSS_BORDER','CLS','RTGS')),
    CONSTRAINT chk_src_comp CHECK (composition_rule IN (
        'ALL_OPEN','ANY_OPEN','PRIMARY','UNION','INTERSECTION')),
    CONSTRAINT fk_src_rule     FOREIGN KEY (settlement_rule_id) REFERENCES mdm.settlement_rule(id) ON DELETE CASCADE,
    CONSTRAINT fk_src_calendar FOREIGN KEY (calendar_id)        REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_src_rule   ON mdm.settlement_rule_calendar (settlement_rule_id);
CREATE INDEX IF NOT EXISTS idx_src_cal    ON mdm.settlement_rule_calendar (calendar_id);
CREATE INDEX IF NOT EXISTS idx_src_tenant ON mdm.settlement_rule_calendar (tenant_id);

-- ── fund_calendar ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fund_calendar (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    fund_calendar_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    fund_id uuid,
    fund_share_class_id uuid,
    umbrella_fund_id uuid,
    dealing_calendar_id uuid NOT NULL,
    valuation_calendar_id uuid,
    nav_publication_calendar_id uuid,
    settlement_calendar_id uuid,
    payment_calendar_id uuid,
    domicile_country_cd varchar(2),
    dealing_timezone_id uuid,
    valuation_timezone_id uuid,
    dealing_cutoff_time time,
    valuation_point time,
    nav_publication_time time,
    dealing_frequency varchar(20),
    valuation_frequency varchar(20),
    settlement_cycle varchar(10),
    is_full_calendar_override bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fund_calendar_pkey PRIMARY KEY (id),
    CONSTRAINT uq_fundcal_cd UNIQUE (tenant_id, fund_calendar_cd),
    CONSTRAINT fk_fundcal_dealing    FOREIGN KEY (dealing_calendar_id)         REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_fundcal_valuation  FOREIGN KEY (valuation_calendar_id)       REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_fundcal_navpub     FOREIGN KEY (nav_publication_calendar_id) REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_fundcal_settlement FOREIGN KEY (settlement_calendar_id)      REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_fundcal_payment    FOREIGN KEY (payment_calendar_id)         REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_fundcal_tz_dealing FOREIGN KEY (dealing_timezone_id)         REFERENCES mdm.time_zone(id),
    CONSTRAINT fk_fundcal_tz_val     FOREIGN KEY (valuation_timezone_id)       REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_fundcal_fund   ON mdm.fund_calendar (fund_id);
CREATE INDEX IF NOT EXISTS idx_fundcal_tenant ON mdm.fund_calendar (tenant_id);

-- ── fund_calendar_rule ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fund_calendar_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    fund_calendar_id uuid NOT NULL,
    rule_type varchar(30) NOT NULL,
    composition_rule varchar(20) NOT NULL,
    participating_calendar_ids uuid[] NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fund_calendar_rule_pkey PRIMARY KEY (id),
    CONSTRAINT chk_fcr_type CHECK (rule_type IN (
        'DEALING_DAY','VALUATION_DAY','NAV_PUBLICATION','CUTOFF','SETTLEMENT')),
    CONSTRAINT chk_fcr_comp CHECK (composition_rule IN (
        'UNION','INTERSECTION','PRIMARY','DOMICILE_AND_MARKET',
        'MARKET_ONLY','DOMICILE_ONLY')),
    CONSTRAINT fk_fcr_fund_calendar FOREIGN KEY (fund_calendar_id) REFERENCES mdm.fund_calendar(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fcr_fund_calendar ON mdm.fund_calendar_rule (fund_calendar_id);
CREATE INDEX IF NOT EXISTS idx_fcr_tenant        ON mdm.fund_calendar_rule (tenant_id);

-- ── fund_calendar_exception ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fund_calendar_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    fund_calendar_id uuid NOT NULL,
    exception_date date NOT NULL,
    exception_type varchar(30) NOT NULL,
    reason varchar(500),
    dealing_cutoff_time time,
    valuation_point time,
    nav_publication_time time,
    is_dealing_day bool,
    is_valuation_day bool,
    is_settlement_day bool,
    announcement_date date,
    notice_days int4,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fund_calendar_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_fce_type CHECK (exception_type IN (
        'CLOSED','HALF_DAY','EARLY_CUTOFF','LATE_CUTOFF',
        'SPECIAL_VALUATION','SPECIAL_DEALING','SUSPENSION')),
    CONSTRAINT fk_fce_fund_calendar FOREIGN KEY (fund_calendar_id) REFERENCES mdm.fund_calendar(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fce_fund_calendar ON mdm.fund_calendar_exception (fund_calendar_id, exception_date);
CREATE INDEX IF NOT EXISTS idx_fce_tenant        ON mdm.fund_calendar_exception (tenant_id);

-- ── central_bank_calendar ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.central_bank_calendar (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    central_bank_calendar_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    central_bank_entity_id uuid,
    central_bank_cd varchar(20),
    country_cd varchar(2),
    calendar_id uuid NOT NULL,
    rtgs_system_cd varchar(30),
    currency_cd varchar(3),
    operating_hours_open time,
    operating_hours_close time,
    time_zone_id uuid,
    settlement_cutoff_time time,
    cross_border_cutoff_time time,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT central_bank_calendar_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cbcal_cd UNIQUE (tenant_id, central_bank_calendar_cd),
    CONSTRAINT fk_cbcal_calendar FOREIGN KEY (calendar_id)  REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_cbcal_tz       FOREIGN KEY (time_zone_id) REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_cbcal_cb       ON mdm.central_bank_calendar (central_bank_cd);
CREATE INDEX IF NOT EXISTS idx_cbcal_rtgs     ON mdm.central_bank_calendar (rtgs_system_cd);
CREATE INDEX IF NOT EXISTS idx_cbcal_currency ON mdm.central_bank_calendar (currency_cd);
CREATE INDEX IF NOT EXISTS idx_cbcal_tenant   ON mdm.central_bank_calendar (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'settlement_calendar','settlement_rule','settlement_rule_calendar',
        'fund_calendar','fund_calendar_rule','fund_calendar_exception',
        'central_bank_calendar'
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

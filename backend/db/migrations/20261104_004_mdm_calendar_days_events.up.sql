-- 20261104_004_mdm_calendar_days_events.up.sql

-- ── calendar_day (materialized) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_day (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
    calendar_date date NOT NULL,
    day_of_week int4 NOT NULL,
    is_weekend bool DEFAULT false NOT NULL,
    is_business_day bool DEFAULT true NOT NULL,
    is_trading_day bool DEFAULT true NOT NULL,
    is_settlement_day bool DEFAULT true NOT NULL,
    is_payment_day bool DEFAULT true NOT NULL,
    is_dealing_day bool DEFAULT true NOT NULL,
    is_valuation_day bool DEFAULT true NOT NULL,
    is_half_day bool DEFAULT false NOT NULL,
    session_open_time time,
    session_close_time time,
    settlement_cutoff_time time,
    is_month_end bool DEFAULT false NOT NULL,
    is_quarter_end bool DEFAULT false NOT NULL,
    is_year_end bool DEFAULT false NOT NULL,
    is_last_business_day_of_month bool DEFAULT false NOT NULL,
    is_first_business_day_of_month bool DEFAULT false NOT NULL,
    business_day_of_month int4,
    business_day_of_quarter int4,
    business_day_of_year int4,
    primary_holiday_id uuid,
    additional_holidays jsonb,
    is_materialized bool DEFAULT true NOT NULL,
    materialization_source varchar(30),
    is_official bool DEFAULT false NOT NULL,
    materialized_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    source_system_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_day_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cday UNIQUE (tenant_id, calendar_id, calendar_date),
    CONSTRAINT fk_cday_calendar FOREIGN KEY (calendar_id)        REFERENCES mdm.calendar_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_cday_holiday  FOREIGN KEY (primary_holiday_id) REFERENCES mdm.holiday_definition(id)
);
CREATE INDEX IF NOT EXISTS idx_cday_calendar   ON mdm.calendar_day (calendar_id, calendar_date);
CREATE INDEX IF NOT EXISTS idx_cday_date       ON mdm.calendar_day (calendar_date);
CREATE INDEX IF NOT EXISTS idx_cday_business   ON mdm.calendar_day (calendar_id, is_business_day) WHERE is_business_day = true;
CREATE INDEX IF NOT EXISTS idx_cday_settlement ON mdm.calendar_day (calendar_id, is_settlement_day) WHERE is_settlement_day = true;
CREATE INDEX IF NOT EXISTS idx_cday_tenant     ON mdm.calendar_day (tenant_id);

-- ── calendar_session ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_day_id uuid NOT NULL,
    session_type varchar(30) NOT NULL,
    session_name varchar(150),
    open_time time,
    close_time time,
    time_zone_id uuid,
    is_official bool DEFAULT true NOT NULL,
    is_early_close bool DEFAULT false NOT NULL,
    early_close_reason varchar(100),
    is_late_open bool DEFAULT false NOT NULL,
    late_open_reason varchar(100),
    display_order int4 DEFAULT 0,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_session_pkey PRIMARY KEY (id),
    CONSTRAINT chk_csess_type CHECK (session_type IN (
        'PRE_MARKET','REGULAR','POST_MARKET','AUCTION','CLOSING_AUCTION',
        'OPENING_AUCTION','BLOCK_TRADING','ODD_LOT','CROSS','FX_FIXING')),
    CONSTRAINT fk_csess_day FOREIGN KEY (calendar_day_id) REFERENCES mdm.calendar_day(id) ON DELETE CASCADE,
    CONSTRAINT fk_csess_tz  FOREIGN KEY (time_zone_id)    REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_csess_day    ON mdm.calendar_session (calendar_day_id);
CREATE INDEX IF NOT EXISTS idx_csess_type   ON mdm.calendar_session (session_type);
CREATE INDEX IF NOT EXISTS idx_csess_tenant ON mdm.calendar_session (tenant_id);

-- ── calendar_special_event ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_special_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    event_type varchar(30) NOT NULL,
    severity varchar(20),
    affected_calendar_ids uuid[] NOT NULL,
    affected_country_cds varchar(2)[],
    affected_mic_cds varchar(4)[],
    start_date date NOT NULL,
    end_date date,
    start_time time,
    end_time time,
    time_zone_id uuid,
    reason text,
    announcement_source varchar(100),
    announcement_date date,
    announcement_url text,
    regulatory_reference varchar(100),
    is_regulator_directed bool DEFAULT false NOT NULL,
    affected_sessions varchar(30)[],
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_special_event_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cse_cd UNIQUE (tenant_id, event_cd),
    CONSTRAINT chk_cse_type CHECK (event_type IN (
        'CLOSURE','LATE_OPEN','EARLY_CLOSE','SPECIAL_SESSION',
        'EXTENDED_HOURS','SUSPENSION','EMERGENCY','HOLIDAY_ADDITION',
        'HOLIDAY_REMOVAL','CALENDAR_OVERRIDE')),
    CONSTRAINT chk_cse_severity CHECK (severity IS NULL OR severity IN (
        'PLANNED','ADVISORY','URGENT','EMERGENCY')),
    CONSTRAINT fk_cse_tz FOREIGN KEY (time_zone_id) REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_cse_dates  ON mdm.calendar_special_event (start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_cse_type   ON mdm.calendar_special_event (event_type, status);
CREATE INDEX IF NOT EXISTS idx_cse_tenant ON mdm.calendar_special_event (tenant_id);

-- ── calendar_special_event_impact ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_special_event_impact (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    special_event_id uuid NOT NULL,
    calendar_id uuid NOT NULL,
    impact_date date NOT NULL,
    impact_type varchar(30) NOT NULL,
    original_open_time time,
    original_close_time time,
    modified_open_time time,
    modified_close_time time,
    session_type varchar(30),
    note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_special_event_impact_pkey PRIMARY KEY (id),
    CONSTRAINT chk_csei_type CHECK (impact_type IN (
        'CLOSED','LATE_OPEN','EARLY_CLOSE','HALF_DAY','NORMAL',
        'SESSION_ADDED','SESSION_REMOVED')),
    CONSTRAINT fk_csei_event    FOREIGN KEY (special_event_id) REFERENCES mdm.calendar_special_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_csei_calendar FOREIGN KEY (calendar_id)      REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_csei_event    ON mdm.calendar_special_event_impact (special_event_id);
CREATE INDEX IF NOT EXISTS idx_csei_calendar ON mdm.calendar_special_event_impact (calendar_id, impact_date);
CREATE INDEX IF NOT EXISTS idx_csei_tenant   ON mdm.calendar_special_event_impact (tenant_id);

-- ── calendar_historical_closure ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_historical_closure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    closure_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    affected_calendar_ids uuid[] NOT NULL,
    closure_start_date date NOT NULL,
    closure_end_date date NOT NULL,
    business_days_lost int4,
    reason text,
    category varchar(30),
    source varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_historical_closure_pkey PRIMARY KEY (id),
    CONSTRAINT uq_chc_closure_cd UNIQUE (tenant_id, closure_cd),
    CONSTRAINT chk_chc_cat CHECK (category IS NULL OR category IN (
        'NATURAL_DISASTER','TERRORISM','PANDEMIC','TECHNICAL',
        'NATIONAL_MOURNING','REGULATORY','WAR','OTHER'))
);
CREATE INDEX IF NOT EXISTS idx_chc_dates  ON mdm.calendar_historical_closure (closure_start_date, closure_end_date);
CREATE INDEX IF NOT EXISTS idx_chc_cat    ON mdm.calendar_historical_closure (category);
CREATE INDEX IF NOT EXISTS idx_chc_tenant ON mdm.calendar_historical_closure (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_day','calendar_session','calendar_special_event',
        'calendar_special_event_impact','calendar_historical_closure'
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

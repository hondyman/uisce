-- 20261104_003_mdm_calendar_holiday_engine.up.sql

-- ── holiday_definition ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.holiday_definition (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    holiday_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    short_name varchar(100),
    holiday_type_id uuid NOT NULL,
    rule_type_id uuid NOT NULL,
    rule_params jsonb NOT NULL,
    description text,
    country_cd varchar(2),
    subdivision_cd varchar(10),
    city_cd varchar(20),
    religion varchar(30),
    is_paid_holiday bool DEFAULT true NOT NULL,
    is_bank_holiday bool DEFAULT true NOT NULL,
    is_market_holiday bool DEFAULT true NOT NULL,
    is_settlement_holiday bool DEFAULT true NOT NULL,
    is_half_day bool DEFAULT false NOT NULL,
    half_day_close_time time,
    substitute_rule varchar(30),
    substitute_rule_params jsonb,
    priority int4 DEFAULT 100 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT holiday_definition_pkey PRIMARY KEY (id),
    CONSTRAINT uq_hd_cd UNIQUE (tenant_id, holiday_cd),
    CONSTRAINT chk_hd_substitute CHECK (substitute_rule IS NULL OR substitute_rule IN (
        'NONE','NEXT_MONDAY','NEXT_WEEKDAY','PREVIOUS_FRIDAY',
        'NEXT_MONDAY_OR_TUESDAY','SAME_DAY')),
    CONSTRAINT fk_hd_type FOREIGN KEY (holiday_type_id) REFERENCES mdm.holiday_type(id),
    CONSTRAINT fk_hd_rule FOREIGN KEY (rule_type_id)    REFERENCES mdm.holiday_rule_type(id)
);
CREATE INDEX IF NOT EXISTS idx_hd_country ON mdm.holiday_definition (country_cd);
CREATE INDEX IF NOT EXISTS idx_hd_type    ON mdm.holiday_definition (holiday_type_id);
CREATE INDEX IF NOT EXISTS idx_hd_rule    ON mdm.holiday_definition (rule_type_id);
CREATE INDEX IF NOT EXISTS idx_hd_tenant  ON mdm.holiday_definition (tenant_id);

-- ── holiday_manual_date ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.holiday_manual_date (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    holiday_definition_id uuid NOT NULL,
    calendar_year int4 NOT NULL,
    observed_date date NOT NULL,
    astronomical_date date,
    is_provisional bool DEFAULT false NOT NULL,
    declaration_source varchar(100),
    declaration_date date,
    declaration_url text,
    is_observed bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT holiday_manual_date_pkey PRIMARY KEY (id),
    CONSTRAINT uq_hmd UNIQUE (tenant_id, holiday_definition_id, calendar_year),
    CONSTRAINT fk_hmd_holiday FOREIGN KEY (holiday_definition_id) REFERENCES mdm.holiday_definition(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_hmd_holiday ON mdm.holiday_manual_date (holiday_definition_id, calendar_year);
CREATE INDEX IF NOT EXISTS idx_hmd_date    ON mdm.holiday_manual_date (observed_date);
CREATE INDEX IF NOT EXISTS idx_hmd_tenant  ON mdm.holiday_manual_date (tenant_id);

-- ── holiday_definition_calendar (mapping) ──────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.holiday_definition_calendar (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    holiday_definition_id uuid NOT NULL,
    calendar_id uuid NOT NULL,
    is_inherited bool DEFAULT false NOT NULL,
    is_overridden bool DEFAULT false NOT NULL,
    override_params jsonb,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT holiday_definition_calendar_pkey PRIMARY KEY (id),
    CONSTRAINT fk_hdc_holiday  FOREIGN KEY (holiday_definition_id) REFERENCES mdm.holiday_definition(id) ON DELETE CASCADE,
    CONSTRAINT fk_hdc_calendar FOREIGN KEY (calendar_id)           REFERENCES mdm.calendar_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_hdc_holiday  ON mdm.holiday_definition_calendar (holiday_definition_id);
CREATE INDEX IF NOT EXISTS idx_hdc_calendar ON mdm.holiday_definition_calendar (calendar_id);
CREATE INDEX IF NOT EXISTS idx_hdc_tenant   ON mdm.holiday_definition_calendar (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_hdc_active
    ON mdm.holiday_definition_calendar (tenant_id, holiday_definition_id, calendar_id)
    WHERE effective_to IS NULL;

-- ── calendar_rule (per-calendar business-day rules) ────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
    rule_type varchar(30) NOT NULL,
    rule_expression jsonb NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_rule_pkey PRIMARY KEY (id),
    CONSTRAINT chk_calrule_type CHECK (rule_type IN (
        'BUSINESS_DAY','HOLIDAY_SOURCE','CUTOFF_TIME','SESSION_HOURS',
        'SETTLEMENT_CYCLE','SUBSTITUTE_APPLIED')),
    CONSTRAINT fk_calrule_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_calrule_calendar ON mdm.calendar_rule (calendar_id);
CREATE INDEX IF NOT EXISTS idx_calrule_tenant   ON mdm.calendar_rule (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'holiday_definition','holiday_manual_date',
        'holiday_definition_calendar','calendar_rule'
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

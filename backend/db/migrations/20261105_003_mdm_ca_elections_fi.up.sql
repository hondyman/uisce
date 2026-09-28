-- 20261105_003_mdm_ca_elections_fi.up.sql

-- ── ca_election_option ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_election_option (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    option_cd varchar(30) NOT NULL,
    election_type_id uuid NOT NULL,
    name varchar(250) NOT NULL,
    description text,
    cash_per_unit numeric(20,12),
    stock_ratio_from numeric(20,12),
    stock_ratio_to numeric(20,12),
    total_value_per_unit numeric(20,12),
    currency varchar(3),
    election_deadline_date date,
    election_deadline_time time,
    min_election_pct numeric(7,4),
    max_election_pct numeric(7,4),
    oversubscription_allowed bool DEFAULT false NOT NULL,
    proration_applies bool DEFAULT false NOT NULL,
    proration_method varchar(30),
    proration_factor numeric(12,8),
    default_election bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_election_option_pkey PRIMARY KEY (id),
    CONSTRAINT uq_caeo UNIQUE (tenant_id, ca_event_id, option_cd),
    CONSTRAINT fk_caeo_ca    FOREIGN KEY (ca_event_id)     REFERENCES mdm.ca_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_caeo_etype FOREIGN KEY (election_type_id) REFERENCES mdm.ca_election_type(id)
);
CREATE INDEX IF NOT EXISTS idx_caeo_ca     ON mdm.ca_election_option (ca_event_id, is_active);
CREATE INDEX IF NOT EXISTS idx_caeo_tenant ON mdm.ca_election_option (tenant_id);

-- ── ca_election_deadline ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_election_deadline (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    deadline_type varchar(30) NOT NULL,
    deadline_date date NOT NULL,
    deadline_time time NOT NULL,
    deadline_timezone_id uuid,
    cutoff_utc timestamptz,
    cutoff_local timestamptz,
    is_guaranteed bool DEFAULT false NOT NULL,
    custodian_deadline date,
    custodian_deadline_time time,
    is_active bool DEFAULT true NOT NULL,
    notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_election_deadline_pkey PRIMARY KEY (id),
    CONSTRAINT chk_caed_type CHECK (deadline_type IN (
        'INITIAL','FINAL','EXTENDED','WITHDRAWN')),
    CONSTRAINT fk_caed_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_caed_tz FOREIGN KEY (deadline_timezone_id) REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_caed_ca     ON mdm.ca_election_deadline (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_caed_date   ON mdm.ca_election_deadline (deadline_date);
CREATE INDEX IF NOT EXISTS idx_caed_tenant ON mdm.ca_election_deadline (tenant_id);

-- ── ca_election_instruction (reference to election execution) ──────────
CREATE TABLE IF NOT EXISTS mdm.ca_election_instruction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    election_option_id uuid NOT NULL,
    account_id uuid,
    security_id uuid NOT NULL,
    instructed_quantity numeric(24,6),
    instructed_pct numeric(7,4),
    instruction_date date NOT NULL,
    instruction_time timestamptz,
    instruction_method varchar(30),
    status varchar(20) NOT NULL,
    is_default_election bool DEFAULT false NOT NULL,
    is_oversubscription bool DEFAULT false NOT NULL,
    rejection_reason varchar(500),
    acknowledged_at timestamptz,
    executed_quantity numeric(24,6),
    executed_amount numeric(24,6),
    executed_currency varchar(3),
    custodian_ref varchar(100),
    proxy_ref varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_election_instruction_pkey PRIMARY KEY (id),
    CONSTRAINT chk_caei_status CHECK (status IN (
        'PENDING','SUBMITTED','ACKNOWLEDGED','ACCEPTED','REJECTED','PRORATED','CANCELLED')),
    CONSTRAINT fk_caei_ca     FOREIGN KEY (ca_event_id)        REFERENCES mdm.ca_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_caei_option FOREIGN KEY (election_option_id) REFERENCES mdm.ca_election_option(id)
);
CREATE INDEX IF NOT EXISTS idx_caei_ca       ON mdm.ca_election_instruction (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_caei_account  ON mdm.ca_election_instruction (account_id);
CREATE INDEX IF NOT EXISTS idx_caei_status   ON mdm.ca_election_instruction (status);
CREATE INDEX IF NOT EXISTS idx_caei_tenant   ON mdm.ca_election_instruction (tenant_id);

-- ── ca_bond_call ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_bond_call (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    security_id uuid NOT NULL,
    call_number int4,
    call_date date NOT NULL,
    call_notice_date date,
    call_notice_days int4,
    call_price numeric(20,12),
    call_premium numeric(20,12),
    accrued_interest numeric(20,12),
    call_type varchar(30) NOT NULL,
    is_mandatory bool DEFAULT false NOT NULL,
    is_partial bool DEFAULT false NOT NULL,
    call_amount numeric(24,4),
    remaining_amount numeric(24,4),
    redemption_currency varchar(3),
    is_exercised bool DEFAULT false NOT NULL,
    exercised_date date,
    announcement_ref varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_bond_call_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cabc_type CHECK (call_type IN (
        'OPTIONAL','MANDATORY','MAKE_WHOLE','REGULATORY','SPECIAL',
        'TAX_CALL','CLEAN_UP_CALL','PARTIAL')),
    CONSTRAINT fk_cabc_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_cabc_ca     ON mdm.ca_bond_call (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cabc_sec    ON mdm.ca_bond_call (security_id, call_date);
CREATE INDEX IF NOT EXISTS idx_cabc_tenant ON mdm.ca_bond_call (tenant_id);

-- ── ca_bond_put ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_bond_put (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    security_id uuid NOT NULL,
    put_number int4,
    put_date date NOT NULL,
    put_notice_date date,
    put_notice_days int4,
    put_price numeric(20,12),
    put_type varchar(30),
    is_mandatory bool DEFAULT false NOT NULL,
    put_amount numeric(24,4),
    redemption_currency varchar(3),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_bond_put_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cabp_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_cabp_ca     ON mdm.ca_bond_put (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cabp_sec    ON mdm.ca_bond_put (security_id, put_date);
CREATE INDEX IF NOT EXISTS idx_cabp_tenant ON mdm.ca_bond_put (tenant_id);

-- ── ca_default_event ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_default_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    issuer_id uuid,
    security_id uuid,
    event_type varchar(50) NOT NULL,
    event_date date NOT NULL,
    announcement_date date,
    cure_deadline date,
    cure_date date,
    is_cured bool,
    grace_period_days int4,
    default_amount numeric(24,4),
    recovery_rate numeric(7,4),
    recovery_estimate_date date,
    is_credit_event bool DEFAULT true NOT NULL,
    is_cds_trigger bool DEFAULT false NOT NULL,
    isdafix_determination varchar(100),
    determination_committee_cd varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_default_event_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cade_type CHECK (event_type IN (
        'FAILURE_TO_PAY','BANKRUPTCY','CROSS_DEFAULT','RESTRUCTURING',
        'OBLIGATION_ACCELERATION','REPUDIATION','MORATORIUM',
        'GOVERNMENT_INTERVENTION')),
    CONSTRAINT fk_cade_ca     FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_cade_issuer FOREIGN KEY (issuer_id)   REFERENCES edm.issuer_master(id)
);
CREATE INDEX IF NOT EXISTS idx_cade_ca     ON mdm.ca_default_event (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cade_issuer ON mdm.ca_default_event (issuer_id, event_date);
CREATE INDEX IF NOT EXISTS idx_cade_tenant ON mdm.ca_default_event (tenant_id);

-- ── ca_sinking_fund ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_sinking_fund (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    security_id uuid NOT NULL,
    payment_date date NOT NULL,
    payment_number int4,
    principal_amount numeric(24,4),
    interest_amount numeric(24,4),
    total_amount numeric(24,4),
    factor_before numeric(18,9),
    factor_after numeric(18,9),
    is_optional bool DEFAULT false NOT NULL,
    is_mandatory bool DEFAULT true NOT NULL,
    remaining_balance numeric(24,4),
    currency varchar(3),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_sinking_fund_pkey PRIMARY KEY (id),
    CONSTRAINT fk_casf_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_casf_ca     ON mdm.ca_sinking_fund (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_casf_sec    ON mdm.ca_sinking_fund (security_id, payment_date);
CREATE INDEX IF NOT EXISTS idx_casf_tenant ON mdm.ca_sinking_fund (tenant_id);

-- ── ca_consent_solicitation ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_consent_solicitation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    security_id uuid NOT NULL,
    solicitation_type varchar(30) NOT NULL,
    solicitation_date date NOT NULL,
    expiration_date date,
    consent_deadline_date date,
    consent_threshold_pct numeric(7,4),
    early_consent_deadline_date date,
    early_consent_fee numeric(20,12),
    late_consent_fee numeric(20,12),
    solicitation_fee numeric(20,12),
    description text,
    is_approved bool,
    approval_date date,
    final_consent_pct numeric(7,4),
    series_affected text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_consent_solicitation_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cacons_type CHECK (solicitation_type IN (
        'CONSENT','AMENDMENT','WAIVER','MODIFICATION',
        'COVENANT_CHANGE','MATURITY_EXTENSION','RATE_CHANGE')),
    CONSTRAINT fk_cacons_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cacons_ca     ON mdm.ca_consent_solicitation (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cacons_tenant ON mdm.ca_consent_solicitation (tenant_id);

-- ── ca_conversion ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_conversion (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    from_security_id uuid NOT NULL,
    to_security_id uuid NOT NULL,
    conversion_type varchar(30) NOT NULL,
    conversion_date date,
    conversion_start_date date,
    conversion_end_date date,
    conversion_ratio numeric(20,12),
    conversion_price numeric(20,12),
    conversion_currency varchar(3),
    is_triggered bool DEFAULT false NOT NULL,
    trigger_date date,
    trigger_reason varchar(255),
    is_cash_settled bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_conversion_pkey PRIMARY KEY (id),
    CONSTRAINT chk_caconv_type CHECK (conversion_type IN (
        'MANDATORY','OPTIONAL','CONTINGENT','FORCED',
        'CONVERSION_RATIO_CHANGE','CONVERSION_PRICE_CHANGE')),
    CONSTRAINT fk_caconv_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_caconv_ca     ON mdm.ca_conversion (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_caconv_sec    ON mdm.ca_conversion (from_security_id, conversion_date);
CREATE INDEX IF NOT EXISTS idx_caconv_tenant ON mdm.ca_conversion (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_election_option','ca_election_deadline','ca_election_instruction',
        'ca_bond_call','ca_bond_put','ca_default_event','ca_sinking_fund',
        'ca_consent_solicitation','ca_conversion'
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

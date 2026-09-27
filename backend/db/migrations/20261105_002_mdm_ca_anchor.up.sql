-- 20261105_002_mdm_ca_anchor.up.sql

-- ── ca_event (anchor) ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_ref varchar(50) NOT NULL,
    ca_official_ref varchar(100),
    corporate_action_id uuid,
    event_type_id uuid NOT NULL,
    mandatory_type_id uuid,
    status_id uuid NOT NULL,
    issuer_id uuid,
    primary_security_id uuid,
    event_currency varchar(3),
    primary_country_cd varchar(2),
    announcement_date date,
    effective_date date,
    ex_date date,
    record_date date,
    payment_date date,
    election_deadline_date date,
    election_deadline_time time,
    election_deadline_timezone_id uuid,
    expiration_date date,
    last_trade_date date,
    tax_rate numeric(12,8),
    is_taxable bool,
    is_qualified bool,
    has_cash_component bool DEFAULT false NOT NULL,
    has_stock_component bool DEFAULT false NOT NULL,
    has_election bool DEFAULT false NOT NULL,
    has_proxy_vote bool DEFAULT false NOT NULL,
    affects_cost_basis bool DEFAULT false NOT NULL,
    affects_position_quantity bool DEFAULT false NOT NULL,
    affects_position_identifier bool DEFAULT false NOT NULL,
    affects_nav bool DEFAULT false NOT NULL,
    affects_tax_lots bool DEFAULT false NOT NULL,
    requires_compliance_review bool DEFAULT false NOT NULL,
    requires_accounting_review bool DEFAULT false NOT NULL,
    requires_fund_action bool DEFAULT false NOT NULL,
    requires_regulatory_filing bool DEFAULT false NOT NULL,
    is_ad_hoc bool DEFAULT false NOT NULL,
    is_historical bool DEFAULT false NOT NULL,
    cancelled_date date,
    cancellation_reason varchar(500),
    superseded_by_id uuid,
    supersedes_id uuid,
    parent_ca_id uuid,
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    version_num int4 DEFAULT 1 NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_event_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_event_ref UNIQUE (tenant_id, ca_ref),
    CONSTRAINT fk_cae_etype      FOREIGN KEY (event_type_id)      REFERENCES mdm.ca_event_type(id),
    CONSTRAINT fk_cae_mtype      FOREIGN KEY (mandatory_type_id)  REFERENCES mdm.ca_mandatory_type(id),
    CONSTRAINT fk_cae_status     FOREIGN KEY (status_id)          REFERENCES mdm.ca_status(id),
    CONSTRAINT fk_cae_issuer     FOREIGN KEY (issuer_id)          REFERENCES edm.issuer_master(id),
    CONSTRAINT fk_cae_tz         FOREIGN KEY (election_deadline_timezone_id) REFERENCES mdm.time_zone(id),
    CONSTRAINT fk_cae_superseded FOREIGN KEY (superseded_by_id)   REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_cae_supersedes FOREIGN KEY (supersedes_id)      REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_cae_parent     FOREIGN KEY (parent_ca_id)       REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_cae_source     FOREIGN KEY (source_system_id)   REFERENCES mdm.source_systems(id),
    CONSTRAINT fk_cae_merged     FOREIGN KEY (merged_into_id)     REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_cae_etype       ON mdm.ca_event (event_type_id);
CREATE INDEX IF NOT EXISTS idx_cae_status      ON mdm.ca_event (status_id);
CREATE INDEX IF NOT EXISTS idx_cae_issuer      ON mdm.ca_event (issuer_id);
CREATE INDEX IF NOT EXISTS idx_cae_dates       ON mdm.ca_event (ex_date, record_date, payment_date);
CREATE INDEX IF NOT EXISTS idx_cae_election_dl ON mdm.ca_event (election_deadline_date) WHERE election_deadline_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cae_compliance  ON mdm.ca_event (requires_compliance_review) WHERE requires_compliance_review = true;
CREATE INDEX IF NOT EXISTS idx_cae_accounting  ON mdm.ca_event (requires_accounting_review) WHERE requires_accounting_review = true;
CREATE INDEX IF NOT EXISTS idx_cae_fund        ON mdm.ca_event (requires_fund_action) WHERE requires_fund_action = true;
CREATE INDEX IF NOT EXISTS idx_cae_tenant      ON mdm.ca_event (tenant_id);

-- Guarded FK to orm.corporate_action if that table exists
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='orm' AND table_name='corporate_action') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='ca_event' AND c.conname='fk_cae_corp_action'
        ) THEN
            ALTER TABLE mdm.ca_event
                ADD CONSTRAINT fk_cae_corp_action
                FOREIGN KEY (corporate_action_id) REFERENCES orm.corporate_action(id)
                NOT VALID;
        END IF;
    END IF;
END $fk$;

-- ── ca_identifier ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(200) NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cai_type CHECK (id_type IN (
        'BBG_CA_ID','RIC_CA_ID','ICE_CA_ID','DTCC_CA_ID','ISIN_REF',
        'EUROCLEAR_REF','CLEARSTREAM_REF','SEC_FILING_REF','INTERNAL',
        'ISSUER_REF','PAYING_AGENT_REF','TAX_EVENT_ID','PROXY_ID')),
    CONSTRAINT fk_cai_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cai_ca     ON mdm.ca_identifier (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cai_lookup ON mdm.ca_identifier (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_cai_tenant ON mdm.ca_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cai_active
    ON mdm.ca_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── ca_security (securities affected) ──────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_security (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    security_id uuid,
    security_identifier varchar(100),
    security_role varchar(30) NOT NULL,
    security_action varchar(30),
    new_isin varchar(15),
    new_cusip varchar(9),
    new_ticker varchar(26),
    new_name varchar(500),
    exchange_ratio numeric(18,9),
    effective_from date,
    effective_to date,
    is_primary bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_security_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cas_role CHECK (security_role IN (
        'PRIMARY','UNDERLYING','RESULTING','AFFECTED','COLLATERAL',
        'CONVERTIBLE_TARGET','ACQUIRER','TARGET','SPINOFF')),
    CONSTRAINT chk_cas_action CHECK (security_action IS NULL OR security_action IN (
        'ADDED','REMOVED','MODIFIED','UNCHANGED')),
    CONSTRAINT fk_cas_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cas_ca     ON mdm.ca_security (ca_event_id, security_role);
CREATE INDEX IF NOT EXISTS idx_cas_sec    ON mdm.ca_security (security_id);
CREATE INDEX IF NOT EXISTS idx_cas_tenant ON mdm.ca_security (tenant_id);

-- ── ca_term (generic term/value) ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_term (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    term_cd varchar(50) NOT NULL,
    term_name varchar(250),
    term_value_text varchar(500),
    term_value_numeric numeric(28,12),
    term_value_date date,
    term_value_boolean bool,
    term_value_json jsonb,
    term_unit varchar(30),
    is_primary bool DEFAULT false NOT NULL,
    is_taxable bool,
    is_qualified bool,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_term_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_term UNIQUE (tenant_id, ca_event_id, term_cd),
    CONSTRAINT fk_caterm_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_caterm_ca     ON mdm.ca_term (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_caterm_cd     ON mdm.ca_term (term_cd);
CREATE INDEX IF NOT EXISTS idx_caterm_tenant ON mdm.ca_term (tenant_id);

-- ── ca_cash_component ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_cash_component (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    component_cd varchar(30) NOT NULL,
    amount_per_unit numeric(20,12),
    total_amount numeric(24,4),
    currency varchar(3) NOT NULL,
    payment_date date,
    tax_rate numeric(12,8),
    withholding_rate numeric(12,8),
    is_net bool DEFAULT false NOT NULL,
    is_gross bool DEFAULT true NOT NULL,
    payment_type_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_cash_component_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cacc_comp CHECK (component_cd IN (
        'GROSS_DIVIDEND','NET_DIVIDEND','RETURN_OF_CAPITAL',
        'INTEREST','PRINCIPAL','PREMIUM','CALL_PRICE','TENDER_PRICE',
        'FRACTIONAL_CASH_IN_LIEU','SCRIP_DIVIDEND_CASH',
        'WITHHOLDING_TAX','FOREIGN_TAX','STAMP_DUTY','FEES')),
    CONSTRAINT fk_cacc_ca    FOREIGN KEY (ca_event_id)     REFERENCES mdm.ca_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_cacc_ptype FOREIGN KEY (payment_type_id) REFERENCES mdm.ca_payment_type(id)
);
CREATE INDEX IF NOT EXISTS idx_cacc_ca     ON mdm.ca_cash_component (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cacc_tenant ON mdm.ca_cash_component (tenant_id);

-- ── ca_stock_component ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_stock_component (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    component_cd varchar(30) NOT NULL,
    from_security_id uuid,
    to_security_id uuid,
    ratio_from numeric(20,12),
    ratio_to numeric(20,12),
    ratio_ratio numeric(20,12),
    resulting_quantity numeric(20,12),
    resulting_security_type varchar(30),
    is_fractional bool DEFAULT false NOT NULL,
    fractional_treatment varchar(30),
    effective_date date,
    payment_date date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_stock_component_pkey PRIMARY KEY (id),
    CONSTRAINT chk_casc_comp CHECK (component_cd IN (
        'NEW_SHARES','RESULTING_SHARES','CANCELLED_SHARES',
        'ACQUIRER_SHARES','SPINOFF_SHARES','RIGHTS','WARRANTS')),
    CONSTRAINT fk_casc_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_casc_ca     ON mdm.ca_stock_component (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_casc_tenant ON mdm.ca_stock_component (tenant_id);

-- ── ca_ratio_history ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_ratio_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    effective_date date NOT NULL,
    ratio_type varchar(30) NOT NULL,
    ratio_from numeric(20,12),
    ratio_to numeric(20,12),
    ratio_ratio numeric(20,12),
    from_security_id uuid,
    to_security_id uuid,
    is_current bool DEFAULT true NOT NULL,
    reason varchar(255),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_ratio_history_pkey PRIMARY KEY (id),
    CONSTRAINT chk_carh_type CHECK (ratio_type IN (
        'SPLIT_RATIO','EXCHANGE_RATIO','CONVERSION_RATIO',
        'PARTICIPATION_RATIO','SUBSCRIPTION_RATIO')),
    CONSTRAINT fk_carh_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_carh_ca     ON mdm.ca_ratio_history (ca_event_id, is_current);
CREATE INDEX IF NOT EXISTS idx_carh_tenant ON mdm.ca_ratio_history (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_event','ca_identifier','ca_security','ca_term',
        'ca_cash_component','ca_stock_component','ca_ratio_history'
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

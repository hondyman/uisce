-- 20261105_001_mdm_ca_reference.up.sql

-- ── ca_event_type ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_event_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    sub_category varchar(30),
    is_income_event bool DEFAULT false NOT NULL,
    is_principal_event bool DEFAULT false NOT NULL,
    is_mandatory bool DEFAULT false NOT NULL,
    is_voluntary bool DEFAULT false NOT NULL,
    requires_election bool DEFAULT false NOT NULL,
    requires_proxy_vote bool DEFAULT false NOT NULL,
    affects_cost_basis bool DEFAULT false NOT NULL,
    affects_nav bool DEFAULT false NOT NULL,
    affects_tax_lots bool DEFAULT false NOT NULL,
    requires_regulatory_notification bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_event_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_et_cd UNIQUE (tenant_id, event_type_cd),
    CONSTRAINT chk_ca_et_cat CHECK (category IN (
        'INCOME','MANDATORY','VOLUNTARY','MANDATORY_WITH_OPTIONS','TAX',
        'LEGAL','REGULATORY','PROXY','FUND','FIXED_INCOME','EQUITY',
        'DERIVATIVE','CLASS_ACTION'))
);
CREATE INDEX IF NOT EXISTS idx_ca_et_cat    ON mdm.ca_event_type (category);
CREATE INDEX IF NOT EXISTS idx_ca_et_tenant ON mdm.ca_event_type (tenant_id);

-- ── ca_status ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    status_category varchar(30) NOT NULL,
    is_terminal bool DEFAULT false NOT NULL,
    requires_processing bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_status_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_status_cd UNIQUE (tenant_id, status_cd),
    CONSTRAINT chk_ca_status_cat CHECK (status_category IN (
        'ANNOUNCED','ACTIVE','COMPLETED','CANCELLED','FAILED','SUPERSEDED'))
);
CREATE INDEX IF NOT EXISTS idx_ca_status_tenant ON mdm.ca_status (tenant_id);

-- ── ca_election_type ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_election_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    election_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    requires_instruction bool DEFAULT true NOT NULL,
    has_default bool DEFAULT false NOT NULL,
    default_election_cd varchar(30),
    proration_possible bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_election_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_elt_cd UNIQUE (tenant_id, election_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_ca_elt_tenant ON mdm.ca_election_type (tenant_id);

-- ── ca_payment_type ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_payment_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    payment_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    payment_method varchar(30),
    is_income bool DEFAULT false NOT NULL,
    is_principal bool DEFAULT false NOT NULL,
    is_interest bool DEFAULT false NOT NULL,
    is_dividend bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_payment_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_pt_cd UNIQUE (tenant_id, payment_type_cd),
    CONSTRAINT chk_ca_pt_method CHECK (payment_method IS NULL OR payment_method IN (
        'CASH','STOCK','BOND','MIXED','IN_KIND','SCRIP'))
);
CREATE INDEX IF NOT EXISTS idx_ca_pt_tenant ON mdm.ca_payment_type (tenant_id);

-- ── ca_tax_treatment ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_tax_treatment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    treatment_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    is_ordinary_income bool DEFAULT false NOT NULL,
    is_qualified_dividend bool DEFAULT false NOT NULL,
    is_capital_gain bool DEFAULT false NOT NULL,
    is_return_of_capital bool DEFAULT false NOT NULL,
    is_tax_free bool DEFAULT false NOT NULL,
    is_deferred bool DEFAULT false NOT NULL,
    affects_cost_basis bool DEFAULT false NOT NULL,
    withholding_required bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_tax_treatment_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_tt_cd UNIQUE (tenant_id, treatment_cd)
);
CREATE INDEX IF NOT EXISTS idx_ca_tt_tenant ON mdm.ca_tax_treatment (tenant_id);

-- ── ca_regulatory_regime ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_regulatory_regime (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    regime_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    jurisdiction varchar(10) NOT NULL,
    regulator_cd varchar(50),
    regulation_name varchar(250),
    regulation_reference varchar(100),
    requires_notification bool DEFAULT false NOT NULL,
    requires_filing bool DEFAULT false NOT NULL,
    requires_shareholder_approval bool DEFAULT false NOT NULL,
    notification_deadline_days int4,
    effective_from date NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_regulatory_regime_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_rr_cd UNIQUE (tenant_id, regime_cd)
);
CREATE INDEX IF NOT EXISTS idx_ca_rr_tenant ON mdm.ca_regulatory_regime (tenant_id);

-- ── ca_mandatory_type ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_mandatory_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mandatory_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_mandatory bool DEFAULT false NOT NULL,
    is_voluntary bool DEFAULT false NOT NULL,
    is_mandatory_with_options bool DEFAULT false NOT NULL,
    requires_election bool DEFAULT false NOT NULL,
    requires_response_by_deadline bool DEFAULT false NOT NULL,
    default_action varchar(30),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_mandatory_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_mt_cd UNIQUE (tenant_id, mandatory_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_ca_mt_tenant ON mdm.ca_mandatory_type (tenant_id);

-- ── ca_entitlement_basis ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_entitlement_basis (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    basis_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    valuation_date_basis varchar(30),
    requires_position_snapshot bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_entitlement_basis_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ca_eb_cd UNIQUE (tenant_id, basis_cd),
    CONSTRAINT chk_ca_eb_basis CHECK (valuation_date_basis IS NULL OR valuation_date_basis IN (
        'RECORD_DATE','EX_DATE','TRADE_DATE','PAYMENT_DATE'))
);
CREATE INDEX IF NOT EXISTS idx_ca_eb_tenant ON mdm.ca_entitlement_basis (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_event_type','ca_status','ca_election_type','ca_payment_type',
        'ca_tax_treatment','ca_regulatory_regime','ca_mandatory_type',
        'ca_entitlement_basis'
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

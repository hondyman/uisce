-- 20261030_003_mdm_counterparty_credit_agreements.up.sql

-- ── counterparty_credit_profile ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_credit_profile (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    as_of_date date NOT NULL,
    internal_rating_cd varchar(30),
    internal_pd numeric(12,8),
    internal_lgd numeric(12,8),
    external_moodys varchar(20),
    external_sp varchar(20),
    external_fitch varchar(20),
    external_dbrs varchar(20),
    external_kbra varchar(20),
    composite_rating_cd varchar(20),
    is_investment_grade bool,
    credit_watch varchar(20),
    capital_adequacy_pct numeric(12,8),
    tier1_ratio_pct numeric(12,8),
    leverage_ratio_pct numeric(12,8),
    cds_spread_bps numeric(12,4),
    last_financial_statement_date date,
    next_financial_statement_date date,
    analyst_party_id uuid,
    approved_by uuid,
    approved_at timestamptz,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_credit_profile_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ccp_watch CHECK (credit_watch IS NULL OR credit_watch IN (
        'NONE','POSITIVE','NEGATIVE','DEVELOPING')),
    CONSTRAINT fk_ccp_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ccp_cpty   ON mdm.counterparty_credit_profile (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_ccp_tenant ON mdm.counterparty_credit_profile (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ccp_current
    ON mdm.counterparty_credit_profile (tenant_id, counterparty_id) WHERE is_current = true;

-- ── counterparty_credit_limit ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_credit_limit (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    limit_type varchar(30) NOT NULL,
    limit_scope varchar(30),
    scope_reference_cd varchar(50),
    limit_amount numeric(28,4) NOT NULL,
    limit_currency varchar(3) NOT NULL,
    current_utilization numeric(28,4),
    available_amount numeric(28,4),
    utilization_pct numeric(7,4),
    warning_threshold_pct numeric(7,4),
    breach_threshold_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    approved_by uuid,
    approved_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_credit_limit_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ccl_type CHECK (limit_type IN (
        'SETTLEMENT','PRE_SETTLEMENT','DERIVATIVE','REPO','SEC_LENDING',
        'FX','CUSTODY','CLEARING','MARGIN','UNCOLLATERALIZED',
        'SECURED','TOTAL')),
    CONSTRAINT fk_ccl_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ccl_cpty   ON mdm.counterparty_credit_limit (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_ccl_type   ON mdm.counterparty_credit_limit (limit_type, is_current);
CREATE INDEX IF NOT EXISTS idx_ccl_tenant ON mdm.counterparty_credit_limit (tenant_id);

-- ── counterparty_agreement ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_agreement (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agreement_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    counterparty_id uuid NOT NULL,
    agreement_type_id uuid NOT NULL,
    parent_agreement_id uuid,
    executed_date date NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    termination_date date,
    termination_reason varchar(500),
    governing_law_jurisdiction varchar(10),
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    is_netting_agreement bool DEFAULT false NOT NULL,
    is_collateralized bool DEFAULT false NOT NULL,
    is_cross_product bool DEFAULT false NOT NULL,
    is_cross_default bool DEFAULT false NOT NULL,
    is_master_agreement bool DEFAULT false NOT NULL,
    document_id uuid,
    document_hash varchar(64),
    negotiation_owner_party_id uuid,
    legal_review_date date,
    is_regulatory_required bool DEFAULT false NOT NULL,
    auto_renewal bool DEFAULT false NOT NULL,
    renewal_notice_days int4,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_agreement_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_agreement_cd_key UNIQUE (tenant_id, agreement_cd),
    CONSTRAINT chk_cpa_status CHECK (status IN (
        'DRAFT','NEGOTIATION','PENDING_SIGNATURE','ACTIVE','AMENDED',
        'TERMINATED','EXPIRED','SUSPENDED')),
    CONSTRAINT fk_cpa_cpty   FOREIGN KEY (counterparty_id)   REFERENCES mdm.counterparty(id) ON DELETE CASCADE,
    CONSTRAINT fk_cpa_type   FOREIGN KEY (agreement_type_id) REFERENCES mdm.agreement_type(id),
    CONSTRAINT fk_cpa_parent FOREIGN KEY (parent_agreement_id) REFERENCES mdm.counterparty_agreement(id)
);
CREATE INDEX IF NOT EXISTS idx_cpa_cpty   ON mdm.counterparty_agreement (counterparty_id, status);
CREATE INDEX IF NOT EXISTS idx_cpa_type   ON mdm.counterparty_agreement (agreement_type_id);
CREATE INDEX IF NOT EXISTS idx_cpa_tenant ON mdm.counterparty_agreement (tenant_id);

-- ── counterparty_agreement_term ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_agreement_term (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agreement_id uuid NOT NULL,
    term_cd varchar(50) NOT NULL,
    term_name varchar(250),
    term_value_text varchar(500),
    term_value_numeric numeric(24,6),
    term_value_date date,
    term_value_boolean bool,
    term_value_json jsonb,
    section_reference varchar(50),
    is_negotiated bool DEFAULT false NOT NULL,
    is_standard_template bool DEFAULT true NOT NULL,
    modified_from_template bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_agreement_term_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cpat_agreement FOREIGN KEY (agreement_id) REFERENCES mdm.counterparty_agreement(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cpat_agreement ON mdm.counterparty_agreement_term (agreement_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpat_term      ON mdm.counterparty_agreement_term (term_cd);
CREATE INDEX IF NOT EXISTS idx_cpat_tenant    ON mdm.counterparty_agreement_term (tenant_id);

-- ── counterparty_due_diligence ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_due_diligence (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    review_date date NOT NULL,
    review_type varchar(30) NOT NULL,
    reviewer_party_id uuid,
    overall_risk_rating varchar(20),
    operational_risk_rating varchar(20),
    credit_risk_rating varchar(20),
    compliance_risk_rating varchar(20),
    reputational_risk_rating varchar(20),
    kyc_verified bool DEFAULT false NOT NULL,
    sanctions_cleared bool DEFAULT false NOT NULL,
    pep_cleared bool DEFAULT false NOT NULL,
    adverse_media_cleared bool DEFAULT false NOT NULL,
    financial_statements_reviewed bool DEFAULT false NOT NULL,
    regulatory_status_verified bool DEFAULT false NOT NULL,
    insurance_verified bool DEFAULT false NOT NULL,
    bcp_verified bool DEFAULT false NOT NULL,
    cyber_verified bool DEFAULT false NOT NULL,
    findings text,
    issues_identified int4 DEFAULT 0,
    remediation_required bool DEFAULT false NOT NULL,
    remediation_deadline date,
    disposition varchar(30),
    next_review_date date,
    approved_by uuid,
    approved_at timestamptz,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_due_diligence_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cdd_review_type CHECK (review_type IN (
        'ONBOARDING','PERIODIC','EVENT_DRIVEN','CREDIT_REVIEW',
        'REGULATORY','ESCALATION','POST_INCIDENT')),
    CONSTRAINT chk_cdd_disposition CHECK (disposition IS NULL OR disposition IN (
        'APPROVED','APPROVED_WITH_CONDITIONS','CONDITIONAL','DEFERRED',
        'REJECTED','ESCALATED')),
    CONSTRAINT fk_cdd_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cdd_cpty   ON mdm.counterparty_due_diligence (counterparty_id, review_date);
CREATE INDEX IF NOT EXISTS idx_cdd_tenant ON mdm.counterparty_due_diligence (tenant_id);

-- ── counterparty_exposure ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_exposure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    as_of_timestamp timestamptz NOT NULL,
    exposure_type varchar(30) NOT NULL,
    exposure_amount numeric(28,4) NOT NULL,
    exposure_currency varchar(3) NOT NULL,
    collateral_held numeric(28,4),
    collateral_posted numeric(28,4),
    net_exposure numeric(28,4),
    credit_limit_id uuid,
    utilization_pct numeric(7,4),
    is_breach bool DEFAULT false NOT NULL,
    breach_severity varchar(10),
    methodology_cd varchar(50),
    source_system_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_exposure_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cex_type CHECK (exposure_type IN (
        'CURRENT','POTENTIAL_FUTURE','GROSS','NET','SECURED','UNSECURED',
        'SETTLEMENT','PRE_SETTLEMENT','DERIVATIVE_MTM','REPO','SEC_LENDING',
        'CASH','CUSTODY_ASSETS')),
    CONSTRAINT fk_cex_cpty  FOREIGN KEY (counterparty_id)  REFERENCES mdm.counterparty(id) ON DELETE CASCADE,
    CONSTRAINT fk_cex_limit FOREIGN KEY (credit_limit_id)  REFERENCES mdm.counterparty_credit_limit(id)
);
CREATE INDEX IF NOT EXISTS idx_cex_cpty   ON mdm.counterparty_exposure (counterparty_id, as_of_timestamp);
CREATE INDEX IF NOT EXISTS idx_cex_breach ON mdm.counterparty_exposure (is_breach) WHERE is_breach = true;
CREATE INDEX IF NOT EXISTS idx_cex_tenant ON mdm.counterparty_exposure (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty_credit_profile','counterparty_credit_limit',
        'counterparty_agreement','counterparty_agreement_term',
        'counterparty_due_diligence','counterparty_exposure'
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

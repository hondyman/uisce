-- 20261030_002_mdm_counterparty_anchor.up.sql
-- Note: counterparty.issuer_id FKs to edm.issuer_master(id) (cross-schema).
-- mdm.issuer does not exist; edm.issuer_master is the canonical issuer table.

-- ── counterparty ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_cd varchar(50) NOT NULL,
    issuer_id uuid,
    party_id uuid,
    counterparty_type_id uuid NOT NULL,
    status_id uuid NOT NULL,
    legal_name varchar(500) NOT NULL,
    display_name varchar(500),
    lei varchar(20),
    bic varchar(11),
    primary_jurisdiction varchar(10),
    country_of_risk varchar(2),
    is_financial_institution bool DEFAULT false NOT NULL,
    is_g_sib bool DEFAULT false NOT NULL,
    is_d_sib bool DEFAULT false NOT NULL,
    is_ccp bool DEFAULT false NOT NULL,
    is_central_bank bool DEFAULT false NOT NULL,
    is_regulated bool DEFAULT true NOT NULL,
    regulator_cd varchar(50),
    primary_contact_party_id uuid,
    relationship_manager_party_id uuid,
    onboarding_date date,
    offboarding_date date,
    approval_date date,
    approved_by uuid,
    next_review_date date,
    review_frequency_months int4,
    is_approved_for_trading bool DEFAULT false NOT NULL,
    is_approved_for_derivatives bool DEFAULT false NOT NULL,
    is_approved_for_repo bool DEFAULT false NOT NULL,
    is_approved_for_sec_lending bool DEFAULT false NOT NULL,
    is_approved_for_clearing bool DEFAULT false NOT NULL,
    is_approved_for_fx bool DEFAULT false NOT NULL,
    is_approved_for_custody bool DEFAULT false NOT NULL,
    is_approved_for_margin bool DEFAULT false NOT NULL,
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_cd_key UNIQUE (tenant_id, counterparty_cd),
    CONSTRAINT fk_cpty_issuer   FOREIGN KEY (issuer_id)              REFERENCES edm.issuer_master(id),
    CONSTRAINT fk_cpty_party    FOREIGN KEY (party_id)               REFERENCES mdm.party(id),
    CONSTRAINT fk_cpty_type     FOREIGN KEY (counterparty_type_id)   REFERENCES mdm.counterparty_type(id),
    CONSTRAINT fk_cpty_status   FOREIGN KEY (status_id)              REFERENCES mdm.counterparty_status(id),
    CONSTRAINT fk_cpty_contact  FOREIGN KEY (primary_contact_party_id) REFERENCES mdm.party(id),
    CONSTRAINT fk_cpty_rm       FOREIGN KEY (relationship_manager_party_id) REFERENCES mdm.party(id),
    CONSTRAINT fk_cpty_merged   FOREIGN KEY (merged_into_id)         REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpty_type      ON mdm.counterparty (counterparty_type_id);
CREATE INDEX IF NOT EXISTS idx_cpty_status    ON mdm.counterparty (status_id);
CREATE INDEX IF NOT EXISTS idx_cpty_lei       ON mdm.counterparty (lei) WHERE lei IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cpty_bic       ON mdm.counterparty (bic) WHERE bic IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cpty_review    ON mdm.counterparty (next_review_date);
CREATE INDEX IF NOT EXISTS idx_cpty_tenant    ON mdm.counterparty (tenant_id);

-- ── counterparty_role_assignment ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_role_assignment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    role_id uuid NOT NULL,
    role_context varchar(30),
    context_reference_cd varchar(50),
    is_primary bool DEFAULT false NOT NULL,
    priority int4 DEFAULT 100,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_role_assignment_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cpra_status CHECK (status IN (
        'PENDING','ACTIVE','SUSPENDED','REVOKED','EXPIRED')),
    CONSTRAINT fk_cpra_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE,
    CONSTRAINT fk_cpra_role FOREIGN KEY (role_id)         REFERENCES mdm.counterparty_role(id)
);
CREATE INDEX IF NOT EXISTS idx_cpra_cpty   ON mdm.counterparty_role_assignment (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpra_role   ON mdm.counterparty_role_assignment (role_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpra_tenant ON mdm.counterparty_role_assignment (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cpra_active
    ON mdm.counterparty_role_assignment (tenant_id, counterparty_id, role_id,
                                          COALESCE(role_context, ''), COALESCE(context_reference_cd, ''))
    WHERE effective_to IS NULL;

-- ── counterparty_identifier ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(100) NOT NULL,
    issuing_authority varchar(250),
    issuing_country_cd varchar(2),
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cpi_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cpi_cpty   ON mdm.counterparty_identifier (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpi_tenant ON mdm.counterparty_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cpi_active
    ON mdm.counterparty_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── counterparty_settlement_instruction ────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_settlement_instruction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    instruction_name varchar(250),
    payment_type varchar(30) NOT NULL,
    currency varchar(3),
    nostro_account_cd varchar(50),
    nostro_account_name varchar(250),
    nostro_bank_name varchar(500),
    nostro_bic varchar(11),
    intermediary_bank_name varchar(500),
    intermediary_bic varchar(11),
    beneficiary_account varchar(50),
    beneficiary_name varchar(500),
    beneficiary_address varchar(500),
    reference_format varchar(250),
    is_default bool DEFAULT false NOT NULL,
    is_verified bool DEFAULT false NOT NULL,
    verified_at timestamptz,
    verified_by uuid,
    verification_method varchar(30),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_settlement_instruction_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cpsi_payment_type CHECK (payment_type IN (
        'WIRE','SWIFT','ACH','CHAPS','TARGET2','FEDWIRE','CHIPS',
        'CUSTODIAL_DVP','CUSTODIAL_FOP','INTERNAL_BOOK','CLS')),
    CONSTRAINT fk_cpsi_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cpsi_cpty     ON mdm.counterparty_settlement_instruction (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpsi_currency ON mdm.counterparty_settlement_instruction (currency, is_current);
CREATE INDEX IF NOT EXISTS idx_cpsi_tenant   ON mdm.counterparty_settlement_instruction (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty','counterparty_role_assignment',
        'counterparty_identifier','counterparty_settlement_instruction'
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

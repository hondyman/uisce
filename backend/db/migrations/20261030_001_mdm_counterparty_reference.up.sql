-- 20261030_001_mdm_counterparty_reference.up.sql

-- ── counterparty_type ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    requires_isda bool DEFAULT false NOT NULL,
    requires_credit_limit bool DEFAULT true NOT NULL,
    requires_due_diligence bool DEFAULT true NOT NULL,
    requires_settlement_instructions bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_type_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_type_cd_key UNIQUE (tenant_id, type_cd),
    CONSTRAINT chk_cpt_category CHECK (category IN (
        'FINANCIAL_INSTITUTION','SERVICE_PROVIDER','COMMERCIAL',
        'GOVERNMENT','INFRASTRUCTURE','INTERMEDIARY'))
);
CREATE INDEX IF NOT EXISTS idx_cpt_tenant ON mdm.counterparty_type (tenant_id);

-- ── counterparty_status ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_active_status bool DEFAULT true NOT NULL,
    is_terminal bool DEFAULT false NOT NULL,
    allows_trading bool DEFAULT false NOT NULL,
    allows_settlement bool DEFAULT false NOT NULL,
    requires_remediation bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_status_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_status_cd_key UNIQUE (tenant_id, status_cd)
);
CREATE INDEX IF NOT EXISTS idx_cps_tenant ON mdm.counterparty_status (tenant_id);

-- ── counterparty_role ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_role (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    role_category varchar(30) NOT NULL,
    requires_agreement bool DEFAULT false NOT NULL,
    requires_credit_limit bool DEFAULT true NOT NULL,
    requires_collateral bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_role_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_role_cd_key UNIQUE (tenant_id, role_cd),
    CONSTRAINT chk_cpr_category CHECK (role_category IN (
        'TRADING','CLEARING','SETTLEMENT','CUSTODY','ADMINISTRATION',
        'FINANCING','ADVISORY','DATA','REGULATORY','DISTRIBUTION'))
);
CREATE INDEX IF NOT EXISTS idx_cpr_tenant ON mdm.counterparty_role (tenant_id);

-- ── agreement_type ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.agreement_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agreement_type_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    is_master_agreement bool DEFAULT false NOT NULL,
    requires_netting bool DEFAULT false NOT NULL,
    requires_collateral bool DEFAULT false NOT NULL,
    standard_template varchar(100),
    regulatory_regime varchar(50),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT agreement_type_pkey PRIMARY KEY (id),
    CONSTRAINT agreement_type_cd_key UNIQUE (tenant_id, agreement_type_cd),
    CONSTRAINT chk_at_category CHECK (category IN (
        'DERIVATIVE','REPO','SECURITIES_LENDING','CLEARING','CUSTODY',
        'SERVICE','DATA','TRADING','FINANCING','OTHER'))
);
CREATE INDEX IF NOT EXISTS idx_at_tenant ON mdm.agreement_type (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty_type','counterparty_status','counterparty_role','agreement_type'
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

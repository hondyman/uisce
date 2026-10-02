-- 20261106_003_mdm_product_distribution.up.sql

-- ── product_platform ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_platform (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    share_class_cd varchar(30),
    platform_id uuid,
    platform_name varchar(250),
    platform_type varchar(30),
    is_available bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    onboarded_date date,
    offboarded_date date,
    onboarding_status varchar(20),
    notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_platform_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_plat_type CHECK (platform_type IS NULL OR platform_type IN (
        'RIA_PLATFORM','BROKER_DEALER_PLATFORM','BANK_PLATFORM',
        'INSURANCE_PLATFORM','RETIREMENT_PLATFORM','SUPERMARKET',
        'WRAP_PLATFORM','INTERNAL')),
    CONSTRAINT fk_prd_platform FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_platform ON mdm.product_platform (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_plt_tnt  ON mdm.product_platform (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_platform
    ON mdm.product_platform (tenant_id, product_id, COALESCE(platform_id::text, ''), COALESCE(share_class_cd, ''), effective_from);

-- ── product_distribution ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_distribution (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    channel_id uuid NOT NULL,
    distributor_id uuid,
    distributor_name varchar(250),
    agreement_ref varchar(100),
    agreement_date date,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_distribution_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prd_dist_product FOREIGN KEY (product_id)   REFERENCES mdm.product(id) ON DELETE CASCADE,
    CONSTRAINT fk_prd_dist_channel FOREIGN KEY (channel_id)   REFERENCES mdm.product_distribution_channel(id),
    CONSTRAINT fk_prd_dist_dist    FOREIGN KEY (distributor_id) REFERENCES edm.issuer_master(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_dist_product ON mdm.product_distribution (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_dist_tenant  ON mdm.product_distribution (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_dist
    ON mdm.product_distribution (tenant_id, product_id, channel_id, COALESCE(distributor_id::text, ''), effective_from);

-- ── product_registration ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_registration (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    share_class_cd varchar(30),
    registration_type_id uuid,
    jurisdiction varchar(10) NOT NULL,
    regulator_cd varchar(50),
    registration_number varchar(100),
    registration_status varchar(20) NOT NULL,
    registration_date date,
    effective_from date,
    effective_to date,
    filing_reference varchar(100),
    filing_url text,
    investor_eligibility text,
    distributor_restrictions text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_registration_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_reg_status CHECK (registration_status IN (
        'PENDING','ACTIVE','SUSPENDED','WITHDRAWN','LAPSED','EXPIRED')),
    CONSTRAINT fk_prd_reg_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE,
    CONSTRAINT fk_prd_reg_type    FOREIGN KEY (registration_type_id) REFERENCES mdm.product_registration_type(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_reg_product ON mdm.product_registration (product_id, registration_status);
CREATE INDEX IF NOT EXISTS idx_prd_reg_jur     ON mdm.product_registration (jurisdiction, regulator_cd);
CREATE INDEX IF NOT EXISTS idx_prd_reg_tenant  ON mdm.product_registration (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_reg
    ON mdm.product_registration (tenant_id, product_id, COALESCE(share_class_cd, ''), jurisdiction, COALESCE(regulator_cd, ''));

-- ── product_eligibility ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_eligibility (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    share_class_cd varchar(30),
    jurisdiction varchar(10),
    eligibility_type varchar(30) NOT NULL,
    is_eligible bool DEFAULT true NOT NULL,
    is_restricted bool DEFAULT false NOT NULL,
    restriction_reason text,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_eligibility_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_elig_type CHECK (eligibility_type IN (
        'RETAIL','ACCREDITED','QUALIFIED_PURCHASER','PROFESSIONAL',
        'ELIGIBLE_COUNTERPARTY','INSTITUTIONAL','HNWI','SOPHISTICATED',
        'PENSION','INSURANCE','SOVEREIGN')),
    CONSTRAINT fk_prd_elig_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_elig_product ON mdm.product_eligibility (product_id);
CREATE INDEX IF NOT EXISTS idx_prd_elig_tenant  ON mdm.product_eligibility (tenant_id);

-- ── product_target_market ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_target_market (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    target_segment varchar(50) NOT NULL,
    risk_tolerance varchar(20),
    investment_horizon varchar(30),
    knowledge_experience varchar(30),
    ability_to_bear_loss varchar(30),
    distribution_strategy varchar(30),
    is_positive_target bool DEFAULT true NOT NULL,
    rationale text,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_target_market_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_tm_ke CHECK (knowledge_experience IS NULL OR knowledge_experience IN (
        'BASIC','INFORMED','ADVANCED','EXPERT')),
    CONSTRAINT chk_prd_tm_ds CHECK (distribution_strategy IS NULL OR distribution_strategy IN (
        'EXECUTION_ONLY','NON_ADVISED','ADVISED','DISCRETIONARY')),
    CONSTRAINT fk_prd_tm_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_tm_product ON mdm.product_target_market (product_id);
CREATE INDEX IF NOT EXISTS idx_prd_tm_tenant  ON mdm.product_target_market (tenant_id);

-- ── product_document ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_document (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    share_class_cd varchar(30),
    document_type_id uuid,
    document_type_cd varchar(30),
    version varchar(30),
    language varchar(10),
    jurisdiction varchar(10),
    effective_date date NOT NULL,
    expiration_date date,
    filing_date date,
    document_url text,
    document_hash varchar(64),
    is_current bool DEFAULT true NOT NULL,
    is_regulatory_filing bool DEFAULT false NOT NULL,
    superseded_by_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_document_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prd_doc_product    FOREIGN KEY (product_id)       REFERENCES mdm.product(id) ON DELETE CASCADE,
    CONSTRAINT fk_prd_doc_type       FOREIGN KEY (document_type_id) REFERENCES mdm.product_document_type(id),
    CONSTRAINT fk_prd_doc_superseded FOREIGN KEY (superseded_by_id) REFERENCES mdm.product_document(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_doc_product ON mdm.product_document (product_id, COALESCE(document_type_cd, ''), is_current);
CREATE INDEX IF NOT EXISTS idx_prd_doc_tenant  ON mdm.product_document (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'product_platform','product_distribution','product_registration',
        'product_eligibility','product_target_market','product_document'
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

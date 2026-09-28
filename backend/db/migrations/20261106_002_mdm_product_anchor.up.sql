-- 20261106_002_mdm_product_anchor.up.sql

-- ── product (anchor) ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    marketing_name varchar(500),
    legal_name varchar(500),
    short_name varchar(150),
    product_type_id uuid NOT NULL,
    product_sub_type_cd varchar(50),
    product_category_id uuid,
    status_id uuid NOT NULL,
    strategy_cd varchar(50),
    product_family_cd varchar(50),
    manager_id uuid,
    sub_advisor_id uuid,
    inception_date date,
    live_date date,
    closed_to_new_date date,
    closed_date date,
    liquidation_date date,
    merger_date date,
    successor_product_id uuid,
    predecessor_product_id uuid,
    base_currency varchar(3),
    domicile varchar(2),
    is_registered bool DEFAULT false NOT NULL,
    is_40_act bool DEFAULT false NOT NULL,
    is_ucits bool DEFAULT false NOT NULL,
    is_aifmd bool DEFAULT false NOT NULL,
    is_hedge_fund bool DEFAULT false NOT NULL,
    is_esg bool DEFAULT false NOT NULL,
    is_sfdr_article_8 bool DEFAULT false NOT NULL,
    is_sfdr_article_9 bool DEFAULT false NOT NULL,
    is_sharia_compliant bool DEFAULT false NOT NULL,
    is_index bool DEFAULT false NOT NULL,
    is_active bool DEFAULT false NOT NULL,
    is_open_to_new bool DEFAULT false NOT NULL,
    risk_profile varchar(20),
    investment_objective text,
    target_return text,
    target_volatility numeric(12,8),
    benchmark_id uuid,
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_cd UNIQUE (tenant_id, product_cd),
    CONSTRAINT fk_prd_type      FOREIGN KEY (product_type_id)       REFERENCES mdm.product_type(id),
    CONSTRAINT fk_prd_category  FOREIGN KEY (product_category_id)   REFERENCES mdm.product_category(id),
    CONSTRAINT fk_prd_status    FOREIGN KEY (status_id)             REFERENCES mdm.product_status(id),
    CONSTRAINT fk_prd_manager   FOREIGN KEY (manager_id)            REFERENCES edm.issuer_master(id),
    CONSTRAINT fk_prd_subadv    FOREIGN KEY (sub_advisor_id)        REFERENCES edm.issuer_master(id),
    CONSTRAINT fk_prd_successor FOREIGN KEY (successor_product_id)  REFERENCES mdm.product(id),
    CONSTRAINT fk_prd_predecess FOREIGN KEY (predecessor_product_id) REFERENCES mdm.product(id),
    CONSTRAINT fk_prd_merged    FOREIGN KEY (merged_into_id)        REFERENCES mdm.product(id),
    CONSTRAINT fk_prd_source    FOREIGN KEY (source_system_id)      REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_type      ON mdm.product (product_type_id);
CREATE INDEX IF NOT EXISTS idx_prd_category  ON mdm.product (product_category_id);
CREATE INDEX IF NOT EXISTS idx_prd_status    ON mdm.product (status_id);
CREATE INDEX IF NOT EXISTS idx_prd_family    ON mdm.product (product_family_cd);
CREATE INDEX IF NOT EXISTS idx_prd_manager   ON mdm.product (manager_id);
CREATE INDEX IF NOT EXISTS idx_prd_benchmark ON mdm.product (benchmark_id);
CREATE INDEX IF NOT EXISTS idx_prd_tenant    ON mdm.product (tenant_id);

-- Guarded FK to mdm.benchmark_master
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='benchmark_master') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='product' AND c.conname='fk_prd_benchmark'
        ) THEN
            ALTER TABLE mdm.product
                ADD CONSTRAINT fk_prd_benchmark
                FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id);
        END IF;
    END IF;
END $fk$;

-- ── product_identifier ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(100) NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_ident_type CHECK (id_type IN (
        'CUSIP','ISIN','TICKER','RIC','BBG_TICKER','BLOOMBERG_ID',
        'MORNINGSTAR_ID','LIPPER_ID','EVESTMENT_ID','SEC_FILE_NUMBER',
        'SEC_CIK','LEI','CRD','NATIONAL_ID','INTERNAL','CUSIP_144A',
        'CINS','SEDOL','WKN','VALOREN','PROVIDER_CODE')),
    CONSTRAINT fk_prd_ident FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_ident       ON mdm.product_identifier (product_id);
CREATE INDEX IF NOT EXISTS idx_prd_ident_look  ON mdm.product_identifier (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_prd_ident_tnt   ON mdm.product_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_ident_active
    ON mdm.product_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── product_share_class ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_share_class (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    share_class_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    class_type_cd varchar(30),
    security_id uuid,
    isin varchar(15),
    cusip varchar(9),
    ticker varchar(26),
    distribution_type varchar(20),
    load_type varchar(30),
    load_amount numeric(12,4),
    is_hedged bool DEFAULT false NOT NULL,
    hedging_currency varchar(3),
    eligibility_restriction text,
    minimum_initial numeric(18,4),
    minimum_subsequent numeric(18,4),
    minimum_balance numeric(18,4),
    currency varchar(3),
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_share_class_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_sc UNIQUE (tenant_id, product_id, share_class_cd, effective_from),
    CONSTRAINT chk_prd_sc_dist CHECK (distribution_type IS NULL OR distribution_type IN (
        'ACC','DIST','MIXED')),
    CONSTRAINT chk_prd_sc_load CHECK (load_type IS NULL OR load_type IN (
        'FRONT_LOAD','BACK_LOAD','LEVEL_LOAD','NO_LOAD')),
    CONSTRAINT fk_prd_sc_prd FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_sc_prd    ON mdm.product_share_class (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_sc_sec    ON mdm.product_share_class (security_id);
CREATE INDEX IF NOT EXISTS idx_prd_sc_tenant ON mdm.product_share_class (tenant_id);

-- Guarded FK to orm.security
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='orm' AND table_name='security') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='product_share_class' AND c.conname='fk_prd_sc_security'
        ) THEN
            ALTER TABLE mdm.product_share_class
                ADD CONSTRAINT fk_prd_sc_security
                FOREIGN KEY (security_id) REFERENCES orm.security(id);
        END IF;
    END IF;
END $fk$;

-- ── product_fee_schedule ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_fee_schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    schedule_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    product_id uuid,
    fee_type varchar(30) NOT NULL,
    calculation_method varchar(30) NOT NULL,
    rate_bps numeric(10,4),
    flat_amount numeric(18,4),
    currency varchar(3),
    frequency varchar(20),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_fee_schedule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_fee_sched_cd UNIQUE (tenant_id, schedule_cd),
    CONSTRAINT chk_prd_fs_type CHECK (fee_type IN (
        'MANAGEMENT','ADMIN','DISTRIBUTION_12B1','SHAREHOLDER_SERVICING',
        'PERFORMANCE','TRAILER','PLATFORM','WRAP','REDEMPTION','PURCHASE',
        'EXCHANGE','ACCOUNT','CUSTODY','ADVISORY')),
    CONSTRAINT chk_prd_fs_method CHECK (calculation_method IN (
        'BPS','FLAT','TIERED','PER_TRANSACTION','PER_ACCOUNT')),
    CONSTRAINT fk_prd_fs_product FOREIGN KEY (product_id) REFERENCES mdm.product(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_fs_product ON mdm.product_fee_schedule (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_fs_tenant  ON mdm.product_fee_schedule (tenant_id);

-- ── product_share_class_fee ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_share_class_fee (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_share_class_id uuid NOT NULL,
    fee_type varchar(30) NOT NULL,
    fee_basis varchar(30) NOT NULL,
    rate_bps numeric(10,4),
    flat_amount numeric(18,4),
    min_amount numeric(18,4),
    max_amount numeric(18,4),
    tier_breakpoints jsonb,
    currency varchar(3),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    waiver_available bool DEFAULT false NOT NULL,
    waiver_terms text,
    cap_amount numeric(18,4),
    expense_cap_date date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_share_class_fee_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prd_scf_sc FOREIGN KEY (product_share_class_id) REFERENCES mdm.product_share_class(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_scf_sc     ON mdm.product_share_class_fee (product_share_class_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_scf_tenant ON mdm.product_share_class_fee (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_fee_sc
    ON mdm.product_share_class_fee (tenant_id, product_share_class_id, fee_type, effective_from);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'product','product_identifier','product_share_class',
        'product_fee_schedule','product_share_class_fee'
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

-- 20261106_001_mdm_product_reference.up.sql

-- ── product_type ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    is_registered bool DEFAULT false NOT NULL,
    is_security bool DEFAULT false NOT NULL,
    requires_vehicle bool DEFAULT true NOT NULL,
    requires_security_class bool DEFAULT false NOT NULL,
    has_share_classes bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_type_cd UNIQUE (tenant_id, type_cd),
    CONSTRAINT chk_prd_type_cat CHECK (category IN (
        'REGISTERED_FUND','UNREGISTERED_FUND','MANAGED_ACCOUNT','INSURANCE',
        'STRUCTURED','ALTERNATIVE','ADVISORY','DEPOSIT'))
);
CREATE INDEX IF NOT EXISTS idx_prd_type_cat    ON mdm.product_type (category);
CREATE INDEX IF NOT EXISTS idx_prd_type_tenant ON mdm.product_type (tenant_id);

-- ── product_sub_type ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_sub_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sub_type_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    parent_type_cd varchar(30) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_sub_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_sub_type_cd UNIQUE (tenant_id, sub_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_sub_type_tenant ON mdm.product_sub_type (tenant_id);

-- ── product_status ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_live bool DEFAULT false NOT NULL,
    is_open_to_new bool DEFAULT false NOT NULL,
    is_terminal bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_status_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_status_cd UNIQUE (tenant_id, status_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_status_tenant ON mdm.product_status (tenant_id);

-- ── product_category ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_category (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    parent_category_id uuid,
    classification_scheme varchar(30),
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_category_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdc_parent FOREIGN KEY (parent_category_id) REFERENCES mdm.product_category(id)
);
-- COALESCE is not allowed inside a table CONSTRAINT UNIQUE — use a unique index (Batch 8 lesson)
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_category_cd
    ON mdm.product_category (tenant_id, category_cd, COALESCE(classification_scheme, ''));
CREATE INDEX IF NOT EXISTS idx_prd_category_tenant ON mdm.product_category (tenant_id);

-- ── product_distribution_channel ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_distribution_channel (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    channel_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    channel_type varchar(30) NOT NULL,
    requires_agreement bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_distribution_channel_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_dist_channel_cd UNIQUE (tenant_id, channel_cd),
    CONSTRAINT chk_prd_dc_type CHECK (channel_type IN (
        'DIRECT','INTERMEDIARY','PLATFORM','INSTITUTIONAL','ADVISOR',
        'RIA','BROKER_DEALER','BANK','INSURANCE','WRAP','RETIREMENT'))
);
CREATE INDEX IF NOT EXISTS idx_prd_dc_tenant ON mdm.product_distribution_channel (tenant_id);

-- ── product_registration_type ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_registration_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reg_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    requires_regulator boolean DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_registration_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_reg_type_cd UNIQUE (tenant_id, reg_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_rt_tenant ON mdm.product_registration_type (tenant_id);

-- ── product_document_type ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_document_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    doc_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    is_mandatory bool DEFAULT false NOT NULL,
    is_public bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_document_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_doc_type_cd UNIQUE (tenant_id, doc_type_cd),
    CONSTRAINT chk_prd_dt_cat CHECK (category IN (
        'REGULATORY','MARKETING','LEGAL','PERFORMANCE','COMPLIANCE','AGREEMENT'))
);
CREATE INDEX IF NOT EXISTS idx_prd_dt_tenant ON mdm.product_document_type (tenant_id);

-- ── product_share_class_type ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_share_class_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    class_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    has_distribution bool DEFAULT false NOT NULL,
    has_load bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_share_class_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_sc_type_cd UNIQUE (tenant_id, class_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_sct_tenant ON mdm.product_share_class_type (tenant_id);

-- ── product_target_market_type ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_target_market_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tm_type_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_positive bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_target_market_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_tm_type_cd UNIQUE (tenant_id, tm_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_tmt_tenant ON mdm.product_target_market_type (tenant_id);

-- ── product_lifecycle_event_type ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_lifecycle_event_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_final bool DEFAULT false NOT NULL,
    requires_approval bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_lifecycle_event_type_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_lc_event_cd UNIQUE (tenant_id, event_cd)
);
CREATE INDEX IF NOT EXISTS idx_prd_lcet_tenant ON mdm.product_lifecycle_event_type (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'product_type','product_sub_type','product_status','product_category',
        'product_distribution_channel','product_registration_type',
        'product_document_type','product_share_class_type',
        'product_target_market_type','product_lifecycle_event_type'
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

-- 20261106_005_mdm_product_golden.up.sql

-- ── product_type_mapping ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    internal_type_cd varchar(30) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdtm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdtm_source ON mdm.product_type_mapping (mdm_source_system_id);
CREATE INDEX IF NOT EXISTS idx_prdtm_tenant ON mdm.product_type_mapping (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_tm_map
    ON mdm.product_type_mapping (tenant_id, mdm_source_system_id, vendor_type_cd,
                                   COALESCE(vendor_sub_type_cd, ''::varchar));

-- ── product_source_priority ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_type_cd varchar(30),
    asset_class_cd varchar(20),
    field_group varchar(50) NOT NULL,
    source_system_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdsp_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdsp_type   ON mdm.product_source_priority (product_type_cd, field_group);
CREATE INDEX IF NOT EXISTS idx_prdsp_tenant ON mdm.product_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_src_pri
    ON mdm.product_source_priority (tenant_id, COALESCE(product_type_cd, ''),
                                      COALESCE(asset_class_cd, ''), field_group, source_system_id);

-- ── product_field_mapping ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    product_type_cd varchar(30),
    vendor_field varchar(150) NOT NULL,
    internal_table varchar(100) NOT NULL,
    internal_field varchar(150) NOT NULL,
    transform_expression text,
    is_required bool DEFAULT false NOT NULL,
    default_value text,
    valid_from date DEFAULT CURRENT_DATE NOT NULL,
    valid_to date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdfm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdfm_source ON mdm.product_field_mapping (mdm_source_system_id, product_type_cd);
CREATE INDEX IF NOT EXISTS idx_prdfm_target ON mdm.product_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_prdfm_tenant ON mdm.product_field_mapping (tenant_id);

-- ── product_survivorship_rule ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_type_cd varchar(30) NOT NULL,
    field_group varchar(50) NOT NULL,
    field_name varchar(150) NOT NULL,
    strategy varchar(30) NOT NULL,
    source_priority jsonb,
    min_confidence numeric(5,2),
    provider_is_authoritative bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_survivorship_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_surv_rule UNIQUE (tenant_id, product_type_cd, field_group, field_name)
);
CREATE INDEX IF NOT EXISTS idx_prdsr_tenant ON mdm.product_survivorship_rule (tenant_id);

-- ── product_survivorship_log ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    field_name varchar(150) NOT NULL,
    winning_source_id uuid,
    winning_value text,
    competing_values jsonb DEFAULT '[]'::jsonb NOT NULL,
    rule_id uuid,
    decision_reason varchar(255),
    decided_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    decided_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdsl_product FOREIGN KEY (product_id) REFERENCES mdm.product(id),
    CONSTRAINT fk_prdsl_rule    FOREIGN KEY (rule_id)    REFERENCES mdm.product_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_prdsl_product ON mdm.product_survivorship_log (product_id, golden_version);
CREATE INDEX IF NOT EXISTS idx_prdsl_tenant  ON mdm.product_survivorship_log (tenant_id);

-- ── product_match_rule ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    product_type_cd varchar(30) NOT NULL,
    rule_name varchar(250) NOT NULL,
    match_keys jsonb NOT NULL,
    deterministic_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    fuzzy_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    threshold_auto_match numeric(5,2) NOT NULL,
    threshold_review numeric(5,2) NOT NULL,
    threshold_no_match numeric(5,2),
    priority int4 DEFAULT 100 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_match_rule UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_prdmr_tenant ON mdm.product_match_rule (tenant_id);

-- ── product_match_candidate ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    product_id_a uuid NOT NULL,
    product_id_b uuid NOT NULL,
    overall_score numeric(5,2) NOT NULL,
    deterministic_match bool DEFAULT false NOT NULL,
    matched_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    conflicting_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamptz,
    review_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prdmc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_prdmc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.product_match_rule(id),
    CONSTRAINT fk_prdmc_a    FOREIGN KEY (product_id_a)  REFERENCES mdm.product(id),
    CONSTRAINT fk_prdmc_b    FOREIGN KEY (product_id_b)  REFERENCES mdm.product(id)
);
CREATE INDEX IF NOT EXISTS idx_prdmc_status ON mdm.product_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_prdmc_tenant ON mdm.product_match_candidate (tenant_id);

-- ── product_merge_log ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_product_id uuid NOT NULL,
    merged_product_id uuid NOT NULL,
    merge_type varchar(20) NOT NULL,
    match_candidate_id uuid,
    merge_reason varchar(255),
    merged_by uuid,
    merged_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reversible bool DEFAULT true NOT NULL,
    reversal_data jsonb,
    downstream_notified bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdml_surv   FOREIGN KEY (surviving_product_id) REFERENCES mdm.product(id),
    CONSTRAINT fk_prdml_merged FOREIGN KEY (merged_product_id)   REFERENCES mdm.product(id)
);
CREATE INDEX IF NOT EXISTS idx_prdml_surv   ON mdm.product_merge_log (surviving_product_id);
CREATE INDEX IF NOT EXISTS idx_prdml_merged ON mdm.product_merge_log (merged_product_id);
CREATE INDEX IF NOT EXISTS idx_prdml_tenant ON mdm.product_merge_log (tenant_id);

-- ── product_golden_record ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    product_type_cd varchar(30) NOT NULL,
    effective_date date NOT NULL,
    knowledge_timestamp timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    golden_attributes jsonb NOT NULL,
    winning_sources jsonb NOT NULL,
    overall_dq_score numeric(5,2),
    identity_confidence numeric(5,2),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    published_at timestamptz,
    published_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_gr_version UNIQUE (tenant_id, product_id, golden_version),
    CONSTRAINT chk_prd_gr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_prd_gr_product FOREIGN KEY (product_id) REFERENCES mdm.product(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_gr_product ON mdm.product_golden_record (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_gr_tenant  ON mdm.product_golden_record (tenant_id);

-- ── product_golden_field ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_golden_field (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    golden_record_id uuid NOT NULL,
    field_name varchar(150) NOT NULL,
    field_value text,
    field_value_numeric numeric(24,6),
    field_value_date date,
    field_value_json jsonb,
    source_system_id uuid,
    source_field varchar(150),
    confidence numeric(5,2),
    rule_applied uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.product_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_prdgf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdgf_record ON mdm.product_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_prdgf_field  ON mdm.product_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_prdgf_tenant ON mdm.product_golden_field (tenant_id);

-- ── product_exception ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid,
    product_type_cd varchar(30),
    identifier_value varchar(100),
    exception_type varchar(50) NOT NULL,
    severity varchar(10) NOT NULL,
    exception_description text NOT NULL,
    source_system_id uuid,
    detected_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    assigned_to uuid,
    resolved_at timestamptz,
    resolution_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prdex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_prdex_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_prdex_product FOREIGN KEY (product_id) REFERENCES mdm.product(id),
    CONSTRAINT fk_prdex_source  FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdex_status  ON mdm.product_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_prdex_product ON mdm.product_exception (product_id);
CREATE INDEX IF NOT EXISTS idx_prdex_tenant  ON mdm.product_exception (tenant_id);

-- ── product_steward ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    product_type_cd varchar(30),
    region varchar(50),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_merge bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_steward_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_prdst_user   ON mdm.product_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_prdst_tenant ON mdm.product_steward (tenant_id);

-- ── product_change_request ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    product_id uuid,
    product_type_cd varchar(30),
    change_type varchar(30) NOT NULL,
    requested_changes jsonb NOT NULL,
    request_reason text,
    source varchar(30),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    requested_by uuid NOT NULL,
    assigned_to uuid,
    approved_by uuid,
    approved_at timestamptz,
    applied_at timestamptz,
    rejection_reason text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_change_ref UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_prdcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_prdcr_product FOREIGN KEY (product_id) REFERENCES mdm.product(id)
);
CREATE INDEX IF NOT EXISTS idx_prdcr_status  ON mdm.product_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_prdcr_product ON mdm.product_change_request (product_id);
CREATE INDEX IF NOT EXISTS idx_prdcr_tenant  ON mdm.product_change_request (tenant_id);

-- ── product_feed_schedule ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_feed_schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    product_type_cd varchar(30),
    feed_name varchar(100) NOT NULL,
    feed_type varchar(20) NOT NULL,
    expected_cadence varchar(50),
    expected_delivery_time time,
    delivery_timezone varchar(50),
    sla_minutes int4,
    grace_period_minutes int4 DEFAULT 15,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_feed_schedule_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdfs_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdfs_tenant ON mdm.product_feed_schedule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_fs
    ON mdm.product_feed_schedule (tenant_id, source_system_id, feed_name);

-- ── product_feed_health ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_feed_health (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    feed_name varchar(100) NOT NULL,
    run_date date NOT NULL,
    delivered_at timestamptz,
    expected_at timestamptz,
    sla_met bool,
    records_expected int4,
    records_received int4,
    records_rejected int4,
    records_promoted int4,
    status varchar(20) NOT NULL,
    error_summary text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_feed_health_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prdfh_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_prdfh_status ON mdm.product_feed_health (status, run_date);
CREATE INDEX IF NOT EXISTS idx_prdfh_tenant ON mdm.product_feed_health (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prd_fh
    ON mdm.product_feed_health (tenant_id, source_system_id, feed_name, run_date);

-- ── product_dq_rule ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_dq_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    product_type_cd varchar(30),
    rule_type varchar(30) NOT NULL,
    rule_expression text,
    severity varchar(10) NOT NULL,
    weight numeric(5,2) DEFAULT 1.0 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_dq_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_prd_dq UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_prddq_tenant ON mdm.product_dq_rule (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'product_type_mapping','product_source_priority','product_field_mapping',
        'product_survivorship_rule','product_survivorship_log',
        'product_match_rule','product_match_candidate','product_merge_log',
        'product_golden_record','product_golden_field','product_exception',
        'product_steward','product_change_request','product_feed_schedule',
        'product_feed_health','product_dq_rule'
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

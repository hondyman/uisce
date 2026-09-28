-- 20261101_003_mdm_benchmark_golden.up.sql

-- ── benchmark_type_mapping ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    internal_type_id uuid NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_btm_source FOREIGN KEY (source_system_id)  REFERENCES mdm.source_systems(id),
    CONSTRAINT fk_btm_type   FOREIGN KEY (internal_type_id)  REFERENCES mdm.benchmark_type(id)
);
CREATE INDEX IF NOT EXISTS idx_btm_source ON mdm.benchmark_type_mapping (source_system_id);
CREATE INDEX IF NOT EXISTS idx_btm_tenant ON mdm.benchmark_type_mapping (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_btm_key
    ON mdm.benchmark_type_mapping (tenant_id, source_system_id, vendor_type_cd,
                                    COALESCE(vendor_sub_type_cd, ''::varchar));

-- ── benchmark_source_priority ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_type_category varchar(30) NOT NULL,
    asset_class_cd varchar(20),
    field_group varchar(50) NOT NULL,
    source_system_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bsp_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bsp_cat    ON mdm.benchmark_source_priority (benchmark_type_category, field_group);
CREATE INDEX IF NOT EXISTS idx_bsp_tenant ON mdm.benchmark_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bsp_key
    ON mdm.benchmark_source_priority (tenant_id, benchmark_type_category,
                                       COALESCE(asset_class_cd, ''::varchar), field_group, source_system_id);

-- ── benchmark_field_mapping ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    benchmark_type_category varchar(30),
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
    CONSTRAINT benchmark_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bfm_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bfm_source ON mdm.benchmark_field_mapping (source_system_id, benchmark_type_category);
CREATE INDEX IF NOT EXISTS idx_bfm_target ON mdm.benchmark_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_bfm_tenant ON mdm.benchmark_field_mapping (tenant_id);

-- ── benchmark_survivorship_rule ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_type_category varchar(30) NOT NULL,
    asset_class_cd varchar(20),
    field_group varchar(50) NOT NULL,
    field_name varchar(150) NOT NULL,
    strategy varchar(30) NOT NULL,
    source_priority jsonb,
    min_confidence numeric(5,2),
    max_staleness_hours int4,
    provider_is_authoritative bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_survivorship_rule_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bsr_strategy CHECK (strategy IN (
        'SOURCE_PRIORITY','MOST_RECENT','MOST_COMPLETE','LONGEST',
        'HIGHEST_CONFIDENCE','MANUAL','PROVIDER_AUTHORITATIVE'))
);
CREATE INDEX IF NOT EXISTS idx_bsr_cat    ON mdm.benchmark_survivorship_rule (benchmark_type_category, field_group);
CREATE INDEX IF NOT EXISTS idx_bsr_tenant ON mdm.benchmark_survivorship_rule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bsr_key
    ON mdm.benchmark_survivorship_rule (tenant_id, benchmark_type_category,
                                         COALESCE(asset_class_cd, ''::varchar), field_group, field_name);

-- ── benchmark_survivorship_log ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_id uuid NOT NULL,
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
    CONSTRAINT benchmark_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bsl_benchmark FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id),
    CONSTRAINT fk_bsl_rule      FOREIGN KEY (rule_id)      REFERENCES mdm.benchmark_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_bsl_benchmark ON mdm.benchmark_survivorship_log (benchmark_id, golden_version);
CREATE INDEX IF NOT EXISTS idx_bsl_field     ON mdm.benchmark_survivorship_log (field_name, decided_at);
CREATE INDEX IF NOT EXISTS idx_bsl_tenant    ON mdm.benchmark_survivorship_log (tenant_id);

-- ── benchmark_match_rule ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    benchmark_type_category varchar(30) NOT NULL,
    rule_name varchar(150) NOT NULL,
    match_keys jsonb NOT NULL,
    deterministic_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    fuzzy_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    threshold_auto_match numeric(5,2) NOT NULL,
    threshold_review numeric(5,2) NOT NULL,
    threshold_no_match numeric(5,2) NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_match_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_bmr_tenant ON mdm.benchmark_match_rule (tenant_id);

-- ── benchmark_match_candidate ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    benchmark_id_a uuid NOT NULL,
    benchmark_id_b uuid NOT NULL,
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
    CONSTRAINT benchmark_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bmc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_bmc_rule FOREIGN KEY (match_rule_id)  REFERENCES mdm.benchmark_match_rule(id),
    CONSTRAINT fk_bmc_a    FOREIGN KEY (benchmark_id_a) REFERENCES mdm.benchmark_master(id),
    CONSTRAINT fk_bmc_b    FOREIGN KEY (benchmark_id_b) REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_bmc_status ON mdm.benchmark_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_bmc_tenant ON mdm.benchmark_match_candidate (tenant_id);

-- ── benchmark_merge_log ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_benchmark_id uuid NOT NULL,
    merged_benchmark_id uuid NOT NULL,
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
    CONSTRAINT benchmark_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bml_surv   FOREIGN KEY (surviving_benchmark_id) REFERENCES mdm.benchmark_master(id),
    CONSTRAINT fk_bml_merged FOREIGN KEY (merged_benchmark_id)    REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_bml_surv   ON mdm.benchmark_merge_log (surviving_benchmark_id);
CREATE INDEX IF NOT EXISTS idx_bml_merged ON mdm.benchmark_merge_log (merged_benchmark_id);
CREATE INDEX IF NOT EXISTS idx_bml_tenant ON mdm.benchmark_merge_log (tenant_id);

-- ── benchmark_golden_record ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    benchmark_type_category varchar(30) NOT NULL,
    asset_class_cd varchar(20),
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
    CONSTRAINT benchmark_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_bgr_version UNIQUE (tenant_id, benchmark_id, golden_version),
    CONSTRAINT chk_bgr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_bgr_benchmark FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_bgr_benchmark ON mdm.benchmark_golden_record (benchmark_id, is_current);
CREATE INDEX IF NOT EXISTS idx_bgr_tenant    ON mdm.benchmark_golden_record (tenant_id);

-- ── benchmark_golden_field ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_golden_field (
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
    CONSTRAINT benchmark_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.benchmark_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_bgf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bgf_record ON mdm.benchmark_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_bgf_field  ON mdm.benchmark_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_bgf_tenant ON mdm.benchmark_golden_field (tenant_id);

-- ── benchmark_golden_publication ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_golden_publication (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    publication_ref varchar(50) NOT NULL,
    publication_type varchar(20) NOT NULL,
    record_count int4 NOT NULL,
    published_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    published_by uuid,
    publication_status varchar(20) DEFAULT 'PUBLISHED' NOT NULL,
    error_summary text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_golden_publication_pkey PRIMARY KEY (id),
    CONSTRAINT uq_bgp_ref UNIQUE (tenant_id, publication_ref)
);
CREATE INDEX IF NOT EXISTS idx_bgp_tenant ON mdm.benchmark_golden_publication (tenant_id);

-- ── benchmark_golden_distribution ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_golden_distribution (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    publication_id uuid NOT NULL,
    golden_record_id uuid NOT NULL,
    target_system_id uuid NOT NULL,
    distributed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    delivery_status varchar(20) NOT NULL,
    ack_at timestamptz,
    retry_count int4 DEFAULT 0,
    error_message text,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_golden_distribution_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bgd_pub    FOREIGN KEY (publication_id)   REFERENCES mdm.benchmark_golden_publication(id),
    CONSTRAINT fk_bgd_record FOREIGN KEY (golden_record_id) REFERENCES mdm.benchmark_golden_record(id),
    CONSTRAINT fk_bgd_target FOREIGN KEY (target_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bgd_target ON mdm.benchmark_golden_distribution (target_system_id, delivery_status);
CREATE INDEX IF NOT EXISTS idx_bgd_tenant ON mdm.benchmark_golden_distribution (tenant_id);

-- ── benchmark_reconciliation ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_reconciliation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_date date NOT NULL,
    benchmark_type_category varchar(30),
    source_system_id_a uuid NOT NULL,
    source_system_id_b uuid NOT NULL,
    records_compared int4 NOT NULL,
    records_matched int4 NOT NULL,
    records_only_in_a int4 NOT NULL,
    records_only_in_b int4 NOT NULL,
    records_field_conflicts int4 NOT NULL,
    constituent_conflicts int4 DEFAULT 0,
    match_pct numeric(5,2),
    status varchar(20) DEFAULT 'COMPLETED' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_reconciliation_pkey PRIMARY KEY (id),
    CONSTRAINT fk_brec_src_a FOREIGN KEY (source_system_id_a) REFERENCES mdm.source_systems(id),
    CONSTRAINT fk_brec_src_b FOREIGN KEY (source_system_id_b) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_brec_date   ON mdm.benchmark_reconciliation (reconciliation_date, benchmark_type_category);
CREATE INDEX IF NOT EXISTS idx_brec_tenant ON mdm.benchmark_reconciliation (tenant_id);

-- ── benchmark_reconciliation_result ────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_reconciliation_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_id uuid NOT NULL,
    benchmark_id uuid,
    identifier_value varchar(100),
    field_name varchar(150) NOT NULL,
    value_a text,
    value_b text,
    values_match bool NOT NULL,
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolved_by uuid,
    resolved_at timestamptz,
    resolution_note text,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_reconciliation_result_pkey PRIMARY KEY (id),
    CONSTRAINT fk_brr_rec FOREIGN KEY (reconciliation_id) REFERENCES mdm.benchmark_reconciliation(id) ON DELETE CASCADE,
    CONSTRAINT fk_brr_bm  FOREIGN KEY (benchmark_id)      REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_brr_rec    ON mdm.benchmark_reconciliation_result (reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_brr_status ON mdm.benchmark_reconciliation_result (status, severity);
CREATE INDEX IF NOT EXISTS idx_brr_tenant ON mdm.benchmark_reconciliation_result (tenant_id);

-- ── benchmark_exception ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_id uuid,
    identifier_value varchar(100),
    benchmark_type_category varchar(30),
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
    CONSTRAINT benchmark_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_bex_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_bex_bm     FOREIGN KEY (benchmark_id)     REFERENCES mdm.benchmark_master(id),
    CONSTRAINT fk_bex_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bex_status    ON mdm.benchmark_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_bex_benchmark ON mdm.benchmark_exception (benchmark_id);
CREATE INDEX IF NOT EXISTS idx_bex_tenant    ON mdm.benchmark_exception (tenant_id);

-- ── benchmark_steward ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    benchmark_type_category varchar(30),
    provider_id uuid,
    asset_class_cd varchar(20),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_merge bool DEFAULT false NOT NULL,
    can_assign_to_portfolio bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_steward_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bst_provider FOREIGN KEY (provider_id) REFERENCES mdm.benchmark_provider(id)
);
CREATE INDEX IF NOT EXISTS idx_bst_user     ON mdm.benchmark_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_bst_provider ON mdm.benchmark_steward (provider_id, steward_role);
CREATE INDEX IF NOT EXISTS idx_bst_tenant   ON mdm.benchmark_steward (tenant_id);

-- ── benchmark_change_request ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    benchmark_id uuid,
    benchmark_type_category varchar(30),
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
    CONSTRAINT benchmark_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_change_request_ref_key UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_bcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_bcr_bm FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_bcr_status ON mdm.benchmark_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_bcr_bm     ON mdm.benchmark_change_request (benchmark_id);
CREATE INDEX IF NOT EXISTS idx_bcr_tenant ON mdm.benchmark_change_request (tenant_id);

-- ── benchmark_feed_schedule ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_feed_schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    benchmark_type_category varchar(30),
    feed_name varchar(100) NOT NULL,
    feed_type varchar(20) NOT NULL,
    expected_cadence varchar(50),
    expected_delivery_time time,
    delivery_timezone varchar(50),
    sla_minutes int4,
    grace_period_minutes int4 DEFAULT 15,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_feed_schedule_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bfs_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bfs_tenant ON mdm.benchmark_feed_schedule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bfs_key
    ON mdm.benchmark_feed_schedule (tenant_id, source_system_id, feed_name);

-- ── benchmark_feed_health ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_feed_health (
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
    records_staged int4,
    records_promoted int4,
    status varchar(20) NOT NULL,
    error_summary text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_feed_health_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bfh_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_bfh_status ON mdm.benchmark_feed_health (status, run_date);
CREATE INDEX IF NOT EXISTS idx_bfh_tenant ON mdm.benchmark_feed_health (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bfh_key
    ON mdm.benchmark_feed_health (tenant_id, source_system_id, feed_name, run_date);

-- ── benchmark_dq_rule ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_dq_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    benchmark_type_category varchar(30),
    rule_type varchar(30) NOT NULL,
    rule_expression text,
    severity varchar(10) NOT NULL,
    weight numeric(5,2) DEFAULT 1.0 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_dq_rule_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_dq_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_bdq_tenant ON mdm.benchmark_dq_rule (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'benchmark_type_mapping','benchmark_source_priority',
        'benchmark_field_mapping','benchmark_survivorship_rule',
        'benchmark_survivorship_log','benchmark_match_rule',
        'benchmark_match_candidate','benchmark_merge_log',
        'benchmark_golden_record','benchmark_golden_field',
        'benchmark_golden_publication','benchmark_golden_distribution',
        'benchmark_reconciliation','benchmark_reconciliation_result',
        'benchmark_exception','benchmark_steward',
        'benchmark_change_request','benchmark_feed_schedule',
        'benchmark_feed_health','benchmark_dq_rule'
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

-- 20261103_005_mdm_price_golden.up.sql
-- Base tables are the anchor. If your implementation prefers golden views
-- over materialized tables (as Batch 6 did for ratings), the six
-- `price_golden_*` tables below can be replaced with views — but the
-- underlying `price`, `price_history`, `price_series` tables must exist.

-- ── price_type_mapping ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    internal_price_type_cd varchar(30) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ptm_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_ptm_source ON mdm.price_type_mapping (source_system_id);
CREATE INDEX IF NOT EXISTS idx_ptm_tenant ON mdm.price_type_mapping (tenant_id);

-- ── price_source_priority ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    asset_class_cd varchar(20),
    sec_sub_typ_cd varchar(30),
    price_type_cd varchar(30),
    currency varchar(3),
    source_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_fallback bool DEFAULT false NOT NULL,
    max_staleness_minutes int4,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_psp_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_psp_class  ON mdm.price_source_priority (price_entity_type, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_psp_tenant ON mdm.price_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_psp_key
    ON mdm.price_source_priority (tenant_id, price_entity_type,
                                   COALESCE(asset_class_cd, ''), COALESCE(sec_sub_typ_cd, ''),
                                   COALESCE(price_type_cd, ''), COALESCE(currency, ''), source_id);

-- ── price_field_mapping ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    asset_class_cd varchar(20),
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
    CONSTRAINT price_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pfm_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_pfm_source ON mdm.price_field_mapping (source_system_id, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_pfm_target ON mdm.price_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_pfm_tenant ON mdm.price_field_mapping (tenant_id);

-- ── price_survivorship_rule ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    asset_class_cd varchar(20),
    field_name varchar(150) NOT NULL,
    strategy varchar(30) NOT NULL,
    source_priority jsonb,
    min_confidence numeric(5,2),
    max_staleness_minutes int4,
    variance_tolerance_pct numeric(12,8),
    manual_override_allowed bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_survivorship_rule_pkey PRIMARY KEY (id),
    CONSTRAINT chk_psr_strategy CHECK (strategy IN (
        'SOURCE_PRIORITY','MOST_RECENT','MOST_CONSERVATIVE',
        'HIGHEST_QUALITY_TIER','MEDIAN','AVERAGE','MANUAL',
        'PROVIDER_OFFICIAL','EXCHANGE_AUTHORITATIVE'))
);
CREATE INDEX IF NOT EXISTS idx_psr_entity ON mdm.price_survivorship_rule (price_entity_type, field_name);
CREATE INDEX IF NOT EXISTS idx_psr_tenant ON mdm.price_survivorship_rule (tenant_id);

-- ── price_survivorship_log ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_date date NOT NULL,
    golden_version int4 NOT NULL,
    field_name varchar(150) NOT NULL,
    winning_source_id uuid,
    winning_value numeric(28,12),
    competing_values jsonb DEFAULT '[]'::jsonb NOT NULL,
    rule_id uuid,
    decision_reason varchar(255),
    decided_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    decided_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_psl_rule FOREIGN KEY (rule_id) REFERENCES mdm.price_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_psl_entity ON mdm.price_survivorship_log (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_psl_tenant ON mdm.price_survivorship_log (tenant_id);

-- ── price_match_rule ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    rule_name varchar(250) NOT NULL,
    match_keys jsonb NOT NULL,
    deterministic_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    fuzzy_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    threshold_auto_match numeric(5,2) NOT NULL,
    threshold_review numeric(5,2) NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT price_match_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_pmr_tenant ON mdm.price_match_rule (tenant_id);

-- ── price_match_candidate ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id_a uuid NOT NULL,
    price_entity_id_b uuid NOT NULL,
    overall_score numeric(5,2) NOT NULL,
    matched_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    conflicting_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamptz,
    review_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pmc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_pmc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.price_match_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_pmc_status ON mdm.price_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_pmc_tenant ON mdm.price_match_candidate (tenant_id);

-- ── price_merge_log ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    surviving_entity_id uuid NOT NULL,
    merged_entity_id uuid NOT NULL,
    merge_type varchar(20) NOT NULL,
    match_candidate_id uuid,
    merge_reason varchar(255),
    merged_by uuid,
    merged_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reversible bool DEFAULT true NOT NULL,
    reversal_data jsonb,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_merge_log_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_pmlg_surv   ON mdm.price_merge_log (surviving_entity_id);
CREATE INDEX IF NOT EXISTS idx_pmlg_merged ON mdm.price_merge_log (merged_entity_id);
CREATE INDEX IF NOT EXISTS idx_pmlg_tenant ON mdm.price_merge_log (tenant_id);

-- ── price_golden_record ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_date date NOT NULL,
    price_time timestamptz,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    price_type_cd varchar(30) NOT NULL,
    golden_value numeric(28,12) NOT NULL,
    currency varchar(3),
    quality_tier_id uuid,
    fair_value_level_id uuid,
    winning_source_id uuid,
    source_count int4 NOT NULL,
    variance_pct numeric(12,8),
    confidence numeric(5,2),
    dq_score numeric(5,2),
    is_stale bool DEFAULT false NOT NULL,
    is_official bool DEFAULT false NOT NULL,
    effective_date date NOT NULL,
    knowledge_timestamp timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    golden_attributes jsonb NOT NULL,
    winning_sources jsonb NOT NULL,
    status varchar(20) DEFAULT 'PUBLISHED' NOT NULL,
    published_at timestamptz,
    published_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_pgr_price_version UNIQUE (tenant_id, price_entity_type, price_entity_id, price_date,
                                             price_type_cd, golden_version),
    CONSTRAINT chk_pgr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED'))
);
CREATE INDEX IF NOT EXISTS idx_pgr_entity ON mdm.price_golden_record (price_entity_type, price_entity_id, price_date, is_current);
CREATE INDEX IF NOT EXISTS idx_pgr_date   ON mdm.price_golden_record (price_date, status);
CREATE INDEX IF NOT EXISTS idx_pgr_tenant ON mdm.price_golden_record (tenant_id);

-- ── price_golden_field ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_golden_field (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    golden_record_id uuid NOT NULL,
    source_id uuid,
    field_name varchar(150) NOT NULL,
    field_value numeric(28,12),
    field_value_text text,
    field_value_date date,
    field_value_json jsonb,
    confidence numeric(5,2),
    rule_applied uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.price_golden_record(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pgf_record ON mdm.price_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_pgf_field  ON mdm.price_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_pgf_tenant ON mdm.price_golden_field (tenant_id);

-- ── price_golden_publication ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_golden_publication (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    publication_ref varchar(50) NOT NULL,
    publication_date date NOT NULL,
    publication_type varchar(20) NOT NULL,
    record_count int4 NOT NULL,
    published_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    published_by uuid,
    publication_status varchar(20) DEFAULT 'PUBLISHED' NOT NULL,
    error_summary text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_golden_publication_pkey PRIMARY KEY (id),
    CONSTRAINT price_golden_publication_ref_key UNIQUE (tenant_id, publication_ref)
);
CREATE INDEX IF NOT EXISTS idx_pgp_tenant ON mdm.price_golden_publication (tenant_id);

-- ── price_golden_distribution ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_golden_distribution (
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
    CONSTRAINT price_golden_distribution_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pgd_pub    FOREIGN KEY (publication_id)   REFERENCES mdm.price_golden_publication(id),
    CONSTRAINT fk_pgd_record FOREIGN KEY (golden_record_id) REFERENCES mdm.price_golden_record(id),
    CONSTRAINT fk_pgd_target FOREIGN KEY (target_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_pgd_target ON mdm.price_golden_distribution (target_system_id, delivery_status);
CREATE INDEX IF NOT EXISTS idx_pgd_tenant ON mdm.price_golden_distribution (tenant_id);

-- ── price_reconciliation ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_reconciliation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_date date NOT NULL,
    price_entity_type varchar(30),
    asset_class_cd varchar(20),
    source_a_id uuid NOT NULL,
    source_b_id uuid NOT NULL,
    price_type_cd varchar(30),
    records_compared int4 NOT NULL,
    records_matched int4 NOT NULL,
    records_only_in_a int4 NOT NULL,
    records_only_in_b int4 NOT NULL,
    records_variance_breach int4 NOT NULL,
    mean_variance_pct numeric(12,8),
    max_variance_pct numeric(12,8),
    p95_variance_pct numeric(12,8),
    match_pct numeric(5,2),
    status varchar(20) DEFAULT 'COMPLETED' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_reconciliation_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prec_src_a FOREIGN KEY (source_a_id) REFERENCES mdm.price_source(id),
    CONSTRAINT fk_prec_src_b FOREIGN KEY (source_b_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_prec_date   ON mdm.price_reconciliation (reconciliation_date, price_entity_type, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_prec_tenant ON mdm.price_reconciliation (tenant_id);

-- ── price_reconciliation_result ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_reconciliation_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_id uuid NOT NULL,
    price_entity_type varchar(30),
    price_entity_id uuid,
    price_type_cd varchar(30),
    value_a numeric(28,12),
    value_b numeric(28,12),
    variance_pct numeric(12,8),
    variance_bps numeric(12,6),
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolved_by uuid,
    resolved_at timestamptz,
    resolution_note text,
    selected_source_id uuid,
    selected_value numeric(28,12),
    tenant_id uuid NOT NULL,
    CONSTRAINT price_reconciliation_result_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prr_rec FOREIGN KEY (reconciliation_id) REFERENCES mdm.price_reconciliation(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prr_rec    ON mdm.price_reconciliation_result (reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_prr_status ON mdm.price_reconciliation_result (status, severity);
CREATE INDEX IF NOT EXISTS idx_prr_tenant ON mdm.price_reconciliation_result (tenant_id);

-- ── price_exception ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30),
    price_entity_id uuid,
    price_date date,
    source_id uuid,
    exception_type varchar(50) NOT NULL,
    severity varchar(10) NOT NULL,
    exception_description text NOT NULL,
    detected_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    assigned_to uuid,
    resolved_at timestamptz,
    resolution_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_pex_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_pex_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pex_status ON mdm.price_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_pex_entity ON mdm.price_exception (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pex_tenant ON mdm.price_exception (tenant_id);

-- ── price_steward ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    asset_class_cd varchar(20),
    source_id uuid,
    region varchar(50),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_approve_challenge bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_steward_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pst_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pst_user   ON mdm.price_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_pst_tenant ON mdm.price_steward (tenant_id);

-- ── price_change_request ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    price_entity_type varchar(30),
    price_entity_id uuid,
    price_date date,
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
    CONSTRAINT price_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT price_change_request_ref_key UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_pcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK'))
);
CREATE INDEX IF NOT EXISTS idx_pcr_status ON mdm.price_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_pcr_entity ON mdm.price_change_request (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pcr_tenant ON mdm.price_change_request (tenant_id);

-- ── price_feed_schedule ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_feed_schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_id uuid NOT NULL,
    asset_class_cd varchar(20),
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
    CONSTRAINT price_feed_schedule_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pfs_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pfs_tenant ON mdm.price_feed_schedule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pfs_key
    ON mdm.price_feed_schedule (tenant_id, source_id, feed_name);

-- ── price_feed_health ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_feed_health (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_id uuid NOT NULL,
    feed_name varchar(100) NOT NULL,
    run_date date NOT NULL,
    delivered_at timestamptz,
    expected_at timestamptz,
    sla_met bool,
    records_expected int4,
    records_received int4,
    records_rejected int4,
    records_promoted int4,
    records_variance_breach int4,
    records_stale int4,
    status varchar(20) NOT NULL,
    error_summary text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_feed_health_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pfh_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pfh_status ON mdm.price_feed_health (status, run_date);
CREATE INDEX IF NOT EXISTS idx_pfh_tenant ON mdm.price_feed_health (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pfh_key
    ON mdm.price_feed_health (tenant_id, source_id, feed_name, run_date);

-- ── price_dq_rule ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_dq_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    price_entity_type varchar(30),
    rule_type varchar(30) NOT NULL,
    rule_expression text,
    severity varchar(10) NOT NULL,
    weight numeric(5,2) DEFAULT 1.0 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_dq_rule_pkey PRIMARY KEY (id),
    CONSTRAINT price_dq_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_pdq_tenant ON mdm.price_dq_rule (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'price_type_mapping','price_source_priority','price_field_mapping',
        'price_survivorship_rule','price_survivorship_log',
        'price_match_rule','price_match_candidate','price_merge_log',
        'price_golden_record','price_golden_field',
        'price_golden_publication','price_golden_distribution',
        'price_reconciliation','price_reconciliation_result',
        'price_exception','price_steward','price_change_request',
        'price_feed_schedule','price_feed_health','price_dq_rule'
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

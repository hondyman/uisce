-- 20261104_006_mdm_calendar_golden.up.sql

-- ── calendar_source_alias ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_source_alias (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_source_id uuid NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    alias_cd varchar(50) NOT NULL,
    alias_name varchar(250),
    is_primary bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_source_alias_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calsa_source FOREIGN KEY (calendar_source_id)   REFERENCES mdm.calendar_source(id),
    CONSTRAINT fk_calsa_mdm    FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_calsa_source ON mdm.calendar_source_alias (calendar_source_id);
CREATE INDEX IF NOT EXISTS idx_calsa_tenant ON mdm.calendar_source_alias (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_calsa_key
    ON mdm.calendar_source_alias (tenant_id, mdm_source_system_id, alias_cd);

-- ── calendar_type_mapping ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    internal_calendar_type_cd varchar(30) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_caltm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_caltm_source ON mdm.calendar_type_mapping (mdm_source_system_id);
CREATE INDEX IF NOT EXISTS idx_caltm_tenant ON mdm.calendar_type_mapping (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_caltm_key
    ON mdm.calendar_type_mapping (tenant_id, mdm_source_system_id, vendor_type_cd,
                                    COALESCE(vendor_sub_type_cd, ''::varchar));

-- ── calendar_source_priority ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_type_cd varchar(30),
    country_cd varchar(2),
    region_cd varchar(30),
    field_group varchar(50) NOT NULL,
    calendar_source_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_authoritative bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calsp_source FOREIGN KEY (calendar_source_id) REFERENCES mdm.calendar_source(id)
);
CREATE INDEX IF NOT EXISTS idx_calsp_type    ON mdm.calendar_source_priority (calendar_type_cd, field_group);
CREATE INDEX IF NOT EXISTS idx_calsp_country ON mdm.calendar_source_priority (country_cd);
CREATE INDEX IF NOT EXISTS idx_calsp_tenant  ON mdm.calendar_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_calsp_key
    ON mdm.calendar_source_priority (tenant_id, COALESCE(calendar_type_cd, ''),
                                       COALESCE(country_cd, ''), COALESCE(region_cd, ''),
                                       field_group, calendar_source_id);

-- ── calendar_field_mapping ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    calendar_type_cd varchar(30),
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
    CONSTRAINT calendar_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calfm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_calfm_source ON mdm.calendar_field_mapping (mdm_source_system_id, calendar_type_cd);
CREATE INDEX IF NOT EXISTS idx_calfm_target ON mdm.calendar_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_calfm_tenant ON mdm.calendar_field_mapping (tenant_id);

-- ── calendar_survivorship_rule ─────────────────────────────────────────
-- Note: COALESCE expressions are not allowed inside table CONSTRAINT UNIQUE;
-- expressed as a unique index instead.
CREATE TABLE IF NOT EXISTS mdm.calendar_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_type_cd varchar(30) NOT NULL,
    country_cd varchar(2),
    field_group varchar(50) NOT NULL,
    field_name varchar(150) NOT NULL,
    strategy varchar(30) NOT NULL,
    source_priority jsonb,
    min_confidence numeric(5,2),
    provider_is_authoritative bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_survivorship_rule_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_calsr_key
    ON mdm.calendar_survivorship_rule (tenant_id, calendar_type_cd,
                                         COALESCE(country_cd, ''), field_group, field_name);
CREATE INDEX IF NOT EXISTS idx_calsr_tenant ON mdm.calendar_survivorship_rule (tenant_id);

-- ── calendar_survivorship_log ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
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
    CONSTRAINT calendar_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calsl_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_calsl_rule     FOREIGN KEY (rule_id)     REFERENCES mdm.calendar_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_calsl_calendar ON mdm.calendar_survivorship_log (calendar_id, golden_version);
CREATE INDEX IF NOT EXISTS idx_calsl_tenant   ON mdm.calendar_survivorship_log (tenant_id);

-- ── calendar_match_rule ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    calendar_type_cd varchar(30) NOT NULL,
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
    CONSTRAINT calendar_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_calmr_cd UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_calmr_tenant ON mdm.calendar_match_rule (tenant_id);

-- ── calendar_match_candidate ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    calendar_id_a uuid NOT NULL,
    calendar_id_b uuid NOT NULL,
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
    CONSTRAINT calendar_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_calmc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_calmc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.calendar_match_rule(id),
    CONSTRAINT fk_calmc_a    FOREIGN KEY (calendar_id_a) REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_calmc_b    FOREIGN KEY (calendar_id_b) REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_calmc_status ON mdm.calendar_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_calmc_tenant ON mdm.calendar_match_candidate (tenant_id);

-- ── calendar_merge_log ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_calendar_id uuid NOT NULL,
    merged_calendar_id uuid NOT NULL,
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
    CONSTRAINT calendar_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calml_surv   FOREIGN KEY (surviving_calendar_id) REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_calml_merged FOREIGN KEY (merged_calendar_id)   REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_calml_surv   ON mdm.calendar_merge_log (surviving_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calml_merged ON mdm.calendar_merge_log (merged_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calml_tenant ON mdm.calendar_merge_log (tenant_id);

-- ── calendar_golden_record ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    calendar_type_cd varchar(30) NOT NULL,
    country_cd varchar(2),
    effective_date date NOT NULL,
    knowledge_timestamp timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    golden_attributes jsonb NOT NULL,
    winning_sources jsonb NOT NULL,
    overall_dq_score numeric(5,2),
    holiday_coverage_years int4,
    last_materialization_date date,
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    published_at timestamptz,
    published_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_calgr_version UNIQUE (tenant_id, calendar_id, golden_version),
    CONSTRAINT chk_calgr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_calgr_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_calgr_calendar ON mdm.calendar_golden_record (calendar_id, is_current);
CREATE INDEX IF NOT EXISTS idx_calgr_tenant   ON mdm.calendar_golden_record (tenant_id);

-- ── calendar_golden_field ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_golden_field (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    golden_record_id uuid NOT NULL,
    field_name varchar(150) NOT NULL,
    field_value text,
    field_value_numeric numeric(24,6),
    field_value_date date,
    field_value_json jsonb,
    calendar_source_id uuid,
    source_field varchar(150),
    confidence numeric(5,2),
    rule_applied uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calgf_record FOREIGN KEY (golden_record_id)   REFERENCES mdm.calendar_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_calgf_source FOREIGN KEY (calendar_source_id) REFERENCES mdm.calendar_source(id)
);
CREATE INDEX IF NOT EXISTS idx_calgf_record ON mdm.calendar_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_calgf_field  ON mdm.calendar_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_calgf_tenant ON mdm.calendar_golden_field (tenant_id);

-- ── calendar_reconciliation ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_reconciliation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_date date NOT NULL,
    calendar_type_cd varchar(30),
    country_cd varchar(2),
    calendar_source_id_a uuid NOT NULL,
    calendar_source_id_b uuid NOT NULL,
    year_compared int4,
    records_compared int4 NOT NULL,
    records_matched int4 NOT NULL,
    records_only_in_a int4 NOT NULL,
    records_only_in_b int4 NOT NULL,
    records_date_conflicts int4 NOT NULL,
    match_pct numeric(5,2),
    status varchar(20) DEFAULT 'COMPLETED' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_reconciliation_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calrec_src_a FOREIGN KEY (calendar_source_id_a) REFERENCES mdm.calendar_source(id),
    CONSTRAINT fk_calrec_src_b FOREIGN KEY (calendar_source_id_b) REFERENCES mdm.calendar_source(id)
);
CREATE INDEX IF NOT EXISTS idx_calrec_date   ON mdm.calendar_reconciliation (reconciliation_date, calendar_type_cd);
CREATE INDEX IF NOT EXISTS idx_calrec_tenant ON mdm.calendar_reconciliation (tenant_id);

-- ── calendar_exception ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid,
    conflict_date date,
    calendar_type_cd varchar(30),
    exception_type varchar(50) NOT NULL,
    severity varchar(10) NOT NULL,
    exception_description text NOT NULL,
    calendar_source_id uuid,
    detected_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    assigned_to uuid,
    resolved_at timestamptz,
    resolution_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_calex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_calex_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_calex_calendar FOREIGN KEY (calendar_id)        REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_calex_source   FOREIGN KEY (calendar_source_id) REFERENCES mdm.calendar_source(id)
);
CREATE INDEX IF NOT EXISTS idx_calex_status   ON mdm.calendar_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_calex_calendar ON mdm.calendar_exception (calendar_id);
CREATE INDEX IF NOT EXISTS idx_calex_tenant   ON mdm.calendar_exception (tenant_id);

-- ── calendar_steward ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    calendar_type_cd varchar(30),
    country_cd varchar(2),
    region_cd varchar(30),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_edit_holidays bool DEFAULT false NOT NULL,
    can_add_ad_hoc_events bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_steward_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_calst_user   ON mdm.calendar_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_calst_tenant ON mdm.calendar_steward (tenant_id);

-- ── calendar_change_request ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    calendar_id uuid,
    calendar_type_cd varchar(30),
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
    CONSTRAINT calendar_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT uq_calcr_ref UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_calcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_calcr_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id)
);
CREATE INDEX IF NOT EXISTS idx_calcr_status   ON mdm.calendar_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_calcr_calendar ON mdm.calendar_change_request (calendar_id);
CREATE INDEX IF NOT EXISTS idx_calcr_tenant   ON mdm.calendar_change_request (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_source_alias','calendar_type_mapping','calendar_source_priority',
        'calendar_field_mapping','calendar_survivorship_rule','calendar_survivorship_log',
        'calendar_match_rule','calendar_match_candidate','calendar_merge_log',
        'calendar_golden_record','calendar_golden_field','calendar_reconciliation',
        'calendar_exception','calendar_steward','calendar_change_request'
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

-- 20261105_005_mdm_ca_golden.up.sql

-- ── ca_type_mapping ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    internal_event_type_cd varchar(30) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_catm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_catm_source ON mdm.ca_type_mapping (mdm_source_system_id);
CREATE INDEX IF NOT EXISTS idx_catm_tenant ON mdm.ca_type_mapping (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_catm_key
    ON mdm.ca_type_mapping (tenant_id, mdm_source_system_id, vendor_type_cd,
                              COALESCE(vendor_sub_type_cd, ''::varchar));

-- ── ca_source_priority ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type_cd varchar(30),
    country_cd varchar(2),
    asset_class_cd varchar(20),
    field_group varchar(50) NOT NULL,
    source_system_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_casp_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_casp_type   ON mdm.ca_source_priority (event_type_cd, field_group);
CREATE INDEX IF NOT EXISTS idx_casp_tenant ON mdm.ca_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_casp_key
    ON mdm.ca_source_priority (tenant_id, COALESCE(event_type_cd, ''),
                                 COALESCE(country_cd, ''), COALESCE(asset_class_cd, ''),
                                 field_group, source_system_id);

-- ── ca_field_mapping ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mdm_source_system_id uuid NOT NULL,
    event_type_cd varchar(30),
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
    CONSTRAINT ca_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cafm_source FOREIGN KEY (mdm_source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_cafm_source ON mdm.ca_field_mapping (mdm_source_system_id, event_type_cd);
CREATE INDEX IF NOT EXISTS idx_cafm_target ON mdm.ca_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_cafm_tenant ON mdm.ca_field_mapping (tenant_id);

-- ── ca_survivorship_rule ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type_cd varchar(30) NOT NULL,
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
    CONSTRAINT ca_survivorship_rule_pkey PRIMARY KEY (id)
);
-- COALESCE is not allowed inside a table CONSTRAINT UNIQUE — use a unique index (Batch 8 lesson)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casr_key
    ON mdm.ca_survivorship_rule (tenant_id, event_type_cd,
                                   COALESCE(country_cd, ''), field_group, field_name);
CREATE INDEX IF NOT EXISTS idx_casr_tenant ON mdm.ca_survivorship_rule (tenant_id);

-- ── ca_survivorship_log ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
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
    CONSTRAINT ca_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_casl_ca   FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_casl_rule FOREIGN KEY (rule_id)     REFERENCES mdm.ca_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_casl_ca     ON mdm.ca_survivorship_log (ca_event_id, golden_version);
CREATE INDEX IF NOT EXISTS idx_casl_field  ON mdm.ca_survivorship_log (field_name, decided_at);
CREATE INDEX IF NOT EXISTS idx_casl_tenant ON mdm.ca_survivorship_log (tenant_id);

-- ── ca_match_rule ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    event_type_cd varchar(30) NOT NULL,
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
    CONSTRAINT ca_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_camr_cd UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_camr_tenant ON mdm.ca_match_rule (tenant_id);

-- ── ca_match_candidate ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    ca_event_id_a uuid NOT NULL,
    ca_event_id_b uuid NOT NULL,
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
    CONSTRAINT ca_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_camc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_camc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.ca_match_rule(id),
    CONSTRAINT fk_camc_a    FOREIGN KEY (ca_event_id_a) REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_camc_b    FOREIGN KEY (ca_event_id_b) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_camc_status ON mdm.ca_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_camc_tenant ON mdm.ca_match_candidate (tenant_id);

-- ── ca_merge_log ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_ca_id uuid NOT NULL,
    merged_ca_id uuid NOT NULL,
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
    CONSTRAINT ca_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_caml_surv   FOREIGN KEY (surviving_ca_id) REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_caml_merged FOREIGN KEY (merged_ca_id)   REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_caml_surv   ON mdm.ca_merge_log (surviving_ca_id);
CREATE INDEX IF NOT EXISTS idx_caml_merged ON mdm.ca_merge_log (merged_ca_id);
CREATE INDEX IF NOT EXISTS idx_caml_tenant ON mdm.ca_merge_log (tenant_id);

-- ── ca_golden_record ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    event_type_cd varchar(30) NOT NULL,
    country_cd varchar(2),
    effective_date date NOT NULL,
    knowledge_timestamp timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    golden_attributes jsonb NOT NULL,
    winning_sources jsonb NOT NULL,
    overall_dq_score numeric(5,2),
    terms_confidence numeric(5,2),
    tax_confidence numeric(5,2),
    compliance_confidence numeric(5,2),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    published_at timestamptz,
    published_by uuid,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cagr_version UNIQUE (tenant_id, ca_event_id, golden_version),
    CONSTRAINT chk_cagr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_cagr_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_cagr_ca     ON mdm.ca_golden_record (ca_event_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cagr_etype  ON mdm.ca_golden_record (event_type_cd);
CREATE INDEX IF NOT EXISTS idx_cagr_tenant ON mdm.ca_golden_record (tenant_id);

-- ── ca_golden_field ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_golden_field (
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
    CONSTRAINT ca_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cagf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.ca_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_cagf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_cagf_record ON mdm.ca_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_cagf_field  ON mdm.ca_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_cagf_tenant ON mdm.ca_golden_field (tenant_id);

-- ── ca_exception ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid,
    event_type_cd varchar(30),
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
    CONSTRAINT ca_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_caex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_caex_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_caex_ca     FOREIGN KEY (ca_event_id)     REFERENCES mdm.ca_event(id),
    CONSTRAINT fk_caex_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_caex_status ON mdm.ca_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_caex_ca     ON mdm.ca_exception (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_caex_tenant ON mdm.ca_exception (tenant_id);

-- ── ca_change_request ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    ca_event_id uuid,
    event_type_cd varchar(30),
    change_type varchar(30) NOT NULL,
    requested_changes jsonb NOT NULL,
    request_reason text,
    source varchar(30),
    consumer_class varchar(30),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    requested_by uuid NOT NULL,
    assigned_to uuid,
    approved_by uuid,
    approved_at timestamptz,
    applied_at timestamptz,
    rejection_reason text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cacr_ref UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_cacr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_cacr_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id)
);
CREATE INDEX IF NOT EXISTS idx_cacr_status ON mdm.ca_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_cacr_ca     ON mdm.ca_change_request (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cacr_tenant ON mdm.ca_change_request (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_type_mapping','ca_source_priority','ca_field_mapping',
        'ca_survivorship_rule','ca_survivorship_log',
        'ca_match_rule','ca_match_candidate','ca_merge_log',
        'ca_golden_record','ca_golden_field','ca_exception','ca_change_request'
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

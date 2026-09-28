-- 20261030_004_mdm_counterparty_golden.up.sql

-- ── counterparty_match_rule ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    counterparty_type varchar(30) NOT NULL,
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
    CONSTRAINT counterparty_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_match_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_cpmr_tenant ON mdm.counterparty_match_rule (tenant_id);

-- ── counterparty_match_candidate ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    counterparty_id_a uuid NOT NULL,
    counterparty_id_b uuid NOT NULL,
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
    CONSTRAINT counterparty_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cpmc_status CHECK (status IN (
        'PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_cpmc_rule FOREIGN KEY (match_rule_id)     REFERENCES mdm.counterparty_match_rule(id),
    CONSTRAINT fk_cpmc_a    FOREIGN KEY (counterparty_id_a) REFERENCES mdm.counterparty(id),
    CONSTRAINT fk_cpmc_b    FOREIGN KEY (counterparty_id_b) REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpmc_status ON mdm.counterparty_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_cpmc_tenant ON mdm.counterparty_match_candidate (tenant_id);

-- ── counterparty_merge_log ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_counterparty_id uuid NOT NULL,
    merged_counterparty_id uuid NOT NULL,
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
    CONSTRAINT counterparty_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cpml_surv   FOREIGN KEY (surviving_counterparty_id) REFERENCES mdm.counterparty(id),
    CONSTRAINT fk_cpml_merged FOREIGN KEY (merged_counterparty_id)   REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpml_surv   ON mdm.counterparty_merge_log (surviving_counterparty_id);
CREATE INDEX IF NOT EXISTS idx_cpml_merged ON mdm.counterparty_merge_log (merged_counterparty_id);
CREATE INDEX IF NOT EXISTS idx_cpml_tenant ON mdm.counterparty_merge_log (tenant_id);

-- ── counterparty_golden_record ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    counterparty_type varchar(30) NOT NULL,
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
    CONSTRAINT counterparty_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cpgr_version UNIQUE (tenant_id, counterparty_id, golden_version),
    CONSTRAINT chk_cpgr_status CHECK (status IN (
        'DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_cpgr_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpgr_cpty   ON mdm.counterparty_golden_record (counterparty_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cpgr_tenant ON mdm.counterparty_golden_record (tenant_id);

-- ── counterparty_golden_field ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_golden_field (
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
    CONSTRAINT counterparty_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cpgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.counterparty_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_cpgf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_cpgf_record ON mdm.counterparty_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_cpgf_field  ON mdm.counterparty_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_cpgf_tenant ON mdm.counterparty_golden_field (tenant_id);

-- ── counterparty_exception ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    counterparty_id uuid,
    identifier_value varchar(100),
    counterparty_type varchar(30),
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
    CONSTRAINT counterparty_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cpex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT chk_cpex_status CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_cpex_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpex_status ON mdm.counterparty_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_cpex_cpty   ON mdm.counterparty_exception (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_cpex_tenant ON mdm.counterparty_exception (tenant_id);

-- ── counterparty_steward ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    counterparty_type varchar(30),
    region varchar(50),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_merge bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    can_approve_credit bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT counterparty_steward_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_cpst_user   ON mdm.counterparty_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_cpst_tenant ON mdm.counterparty_steward (tenant_id);

-- ── counterparty_change_request ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.counterparty_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    counterparty_id uuid,
    counterparty_type varchar(30),
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
    CONSTRAINT counterparty_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT counterparty_change_request_ref_key UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_cpcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_cpcr_cpty FOREIGN KEY (counterparty_id) REFERENCES mdm.counterparty(id)
);
CREATE INDEX IF NOT EXISTS idx_cpcr_status ON mdm.counterparty_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_cpcr_cpty   ON mdm.counterparty_change_request (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_cpcr_tenant ON mdm.counterparty_change_request (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'counterparty_match_rule','counterparty_match_candidate',
        'counterparty_merge_log','counterparty_golden_record',
        'counterparty_golden_field','counterparty_exception',
        'counterparty_steward','counterparty_change_request'
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

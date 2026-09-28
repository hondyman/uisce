-- ── party_role_assignment ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_role_assignment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    role_id uuid NOT NULL,
    entity_type varchar(30) NOT NULL,
    entity_id uuid NOT NULL,
    role_sub_type varchar(50),
    priority int4 DEFAULT 100,
    is_primary bool DEFAULT false NOT NULL,
    ownership_pct numeric(7,4),
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_role_assignment_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pra_status CHECK (status IN ('PENDING','ACTIVE','SUSPENDED','REVOKED','EXPIRED')),
    CONSTRAINT fk_pra_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_pra_role  FOREIGN KEY (role_id)  REFERENCES mdm.party_role(id)
);
CREATE INDEX IF NOT EXISTS idx_pra_party  ON mdm.party_role_assignment (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pra_role   ON mdm.party_role_assignment (role_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pra_entity ON mdm.party_role_assignment (entity_type, entity_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pra_tenant ON mdm.party_role_assignment (tenant_id);

-- ── party_history (SCD2) ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    version_num int4 NOT NULL,
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    is_current bool DEFAULT true NOT NULL,
    record_snapshot jsonb NOT NULL,
    changed_columns text[],
    change_source varchar(30),
    change_reason varchar(255),
    changed_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_history_pkey PRIMARY KEY (id),
    CONSTRAINT uq_partyh_version UNIQUE (tenant_id, party_id, version_num),
    CONSTRAINT fk_partyh_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_partyh_party  ON mdm.party_history (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_partyh_tenant ON mdm.party_history (tenant_id);

-- ── party_match_rule ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    party_type varchar(30) NOT NULL,
    party_sub_type varchar(50),
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
    CONSTRAINT party_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_pmr_cd UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_pmr_tenant ON mdm.party_match_rule (tenant_id);

-- ── party_match_candidate ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    party_id_a uuid NOT NULL,
    party_id_b uuid NOT NULL,
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
    CONSTRAINT party_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pmc_status CHECK (status IN ('PENDING','AUTO_MERGED','APPROVED','REJECTED','DEFERRED')),
    CONSTRAINT fk_pmc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.party_match_rule(id),
    CONSTRAINT fk_pmc_a    FOREIGN KEY (party_id_a)    REFERENCES mdm.party(id),
    CONSTRAINT fk_pmc_b    FOREIGN KEY (party_id_b)    REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pmc_status ON mdm.party_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_pmc_tenant ON mdm.party_match_candidate (tenant_id);

-- ── party_merge_log ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_party_id uuid NOT NULL,
    merged_party_id uuid NOT NULL,
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
    CONSTRAINT party_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pml_surv   FOREIGN KEY (surviving_party_id) REFERENCES mdm.party(id),
    CONSTRAINT fk_pml_merged FOREIGN KEY (merged_party_id)   REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pml_surv   ON mdm.party_merge_log (surviving_party_id);
CREATE INDEX IF NOT EXISTS idx_pml_merged ON mdm.party_merge_log (merged_party_id);
CREATE INDEX IF NOT EXISTS idx_pml_tenant ON mdm.party_merge_log (tenant_id);

-- ── party_golden_record ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    party_type varchar(30) NOT NULL,
    party_sub_type varchar(50),
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
    CONSTRAINT party_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT uq_pgr_version UNIQUE (tenant_id, party_id, golden_version),
    CONSTRAINT chk_pgr_status CHECK (status IN ('DRAFT','REVIEW','PUBLISHED','SUPERSEDED','RETRACTED')),
    CONSTRAINT fk_pgr_party FOREIGN KEY (party_id) REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pgr_party  ON mdm.party_golden_record (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pgr_tenant ON mdm.party_golden_record (tenant_id);

-- ── party_golden_field ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_golden_field (
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
    CONSTRAINT party_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.party_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_pgf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_pgf_record ON mdm.party_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_pgf_field  ON mdm.party_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_pgf_tenant ON mdm.party_golden_field (tenant_id);

-- ── party_exception ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid,
    identifier_value varchar(100),
    party_type varchar(30),
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
    CONSTRAINT party_exception_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pex_status CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT chk_pex_severity CHECK (severity IN ('ERROR','WARNING','INFO')),
    CONSTRAINT fk_pex_party FOREIGN KEY (party_id) REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pex_status ON mdm.party_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_pex_party  ON mdm.party_exception (party_id);
CREATE INDEX IF NOT EXISTS idx_pex_tenant ON mdm.party_exception (tenant_id);

-- ── party_change_request ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    party_id uuid,
    party_type varchar(30),
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
    CONSTRAINT party_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT uq_pcr_ref UNIQUE (tenant_id, request_ref),
    CONSTRAINT chk_pcr_status CHECK (status IN (
        'DRAFT','SUBMITTED','REVIEW','APPROVED','REJECTED','APPLIED','ROLLED_BACK')),
    CONSTRAINT fk_pcr_party FOREIGN KEY (party_id) REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pcr_status ON mdm.party_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_pcr_party  ON mdm.party_change_request (party_id);
CREATE INDEX IF NOT EXISTS idx_pcr_tenant ON mdm.party_change_request (tenant_id);

-- ── party_steward ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    party_type varchar(30),
    party_sub_type varchar(50),
    region varchar(50),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_merge bool DEFAULT false NOT NULL,
    can_view_pii bool DEFAULT false NOT NULL,
    can_view_kyc bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_steward_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_pst_user   ON mdm.party_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_pst_tenant ON mdm.party_steward (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'party_role_assignment','party_history','party_match_rule',
        'party_match_candidate','party_merge_log','party_golden_record',
        'party_golden_field','party_exception','party_change_request',
        'party_steward'
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

-- ============================================================================
-- 0001_mdm_security.sql  -  the security MDM layer of the CRIMS `mdm` schema (38 tables)
--
-- NOT applied by the migration runner (that targets the `alpha` database). Run by hand against `crims`:
--     psql "<crims url>" -X -v ON_ERROR_STOP=1 --single-transaction -f 0001_mdm_security.sql
-- or use the wrapper that checks its prerequisites first (scratchpad apply_crims_mdm_security.sh).
--
-- Re-runnable: CREATE ... IF NOT EXISTS, and policies are dropped and recreated.
-- Needs: mdm.source_system, orm.corporate_action (both exist in CRIMS).
--
-- Follows the existing mdm tables exactly: tenant_id on every row, a tenant-read policy that also lets a
-- tenant read the shared reference tenant (app.shared_reference_tenant), a tenant-only write policy, and
-- FORCE ROW LEVEL SECURITY (the issuer_* tables are forced too, so the owner does not bypass it).
--
-- security_id columns point at orm.security(id) by convention only: there is no FK, matching how the issuer
-- golden-record tables reference their master.
-- ============================================================================

-- Part A: taxonomy and vocabularies ------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_asset_class (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    display_order int4 DEFAULT 0,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_asset_class_pkey PRIMARY KEY (id),
    CONSTRAINT security_asset_class_cd_key UNIQUE (tenant_id, asset_class_cd)
);
CREATE INDEX IF NOT EXISTS idx_sac_tenant ON mdm.security_asset_class (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_typ_cd varchar(15) NOT NULL,
    sec_sub_typ_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    description text,
    detail_table varchar(100),
    satellite_tables jsonb DEFAULT '[]'::jsonb NOT NULL,
    requires_maturity bool DEFAULT false NOT NULL,
    requires_coupon bool DEFAULT false NOT NULL,
    requires_underlying bool DEFAULT false NOT NULL,
    requires_issuer bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_type_pkey PRIMARY KEY (id),
    CONSTRAINT security_type_key UNIQUE (tenant_id, sec_typ_cd, sec_sub_typ_cd)
);
CREATE INDEX IF NOT EXISTS idx_st_ac     ON mdm.security_type (asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_st_tenant ON mdm.security_type (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_type_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    vendor_type_cd varchar(100) NOT NULL,
    vendor_sub_type_cd varchar(100),
    vendor_description varchar(255),
    internal_asset_class_cd varchar(20) NOT NULL,
    internal_sec_typ_cd varchar(15) NOT NULL,
    internal_sec_sub_typ_cd varchar(30) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_type_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_stm_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_stm_source   ON mdm.security_type_mapping (source_system_id);
CREATE INDEX IF NOT EXISTS idx_stm_internal ON mdm.security_type_mapping (internal_sec_typ_cd, internal_sec_sub_typ_cd);
CREATE INDEX IF NOT EXISTS idx_stm_tenant   ON mdm.security_type_mapping (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_stm_key
    ON mdm.security_type_mapping (tenant_id, source_system_id, vendor_type_cd, COALESCE(vendor_sub_type_cd, ''::varchar));

CREATE TABLE IF NOT EXISTS mdm.classification_scheme (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    scheme_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    version varchar(20),
    hierarchy_depth int4 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT classification_scheme_pkey PRIMARY KEY (id),
    CONSTRAINT classification_scheme_cd_key UNIQUE (tenant_id, scheme_cd, version)
);
CREATE INDEX IF NOT EXISTS idx_cs_tenant ON mdm.classification_scheme (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.classification_scheme_map (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    from_scheme_cd varchar(30) NOT NULL,
    from_code varchar(30) NOT NULL,
    from_level int4 NOT NULL,
    to_scheme_cd varchar(30) NOT NULL,
    to_code varchar(30) NOT NULL,
    to_level int4 NOT NULL,
    mapping_type varchar(20) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT classification_scheme_map_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_csm_from   ON mdm.classification_scheme_map (from_scheme_cd, from_code);
CREATE INDEX IF NOT EXISTS idx_csm_to     ON mdm.classification_scheme_map (to_scheme_cd, to_code);
CREATE INDEX IF NOT EXISTS idx_csm_tenant ON mdm.classification_scheme_map (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.rating_scale (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_cd varchar(20) NOT NULL,
    rating_value varchar(20) NOT NULL,
    rating_scale varchar(20) NOT NULL,
    numeric_equivalent int4 NOT NULL,
    is_investment_grade bool NOT NULL,
    is_default bool DEFAULT false NOT NULL,
    is_high_yield bool DEFAULT false NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_scale_pkey PRIMARY KEY (id),
    CONSTRAINT rating_scale_key UNIQUE (tenant_id, agency_cd, rating_scale, rating_value)
);
CREATE INDEX IF NOT EXISTS idx_rs_equiv  ON mdm.rating_scale (numeric_equivalent);
CREATE INDEX IF NOT EXISTS idx_rs_tenant ON mdm.rating_scale (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.day_count_convention (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    convention_cd varchar(20) NOT NULL,
    name varchar(100) NOT NULL,
    formula text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT day_count_convention_pkey PRIMARY KEY (id),
    CONSTRAINT day_count_convention_cd_key UNIQUE (tenant_id, convention_cd)
);
CREATE INDEX IF NOT EXISTS idx_dcc_tenant ON mdm.day_count_convention (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.business_day_convention (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    convention_cd varchar(20) NOT NULL,
    name varchar(100) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT business_day_convention_pkey PRIMARY KEY (id),
    CONSTRAINT business_day_convention_cd_key UNIQUE (tenant_id, convention_cd)
);
CREATE INDEX IF NOT EXISTS idx_bdc_tenant ON mdm.business_day_convention (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.payment_frequency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    frequency_cd varchar(20) NOT NULL,
    name varchar(100) NOT NULL,
    periods_per_year numeric(6,2) NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT payment_frequency_pkey PRIMARY KEY (id),
    CONSTRAINT payment_frequency_cd_key UNIQUE (tenant_id, frequency_cd)
);
CREATE INDEX IF NOT EXISTS idx_pf_tenant ON mdm.payment_frequency (tenant_id);

-- Part B: source management per asset class ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_source_priority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_sub_typ_cd varchar(30),
    field_group varchar(50) NOT NULL,
    source_system_id uuid NOT NULL,
    priority int4 NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_source_priority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ssp_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_ssp_asset  ON mdm.security_source_priority (asset_class_cd, field_group);
CREATE INDEX IF NOT EXISTS idx_ssp_tenant ON mdm.security_source_priority (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ssp_key
    ON mdm.security_source_priority (tenant_id, asset_class_cd, COALESCE(sec_sub_typ_cd, ''::varchar), field_group, source_system_id);

CREATE TABLE IF NOT EXISTS mdm.security_feed_schedule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
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
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_feed_schedule_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sfs_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sfs_tenant ON mdm.security_feed_schedule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_sfs_key ON mdm.security_feed_schedule (tenant_id, source_system_id, feed_name);

CREATE TABLE IF NOT EXISTS mdm.security_feed_health (
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
    CONSTRAINT security_feed_health_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sfh_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sfh_status ON mdm.security_feed_health (status, run_date);
CREATE INDEX IF NOT EXISTS idx_sfh_tenant ON mdm.security_feed_health (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_sfh_key ON mdm.security_feed_health (tenant_id, source_system_id, feed_name, run_date);

CREATE TABLE IF NOT EXISTS mdm.security_asset_class_routing (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sec_typ_cd varchar(15) NOT NULL,
    sec_sub_typ_cd varchar(30) NOT NULL,
    detail_table varchar(100) NOT NULL,
    satellite_tables jsonb DEFAULT '[]'::jsonb NOT NULL,
    requires_underlying bool DEFAULT false NOT NULL,
    requires_term_sheet bool DEFAULT false NOT NULL,
    requires_legal_review bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_asset_class_routing_pkey PRIMARY KEY (id),
    CONSTRAINT security_asset_class_routing_key UNIQUE (tenant_id, sec_typ_cd, sec_sub_typ_cd)
);
CREATE INDEX IF NOT EXISTS idx_sacr_tenant ON mdm.security_asset_class_routing (tenant_id);

-- Part C: identifier authority -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_identifier_authority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    id_type varchar(20) NOT NULL,
    asset_class_cd varchar(20),
    source_system_id uuid NOT NULL,
    is_authoritative bool DEFAULT true NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    requires_checksum_validation bool DEFAULT true NOT NULL,
    checksum_algorithm varchar(30),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_identifier_authority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sia_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sia_type   ON mdm.security_identifier_authority (id_type, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_sia_tenant ON mdm.security_identifier_authority (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_identifier_issuance (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    id_type varchar(20) NOT NULL,
    id_value varchar(100) NOT NULL,
    issuing_authority varchar(50),
    source_system_id uuid NOT NULL,
    first_seen_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_confirmed_at timestamptz,
    is_valid bool DEFAULT true NOT NULL,
    validation_method varchar(30),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_identifier_issuance_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sii_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sii_sec    ON mdm.security_identifier_issuance (security_id, id_type);
CREATE INDEX IF NOT EXISTS idx_sii_value  ON mdm.security_identifier_issuance (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_sii_tenant ON mdm.security_identifier_issuance (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_identifier_conflict (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    id_type varchar(20) NOT NULL,
    id_value varchar(100) NOT NULL,
    source_system_id_a uuid NOT NULL,
    source_value_a varchar(100),
    source_system_id_b uuid NOT NULL,
    source_value_b varchar(100),
    security_id_a uuid,
    security_id_b uuid,
    conflict_type varchar(30) NOT NULL,
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolution_note text,
    resolved_by uuid,
    resolved_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_identifier_conflict_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sic_src_a FOREIGN KEY (source_system_id_a) REFERENCES mdm.source_system(id),
    CONSTRAINT fk_sic_src_b FOREIGN KEY (source_system_id_b) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sic_status ON mdm.security_identifier_conflict (status, severity);
CREATE INDEX IF NOT EXISTS idx_sic_value  ON mdm.security_identifier_conflict (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_sic_tenant ON mdm.security_identifier_conflict (tenant_id);

-- Part D: field mapping and survivorship -------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_field_mapping (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_system_id uuid NOT NULL,
    asset_class_cd varchar(20),
    vendor_field varchar(150) NOT NULL,
    vendor_data_type varchar(30),
    internal_table varchar(100) NOT NULL,
    internal_field varchar(150) NOT NULL,
    transform_expression text,
    is_required bool DEFAULT false NOT NULL,
    default_value text,
    valid_from date DEFAULT CURRENT_DATE NOT NULL,
    valid_to date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_field_mapping_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sfm_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sfm_source ON mdm.security_field_mapping (source_system_id, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_sfm_target ON mdm.security_field_mapping (internal_table, internal_field);
CREATE INDEX IF NOT EXISTS idx_sfm_tenant ON mdm.security_field_mapping (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_field_transform (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    transform_cd varchar(50) NOT NULL,
    name varchar(150) NOT NULL,
    transform_type varchar(30) NOT NULL,
    transform_logic text NOT NULL,
    input_data_type varchar(30),
    output_data_type varchar(30),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_field_transform_pkey PRIMARY KEY (id),
    CONSTRAINT security_field_transform_cd_key UNIQUE (tenant_id, transform_cd)
);
CREATE INDEX IF NOT EXISTS idx_sft_tenant ON mdm.security_field_transform (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_survivorship_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_sub_typ_cd varchar(30),
    field_group varchar(50) NOT NULL,
    field_name varchar(150) NOT NULL,
    strategy varchar(30) NOT NULL,
    source_priority jsonb,
    min_confidence numeric(5,2),
    max_staleness_hours int4,
    manual_override_allowed bool DEFAULT true NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_survivorship_rule_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_ssr_asset  ON mdm.security_survivorship_rule (asset_class_cd, field_group);
CREATE INDEX IF NOT EXISTS idx_ssr_tenant ON mdm.security_survivorship_rule (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ssr_key
    ON mdm.security_survivorship_rule (tenant_id, asset_class_cd, COALESCE(sec_sub_typ_cd, ''::varchar), field_group, field_name);

CREATE TABLE IF NOT EXISTS mdm.security_survivorship_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
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
    CONSTRAINT security_survivorship_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ssl_rule FOREIGN KEY (rule_id) REFERENCES mdm.security_survivorship_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_ssl_sec    ON mdm.security_survivorship_log (security_id, golden_version);
CREATE INDEX IF NOT EXISTS idx_ssl_field  ON mdm.security_survivorship_log (field_name, decided_at);
CREATE INDEX IF NOT EXISTS idx_ssl_tenant ON mdm.security_survivorship_log (tenant_id);

-- Part E: match, merge, split ------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_match_rule (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_sub_typ_cd varchar(30),
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
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_match_rule_pkey PRIMARY KEY (id),
    CONSTRAINT security_match_rule_cd_key UNIQUE (tenant_id, rule_cd)
);
CREATE INDEX IF NOT EXISTS idx_smr_asset  ON mdm.security_match_rule (asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_smr_tenant ON mdm.security_match_rule (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_match_candidate (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_rule_id uuid NOT NULL,
    security_id_a uuid NOT NULL,
    security_id_b uuid NOT NULL,
    overall_score numeric(5,2) NOT NULL,
    deterministic_match bool DEFAULT false NOT NULL,
    matched_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    conflicting_keys jsonb DEFAULT '[]'::jsonb NOT NULL,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamptz,
    review_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_match_candidate_pkey PRIMARY KEY (id),
    CONSTRAINT fk_smc_rule FOREIGN KEY (match_rule_id) REFERENCES mdm.security_match_rule(id)
);
CREATE INDEX IF NOT EXISTS idx_smc_status ON mdm.security_match_candidate (status, overall_score);
CREATE INDEX IF NOT EXISTS idx_smc_pairs  ON mdm.security_match_candidate (security_id_a, security_id_b);
CREATE INDEX IF NOT EXISTS idx_smc_tenant ON mdm.security_match_candidate (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_merge_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surviving_security_id uuid NOT NULL,
    merged_security_id uuid NOT NULL,
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
    CONSTRAINT security_merge_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sml_candidate FOREIGN KEY (match_candidate_id) REFERENCES mdm.security_match_candidate(id)
);
CREATE INDEX IF NOT EXISTS idx_sml_surviving ON mdm.security_merge_log (surviving_security_id);
CREATE INDEX IF NOT EXISTS idx_sml_merged    ON mdm.security_merge_log (merged_security_id);
CREATE INDEX IF NOT EXISTS idx_sml_tenant    ON mdm.security_merge_log (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_split_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    original_security_id uuid NOT NULL,
    new_security_id uuid NOT NULL,
    split_type varchar(30) NOT NULL,
    split_ratio numeric(18,9),
    effective_date date NOT NULL,
    corporate_action_id uuid,
    split_by uuid,
    split_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reversal_data jsonb,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_split_log_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ssl_ca FOREIGN KEY (corporate_action_id) REFERENCES orm.corporate_action(id)
);
CREATE INDEX IF NOT EXISTS idx_sspl_orig   ON mdm.security_split_log (original_security_id);
CREATE INDEX IF NOT EXISTS idx_sspl_new    ON mdm.security_split_log (new_security_id);
CREATE INDEX IF NOT EXISTS idx_sspl_tenant ON mdm.security_split_log (tenant_id);

-- Part F: golden record and publication --------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_golden_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    golden_version int4 NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_typ_cd varchar(15) NOT NULL,
    sec_sub_typ_cd varchar(30) NOT NULL,
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
    CONSTRAINT security_golden_record_pkey PRIMARY KEY (id),
    CONSTRAINT security_golden_record_key UNIQUE (tenant_id, security_id, golden_version)
);
CREATE INDEX IF NOT EXISTS idx_sgr_sec    ON mdm.security_golden_record (security_id, is_current);
CREATE INDEX IF NOT EXISTS idx_sgr_pub    ON mdm.security_golden_record (status, published_at);
CREATE INDEX IF NOT EXISTS idx_sgr_class  ON mdm.security_golden_record (asset_class_cd, sec_sub_typ_cd);
CREATE INDEX IF NOT EXISTS idx_sgr_tenant ON mdm.security_golden_record (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_golden_field (
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
    CONSTRAINT security_golden_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sgf_record FOREIGN KEY (golden_record_id) REFERENCES mdm.security_golden_record(id) ON DELETE CASCADE,
    CONSTRAINT fk_sgf_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sgf_record ON mdm.security_golden_field (golden_record_id);
CREATE INDEX IF NOT EXISTS idx_sgf_field  ON mdm.security_golden_field (field_name);
CREATE INDEX IF NOT EXISTS idx_sgf_tenant ON mdm.security_golden_field (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_golden_publication (
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
    CONSTRAINT security_golden_publication_pkey PRIMARY KEY (id),
    CONSTRAINT uq_sgp_ref UNIQUE (tenant_id, publication_ref)
);
CREATE INDEX IF NOT EXISTS idx_sgp_tenant ON mdm.security_golden_publication (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_golden_distribution (
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
    CONSTRAINT security_golden_distribution_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sgd_pub    FOREIGN KEY (publication_id)   REFERENCES mdm.security_golden_publication(id),
    CONSTRAINT fk_sgd_record FOREIGN KEY (golden_record_id) REFERENCES mdm.security_golden_record(id),
    CONSTRAINT fk_sgd_target FOREIGN KEY (target_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_sgd_target ON mdm.security_golden_distribution (target_system_id, delivery_status);
CREATE INDEX IF NOT EXISTS idx_sgd_tenant ON mdm.security_golden_distribution (tenant_id);

-- Part G: term sheet extraction ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_term_sheet (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    document_type varchar(30) NOT NULL,
    document_name varchar(255) NOT NULL,
    document_url text,
    document_hash varchar(64),
    language varchar(10),
    effective_date date,
    extraction_status varchar(20) DEFAULT 'PENDING' NOT NULL,
    extracted_at timestamptz,
    extracted_by uuid,
    reviewed_by uuid,
    reviewed_at timestamptz,
    confidence numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_term_sheet_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_sts_sec    ON mdm.security_term_sheet (security_id, document_type);
CREATE INDEX IF NOT EXISTS idx_sts_status ON mdm.security_term_sheet (extraction_status);
CREATE INDEX IF NOT EXISTS idx_sts_tenant ON mdm.security_term_sheet (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_term_extraction_job (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    term_sheet_id uuid NOT NULL,
    job_type varchar(30) NOT NULL,
    model_version varchar(50),
    started_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at timestamptz,
    status varchar(20) DEFAULT 'RUNNING' NOT NULL,
    fields_extracted int4,
    fields_pending_review int4,
    error_message text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_term_extraction_job_pkey PRIMARY KEY (id),
    CONSTRAINT fk_stej_ts FOREIGN KEY (term_sheet_id) REFERENCES mdm.security_term_sheet(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_stej_tenant ON mdm.security_term_extraction_job (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_term_extraction_field (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    job_id uuid NOT NULL,
    field_name varchar(150) NOT NULL,
    extracted_value text,
    extracted_value_json jsonb,
    confidence numeric(5,2),
    source_page int4,
    source_snippet text,
    validation_status varchar(20) DEFAULT 'PENDING' NOT NULL,
    validation_note text,
    validated_by uuid,
    validated_at timestamptz,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_term_extraction_field_pkey PRIMARY KEY (id),
    CONSTRAINT fk_stef_job FOREIGN KEY (job_id) REFERENCES mdm.security_term_extraction_job(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_stef_job    ON mdm.security_term_extraction_field (job_id);
CREATE INDEX IF NOT EXISTS idx_stef_field  ON mdm.security_term_extraction_field (field_name);
CREATE INDEX IF NOT EXISTS idx_stef_tenant ON mdm.security_term_extraction_field (tenant_id);

-- Part H: reconciliation -----------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_reconciliation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_date date NOT NULL,
    asset_class_cd varchar(20),
    source_system_id_a uuid NOT NULL,
    source_system_id_b uuid NOT NULL,
    records_compared int4 NOT NULL,
    records_matched int4 NOT NULL,
    records_only_in_a int4 NOT NULL,
    records_only_in_b int4 NOT NULL,
    records_field_conflicts int4 NOT NULL,
    match_pct numeric(5,2),
    tolerance_pct numeric(5,2),
    status varchar(20) DEFAULT 'COMPLETED' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_reconciliation_pkey PRIMARY KEY (id),
    CONSTRAINT fk_srec_src_a FOREIGN KEY (source_system_id_a) REFERENCES mdm.source_system(id),
    CONSTRAINT fk_srec_src_b FOREIGN KEY (source_system_id_b) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_srec_date   ON mdm.security_reconciliation (reconciliation_date, asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_srec_tenant ON mdm.security_reconciliation (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_reconciliation_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    reconciliation_id uuid NOT NULL,
    security_id uuid,
    identifier_value varchar(100),
    field_name varchar(150) NOT NULL,
    value_a text,
    value_b text,
    values_match bool NOT NULL,
    variance_pct numeric(12,8),
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolved_by uuid,
    resolved_at timestamptz,
    resolution_note text,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_reconciliation_result_pkey PRIMARY KEY (id),
    CONSTRAINT fk_srr_rec FOREIGN KEY (reconciliation_id) REFERENCES mdm.security_reconciliation(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_srr_rec    ON mdm.security_reconciliation_result (reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_srr_status ON mdm.security_reconciliation_result (status, severity);
CREATE INDEX IF NOT EXISTS idx_srr_sec    ON mdm.security_reconciliation_result (security_id);
CREATE INDEX IF NOT EXISTS idx_srr_tenant ON mdm.security_reconciliation_result (tenant_id);

-- Part I: governance ---------------------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mdm.security_status_authority (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_sub_typ_cd varchar(30),
    status_field varchar(30) NOT NULL,
    source_system_id uuid NOT NULL,
    authority_level varchar(20) NOT NULL,
    auto_apply bool DEFAULT true NOT NULL,
    requires_review bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_status_authority_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ssa_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_ssa_tenant ON mdm.security_status_authority (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_exception (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid,
    identifier_value varchar(100),
    asset_class_cd varchar(20),
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
    CONSTRAINT security_exception_pkey PRIMARY KEY (id),
    CONSTRAINT fk_se_source FOREIGN KEY (source_system_id) REFERENCES mdm.source_system(id)
);
CREATE INDEX IF NOT EXISTS idx_se_status ON mdm.security_exception (status, severity);
CREATE INDEX IF NOT EXISTS idx_se_type   ON mdm.security_exception (exception_type, detected_at);
CREATE INDEX IF NOT EXISTS idx_se_sec    ON mdm.security_exception (security_id);
CREATE INDEX IF NOT EXISTS idx_se_tenant ON mdm.security_exception (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_steward (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    asset_class_cd varchar(20),
    sec_sub_typ_cd varchar(30),
    region varchar(50),
    steward_role varchar(30) NOT NULL,
    can_override_survivorship bool DEFAULT false NOT NULL,
    can_merge bool DEFAULT false NOT NULL,
    can_publish bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_steward_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_sst_user   ON mdm.security_steward (user_id);
CREATE INDEX IF NOT EXISTS idx_sst_ac     ON mdm.security_steward (asset_class_cd, steward_role);
CREATE INDEX IF NOT EXISTS idx_sst_tenant ON mdm.security_steward (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_change_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    request_ref varchar(50) NOT NULL,
    security_id uuid,
    asset_class_cd varchar(20),
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
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_change_request_pkey PRIMARY KEY (id),
    CONSTRAINT uq_scr_ref UNIQUE (tenant_id, request_ref)
);
CREATE INDEX IF NOT EXISTS idx_scr_status ON mdm.security_change_request (status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_scr_sec    ON mdm.security_change_request (security_id);
CREATE INDEX IF NOT EXISTS idx_scr_tenant ON mdm.security_change_request (tenant_id);

CREATE TABLE IF NOT EXISTS mdm.security_ca_linkage (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    corporate_action_id uuid NOT NULL,
    linkage_type varchar(30) NOT NULL,
    source_system_id uuid,
    confidence numeric(5,2),
    is_primary bool DEFAULT false NOT NULL,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT security_ca_linkage_pkey PRIMARY KEY (id),
    CONSTRAINT fk_scal_ca FOREIGN KEY (corporate_action_id) REFERENCES orm.corporate_action(id)
);
CREATE INDEX IF NOT EXISTS idx_scal_sec    ON mdm.security_ca_linkage (security_id);
CREATE INDEX IF NOT EXISTS idx_scal_ca     ON mdm.security_ca_linkage (corporate_action_id);
CREATE INDEX IF NOT EXISTS idx_scal_tenant ON mdm.security_ca_linkage (tenant_id);

-- Row-level security: the same read/write pair on every table above, and FORCE like the existing mdm tables.
-- Read: the tenant's own rows plus the shared reference tenant's. Write: the tenant's own rows only.
DO $rls$
DECLARE
    t text;
    tables text[] := ARRAY[
        'security_asset_class', 'security_type', 'security_type_mapping', 'classification_scheme',
        'classification_scheme_map', 'rating_scale', 'day_count_convention', 'business_day_convention',
        'payment_frequency', 'security_source_priority', 'security_feed_schedule', 'security_feed_health',
        'security_asset_class_routing', 'security_identifier_authority', 'security_identifier_issuance',
        'security_identifier_conflict', 'security_field_mapping', 'security_field_transform',
        'security_survivorship_rule', 'security_survivorship_log', 'security_match_rule',
        'security_match_candidate', 'security_merge_log', 'security_split_log', 'security_golden_record',
        'security_golden_field', 'security_golden_publication', 'security_golden_distribution',
        'security_term_sheet', 'security_term_extraction_job', 'security_term_extraction_field',
        'security_reconciliation', 'security_reconciliation_result', 'security_status_authority',
        'security_exception', 'security_steward', 'security_change_request', 'security_ca_linkage'];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$, t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$, t || '_tenant_write', t);
    END LOOP;
END
$rls$;

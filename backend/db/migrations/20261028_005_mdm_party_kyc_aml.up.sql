-- ── kyc_profile ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.kyc_profile (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    kyc_status_id uuid NOT NULL,
    risk_rating_id uuid,
    overall_risk_score numeric(7,4),
    customer_risk_score numeric(7,4),
    geographic_risk_score numeric(7,4),
    product_risk_score numeric(7,4),
    channel_risk_score numeric(7,4),
    pep_risk_score numeric(7,4),
    sanctions_risk_score numeric(7,4),
    onboarding_date date,
    onboarding_approved_by uuid,
    onboarding_approved_at timestamptz,
    last_review_date date,
    next_review_date date,
    review_frequency_months int4,
    requires_edd bool DEFAULT false NOT NULL,
    edd_reason varchar(500),
    edd_completed_date date,
    is_pep bool DEFAULT false NOT NULL,
    is_sanctioned bool DEFAULT false NOT NULL,
    is_adverse_media bool DEFAULT false NOT NULL,
    source_of_wealth_verified bool DEFAULT false NOT NULL,
    source_of_funds_verified bool DEFAULT false NOT NULL,
    is_high_risk_jurisdiction bool DEFAULT false NOT NULL,
    onboarding_channel varchar(30),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT kyc_profile_pkey PRIMARY KEY (id),
    CONSTRAINT uq_kyc_profile UNIQUE (tenant_id, party_id),
    CONSTRAINT fk_kyc_party  FOREIGN KEY (party_id)      REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_kyc_status FOREIGN KEY (kyc_status_id) REFERENCES mdm.kyc_status(id),
    CONSTRAINT fk_kyc_risk   FOREIGN KEY (risk_rating_id) REFERENCES mdm.risk_rating(id)
);
CREATE INDEX IF NOT EXISTS idx_kyc_party  ON mdm.kyc_profile (party_id);
CREATE INDEX IF NOT EXISTS idx_kyc_tenant ON mdm.kyc_profile (tenant_id);

-- ── kyc_document ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.kyc_document (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    document_type_id uuid NOT NULL,
    document_number varchar(100),
    document_url text,
    document_hash varchar(64),
    issue_date date,
    expiry_date date,
    issuing_country_cd varchar(2),
    issuing_authority varchar(250),
    is_verified bool DEFAULT false NOT NULL,
    verified_at timestamptz,
    verified_by uuid,
    verification_method varchar(30),
    is_current bool DEFAULT true NOT NULL,
    superseded_by_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT kyc_document_pkey PRIMARY KEY (id),
    CONSTRAINT fk_kycd_party  FOREIGN KEY (party_id)         REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_kycd_type   FOREIGN KEY (document_type_id) REFERENCES mdm.party_document_type(id)
);
CREATE INDEX IF NOT EXISTS idx_kycd_party  ON mdm.kyc_document (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_kycd_expiry ON mdm.kyc_document (expiry_date);
CREATE INDEX IF NOT EXISTS idx_kycd_tenant ON mdm.kyc_document (tenant_id);

-- ── sanctions_screening ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.sanctions_screening (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    list_source_id uuid NOT NULL,
    screening_date date NOT NULL,
    screening_timestamp timestamptz,
    screening_provider varchar(100),
    screened_name varchar(500),
    match_score numeric(7,4),
    match_status varchar(30) NOT NULL,
    is_priority bool DEFAULT false NOT NULL,
    is_continuous bool DEFAULT false NOT NULL,
    hit_count int4 DEFAULT 0,
    reviewed_by uuid,
    reviewed_at timestamptz,
    review_notes text,
    disposition_reason varchar(500),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT sanctions_screening_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ss_status CHECK (match_status IN (
        'NO_MATCH','MATCH','POTENTIAL_MATCH','FALSE_POSITIVE',
        'CONFIRMED_MATCH','PENDING_REVIEW','CLEARED')),
    CONSTRAINT fk_ss_party FOREIGN KEY (party_id)       REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_ss_list  FOREIGN KEY (list_source_id) REFERENCES mdm.sanctions_list_source(id)
);
CREATE INDEX IF NOT EXISTS idx_ss_party  ON mdm.sanctions_screening (party_id, screening_date);
CREATE INDEX IF NOT EXISTS idx_ss_status ON mdm.sanctions_screening (match_status);
CREATE INDEX IF NOT EXISTS idx_ss_tenant ON mdm.sanctions_screening (tenant_id);

-- ── sanctions_hit ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.sanctions_hit (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sanctions_screening_id uuid NOT NULL,
    party_id uuid NOT NULL,
    list_source_id uuid NOT NULL,
    hit_reference varchar(100),
    hit_name varchar(500),
    hit_type varchar(30),
    matched_field varchar(50),
    match_score numeric(7,4),
    list_entry_id varchar(100),
    list_entry_data jsonb,
    disposition varchar(30),
    disposition_by uuid,
    disposition_at timestamptz,
    disposition_notes text,
    escalation_required bool DEFAULT false NOT NULL,
    sar_filed bool DEFAULT false NOT NULL,
    sar_reference varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT sanctions_hit_pkey PRIMARY KEY (id),
    CONSTRAINT fk_sh_screening FOREIGN KEY (sanctions_screening_id) REFERENCES mdm.sanctions_screening(id) ON DELETE CASCADE,
    CONSTRAINT fk_sh_party     FOREIGN KEY (party_id)               REFERENCES mdm.party(id),
    CONSTRAINT fk_sh_list      FOREIGN KEY (list_source_id)         REFERENCES mdm.sanctions_list_source(id)
);
CREATE INDEX IF NOT EXISTS idx_sh_screening ON mdm.sanctions_hit (sanctions_screening_id);
CREATE INDEX IF NOT EXISTS idx_sh_party     ON mdm.sanctions_hit (party_id);
CREATE INDEX IF NOT EXISTS idx_sh_tenant    ON mdm.sanctions_hit (tenant_id);

-- ── pep_screening ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.pep_screening (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    screening_date date NOT NULL,
    screening_timestamp timestamptz,
    screening_provider varchar(100),
    is_pep bool NOT NULL,
    pep_category varchar(30),
    pep_level varchar(20),
    position_title varchar(500),
    country_cd varchar(2),
    organization varchar(500),
    match_score numeric(7,4),
    disposition varchar(30),
    disposition_by uuid,
    disposition_at timestamptz,
    disposition_notes text,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT pep_screening_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ps_level CHECK (pep_level IS NULL OR pep_level IN (
        'DOMESTIC','FOREIGN','INTERNATIONAL_ORG','FAMILY','ASSOCIATE')),
    CONSTRAINT fk_ps_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ps_party  ON mdm.pep_screening (party_id, screening_date);
CREATE INDEX IF NOT EXISTS idx_ps_ispep  ON mdm.pep_screening (is_pep) WHERE is_pep = true;
CREATE INDEX IF NOT EXISTS idx_ps_tenant ON mdm.pep_screening (tenant_id);

-- ── adverse_media_check ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.adverse_media_check (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    check_date date NOT NULL,
    check_provider varchar(100),
    match_found bool DEFAULT false NOT NULL,
    match_count int4 DEFAULT 0,
    severity varchar(20),
    category varchar(50),
    article_url text,
    article_source varchar(250),
    article_date date,
    article_summary text,
    disposition varchar(30),
    disposition_by uuid,
    disposition_at timestamptz,
    disposition_notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT adverse_media_check_pkey PRIMARY KEY (id),
    CONSTRAINT chk_amc_sev CHECK (severity IS NULL OR severity IN ('LOW','MEDIUM','HIGH','CRITICAL')),
    CONSTRAINT fk_amc_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_amc_party  ON mdm.adverse_media_check (party_id, check_date);
CREATE INDEX IF NOT EXISTS idx_amc_sev    ON mdm.adverse_media_check (severity);
CREATE INDEX IF NOT EXISTS idx_amc_tenant ON mdm.adverse_media_check (tenant_id);

-- ── fatca_crs_classification ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fatca_crs_classification (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    regime varchar(20) NOT NULL,
    status_id uuid NOT NULL,
    classification_type varchar(50),
    controlling_person_type varchar(30),
    self_certification_date date,
    self_certification_document_id uuid,
    certification_type varchar(30),
    w8_w9_expiry_date date,
    w8_w9_benefit_type varchar(30),
    treaty_country_cd varchar(2),
    is_reportable bool DEFAULT false NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    superseded_by_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT fatca_crs_classification_pkey PRIMARY KEY (id),
    CONSTRAINT chk_fcc_regime CHECK (regime IN ('FATCA','CRS','UK_CDOT')),
    CONSTRAINT fk_fcc_party  FOREIGN KEY (party_id)  REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_fcc_status FOREIGN KEY (status_id) REFERENCES mdm.fatca_crs_status(id)
);
CREATE INDEX IF NOT EXISTS idx_fcc_party  ON mdm.fatca_crs_classification (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_fcc_tenant ON mdm.fatca_crs_classification (tenant_id);

-- ── party_consent ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_consent (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    consent_type varchar(50) NOT NULL,
    consent_version varchar(30) NOT NULL,
    consent_status varchar(20) NOT NULL,
    consent_basis varchar(30),
    granted_at timestamptz,
    withdrawn_at timestamptz,
    expires_at timestamptz,
    collection_method varchar(30),
    ip_address varchar(45),
    user_agent varchar(500),
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_consent_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pcon_status CHECK (consent_status IN (
        'GRANTED','WITHDRAWN','EXPIRED','PENDING','IMPLIED','NOT_REQUIRED')),
    CONSTRAINT chk_pcon_basis CHECK (consent_basis IS NULL OR consent_basis IN (
        'CONSENT','CONTRACT','LEGAL_OBLIGATION','VITAL_INTEREST',
        'PUBLIC_TASK','LEGITIMATE_INTEREST')),
    CONSTRAINT fk_pcon_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pcon_party  ON mdm.party_consent (party_id, consent_type, is_current);
CREATE INDEX IF NOT EXISTS idx_pcon_tenant ON mdm.party_consent (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'kyc_profile','kyc_document','sanctions_screening','sanctions_hit',
        'pep_screening','adverse_media_check','fatca_crs_classification',
        'party_consent'
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

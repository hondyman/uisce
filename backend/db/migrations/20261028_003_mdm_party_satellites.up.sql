-- Multi-valued, time-bounded party attributes.

-- ── party_name ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_name (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    name_type varchar(30) NOT NULL,
    name varchar(500) NOT NULL,
    first_name varchar(150),
    middle_name varchar(150),
    last_name varchar(150),
    prefix varchar(30),
    suffix varchar(30),
    language_cd varchar(10),
    script_cd varchar(20),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_primary bool DEFAULT false NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_name_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pname_type CHECK (name_type IN (
        'LEGAL','TRADING','DISPLAY','FORMER','DBA','NICKNAME','MAIDEN',
        'TRANSLITERATED','NATIVE_SCRIPT','SHORT','ABBREVIATED')),
    CONSTRAINT fk_pname_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pname_party  ON mdm.party_name (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pname_name   ON mdm.party_name (last_name, first_name);
CREATE INDEX IF NOT EXISTS idx_pname_tenant ON mdm.party_name (tenant_id);

-- ── party_identifier ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(100) NOT NULL,
    issuing_country_cd varchar(2),
    issuing_authority varchar(250),
    issue_date date,
    expiry_date date,
    is_primary bool DEFAULT false NOT NULL,
    is_verified bool DEFAULT false NOT NULL,
    verified_at timestamptz,
    verified_by uuid,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pident_type CHECK (id_type IN (
        'LEI','BIC','MIC','SEC_CIK','DUNS','TAX_ID_TIN','TAX_ID_EIN',
        'TAX_ID_SSN','TAX_ID_ITIN','TAX_ID_VAT','TAX_ID_NATIONAL',
        'PASSPORT','NATIONAL_ID','DRIVERS_LICENSE','RESIDENCE_PERMIT',
        'SOCIAL_SECURITY','GIIN','CRD','IARD','NFA_ID','FCA_REF',
        'COMPANY_REGISTRATION','CHARITY_NUMBER','TRUST_REGISTRATION',
        'CUSIP_ISSUER','ISIN_ISSUER','MORNINGSTAR_ID','CRM_ID','INTERNAL')),
    CONSTRAINT fk_pident_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pident_party  ON mdm.party_identifier (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pident_lookup ON mdm.party_identifier (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_pident_tenant ON mdm.party_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pident_active
    ON mdm.party_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── party_address ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_address (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    address_type varchar(30) NOT NULL,
    address_line_1 varchar(255),
    address_line_2 varchar(255),
    address_line_3 varchar(255),
    city varchar(100),
    state_province varchar(100),
    postal_code varchar(20),
    country_cd varchar(2),
    is_primary bool DEFAULT false NOT NULL,
    is_verified bool DEFAULT false NOT NULL,
    verified_at timestamptz,
    verified_method varchar(30),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_address_pkey PRIMARY KEY (id),
    CONSTRAINT chk_paddr_type CHECK (address_type IN (
        'RESIDENTIAL','MAILING','BUSINESS','REGISTERED','LEGAL',
        'OPERATIONS','SERVICE_OF_PROCESS','TAX','CORRESPONDENCE',
        'FORMER','TEMPORARY')),
    CONSTRAINT fk_paddr_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_paddr_party  ON mdm.party_address (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_paddr_tenant ON mdm.party_address (tenant_id);

-- ── party_contact ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_contact (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    contact_type varchar(30) NOT NULL,
    contact_value varchar(500) NOT NULL,
    country_code varchar(10),
    extension varchar(20),
    is_primary bool DEFAULT false NOT NULL,
    is_verified bool DEFAULT false NOT NULL,
    is_opted_in bool DEFAULT true NOT NULL,
    is_preferred bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_contact_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pcont_type CHECK (contact_type IN (
        'PHONE_MOBILE','PHONE_HOME','PHONE_WORK','PHONE_FAX',
        'EMAIL_PERSONAL','EMAIL_WORK','EMAIL_ALERTS',
        'WEB_URL','SOCIAL_LINKEDIN','SOCIAL_TWITTER','SECURE_MESSAGE',
        'VIDEO_CONFERENCE','WHATSAPP','WECHAT','TELEGRAM')),
    CONSTRAINT fk_pcont_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pcont_party  ON mdm.party_contact (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pcont_tenant ON mdm.party_contact (tenant_id);

-- ── party_individual ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_individual (
    party_id uuid NOT NULL,
    place_of_birth varchar(250),
    country_of_birth_cd varchar(2),
    gender varchar(20),
    marital_status varchar(20),
    marital_status_date date,
    secondary_nationality_cd varchar(2),
    dual_nationality bool DEFAULT false NOT NULL,
    employment_status varchar(20),
    occupation varchar(250),
    employer_name varchar(500),
    annual_income_usd numeric(24,4),
    net_worth_usd numeric(24,4),
    liquid_net_worth_usd numeric(24,4),
    is_politically_exposed bool DEFAULT false NOT NULL,
    is_family_member_of_pep bool DEFAULT false NOT NULL,
    is_close_associate_of_pep bool DEFAULT false NOT NULL,
    is_vulnerable_customer bool DEFAULT false NOT NULL,
    vulnerability_notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_individual_pkey PRIMARY KEY (party_id),
    CONSTRAINT chk_pind_gender CHECK (gender IS NULL OR gender IN (
        'MALE','FEMALE','NON_BINARY','UNDISCLOSED','OTHER')),
    CONSTRAINT fk_pind_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pind_tenant ON mdm.party_individual (tenant_id);

-- ── party_legal_entity ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_legal_entity (
    party_id uuid NOT NULL,
    legal_form varchar(50),
    jurisdiction_of_incorporation varchar(10),
    country_of_incorporation_cd varchar(2),
    dissolution_date date,
    registration_authority varchar(250),
    share_capital numeric(24,4),
    share_capital_currency varchar(3),
    fiscal_year_end varchar(10),
    legal_entity_identifier_registered bool DEFAULT false NOT NULL,
    sic_code varchar(10),
    naics_code varchar(10),
    gics_code varchar(20),
    industry_description varchar(500),
    employee_count int4,
    is_listed bool DEFAULT false NOT NULL,
    is_public bool DEFAULT false NOT NULL,
    is_non_profit bool DEFAULT false NOT NULL,
    is_financial_institution_type varchar(30),
    is_shell_company bool DEFAULT false NOT NULL,
    is_spv bool DEFAULT false NOT NULL,
    is_spac bool DEFAULT false NOT NULL,
    is_operating_company bool DEFAULT true NOT NULL,
    is_holding_company bool DEFAULT false NOT NULL,
    has_bearer_shares bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_legal_entity_pkey PRIMARY KEY (party_id),
    CONSTRAINT fk_ple_party FOREIGN KEY (party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ple_tenant ON mdm.party_legal_entity (tenant_id);

-- ── party_tax_residency ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_tax_residency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    party_id uuid NOT NULL,
    country_cd varchar(2) NOT NULL,
    residency_type varchar(30) NOT NULL,
    tin varchar(50),
    tin_issuing_country_cd varchar(2),
    tin_not_provided_reason varchar(30),
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    self_certified_date date,
    self_certification_document_id uuid,
    certified_by uuid,
    certified_at timestamptz,
    fatca_status_id uuid,
    crs_status_id uuid,
    treaty_article varchar(20),
    treaty_rate_pct numeric(7,4),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_tax_residency_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ptr_type CHECK (residency_type IN (
        'PRIMARY','SECONDARY','TREATY_TIE_BREAKER','CITIZENSHIP_BASED',
        'DOMICILE','RESIDENCE','PLACE_OF_MANAGEMENT')),
    CONSTRAINT fk_ptr_party  FOREIGN KEY (party_id)        REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_ptr_fatca  FOREIGN KEY (fatca_status_id) REFERENCES mdm.fatca_crs_status(id),
    CONSTRAINT fk_ptr_crs    FOREIGN KEY (crs_status_id)   REFERENCES mdm.fatca_crs_status(id)
);
CREATE INDEX IF NOT EXISTS idx_ptr_party   ON mdm.party_tax_residency (party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_ptr_country ON mdm.party_tax_residency (country_cd);
CREATE INDEX IF NOT EXISTS idx_ptr_tenant  ON mdm.party_tax_residency (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ptr_current
    ON mdm.party_tax_residency (tenant_id, party_id, country_cd, residency_type)
    WHERE effective_to IS NULL;

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'party_name','party_identifier','party_address','party_contact',
        'party_individual','party_legal_entity','party_tax_residency'
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

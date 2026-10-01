-- Reference/lookup tables for the party master. All use the two-policy RLS
-- pattern (tenant read + shared-reference read; tenant-only write).

-- ── party_type ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    sub_category varchar(30),
    is_individual bool DEFAULT false NOT NULL,
    is_legal_entity bool DEFAULT false NOT NULL,
    is_trust bool DEFAULT false NOT NULL,
    is_government bool DEFAULT false NOT NULL,
    is_fund bool DEFAULT false NOT NULL,
    is_spv bool DEFAULT false NOT NULL,
    requires_lei bool DEFAULT false NOT NULL,
    requires_kyc bool DEFAULT true NOT NULL,
    requires_tax_residency bool DEFAULT true NOT NULL,
    requires_ubo_lookup bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_type_pkey PRIMARY KEY (id),
    CONSTRAINT party_type_cd_key UNIQUE (tenant_id, type_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_type_tenant ON mdm.party_type (tenant_id);

-- ── party_sub_type ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_sub_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sub_type_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    parent_type_cd varchar(30) NOT NULL,
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_sub_type_pkey PRIMARY KEY (id),
    CONSTRAINT party_sub_type_cd_key UNIQUE (tenant_id, sub_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_sub_type_tenant ON mdm.party_sub_type (tenant_id);

-- ── party_status ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_active_status bool DEFAULT true NOT NULL,
    is_terminal bool DEFAULT false NOT NULL,
    requires_kyc_review bool DEFAULT true NOT NULL,
    is_blocked bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_status_pkey PRIMARY KEY (id),
    CONSTRAINT party_status_cd_key UNIQUE (tenant_id, status_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_status_lookup ON mdm.party_status (tenant_id);

-- ── party_segment ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_segment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    segment_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    segment_type varchar(30) NOT NULL,
    description text,
    min_aum_usd numeric(24,4),
    max_aum_usd numeric(24,4),
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_segment_pkey PRIMARY KEY (id),
    CONSTRAINT party_segment_cd_key UNIQUE (tenant_id, segment_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_segment_tenant ON mdm.party_segment (tenant_id);

-- ── party_role ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_role (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    role_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    role_category varchar(30) NOT NULL,
    description text,
    applies_to_entity_type varchar(30),
    requires_kyc bool DEFAULT true NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_role_pkey PRIMARY KEY (id),
    CONSTRAINT party_role_cd_key UNIQUE (tenant_id, role_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_role_tenant ON mdm.party_role (tenant_id);

-- ── party_relationship_type ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_relationship_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    relationship_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    category varchar(30) NOT NULL,
    is_directional bool DEFAULT true NOT NULL,
    inverse_relationship_cd varchar(30),
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_relationship_type_pkey PRIMARY KEY (id),
    CONSTRAINT party_relationship_type_cd_key UNIQUE (tenant_id, relationship_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_relationship_type_tenant ON mdm.party_relationship_type (tenant_id);

-- ── kyc_status ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.kyc_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    is_complete bool DEFAULT false NOT NULL,
    is_pending bool DEFAULT false NOT NULL,
    is_failed bool DEFAULT false NOT NULL,
    allows_transactions bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT kyc_status_pkey PRIMARY KEY (id),
    CONSTRAINT kyc_status_cd_key UNIQUE (tenant_id, status_cd)
);
CREATE INDEX IF NOT EXISTS idx_kyc_status_tenant ON mdm.kyc_status (tenant_id);

-- ── risk_rating ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.risk_rating (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    rating_type varchar(30) NOT NULL,
    rating_level int4 NOT NULL,
    review_frequency_months int4,
    requires_edd bool DEFAULT false NOT NULL,
    requires_senior_approval bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT risk_rating_pkey PRIMARY KEY (id),
    CONSTRAINT risk_rating_cd_key UNIQUE (tenant_id, rating_cd, rating_type)
);
CREATE INDEX IF NOT EXISTS idx_risk_rating_tenant ON mdm.risk_rating (tenant_id);

-- ── source_of_wealth_type ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.source_of_wealth_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    requires_documentation bool DEFAULT true NOT NULL,
    risk_weight numeric(5,2),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT source_of_wealth_type_pkey PRIMARY KEY (id),
    CONSTRAINT source_of_wealth_type_cd_key UNIQUE (tenant_id, source_cd)
);
CREATE INDEX IF NOT EXISTS idx_source_of_wealth_type_tenant ON mdm.source_of_wealth_type (tenant_id);

-- ── sanctions_list_source ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.sanctions_list_source (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    list_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    jurisdiction varchar(10),
    issuing_authority varchar(250),
    list_type varchar(30) NOT NULL,
    update_frequency varchar(20),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT sanctions_list_source_pkey PRIMARY KEY (id),
    CONSTRAINT sanctions_list_source_cd_key UNIQUE (tenant_id, list_cd)
);
CREATE INDEX IF NOT EXISTS idx_sanctions_list_source_tenant ON mdm.sanctions_list_source (tenant_id);

-- ── party_document_type ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_document_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    doc_type_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    category varchar(30) NOT NULL,
    applies_to varchar(30) NOT NULL,
    is_mandatory_for_kyc bool DEFAULT false NOT NULL,
    has_expiry bool DEFAULT true NOT NULL,
    validity_months int4,
    requires_certification bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_document_type_pkey PRIMARY KEY (id),
    CONSTRAINT party_document_type_cd_key UNIQUE (tenant_id, doc_type_cd)
);
CREATE INDEX IF NOT EXISTS idx_party_document_type_tenant ON mdm.party_document_type (tenant_id);

-- ── fatca_crs_status ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fatca_crs_status (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    status_cd varchar(30) NOT NULL,
    name varchar(150) NOT NULL,
    regime varchar(20) NOT NULL,
    description text,
    requires_reporting bool DEFAULT false NOT NULL,
    requires_withholding bool DEFAULT false NOT NULL,
    display_order int4 DEFAULT 0,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT fatca_crs_status_pkey PRIMARY KEY (id),
    CONSTRAINT fatca_crs_status_cd_key UNIQUE (tenant_id, status_cd, regime)
);
CREATE INDEX IF NOT EXISTS idx_fatca_crs_status_tenant ON mdm.fatca_crs_status (tenant_id);

-- ── RLS: apply the standard two-policy pattern to all 12 reference tables ─
DO $rls$
DECLARE
    t text;
    tables text[] := ARRAY[
        'party_type','party_sub_type','party_status','party_segment',
        'party_role','party_relationship_type','kyc_status','risk_rating',
        'source_of_wealth_type','sanctions_list_source','party_document_type',
        'fatca_crs_status'
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

-- 20261102_003_mdm_rating_specialized.up.sql
-- Specialized rating tables: internal, insurance, bank, fund, proprietary model.

-- ── rating_internal (house / proprietary ratings) ──────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_internal (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    internal_scale_cd varchar(50) NOT NULL,
    rated_party_type varchar(30) NOT NULL,
    rated_party_id uuid NOT NULL,
    internal_value varchar(20) NOT NULL,
    internal_rank int4,
    score numeric(8,3),
    score_band varchar(30),
    model_name varchar(100) NOT NULL,
    model_version varchar(30),
    analyst_id uuid,
    peer_group_id uuid,
    rationale text,
    review_frequency_months int4 DEFAULT 12,
    next_review_date date,
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_internal_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ri_party CHECK (rated_party_type IN (
        'ISSUER','ISSUE','COUNTERPARTY','SOVEREIGN','FUND','SPV','INTERNAL'))
);
CREATE INDEX IF NOT EXISTS idx_ri_party   ON mdm.rating_internal (rated_party_type, rated_party_id) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_ri_model   ON mdm.rating_internal (model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_ri_next    ON mdm.rating_internal (next_review_date) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_ri_tenant  ON mdm.rating_internal (tenant_id);

-- ── rating_insurance (AM Best / S&P / Moody's insurer) ─────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_insurance (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_id uuid,
    insurer_id uuid NOT NULL,
    agency_id uuid NOT NULL,
    financial_strength varchar(20) NOT NULL,
    fs_rank int4,
    credit_rating varchar(20),
    cr_rank int4,
    debt_rating varchar(20),
    long_term_outlook_id uuid,
    short_term_rating varchar(20),
    is_licensed bool DEFAULT false NOT NULL,
    naic_category int4,
    am_best_rating varchar(10),
    am_best_financial_size varchar(10),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_insurance_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ri_ins_rating FOREIGN KEY (rating_id)          REFERENCES mdm.rating(id) ON DELETE SET NULL,
    CONSTRAINT fk_ri_agency     FOREIGN KEY (agency_id)          REFERENCES mdm.rating_agency(id),
    CONSTRAINT fk_ri_outlook    FOREIGN KEY (long_term_outlook_id) REFERENCES mdm.rating_outlook(id),
    CONSTRAINT chk_ri_naic CHECK (naic_category IS NULL OR naic_category BETWEEN 1 AND 6)
);
CREATE INDEX IF NOT EXISTS idx_riins_insurer ON mdm.rating_insurance (insurer_id, is_active);
CREATE INDEX IF NOT EXISTS idx_riins_agency  ON mdm.rating_insurance (agency_id, is_active);
CREATE INDEX IF NOT EXISTS idx_riins_naic    ON mdm.rating_insurance (naic_category) WHERE naic_category IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_riins_tenant  ON mdm.rating_insurance (tenant_id);

-- ── rating_bank (bank / holdco ratings) ────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_bank (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_id uuid,
    bank_id uuid NOT NULL,
    agency_id uuid NOT NULL,
    bank_rating varchar(20) NOT NULL,
    bank_rank int4,
    holdco_rating varchar(20),
    holdco_rank int4,
    deposit_rating varchar(20),
    long_term_rating varchar(20),
    short_term_rating varchar(20),
    outlook_id uuid,
    baa_rating varchar(10),
    capital_ratio_pct numeric(6,3),
    asset_quality_grade int4,
    is_reg_capitalized bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_bank_pkey PRIMARY KEY (id),
    CONSTRAINT fk_rb_rating FOREIGN KEY (rating_id) REFERENCES mdm.rating(id) ON DELETE SET NULL,
    CONSTRAINT fk_rb_agency FOREIGN KEY (agency_id) REFERENCES mdm.rating_agency(id),
    CONSTRAINT fk_rb_outlook FOREIGN KEY (outlook_id) REFERENCES mdm.rating_outlook(id)
);
CREATE INDEX IF NOT EXISTS idx_rb_bank    ON mdm.rating_bank (bank_id, is_active);
CREATE INDEX IF NOT EXISTS idx_rb_agency  ON mdm.rating_bank (agency_id, is_active);
CREATE INDEX IF NOT EXISTS idx_rb_tenant  ON mdm.rating_bank (tenant_id);

-- ── rating_fund (fund / collective investment ratings) ─────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_fund (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rating_id uuid,
    fund_id uuid NOT NULL,
    agency_id uuid NOT NULL,
    fund_rating varchar(20) NOT NULL,
    fund_rank int4,
    share_class_rating varchar(20),
    category_rank int4,
    category_cd varchar(50),
    morningstar_overall int4,
    morningstar_sustainability int4,
    outlook_id uuid,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_fund_pkey PRIMARY KEY (id),
    CONSTRAINT fk_rf_rating FOREIGN KEY (rating_id) REFERENCES mdm.rating(id) ON DELETE SET NULL,
    CONSTRAINT fk_rf_agency FOREIGN KEY (agency_id) REFERENCES mdm.rating_agency(id),
    CONSTRAINT fk_rf_outlook FOREIGN KEY (outlook_id) REFERENCES mdm.rating_outlook(id),
    CONSTRAINT chk_rf_ms CHECK (morningstar_overall IS NULL OR morningstar_overall BETWEEN 1 AND 5)
);
CREATE INDEX IF NOT EXISTS idx_rf_fund    ON mdm.rating_fund (fund_id, is_active);
CREATE INDEX IF NOT EXISTS idx_rf_agency  ON mdm.rating_fund (agency_id, is_active);
CREATE INDEX IF NOT EXISTS idx_rf_tenant  ON mdm.rating_fund (tenant_id);

-- ── rating_proprietary (firm's own model output vs agency) ─────────────
CREATE TABLE IF NOT EXISTS mdm.rating_proprietary (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rated_party_type varchar(30) NOT NULL,
    rated_party_id uuid NOT NULL,
    prop_scale_cd varchar(50) NOT NULL,
    prop_value varchar(20) NOT NULL,
    prop_rank int4,
    implied_agency_value varchar(20),
    divergence_rank int4,
    model_cd varchar(50) NOT NULL,
    model_version varchar(30),
    input_as_of date NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    notes text,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_proprietary_pkey PRIMARY KEY (id),
    CONSTRAINT chk_rp_party CHECK (rated_party_type IN (
        'ISSUER','ISSUE','COUNTERPARTY','SOVEREIGN','FUND','SPV'))
);
CREATE INDEX IF NOT EXISTS idx_rp_party    ON mdm.rating_proprietary (rated_party_type, rated_party_id) WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_rp_model    ON mdm.rating_proprietary (model_cd, input_as_of DESC);
CREATE INDEX IF NOT EXISTS idx_rp_diverge  ON mdm.rating_proprietary (divergence_rank DESC NULLS LAST) WHERE is_current;
CREATE INDEX IF NOT EXISTS idx_rp_tenant   ON mdm.rating_proprietary (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'rating_internal','rating_insurance','rating_bank',
        'rating_fund','rating_proprietary'
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

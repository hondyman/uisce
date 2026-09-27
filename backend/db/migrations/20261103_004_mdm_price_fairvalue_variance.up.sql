-- 20261103_004_mdm_price_fairvalue_variance.up.sql

-- ── fair_value_classification ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fair_value_classification (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    fair_value_level_id uuid NOT NULL,
    as_of_date date NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    valuation_methodology varchar(100),
    valuation_technique varchar(100),
    unobservable_inputs jsonb,
    observable_inputs jsonb,
    significance_assessment text,
    transfer_in bool DEFAULT false NOT NULL,
    transfer_out bool DEFAULT false NOT NULL,
    transfer_date date,
    transfer_reason varchar(255),
    classification_source varchar(30),
    approved_by uuid,
    approved_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fair_value_classification_pkey PRIMARY KEY (id),
    CONSTRAINT fk_fvc_level FOREIGN KEY (fair_value_level_id) REFERENCES mdm.fair_value_level(id)
);
CREATE INDEX IF NOT EXISTS idx_fvc_entity ON mdm.fair_value_classification (price_entity_type, price_entity_id, is_current);
CREATE INDEX IF NOT EXISTS idx_fvc_level  ON mdm.fair_value_classification (fair_value_level_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_fvc_tenant ON mdm.fair_value_classification (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_fvc_current
    ON mdm.fair_value_classification (tenant_id, price_entity_type, price_entity_id)
    WHERE effective_to IS NULL;

-- ── valuation_model ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.valuation_model (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    model_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    model_type varchar(30) NOT NULL,
    asset_class_cd varchar(20),
    sec_sub_typ_cd varchar(30),
    methodology_description text,
    model_version varchar(30),
    validation_status varchar(20),
    validated_by varchar(100),
    validation_date date,
    approval_date date,
    requires_calibration bool DEFAULT false NOT NULL,
    calibration_frequency_id uuid,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT valuation_model_pkey PRIMARY KEY (id),
    CONSTRAINT valuation_model_cd_key UNIQUE (tenant_id, model_cd, model_version),
    CONSTRAINT chk_vm_type CHECK (model_type IN (
        'DCF','BLACK_SCHOLES','BINOMIAL','MONTE_CARLO','COMPARABLE',
        'MARKET_MULTIPLE','RECENT_TRANSACTION','NAV','APPRAISAL')),
    CONSTRAINT chk_vm_validation CHECK (validation_status IS NULL OR validation_status IN (
        'DRAFT','VALIDATED','APPROVED','IN_USE','RETIRED'))
);
CREATE INDEX IF NOT EXISTS idx_vm_type   ON mdm.valuation_model (model_type);
CREATE INDEX IF NOT EXISTS idx_vm_class  ON mdm.valuation_model (asset_class_cd, sec_sub_typ_cd);
CREATE INDEX IF NOT EXISTS idx_vm_tenant ON mdm.valuation_model (tenant_id);

-- ── valuation_input ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.valuation_input (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    valuation_model_id uuid,
    valuation_date date NOT NULL,
    input_cd varchar(100) NOT NULL,
    input_name varchar(250),
    input_value numeric(28,12),
    input_value_text varchar(500),
    input_unit varchar(30),
    is_unobservable bool DEFAULT false NOT NULL,
    observable_range_low numeric(28,12),
    observable_range_high numeric(28,12),
    sensitivity_to_fair_value numeric(28,12),
    source_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT valuation_input_pkey PRIMARY KEY (id),
    CONSTRAINT fk_vi_model  FOREIGN KEY (valuation_model_id) REFERENCES mdm.valuation_model(id),
    CONSTRAINT fk_vi_source FOREIGN KEY (source_id)          REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_vi_entity ON mdm.valuation_input (price_entity_type, price_entity_id, valuation_date);
CREATE INDEX IF NOT EXISTS idx_vi_input  ON mdm.valuation_input (input_cd, valuation_date);
CREATE INDEX IF NOT EXISTS idx_vi_tenant ON mdm.valuation_input (tenant_id);

-- ── valuation_sensitivity ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.valuation_sensitivity (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    valuation_date date NOT NULL,
    scenario_cd varchar(50) NOT NULL,
    scenario_description text,
    fair_value numeric(28,12) NOT NULL,
    change_from_base numeric(28,12),
    change_pct_from_base numeric(12,8),
    key_assumptions jsonb,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT valuation_sensitivity_pkey PRIMARY KEY (id),
    CONSTRAINT chk_vsens_scenario CHECK (scenario_cd IN (
        'BASE','UPSIDE','DOWNSIDE','STRESS','REASONABLE_POSSIBLE'))
);
CREATE INDEX IF NOT EXISTS idx_vsens_entity ON mdm.valuation_sensitivity (price_entity_type, price_entity_id, valuation_date);
CREATE INDEX IF NOT EXISTS idx_vsens_tenant ON mdm.valuation_sensitivity (tenant_id);

-- ── price_variance_threshold ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_variance_threshold (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_class_cd varchar(20) NOT NULL,
    sec_sub_typ_cd varchar(30),
    price_type_cd varchar(30),
    threshold_type varchar(20) NOT NULL,
    warning_threshold numeric(18,6),
    error_threshold numeric(18,6),
    critical_threshold numeric(18,6),
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_variance_threshold_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pvt_type CHECK (threshold_type IN (
        'ABSOLUTE','PERCENTAGE','BPS','YIELD_BPS'))
);
CREATE INDEX IF NOT EXISTS idx_pvt_class  ON mdm.price_variance_threshold (asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_pvt_tenant ON mdm.price_variance_threshold (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pvt
    ON mdm.price_variance_threshold (tenant_id, asset_class_cd,
                                      COALESCE(sec_sub_typ_cd, ''), COALESCE(price_type_cd, ''));

-- ── price_variance_event ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_variance_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_type_id uuid,
    price_date date NOT NULL,
    source_a_id uuid NOT NULL,
    source_b_id uuid NOT NULL,
    price_a numeric(28,12) NOT NULL,
    price_b numeric(28,12) NOT NULL,
    variance_absolute numeric(28,12),
    variance_pct numeric(12,8),
    variance_bps numeric(12,6),
    threshold_id uuid,
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolved_by uuid,
    resolved_at timestamptz,
    resolution_note text,
    selected_source_id uuid,
    selected_price numeric(28,12),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_variance_event_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pve_severity CHECK (severity IN ('INFO','WARNING','ERROR','CRITICAL')),
    CONSTRAINT chk_pve_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_pve_src_a FOREIGN KEY (source_a_id)        REFERENCES mdm.price_source(id),
    CONSTRAINT fk_pve_src_b FOREIGN KEY (source_b_id)        REFERENCES mdm.price_source(id),
    CONSTRAINT fk_pve_thresh FOREIGN KEY (threshold_id)      REFERENCES mdm.price_variance_threshold(id)
);
CREATE INDEX IF NOT EXISTS idx_pve_entity ON mdm.price_variance_event (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pve_status ON mdm.price_variance_event (status, severity);
CREATE INDEX IF NOT EXISTS idx_pve_tenant ON mdm.price_variance_event (tenant_id);

-- ── price_stale_event ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_stale_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_id uuid,
    source_id uuid,
    price_date date NOT NULL,
    last_valid_price_date date,
    staleness_days int4 NOT NULL,
    staleness_minutes int4,
    severity varchar(10) NOT NULL,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    resolved_by uuid,
    resolved_at timestamptz,
    resolution_note text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_stale_event_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pse_severity CHECK (severity IN ('INFO','WARNING','ERROR','CRITICAL')),
    CONSTRAINT chk_pse_status   CHECK (status IN ('OPEN','IN_REVIEW','RESOLVED','WAIVED')),
    CONSTRAINT fk_pse_price  FOREIGN KEY (price_id)  REFERENCES mdm.price(id),
    CONSTRAINT fk_pse_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pse_entity ON mdm.price_stale_event (price_entity_type, price_entity_id);
CREATE INDEX IF NOT EXISTS idx_pse_status ON mdm.price_stale_event (status, severity);
CREATE INDEX IF NOT EXISTS idx_pse_tenant ON mdm.price_stale_event (tenant_id);

-- ── price_challenge ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_challenge (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_date date NOT NULL,
    source_id uuid NOT NULL,
    challenged_price numeric(28,12) NOT NULL,
    proposed_price numeric(28,12) NOT NULL,
    challenge_reason text,
    challenge_evidence jsonb,
    status varchar(20) DEFAULT 'OPEN' NOT NULL,
    submitted_by uuid,
    submitted_at timestamptz,
    vendor_reference varchar(100),
    vendor_response text,
    resolution_date date,
    final_price numeric(28,12),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_challenge_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pch_status CHECK (status IN (
        'OPEN','SUBMITTED','ACCEPTED','REJECTED','WITHDRAWN','ESCALATED')),
    CONSTRAINT fk_pch_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pch_entity ON mdm.price_challenge (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pch_status ON mdm.price_challenge (status);
CREATE INDEX IF NOT EXISTS idx_pch_tenant ON mdm.price_challenge (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'fair_value_classification','valuation_model','valuation_input',
        'valuation_sensitivity','price_variance_threshold','price_variance_event',
        'price_stale_event','price_challenge'
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

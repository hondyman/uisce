-- 20261103_003_mdm_price_curves.up.sql

-- timestamptz::text is STABLE (depends on TimeZone GUC) and cannot appear in
-- index expressions. Epoch seconds are timezone-independent, so wrap them in
-- an IMMUTABLE function for use in the unique indexes below.
CREATE OR REPLACE FUNCTION mdm.immutable_tstz_text(t timestamptz)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$ SELECT trunc(extract(epoch FROM t))::bigint::text $$;

-- ── curve_master ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_master (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    curve_cd varchar(100) NOT NULL,
    name varchar(500) NOT NULL,
    short_name varchar(150),
    category_id uuid NOT NULL,
    source_id uuid,
    currency varchar(3),
    index_name varchar(100),
    tenor_standard varchar(30),
    interpolation_method_id uuid,
    compounding_frequency_id uuid,
    day_count varchar(30),
    business_day_conv varchar(30),
    calendar_cd varchar(10),
    spot_lag_days int4,
    curve_type varchar(30),
    is_bootstrapped bool DEFAULT false NOT NULL,
    bootstrap_method varchar(50),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    parent_curve_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_master_pkey PRIMARY KEY (id),
    CONSTRAINT curve_master_cd_key UNIQUE (tenant_id, curve_cd),
    CONSTRAINT chk_cm_type CHECK (curve_type IS NULL OR curve_type IN (
        'PAR','ZERO','FORWARD','DISCOUNT','SPOT')),
    CONSTRAINT fk_cm_category FOREIGN KEY (category_id)              REFERENCES mdm.curve_category(id),
    CONSTRAINT fk_cm_source   FOREIGN KEY (source_id)                REFERENCES mdm.price_source(id),
    CONSTRAINT fk_cm_interp   FOREIGN KEY (interpolation_method_id)  REFERENCES mdm.curve_interpolation_method(id),
    CONSTRAINT fk_cm_comp     FOREIGN KEY (compounding_frequency_id) REFERENCES mdm.curve_compounding_frequency(id),
    CONSTRAINT fk_cm_parent   FOREIGN KEY (parent_curve_id)          REFERENCES mdm.curve_master(id)
);
CREATE INDEX IF NOT EXISTS idx_cm_category ON mdm.curve_master (category_id);
CREATE INDEX IF NOT EXISTS idx_cm_currency ON mdm.curve_master (currency);
CREATE INDEX IF NOT EXISTS idx_cm_source   ON mdm.curve_master (source_id);
CREATE INDEX IF NOT EXISTS idx_cm_tenant   ON mdm.curve_master (tenant_id);

-- ── curve_point ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_point (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    curve_id uuid NOT NULL,
    curve_date date NOT NULL,
    curve_time timestamptz,
    tenor_cd varchar(30) NOT NULL,
    tenor_days int4,
    tenor_years numeric(12,8),
    rate numeric(20,12) NOT NULL,
    prior_rate numeric(20,12),
    change_bps numeric(12,6),
    observation_type_id uuid,
    quality_tier_id uuid,
    is_stale bool DEFAULT false NOT NULL,
    source_timestamp timestamptz,
    confidence numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_point_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cpt_curve FOREIGN KEY (curve_id)            REFERENCES mdm.curve_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_cpt_obs   FOREIGN KEY (observation_type_id) REFERENCES mdm.price_observation_type(id),
    CONSTRAINT fk_cpt_tier  FOREIGN KEY (quality_tier_id)     REFERENCES mdm.price_quality_tier(id)
);
CREATE INDEX IF NOT EXISTS idx_cpt_curve  ON mdm.curve_point (curve_id, curve_date, tenor_cd);
CREATE INDEX IF NOT EXISTS idx_cpt_date   ON mdm.curve_point (curve_date);
CREATE INDEX IF NOT EXISTS idx_cpt_tenant ON mdm.curve_point (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cpt
    ON mdm.curve_point (tenant_id, curve_id, curve_date, tenor_cd,
                         COALESCE(mdm.immutable_tstz_text(curve_time), ''));

-- ── curve_snapshot ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.curve_snapshot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    curve_id uuid NOT NULL,
    snapshot_date date NOT NULL,
    snapshot_time timestamptz,
    snapshot_type varchar(30) NOT NULL,
    point_count int4 NOT NULL,
    snapshot_hash varchar(64),
    is_official bool DEFAULT false NOT NULL,
    published_at timestamptz,
    source_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT curve_snapshot_pkey PRIMARY KEY (id),
    CONSTRAINT fk_csnap_curve  FOREIGN KEY (curve_id)  REFERENCES mdm.curve_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_csnap_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_csnap_curve  ON mdm.curve_snapshot (curve_id, snapshot_date);
CREATE INDEX IF NOT EXISTS idx_csnap_tenant ON mdm.curve_snapshot (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_csnap
    ON mdm.curve_snapshot (tenant_id, curve_id, snapshot_date,
                            COALESCE(mdm.immutable_tstz_text(snapshot_time), ''), snapshot_type);

-- ── vol_surface ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.vol_surface (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surface_cd varchar(100) NOT NULL,
    name varchar(500) NOT NULL,
    underlying_type varchar(30) NOT NULL,
    underlying_security_id uuid,
    underlying_index_cd varchar(50),
    currency varchar(3),
    surface_method varchar(50),
    model_params jsonb,
    is_calibrated bool DEFAULT false NOT NULL,
    calibration_date date,
    calibration_source_id uuid,
    calibration_rmse numeric(12,8),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT vol_surface_pkey PRIMARY KEY (id),
    CONSTRAINT vol_surface_cd_key UNIQUE (tenant_id, surface_cd),
    CONSTRAINT fk_vs_source FOREIGN KEY (calibration_source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_vs_underlying ON mdm.vol_surface (underlying_type, underlying_security_id);
CREATE INDEX IF NOT EXISTS idx_vs_currency   ON mdm.vol_surface (currency);
CREATE INDEX IF NOT EXISTS idx_vs_tenant     ON mdm.vol_surface (tenant_id);

-- ── vol_surface_point ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.vol_surface_point (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    surface_id uuid NOT NULL,
    surface_date date NOT NULL,
    surface_time timestamptz,
    expiry_tenor varchar(30) NOT NULL,
    expiry_days int4,
    strike_type varchar(20) NOT NULL,
    strike_value numeric(24,12) NOT NULL,
    implied_vol numeric(20,12) NOT NULL,
    prior_implied_vol numeric(20,12),
    change_bps numeric(12,6),
    quality_tier_id uuid,
    is_stale bool DEFAULT false NOT NULL,
    source_timestamp timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT vol_surface_point_pkey PRIMARY KEY (id),
    CONSTRAINT fk_vsp_surface FOREIGN KEY (surface_id) REFERENCES mdm.vol_surface(id) ON DELETE CASCADE,
    CONSTRAINT fk_vsp_tier    FOREIGN KEY (quality_tier_id) REFERENCES mdm.price_quality_tier(id)
);
CREATE INDEX IF NOT EXISTS idx_vsp_surface ON mdm.vol_surface_point (surface_id, surface_date, expiry_tenor);
CREATE INDEX IF NOT EXISTS idx_vsp_tenant  ON mdm.vol_surface_point (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_vsp
    ON mdm.vol_surface_point (tenant_id, surface_id, surface_date, expiry_tenor, strike_type, strike_value);

-- ── fx_rate_master ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.fx_rate_master (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    base_currency varchar(3) NOT NULL,
    quote_currency varchar(3) NOT NULL,
    rate_type varchar(30) NOT NULL,
    tenor_cd varchar(30),
    rate_date date NOT NULL,
    rate_time timestamptz,
    rate numeric(24,12) NOT NULL,
    bid numeric(24,12),
    ask numeric(24,12),
    mid numeric(24,12),
    prior_rate numeric(24,12),
    change_pct numeric(12,8),
    source_id uuid,
    quality_tier_id uuid,
    fixing_source varchar(50),
    is_official bool DEFAULT false NOT NULL,
    is_stale bool DEFAULT false NOT NULL,
    source_timestamp timestamptz,
    confidence numeric(5,2),
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fx_rate_master_pkey PRIMARY KEY (id),
    CONSTRAINT chk_fxrm_type CHECK (rate_type IN (
        'SPOT','FORWARD','SWAP','FIXING','NDF','INDICATIVE')),
    CONSTRAINT fk_fxrm_source FOREIGN KEY (source_id)       REFERENCES mdm.price_source(id),
    CONSTRAINT fk_fxrm_tier   FOREIGN KEY (quality_tier_id) REFERENCES mdm.price_quality_tier(id)
);
CREATE INDEX IF NOT EXISTS idx_fxrm_pair     ON mdm.fx_rate_master (base_currency, quote_currency, rate_date);
CREATE INDEX IF NOT EXISTS idx_fxrm_current  ON mdm.fx_rate_master (base_currency, quote_currency, is_current);
CREATE INDEX IF NOT EXISTS idx_fxrm_official ON mdm.fx_rate_master (is_official) WHERE is_official = true;
CREATE INDEX IF NOT EXISTS idx_fxrm_tenant   ON mdm.fx_rate_master (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_fxrm_current
    ON mdm.fx_rate_master (tenant_id, base_currency, quote_currency, rate_type,
                            COALESCE(tenor_cd, ''), rate_date)
    WHERE is_current = true;

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'curve_master','curve_point','curve_snapshot',
        'vol_surface','vol_surface_point','fx_rate_master'
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

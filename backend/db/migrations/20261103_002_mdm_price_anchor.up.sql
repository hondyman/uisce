-- 20261103_002_mdm_price_anchor.up.sql

-- ── price ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_type_id uuid NOT NULL,
    source_id uuid NOT NULL,
    observation_type_id uuid,
    quality_tier_id uuid,
    fair_value_level_id uuid,
    price_date date NOT NULL,
    price_time timestamptz,
    value numeric(28,12) NOT NULL,
    prior_value numeric(28,12),
    change_pct numeric(12,8),
    currency varchar(3),
    quote_basis varchar(20),
    quantity numeric(24,4),
    price_unit varchar(20),
    is_official bool DEFAULT false NOT NULL,
    is_executable bool DEFAULT false NOT NULL,
    is_stale bool DEFAULT false NOT NULL,
    is_indicative bool DEFAULT false NOT NULL,
    is_adjusted bool DEFAULT false NOT NULL,
    staleness_minutes int4,
    source_timestamp timestamptz,
    confidence numeric(5,2),
    source_system_id uuid,
    predecessor_price_id uuid,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_pkey PRIMARY KEY (id),
    CONSTRAINT chk_price_entity_type CHECK (price_entity_type IN (
        'SECURITY','CURVE','SURFACE','INDEX','FX_RATE','DERIVED')),
    CONSTRAINT fk_price_type   FOREIGN KEY (price_type_id)       REFERENCES mdm.price_type(id),
    CONSTRAINT fk_price_source FOREIGN KEY (source_id)           REFERENCES mdm.price_source(id),
    CONSTRAINT fk_price_obs    FOREIGN KEY (observation_type_id) REFERENCES mdm.price_observation_type(id),
    CONSTRAINT fk_price_tier   FOREIGN KEY (quality_tier_id)     REFERENCES mdm.price_quality_tier(id),
    CONSTRAINT fk_price_fvl    FOREIGN KEY (fair_value_level_id) REFERENCES mdm.fair_value_level(id)
);
CREATE INDEX IF NOT EXISTS idx_price_entity   ON mdm.price (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_price_current  ON mdm.price (price_entity_type, price_entity_id, is_current);
CREATE INDEX IF NOT EXISTS idx_price_source   ON mdm.price (source_id, price_date);
CREATE INDEX IF NOT EXISTS idx_price_official ON mdm.price (is_official) WHERE is_official = true;
CREATE INDEX IF NOT EXISTS idx_price_stale    ON mdm.price (is_stale) WHERE is_stale = true;
CREATE INDEX IF NOT EXISTS idx_price_tenant   ON mdm.price (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_price_current
    ON mdm.price (tenant_id, price_entity_type, price_entity_id, price_type_id, source_id, price_date)
    WHERE is_current = true;

-- ── price_history (SCD2) ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_id uuid NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_type_id uuid NOT NULL,
    source_id uuid NOT NULL,
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
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_history_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ph_version UNIQUE (tenant_id, price_id, version_num),
    CONSTRAINT fk_ph_price FOREIGN KEY (price_id) REFERENCES mdm.price(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ph_entity ON mdm.price_history (price_entity_type, price_entity_id, is_current);
CREATE INDEX IF NOT EXISTS idx_ph_valid  ON mdm.price_history (price_id, valid_from);
CREATE INDEX IF NOT EXISTS idx_ph_tenant ON mdm.price_history (tenant_id);

-- ── price_series (denormalized for fast analytics) ────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_series (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid NOT NULL,
    price_type_id uuid NOT NULL,
    source_id uuid,
    price_date date NOT NULL,
    price_time timestamptz,
    open_value numeric(28,12),
    high_value numeric(28,12),
    low_value numeric(28,12),
    close_value numeric(28,12),
    vwap_value numeric(28,12),
    volume numeric(28,4),
    trade_count int4,
    daily_return numeric(18,12),
    cumulative_return numeric(18,12),
    currency varchar(3),
    quality_tier_id uuid,
    is_finalized bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_series_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pser_ptype FOREIGN KEY (price_type_id)   REFERENCES mdm.price_type(id),
    CONSTRAINT fk_pser_src   FOREIGN KEY (source_id)       REFERENCES mdm.price_source(id),
    CONSTRAINT fk_pser_tier  FOREIGN KEY (quality_tier_id) REFERENCES mdm.price_quality_tier(id)
);
CREATE INDEX IF NOT EXISTS idx_pser_entity ON mdm.price_series (price_entity_type, price_entity_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pser_source ON mdm.price_series (source_id, price_date);
CREATE INDEX IF NOT EXISTS idx_pser_tenant ON mdm.price_series (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pser
    ON mdm.price_series (tenant_id, price_entity_type, price_entity_id, price_type_id,
                          COALESCE(source_id::text, ''), price_date);

-- ── price_vendor_symbol ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.price_vendor_symbol (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    price_entity_type varchar(30) NOT NULL,
    price_entity_id uuid,
    curve_id uuid,
    surface_id uuid,
    source_id uuid NOT NULL,
    vendor_symbol varchar(200) NOT NULL,
    vendor_field varchar(100),
    vendor_type varchar(30),
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT price_vendor_symbol_pkey PRIMARY KEY (id),
    CONSTRAINT fk_pvs_source FOREIGN KEY (source_id) REFERENCES mdm.price_source(id)
);
CREATE INDEX IF NOT EXISTS idx_pvs_entity ON mdm.price_vendor_symbol (price_entity_type, price_entity_id);
CREATE INDEX IF NOT EXISTS idx_pvs_lookup ON mdm.price_vendor_symbol (source_id, vendor_symbol);
CREATE INDEX IF NOT EXISTS idx_pvs_tenant ON mdm.price_vendor_symbol (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pvs
    ON mdm.price_vendor_symbol (tenant_id, source_id, vendor_symbol, COALESCE(vendor_field, ''));

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY['price','price_history','price_series','price_vendor_symbol'];
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

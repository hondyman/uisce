-- 20261101_002_mdm_benchmark_anchor.up.sql

-- ── benchmark_master ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_master (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    short_name varchar(150),
    legal_name varchar(500),
    provider_id uuid NOT NULL,
    benchmark_type_id uuid NOT NULL,
    return_variant_id uuid,
    currency_variant_id uuid,
    base_currency varchar(3) NOT NULL,
    asset_class_cd varchar(20),
    region_cd varchar(30),
    country_cd varchar(2),
    weighting_method_id uuid,
    rebalance_frequency_id uuid,
    inception_date date,
    termination_date date,
    base_value numeric(18,4),
    base_date date,
    divisor numeric(24,9),
    is_investable bool DEFAULT false NOT NULL,
    is_publicly_available bool DEFAULT false NOT NULL,
    is_custom bool DEFAULT false NOT NULL,
    is_esg bool DEFAULT false NOT NULL,
    is_sharia_compliant bool DEFAULT false NOT NULL,
    is_net_zero_aligned bool DEFAULT false NOT NULL,
    is_regulated bool DEFAULT false NOT NULL,
    is_significant_benchmark bool DEFAULT false NOT NULL,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    status_reason varchar(500),
    discontinuation_date date,
    successor_benchmark_id uuid,
    methodology_url text,
    factsheet_url text,
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_master_pkey PRIMARY KEY (id),
    CONSTRAINT benchmark_master_cd_key UNIQUE (tenant_id, benchmark_cd),
    CONSTRAINT chk_bm_status CHECK (status IN (
        'ACTIVE','DORMANT','DISCONTINUED','SUSPENDED','MERGED','REPLACED')),
    CONSTRAINT fk_bm_provider      FOREIGN KEY (provider_id)             REFERENCES mdm.benchmark_provider(id),
    CONSTRAINT fk_bm_type          FOREIGN KEY (benchmark_type_id)       REFERENCES mdm.benchmark_type(id),
    CONSTRAINT fk_bm_return_var    FOREIGN KEY (return_variant_id)       REFERENCES mdm.benchmark_return_variant(id),
    CONSTRAINT fk_bm_currency_var  FOREIGN KEY (currency_variant_id)     REFERENCES mdm.benchmark_currency_variant(id),
    CONSTRAINT fk_bm_weight_method FOREIGN KEY (weighting_method_id)     REFERENCES mdm.benchmark_weighting_method(id),
    CONSTRAINT fk_bm_rebal_freq    FOREIGN KEY (rebalance_frequency_id)  REFERENCES mdm.benchmark_rebalance_frequency(id),
    CONSTRAINT fk_bm_successor     FOREIGN KEY (successor_benchmark_id)  REFERENCES mdm.benchmark_master(id),
    CONSTRAINT fk_bm_source        FOREIGN KEY (source_system_id)        REFERENCES mdm.source_systems(id),
    CONSTRAINT fk_bm_merged        FOREIGN KEY (merged_into_id)          REFERENCES mdm.benchmark_master(id)
);
CREATE INDEX IF NOT EXISTS idx_bm_provider ON mdm.benchmark_master (provider_id);
CREATE INDEX IF NOT EXISTS idx_bm_type     ON mdm.benchmark_master (benchmark_type_id);
CREATE INDEX IF NOT EXISTS idx_bm_asset    ON mdm.benchmark_master (asset_class_cd);
CREATE INDEX IF NOT EXISTS idx_bm_region   ON mdm.benchmark_master (region_cd);
CREATE INDEX IF NOT EXISTS idx_bm_status   ON mdm.benchmark_master (status);
CREATE INDEX IF NOT EXISTS idx_bm_esg      ON mdm.benchmark_master (is_esg) WHERE is_esg = true;
CREATE INDEX IF NOT EXISTS idx_bm_tenant   ON mdm.benchmark_master (tenant_id);

-- ── benchmark_identifier ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    benchmark_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(100) NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bid_type CHECK (id_type IN (
        'BBG_TICKER','RIC','ISIN','SEDOL','CUSIP','FIGI','WKN',
        'PROVIDER_CODE','ESMA_ISIN','LEI','INTERNAL','MORNINGSTAR_ID',
        'LIPPER_ID','EVESTMENT_ID')),
    CONSTRAINT fk_bid_benchmark FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_bid_benchmark ON mdm.benchmark_identifier (benchmark_id, is_current);
CREATE INDEX IF NOT EXISTS idx_bid_lookup    ON mdm.benchmark_identifier (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_bid_tenant    ON mdm.benchmark_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bid_active
    ON mdm.benchmark_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── benchmark_hierarchy ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_hierarchy (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_benchmark_id uuid NOT NULL,
    child_benchmark_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    relationship_description varchar(255),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_hierarchy_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bh_no_self CHECK (parent_benchmark_id <> child_benchmark_id),
    CONSTRAINT chk_bh_type CHECK (hierarchy_type IN (
        'FAMILY','UMBRELLA','SUB_INDEX','SECTOR_BREAKOUT',
        'COUNTRY_BREAKOUT','CURRENCY_VARIANT','RETURN_VARIANT',
        'ESG_VARIANT','HEDGED_VARIANT','SIZE_BREAKOUT','STYLE_BREAKOUT')),
    CONSTRAINT fk_bh_parent FOREIGN KEY (parent_benchmark_id) REFERENCES mdm.benchmark_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_bh_child  FOREIGN KEY (child_benchmark_id)  REFERENCES mdm.benchmark_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_bh_parent  ON mdm.benchmark_hierarchy (parent_benchmark_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_bh_child   ON mdm.benchmark_hierarchy (child_benchmark_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_bh_current ON mdm.benchmark_hierarchy (is_current);
CREATE INDEX IF NOT EXISTS idx_bh_tenant  ON mdm.benchmark_hierarchy (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bh_active_edge
    ON mdm.benchmark_hierarchy (tenant_id, parent_benchmark_id, child_benchmark_id, hierarchy_type)
    WHERE effective_to IS NULL;

-- ── benchmark_hierarchy_closure ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.benchmark_hierarchy_closure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ancestor_benchmark_id uuid NOT NULL,
    descendant_benchmark_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    depth int4 NOT NULL,
    path text,
    computed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT benchmark_hierarchy_closure_pkey PRIMARY KEY (id),
    CONSTRAINT fk_bhc_anc  FOREIGN KEY (ancestor_benchmark_id)   REFERENCES mdm.benchmark_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_bhc_desc FOREIGN KEY (descendant_benchmark_id) REFERENCES mdm.benchmark_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_bhc_anc    ON mdm.benchmark_hierarchy_closure (ancestor_benchmark_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_bhc_desc   ON mdm.benchmark_hierarchy_closure (descendant_benchmark_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_bhc_tenant ON mdm.benchmark_hierarchy_closure (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bhc_edge
    ON mdm.benchmark_hierarchy_closure (tenant_id, ancestor_benchmark_id, descendant_benchmark_id, hierarchy_type);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'benchmark_master','benchmark_identifier',
        'benchmark_hierarchy','benchmark_hierarchy_closure'
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

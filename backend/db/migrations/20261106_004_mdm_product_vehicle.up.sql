-- 20261106_004_mdm_product_vehicle.up.sql

-- ── product_vehicle ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_vehicle (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    vehicle_type varchar(30) NOT NULL,
    vehicle_entity_type varchar(30) NOT NULL,
    vehicle_entity_id uuid,
    umbrella_fund_id uuid,
    legal_entity_id uuid,
    role varchar(30) NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_vehicle_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_veh_type CHECK (vehicle_type IN (
        'FUND','TRUST','SERIES','SUB_FUND','UMBRELLA','SEPARATE_ACCOUNT',
        'INSURANCE_COMPANY','GENERAL_ACCOUNT','SPV','LLC','LP','PARTNERSHIP')),
    CONSTRAINT chk_prd_veh_entity CHECK (vehicle_entity_type IN (
        'FUND','ISSUER','LEGAL_ENTITY')),
    CONSTRAINT chk_prd_veh_role CHECK (role IN (
        'PRIMARY_VEHICLE','RELATED_VEHICLE','FEEDER','MASTER','SUB_FUND')),
    CONSTRAINT fk_prd_veh_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_veh_product ON mdm.product_vehicle (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_veh_fund    ON mdm.product_vehicle (umbrella_fund_id);
CREATE INDEX IF NOT EXISTS idx_prd_veh_tenant  ON mdm.product_vehicle (tenant_id);

-- Guarded FK to orm.fund
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='orm' AND table_name='fund') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='product_vehicle' AND c.conname='fk_prd_veh_fund'
        ) THEN
            ALTER TABLE mdm.product_vehicle
                ADD CONSTRAINT fk_prd_veh_fund
                FOREIGN KEY (umbrella_fund_id) REFERENCES orm.fund(id);
        END IF;
    END IF;
END $fk$;

-- ── product_security ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_security (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    security_id uuid,
    security_identifier varchar(100),
    security_role varchar(30) NOT NULL,
    share_class_cd varchar(30),
    is_primary bool DEFAULT false NOT NULL,
    is_listed bool DEFAULT false NOT NULL,
    is_etf bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_security_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_sec_role CHECK (security_role IN (
        'SHARE_CLASS','PRIMARY_SECURITY','LISTED_VEHICLE','REFERENCE',
        'UNDERLYING','HEDGE_INSTRUMENT')),
    CONSTRAINT fk_prd_sec_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_sec_product ON mdm.product_security (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_sec_sec     ON mdm.product_security (security_id);
CREATE INDEX IF NOT EXISTS idx_prd_sec_tenant  ON mdm.product_security (tenant_id);

-- Guarded FK to orm.security
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='orm' AND table_name='security') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='product_security' AND c.conname='fk_prd_sec_security'
        ) THEN
            ALTER TABLE mdm.product_security
                ADD CONSTRAINT fk_prd_sec_security
                FOREIGN KEY (security_id) REFERENCES orm.security(id);
        END IF;
    END IF;
END $fk$;

-- ── product_benchmark ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_benchmark (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    benchmark_id uuid NOT NULL,
    role varchar(30) NOT NULL,
    weight numeric(7,4) DEFAULT 100.00 NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_benchmark_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_bmk_role CHECK (role IN (
        'PRIMARY','SECONDARY','REGULATORY','MARKETING','PEER','RISK')),
    CONSTRAINT fk_prd_bmk_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_bmk_product ON mdm.product_benchmark (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_bmk_bench   ON mdm.product_benchmark (benchmark_id);
CREATE INDEX IF NOT EXISTS idx_prd_bmk_tenant  ON mdm.product_benchmark (tenant_id);

DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='benchmark_master') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class cl ON cl.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = cl.relnamespace
            WHERE n.nspname='mdm' AND cl.relname='product_benchmark' AND c.conname='fk_prd_bmk_benchmark'
        ) THEN
            ALTER TABLE mdm.product_benchmark
                ADD CONSTRAINT fk_prd_bmk_benchmark
                FOREIGN KEY (benchmark_id) REFERENCES mdm.benchmark_master(id);
        END IF;
    END IF;
END $fk$;

-- ── product_strategy ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_strategy (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    strategy_cd varchar(50) NOT NULL,
    strategy_name varchar(250),
    strategy_type varchar(30),
    asset_class_focus varchar(20),
    geographic_focus varchar(30),
    sector_focus jsonb,
    market_cap_focus varchar(20),
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_strategy_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prd_strat_type CHECK (strategy_type IS NULL OR strategy_type IN (
        'ACTIVE','PASSIVE','ENHANCED_INDEX','QUANT','FUNDAMENTAL','ESG',
        'LONG_ONLY','LONG_SHORT','MARKET_NEUTRAL','ARBITRAGE')),
    CONSTRAINT fk_prd_strat_product FOREIGN KEY (product_id) REFERENCES mdm.product(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_prd_strat_product ON mdm.product_strategy (product_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prd_strat_tenant  ON mdm.product_strategy (tenant_id);

-- ── product_lifecycle_event ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.product_lifecycle_event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    product_id uuid NOT NULL,
    event_type_id uuid,
    event_cd varchar(30),
    event_date date NOT NULL,
    effective_date date,
    reason varchar(500),
    announcement_ref varchar(100),
    board_approval_date date,
    board_approval_ref varchar(100),
    related_product_id uuid,
    related_document_id uuid,
    approved_by uuid,
    approved_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT product_lifecycle_event_pkey PRIMARY KEY (id),
    CONSTRAINT fk_prd_lc_product    FOREIGN KEY (product_id)         REFERENCES mdm.product(id) ON DELETE CASCADE,
    CONSTRAINT fk_prd_lc_type       FOREIGN KEY (event_type_id)      REFERENCES mdm.product_lifecycle_event_type(id),
    CONSTRAINT fk_prd_lc_related    FOREIGN KEY (related_product_id) REFERENCES mdm.product(id),
    CONSTRAINT fk_prd_lc_doc        FOREIGN KEY (related_document_id) REFERENCES mdm.product_document(id)
);
CREATE INDEX IF NOT EXISTS idx_prd_lc_product ON mdm.product_lifecycle_event (product_id, event_date);
CREATE INDEX IF NOT EXISTS idx_prd_lc_type    ON mdm.product_lifecycle_event (event_cd, event_date);
CREATE INDEX IF NOT EXISTS idx_prd_lc_tenant  ON mdm.product_lifecycle_event (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'product_vehicle','product_security','product_benchmark',
        'product_strategy','product_lifecycle_event'
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

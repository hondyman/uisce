-- 20261102_001_mdm_rating_reference.up.sql
-- Rating reference tables. mdm.rating_scale does not exist in this DB, so it is
-- created here (with a guarded ALTER for environments where it pre-exists).

-- ── rating_agency ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_agency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_cd varchar(30) NOT NULL,
    name varchar(250) NOT NULL,
    short_name varchar(50),
    agency_type varchar(30) NOT NULL,
    domicile varchar(2),
    parent_agency_id uuid,
    is_nrsro bool DEFAULT false NOT NULL,
    is_ecai bool DEFAULT false NOT NULL,
    is_naic_acceptable bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    registration_number varchar(50),
    registration_authority varchar(100),
    website varchar(250),
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_agency_pkey PRIMARY KEY (id),
    CONSTRAINT rating_agency_cd_key UNIQUE (tenant_id, agency_cd),
    CONSTRAINT chk_ra_type CHECK (agency_type IN (
        'NRSRO','ECAI','NATIONAL','REGIONAL','INTERNAL','MODEL','MARKET_IMPLIED')),
    CONSTRAINT chk_ra_status CHECK (status IN (
        'ACTIVE','DORMANT','WITHDRAWN','SUSPENDED')),
    CONSTRAINT fk_ra_parent FOREIGN KEY (parent_agency_id) REFERENCES mdm.rating_agency(id)
);
CREATE INDEX IF NOT EXISTS idx_ra_type   ON mdm.rating_agency (agency_type);
CREATE INDEX IF NOT EXISTS idx_ra_active ON mdm.rating_agency (is_active, is_nrsro);
CREATE INDEX IF NOT EXISTS idx_ra_tenant ON mdm.rating_agency (tenant_id);

-- ── rating_type ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd varchar(50) NOT NULL,
    name varchar(150) NOT NULL,
    applies_to varchar(30) NOT NULL,
    rating_basis varchar(30),
    description text,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_type_pkey PRIMARY KEY (id),
    CONSTRAINT rating_type_cd_key UNIQUE (tenant_id, type_cd),
    CONSTRAINT chk_rt_applies CHECK (applies_to IN (
        'ISSUER','ISSUE','TRANCHES','COUNTERPARTY','SOVEREIGN',
        'SUB_SOVEREIGN','FINANCIAL_STRENGTH','INSURER','BANK','FUND'))
);
CREATE INDEX IF NOT EXISTS idx_rt_tenant ON mdm.rating_type (tenant_id);

-- ── rating_action_type ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_action_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    action_cd varchar(30) NOT NULL,
    name varchar(100) NOT NULL,
    direction varchar(20) NOT NULL,
    is_credit_event bool DEFAULT false NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_action_type_pkey PRIMARY KEY (id),
    CONSTRAINT rating_action_type_cd_key UNIQUE (tenant_id, action_cd),
    CONSTRAINT chk_rat_direction CHECK (direction IN (
        'UPGRADE','DOWNGRADE','AFFIRMATION','INITIAL','WITHDRAWN',
        'PLACED_ON_WATCH','REMOVED_FROM_WATCH','DEFAULT','CURE','OTHER'))
);
CREATE INDEX IF NOT EXISTS idx_rat_tenant ON mdm.rating_action_type (tenant_id);

-- ── rating_outlook ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_outlook (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    outlook_cd varchar(20) NOT NULL,
    name varchar(50) NOT NULL,
    direction varchar(20) NOT NULL,
    horizon_months int4,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_outlook_pkey PRIMARY KEY (id),
    CONSTRAINT rating_outlook_cd_key UNIQUE (tenant_id, outlook_cd),
    CONSTRAINT chk_ro_direction CHECK (direction IN (
        'POSITIVE','NEGATIVE','STABLE','DEVELOPING','NEUTRAL'))
);
CREATE INDEX IF NOT EXISTS idx_ro_tenant ON mdm.rating_outlook (tenant_id);

-- ── rating_watch ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_watch (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    watch_cd varchar(30) NOT NULL,
    name varchar(50) NOT NULL,
    direction varchar(20) NOT NULL,
    horizon_days int4,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_watch_pkey PRIMARY KEY (id),
    CONSTRAINT rating_watch_cd_key UNIQUE (tenant_id, watch_cd),
    CONSTRAINT chk_rw_direction CHECK (direction IN (
        'POSITIVE','NEGATIVE','DEVELOPING','EVOLVING'))
);
CREATE INDEX IF NOT EXISTS idx_rw_tenant ON mdm.rating_watch (tenant_id);

-- ── rating_scale_map (agency-to-agency equivalence) ────────────────────
CREATE TABLE IF NOT EXISTS mdm.rating_scale_map (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    from_agency_cd varchar(30) NOT NULL,
    from_scale_cd varchar(50) NOT NULL,
    from_value varchar(20) NOT NULL,
    to_agency_cd varchar(30) NOT NULL,
    to_scale_cd varchar(50) NOT NULL,
    to_value varchar(20) NOT NULL,
    mapping_type varchar(20) NOT NULL,
    confidence numeric(5,2) DEFAULT 100.00,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_scale_map_pkey PRIMARY KEY (id),
    CONSTRAINT chk_rsm_type CHECK (mapping_type IN (
        'EXACT','APPROXIMATE','CONSERVATIVE','REGULATORY'))
);
CREATE INDEX IF NOT EXISTS idx_rsm_from   ON mdm.rating_scale_map (from_agency_cd, from_value);
CREATE INDEX IF NOT EXISTS idx_rsm_to     ON mdm.rating_scale_map (to_agency_cd, to_value);
CREATE INDEX IF NOT EXISTS idx_rsm_tenant ON mdm.rating_scale_map (tenant_id);

-- ── rating_scale (create if missing; ALTER guarded if pre-existing) ────
CREATE TABLE IF NOT EXISTS mdm.rating_scale (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_id uuid,
    agency_cd varchar(30),
    scale_cd varchar(50) NOT NULL,
    scale_name varchar(150),
    agency_scale_type varchar(30),
    value varchar(20) NOT NULL,
    rank_no int4,
    rating_category varchar(20),
    is_not_rated bool DEFAULT false,
    is_withdrawn bool DEFAULT false,
    is_na bool DEFAULT false,
    display_order int4 DEFAULT 0,
    effective_from date DEFAULT CURRENT_DATE,
    effective_to date,
    is_active bool DEFAULT true,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_scale_pkey PRIMARY KEY (id),
    CONSTRAINT rating_scale_scale_value_key UNIQUE (tenant_id, scale_cd, value)
);

DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='rating_scale') THEN
        ALTER TABLE mdm.rating_scale
            ADD COLUMN IF NOT EXISTS agency_id         uuid,
            ADD COLUMN IF NOT EXISTS agency_scale_type varchar(30),
            ADD COLUMN IF NOT EXISTS rating_category   varchar(20),
            ADD COLUMN IF NOT EXISTS is_not_rated      bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS is_withdrawn      bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS is_na             bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS display_order     int4 DEFAULT 0,
            ADD COLUMN IF NOT EXISTS effective_from    date DEFAULT CURRENT_DATE,
            ADD COLUMN IF NOT EXISTS effective_to      date,
            ADD COLUMN IF NOT EXISTS is_active         bool DEFAULT true;
        CREATE INDEX IF NOT EXISTS idx_rs_agency_id
            ON mdm.rating_scale (agency_id) WHERE agency_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_rs_category
            ON mdm.rating_scale (rating_category) WHERE rating_category IS NOT NULL;
    END IF;
END $do$;

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'rating_agency','rating_type','rating_action_type',
        'rating_outlook','rating_watch','rating_scale','rating_scale_map'
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

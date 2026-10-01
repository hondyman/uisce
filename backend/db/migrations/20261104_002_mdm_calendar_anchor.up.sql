-- 20261104_002_mdm_calendar_anchor.up.sql

-- ── calendar_master ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_master (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    short_name varchar(150),
    calendar_type_id uuid NOT NULL,
    primary_source_id uuid,
    primary_time_zone_id uuid,
    country_cd varchar(2),
    subdivision_cd varchar(10),
    city_cd varchar(20),
    mic varchar(4),
    currency_cd varchar(3),
    parent_calendar_id uuid,
    hierarchy_type_id uuid,
    inherits_parent bool DEFAULT false NOT NULL,
    week_start_day int4 DEFAULT 1 NOT NULL,
    weekend_days int4[] DEFAULT ARRAY[6,7] NOT NULL,
    is_regulatory_mandated bool DEFAULT false NOT NULL,
    regulatory_regime_cd varchar(50),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    replaced_by_calendar_id uuid,
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_master_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cal_master_cd UNIQUE (tenant_id, calendar_cd),
    CONSTRAINT chk_cal_master_status CHECK (status IN (
        'ACTIVE','DORMANT','DISCONTINUED','REPLACED','SUSPENDED')),
    CONSTRAINT fk_cal_type     FOREIGN KEY (calendar_type_id)    REFERENCES mdm.calendar_type(id),
    CONSTRAINT fk_cal_src      FOREIGN KEY (primary_source_id)   REFERENCES mdm.calendar_source(id),
    CONSTRAINT fk_cal_tz       FOREIGN KEY (primary_time_zone_id) REFERENCES mdm.time_zone(id),
    CONSTRAINT fk_cal_parent   FOREIGN KEY (parent_calendar_id)  REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_cal_hier     FOREIGN KEY (hierarchy_type_id)   REFERENCES mdm.calendar_hierarchy_type(id),
    CONSTRAINT fk_cal_replaced FOREIGN KEY (replaced_by_calendar_id) REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_cal_merged   FOREIGN KEY (merged_into_id)      REFERENCES mdm.calendar_master(id),
    CONSTRAINT fk_cal_srcsys   FOREIGN KEY (source_system_id)    REFERENCES mdm.source_systems(id)
);
CREATE INDEX IF NOT EXISTS idx_cal_master_type    ON mdm.calendar_master (calendar_type_id);
CREATE INDEX IF NOT EXISTS idx_cal_master_country ON mdm.calendar_master (country_cd);
CREATE INDEX IF NOT EXISTS idx_cal_master_mic     ON mdm.calendar_master (mic);
CREATE INDEX IF NOT EXISTS idx_cal_master_parent  ON mdm.calendar_master (parent_calendar_id);
CREATE INDEX IF NOT EXISTS idx_cal_master_status  ON mdm.calendar_master (status);
CREATE INDEX IF NOT EXISTS idx_cal_master_tenant  ON mdm.calendar_master (tenant_id);

-- ── calendar_identifier ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_identifier (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
    id_type varchar(30) NOT NULL,
    id_value varchar(100) NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_identifier_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cali_id_type CHECK (id_type IN (
        'BBG_CALENDAR','RIC_CALENDAR','MIC','ISO_MIC','PROVIDER_CODE',
        'ISDA_CODE','ICMA_CODE','FIX_CALENDAR','INTERNAL','IANA_TZ')),
    CONSTRAINT fk_cali_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cali_calendar ON mdm.calendar_identifier (calendar_id);
CREATE INDEX IF NOT EXISTS idx_cali_lookup   ON mdm.calendar_identifier (id_type, id_value);
CREATE INDEX IF NOT EXISTS idx_cali_tenant   ON mdm.calendar_identifier (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cali_active
    ON mdm.calendar_identifier (tenant_id, id_type, id_value) WHERE effective_to IS NULL;

-- ── calendar_classification ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_classification (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    calendar_id uuid NOT NULL,
    scheme varchar(30) NOT NULL,
    code varchar(50) NOT NULL,
    name varchar(250),
    level int4 DEFAULT 1 NOT NULL,
    parent_code varchar(50),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_primary bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_classification_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calclass_calendar FOREIGN KEY (calendar_id) REFERENCES mdm.calendar_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_calclass_calendar ON mdm.calendar_classification (calendar_id);
CREATE INDEX IF NOT EXISTS idx_calclass_scheme   ON mdm.calendar_classification (scheme, code);
CREATE INDEX IF NOT EXISTS idx_calclass_tenant   ON mdm.calendar_classification (tenant_id);

-- ── calendar_hierarchy ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_hierarchy (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_calendar_id uuid NOT NULL,
    child_calendar_id uuid NOT NULL,
    hierarchy_type_id uuid NOT NULL,
    inheritance_mode varchar(20) NOT NULL,
    precedence int4 DEFAULT 100 NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_hierarchy_pkey PRIMARY KEY (id),
    CONSTRAINT chk_calh_no_self CHECK (parent_calendar_id <> child_calendar_id),
    CONSTRAINT chk_calh_mode CHECK (inheritance_mode IN (
        'INHERIT_ALL','INHERIT_EXCEPT','ADDITIVE','OVERRIDE')),
    CONSTRAINT fk_calh_parent FOREIGN KEY (parent_calendar_id) REFERENCES mdm.calendar_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_calh_child  FOREIGN KEY (child_calendar_id)  REFERENCES mdm.calendar_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_calh_type   FOREIGN KEY (hierarchy_type_id)  REFERENCES mdm.calendar_hierarchy_type(id)
);
CREATE INDEX IF NOT EXISTS idx_calh_parent  ON mdm.calendar_hierarchy (parent_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calh_child   ON mdm.calendar_hierarchy (child_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calh_tenant  ON mdm.calendar_hierarchy (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_calh_active_edge
    ON mdm.calendar_hierarchy (tenant_id, parent_calendar_id, child_calendar_id, hierarchy_type_id)
    WHERE effective_to IS NULL;

-- ── calendar_hierarchy_closure ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.calendar_hierarchy_closure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ancestor_calendar_id uuid NOT NULL,
    descendant_calendar_id uuid NOT NULL,
    hierarchy_type_id uuid NOT NULL,
    depth int4 NOT NULL,
    path text,
    computed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT calendar_hierarchy_closure_pkey PRIMARY KEY (id),
    CONSTRAINT fk_calhc_anc  FOREIGN KEY (ancestor_calendar_id)   REFERENCES mdm.calendar_master(id) ON DELETE CASCADE,
    CONSTRAINT fk_calhc_desc FOREIGN KEY (descendant_calendar_id) REFERENCES mdm.calendar_master(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_calhc_anc    ON mdm.calendar_hierarchy_closure (ancestor_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calhc_desc   ON mdm.calendar_hierarchy_closure (descendant_calendar_id);
CREATE INDEX IF NOT EXISTS idx_calhc_tenant ON mdm.calendar_hierarchy_closure (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_calhc_edge
    ON mdm.calendar_hierarchy_closure (tenant_id, ancestor_calendar_id, descendant_calendar_id, hierarchy_type_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'calendar_master','calendar_identifier','calendar_classification',
        'calendar_hierarchy','calendar_hierarchy_closure'
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

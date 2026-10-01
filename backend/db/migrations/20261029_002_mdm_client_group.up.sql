-- 20261029_002_mdm_client_group.up.sql
-- Client Group: the middle layer between tenant and party. Groups a family,
-- institutional client, or fund complex into a single relationship unit.
-- Every party, account, portfolio, trust, and entity is assigned to exactly
-- one client group.

-- ── client_group ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_group_cd varchar(50) NOT NULL,
    name varchar(500) NOT NULL,
    short_name varchar(150),
    group_type varchar(30) NOT NULL,
    parent_client_group_id uuid,
    primary_party_id uuid,
    primary_jurisdiction varchar(10),
    base_currency varchar(3),
    is_confidential bool DEFAULT false NOT NULL,
    is_cross_border bool DEFAULT false NOT NULL,
    is_erisa bool DEFAULT false NOT NULL,
    is_regulated bool DEFAULT false NOT NULL,
    regulatory_regime_cd varchar(50),
    relationship_since date,
    onboarding_date date,
    servicing_team_id uuid,
    relationship_manager_id uuid,
    aum numeric(28,4),
    aum_currency varchar(3),
    aum_date date,
    status varchar(20) DEFAULT 'ACTIVE' NOT NULL,
    lifecycle_stage varchar(20) DEFAULT 'ACTIVE',
    source_system_id uuid,
    is_golden_record bool DEFAULT true NOT NULL,
    merged_into_id uuid,
    dq_score numeric(5,2),
    steward_id uuid,
    last_reviewed_at timestamptz,
    review_frequency_months int4,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_pkey PRIMARY KEY (id),
    CONSTRAINT client_group_cd_key UNIQUE (tenant_id, client_group_cd),
    CONSTRAINT chk_cg_type CHECK (group_type IN (
        'FAMILY','FAMILY_OFFICE','INSTITUTIONAL','PENSION','INSURANCE',
        'SOVEREIGN','ENDOWMENT','FOUNDATION','CORPORATE','FUND_COMPLEX',
        'PARTNERSHIP','JOINT_VENTURE','MULTI_FAMILY_OFFICE')),
    CONSTRAINT chk_cg_status CHECK (status IN (
        'PROSPECT','ONBOARDING','ACTIVE','DORMANT','SUSPENDED','CLOSED')),
    CONSTRAINT chk_cg_lifecycle CHECK (lifecycle_stage IS NULL OR lifecycle_stage IN (
        'PROSPECT','ONBOARDING','ACTIVE','DORMANT','CLOSED','MERGED')),
    CONSTRAINT fk_cg_parent   FOREIGN KEY (parent_client_group_id) REFERENCES mdm.client_group(id),
    CONSTRAINT fk_cg_primary  FOREIGN KEY (primary_party_id)       REFERENCES mdm.party(id),
    CONSTRAINT fk_cg_merged   FOREIGN KEY (merged_into_id)         REFERENCES mdm.client_group(id)
);
CREATE INDEX IF NOT EXISTS idx_cg_tenant       ON mdm.client_group (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cg_type         ON mdm.client_group (group_type);
CREATE INDEX IF NOT EXISTS idx_cg_parent       ON mdm.client_group (parent_client_group_id);
CREATE INDEX IF NOT EXISTS idx_cg_status       ON mdm.client_group (status);
CREATE INDEX IF NOT EXISTS idx_cg_confidential ON mdm.client_group (is_confidential) WHERE is_confidential = true;

-- ── client_group_hierarchy ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group_hierarchy (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_client_group_id uuid NOT NULL,
    child_client_group_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    ownership_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_hierarchy_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cgh_no_self CHECK (parent_client_group_id <> child_client_group_id),
    CONSTRAINT chk_cgh_type CHECK (hierarchy_type IN (
        'FAMILY_BRANCH','SUB_FUND','MASTER_FEEDER','SUBSIDIARY',
        'TRUST_STRUCTURE','PHILANTHROPIC_AFFILIATION','CORPORATE_GROUP')),
    CONSTRAINT fk_cgh_parent FOREIGN KEY (parent_client_group_id) REFERENCES mdm.client_group(id) ON DELETE CASCADE,
    CONSTRAINT fk_cgh_child  FOREIGN KEY (child_client_group_id)  REFERENCES mdm.client_group(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cgh_parent ON mdm.client_group_hierarchy (parent_client_group_id);
CREATE INDEX IF NOT EXISTS idx_cgh_child  ON mdm.client_group_hierarchy (child_client_group_id);
CREATE INDEX IF NOT EXISTS idx_cgh_tenant ON mdm.client_group_hierarchy (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cgh_active_edge
    ON mdm.client_group_hierarchy (tenant_id, parent_client_group_id, child_client_group_id, hierarchy_type)
    WHERE effective_to IS NULL;

-- ── client_group_hierarchy_closure ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group_hierarchy_closure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ancestor_client_group_id uuid NOT NULL,
    descendant_client_group_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    depth int4 NOT NULL,
    path text,
    computed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_hierarchy_closure_pkey PRIMARY KEY (id),
    CONSTRAINT fk_cghc_anc  FOREIGN KEY (ancestor_client_group_id)   REFERENCES mdm.client_group(id) ON DELETE CASCADE,
    CONSTRAINT fk_cghc_desc FOREIGN KEY (descendant_client_group_id) REFERENCES mdm.client_group(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cghc_anc    ON mdm.client_group_hierarchy_closure (ancestor_client_group_id);
CREATE INDEX IF NOT EXISTS idx_cghc_desc   ON mdm.client_group_hierarchy_closure (descendant_client_group_id);
CREATE INDEX IF NOT EXISTS idx_cghc_tenant ON mdm.client_group_hierarchy_closure (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cghc_edge
    ON mdm.client_group_hierarchy_closure (tenant_id, ancestor_client_group_id, descendant_client_group_id, hierarchy_type);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'client_group','client_group_hierarchy','client_group_hierarchy_closure'
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

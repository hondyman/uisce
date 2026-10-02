-- ── party_hierarchy ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_hierarchy (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_party_id uuid NOT NULL,
    child_party_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    ownership_pct numeric(7,4),
    voting_pct numeric(7,4),
    economic_pct numeric(7,4),
    control_indicator varchar(20),
    is_direct bool DEFAULT true NOT NULL,
    is_nominee_arrangement bool DEFAULT false NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    source varchar(50),
    confidence numeric(5,2),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_hierarchy_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ph_no_self CHECK (parent_party_id <> child_party_id),
    CONSTRAINT chk_ph_type CHECK (hierarchy_type IN (
        'LEGAL_OWNERSHIP','BENEFICIAL_OWNERSHIP','VOTING_CONTROL',
        'MANAGEMENT_CONTROL','REGULATORY_GROUP','FAMILY_GROUP',
        'TRUST_STRUCTURE','NOMINEE_ARRANGEMENT')),
    CONSTRAINT fk_ph_parent FOREIGN KEY (parent_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_ph_child  FOREIGN KEY (child_party_id)  REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ph_parent ON mdm.party_hierarchy (parent_party_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_ph_child  ON mdm.party_hierarchy (child_party_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_ph_tenant ON mdm.party_hierarchy (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ph_active
    ON mdm.party_hierarchy (tenant_id, parent_party_id, child_party_id, hierarchy_type)
    WHERE effective_to IS NULL;

-- ── party_hierarchy_closure ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_hierarchy_closure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ancestor_party_id uuid NOT NULL,
    descendant_party_id uuid NOT NULL,
    hierarchy_type varchar(30) NOT NULL,
    depth int4 NOT NULL,
    min_ownership_pct numeric(7,4),
    cumulative_ownership_pct numeric(7,4),
    path text,
    computed_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_hierarchy_closure_pkey PRIMARY KEY (id),
    CONSTRAINT fk_phc_anc  FOREIGN KEY (ancestor_party_id)   REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_phc_desc FOREIGN KEY (descendant_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_phc_anc    ON mdm.party_hierarchy_closure (ancestor_party_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_phc_desc   ON mdm.party_hierarchy_closure (descendant_party_id, hierarchy_type);
CREATE INDEX IF NOT EXISTS idx_phc_tenant ON mdm.party_hierarchy_closure (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_phc_edge
    ON mdm.party_hierarchy_closure (tenant_id, ancestor_party_id, descendant_party_id, hierarchy_type);

-- ── party_ownership (shareholder register side) ─────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_ownership (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    owned_party_id uuid NOT NULL,
    owner_party_id uuid,
    owner_name_if_unregistered varchar(500),
    owner_type_if_unregistered varchar(30),
    ownership_type varchar(30) NOT NULL,
    shares_held numeric(24,4),
    share_class varchar(30),
    ownership_pct numeric(7,4),
    voting_pct numeric(7,4),
    economic_pct numeric(7,4),
    is_beneficial_owner bool DEFAULT false NOT NULL,
    is_controlling_shareholder bool DEFAULT false NOT NULL,
    is_nominee bool DEFAULT false NOT NULL,
    is_pledged bool DEFAULT false NOT NULL,
    as_of_date date NOT NULL,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_ownership_pkey PRIMARY KEY (id),
    CONSTRAINT chk_po_type CHECK (ownership_type IN (
        'DIRECT','INDIRECT','BENEFICIAL','NOMINEE','PLEDGED','ENCUMBERED')),
    CONSTRAINT fk_po_owned FOREIGN KEY (owned_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_po_owner FOREIGN KEY (owner_party_id) REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_po_owned  ON mdm.party_ownership (owned_party_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_po_owner  ON mdm.party_ownership (owner_party_id);
CREATE INDEX IF NOT EXISTS idx_po_tenant ON mdm.party_ownership (tenant_id);

-- ── party_control ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_control (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    controlled_party_id uuid NOT NULL,
    controller_party_id uuid NOT NULL,
    control_type varchar(30) NOT NULL,
    control_basis varchar(30),
    is_effective_control bool DEFAULT false NOT NULL,
    is_significant_influence bool DEFAULT false NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_control_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pctl_type CHECK (control_type IN (
        'VOTING_RIGHTS','BOARD_CONTROL','MANAGEMENT_CONTROL',
        'CONTRACTUAL_CONTROL','SHAREHOLDER_AGREEMENT','FAMILY_CONTROL',
        'TRUST_CONTROL','GOVERNMENT_CONTROL','NEGATIVE_CONTROL')),
    CONSTRAINT fk_pctl_controlled FOREIGN KEY (controlled_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_pctl_controller FOREIGN KEY (controller_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pctl_controlled ON mdm.party_control (controlled_party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pctl_controller ON mdm.party_control (controller_party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pctl_tenant     ON mdm.party_control (tenant_id);

-- ── party_ubo ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_ubo (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    subject_party_id uuid NOT NULL,
    ubo_party_id uuid,
    ubo_name_if_unregistered varchar(500),
    ubo_type varchar(30) NOT NULL,
    ubo_basis varchar(30) NOT NULL,
    ownership_pct numeric(7,4),
    voting_pct numeric(7,4),
    economic_pct numeric(7,4),
    indirect_pct numeric(7,4),
    is_direct bool DEFAULT true NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    jurisdiction_rule varchar(10) NOT NULL,
    threshold_applied numeric(7,4) NOT NULL,
    computation_method varchar(30) NOT NULL,
    computation_date date NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    verified_by uuid,
    verified_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_ubo_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pubo_type CHECK (ubo_type IN (
        'NATURAL_PERSON','LEGAL_ENTITY','TRUST','NOMINEE',
        'SENIOR_MANAGING_OFFICIAL','CATEGORY')),
    CONSTRAINT chk_pubo_basis CHECK (ubo_basis IN (
        'OWNERSHIP_25','OWNERSHIP_10','OWNERSHIP_OTHER','VOTING_CONTROL',
        'MANAGEMENT_CONTROL','SENIOR_MANAGING_OFFICIAL_FALLBACK',
        'TRUST_BENEFICIARY','TRUST_SETTLOR','TRUST_PROTECTOR')),
    CONSTRAINT chk_pubo_method CHECK (computation_method IN (
        'OWNERSHIP_CHAIN','CONTROL_CHAIN','FALLBACK_SMO',
        'TRUST_LOOKTHROUGH','NOMINEE_LOOKTHROUGH')),
    CONSTRAINT fk_pubo_subject FOREIGN KEY (subject_party_id) REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_pubo_ubo     FOREIGN KEY (ubo_party_id)     REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_pubo_subject ON mdm.party_ubo (subject_party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_pubo_ubo     ON mdm.party_ubo (ubo_party_id);
CREATE INDEX IF NOT EXISTS idx_pubo_tenant  ON mdm.party_ubo (tenant_id);

-- ── party_relationship (non-hierarchical junctions) ─────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_relationship (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    from_party_id uuid NOT NULL,
    to_party_id uuid NOT NULL,
    relationship_type_id uuid NOT NULL,
    relationship_sub_type varchar(50),
    is_primary bool DEFAULT false NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    description text,
    verified_by uuid,
    verified_at timestamptz,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_relationship_pkey PRIMARY KEY (id),
    CONSTRAINT chk_prel_no_self CHECK (from_party_id <> to_party_id),
    CONSTRAINT fk_prel_from FOREIGN KEY (from_party_id)         REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_prel_to   FOREIGN KEY (to_party_id)           REFERENCES mdm.party(id) ON DELETE CASCADE,
    CONSTRAINT fk_prel_type FOREIGN KEY (relationship_type_id)  REFERENCES mdm.party_relationship_type(id)
);
CREATE INDEX IF NOT EXISTS idx_prel_from   ON mdm.party_relationship (from_party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prel_to     ON mdm.party_relationship (to_party_id, is_current);
CREATE INDEX IF NOT EXISTS idx_prel_type   ON mdm.party_relationship (relationship_type_id);
CREATE INDEX IF NOT EXISTS idx_prel_tenant ON mdm.party_relationship (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prel_active
    ON mdm.party_relationship (tenant_id, from_party_id, to_party_id, relationship_type_id)
    WHERE effective_to IS NULL;

-- ── party_successor ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.party_successor (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    predecessor_party_id uuid NOT NULL,
    successor_party_id uuid NOT NULL,
    succession_type varchar(30) NOT NULL,
    effective_date date NOT NULL,
    succession_pct numeric(7,4),
    assumes_assets bool DEFAULT false NOT NULL,
    assumes_liabilities bool DEFAULT false NOT NULL,
    assumes_contracts bool DEFAULT false NOT NULL,
    related_corporate_action_id uuid,
    regulatory_filing_ref varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT party_successor_pkey PRIMARY KEY (id),
    CONSTRAINT fk_psucc_pred FOREIGN KEY (predecessor_party_id) REFERENCES mdm.party(id),
    CONSTRAINT fk_psucc_succ FOREIGN KEY (successor_party_id)   REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_psucc_pred   ON mdm.party_successor (predecessor_party_id);
CREATE INDEX IF NOT EXISTS idx_psucc_succ   ON mdm.party_successor (successor_party_id);
CREATE INDEX IF NOT EXISTS idx_psucc_tenant ON mdm.party_successor (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'party_hierarchy','party_hierarchy_closure','party_ownership',
        'party_control','party_ubo','party_relationship','party_successor'
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

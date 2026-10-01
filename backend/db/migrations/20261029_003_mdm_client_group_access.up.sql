-- 20261029_003_mdm_client_group_access.up.sql
-- Client group membership (which entities belong), user access control
-- (for family-office logins), and manager-side servicing team assignments.

-- ── client_group_membership ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group_membership (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_group_id uuid NOT NULL,
    member_type varchar(30) NOT NULL,
    member_id uuid NOT NULL,
    membership_role varchar(30) NOT NULL,
    ownership_pct numeric(7,4),
    consolidation_method varchar(30),
    effective_from date NOT NULL,
    effective_to date,
    is_primary bool DEFAULT false NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    added_by uuid,
    added_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    removal_reason varchar(500),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_membership_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cgm_member_type CHECK (member_type IN (
        'PARTY','ACCOUNT','TRUST','ISSUER','PORTFOLIO','FUND','MANDATE',
        'LIFESTYLE_ASSET','PHILANTHROPIC_VEHICLE','FAMILY_BUSINESS',
        'FAMILY_BANK','FAMILY_OFFICE')),
    CONSTRAINT chk_cgm_role CHECK (membership_role IN (
        'PRIMARY','LINKED','CUSTODY','INVESTMENT','PHILANTHROPIC',
        'STRUCTURAL','OPERATING','BENEFICIAL','NOMINEE')),
    CONSTRAINT chk_cgm_consolidation CHECK (consolidation_method IS NULL OR consolidation_method IN (
        'FULL','EQUITY','PROPORTIONAL','COST','LOOKTHROUGH','NONE')),
    CONSTRAINT fk_cgm_group FOREIGN KEY (client_group_id) REFERENCES mdm.client_group(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cgm_group  ON mdm.client_group_membership (client_group_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cgm_member ON mdm.client_group_membership (member_type, member_id, is_current);
CREATE INDEX IF NOT EXISTS idx_cgm_tenant ON mdm.client_group_membership (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cgm_active
    ON mdm.client_group_membership (tenant_id, client_group_id, member_type, member_id)
    WHERE effective_to IS NULL;

-- ── client_group_user ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group_user (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_group_id uuid NOT NULL,
    user_id uuid NOT NULL,
    party_id uuid,
    access_role varchar(30) NOT NULL,
    is_primary_admin bool DEFAULT false NOT NULL,
    can_invite_users bool DEFAULT false NOT NULL,
    can_view_all_members bool DEFAULT false NOT NULL,
    can_view_trusts bool DEFAULT false NOT NULL,
    can_view_lifestyle_assets bool DEFAULT false NOT NULL,
    can_view_financials bool DEFAULT true NOT NULL,
    can_view_documents bool DEFAULT true NOT NULL,
    can_view_governance bool DEFAULT false NOT NULL,
    restricted_to_entity_ids uuid[],
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    invited_by uuid,
    invitation_accepted_at timestamptz,
    last_login_at timestamptz,
    mfa_enabled bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_user_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cgu UNIQUE (tenant_id, client_group_id, user_id),
    CONSTRAINT chk_cgu_role CHECK (access_role IN (
        'FAMILY_ADMIN','FAMILY_MEMBER','FAMILY_VIEWER',
        'TRUSTEE_EXTERNAL','FAMILY_OFFICE_EXEC','FAMILY_ADVISOR',
        'FOUNDATION_BOARD','EXTERNAL_ACCOUNTANT','EXTERNAL_ATTORNEY')),
    CONSTRAINT fk_cgu_group FOREIGN KEY (client_group_id) REFERENCES mdm.client_group(id) ON DELETE CASCADE,
    CONSTRAINT fk_cgu_party FOREIGN KEY (party_id)        REFERENCES mdm.party(id)
);
CREATE INDEX IF NOT EXISTS idx_cgu_group  ON mdm.client_group_user (client_group_id, is_active);
CREATE INDEX IF NOT EXISTS idx_cgu_user   ON mdm.client_group_user (user_id);
CREATE INDEX IF NOT EXISTS idx_cgu_tenant ON mdm.client_group_user (tenant_id);

-- ── client_group_servicing ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.client_group_servicing (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    client_group_id uuid NOT NULL,
    user_id uuid NOT NULL,
    servicing_role varchar(30) NOT NULL,
    is_primary bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT client_group_servicing_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cgs_role CHECK (servicing_role IN (
        'RELATIONSHIP_MANAGER','INVESTMENT_MANAGER','PORTFOLIO_MANAGER',
        'TRUST_OFFICER','PHILANTHROPIC_ADVISOR','TAX_ADVISOR',
        'ESTATE_PLANNER','FAMILY_OFFICE_CEO','CFO','COO',
        'COMPLIANCE_OFFICER','OPERATIONS','CLIENT_SERVICE')),
    CONSTRAINT fk_cgs_group FOREIGN KEY (client_group_id) REFERENCES mdm.client_group(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cgs_group  ON mdm.client_group_servicing (client_group_id, is_active);
CREATE INDEX IF NOT EXISTS idx_cgs_user   ON mdm.client_group_servicing (user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_cgs_tenant ON mdm.client_group_servicing (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'client_group_membership','client_group_user','client_group_servicing'
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

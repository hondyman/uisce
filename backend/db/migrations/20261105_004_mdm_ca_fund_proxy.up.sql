-- 20261105_004_mdm_ca_fund_proxy.up.sql

-- ── ca_fund_distribution ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_fund_distribution (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    fund_id uuid,
    fund_share_class_id uuid,
    distribution_type varchar(30) NOT NULL,
    distribution_per_share numeric(20,12),
    income_per_share numeric(20,12),
    st_cg_per_share numeric(20,12),
    lt_cg_per_share numeric(20,12),
    roc_per_share numeric(20,12),
    foreign_tax_per_share numeric(20,12),
    foreign_source_income_pct numeric(7,4),
    currency varchar(3),
    ex_date date NOT NULL,
    record_date date,
    reinvestment_date date,
    payable_date date,
    reinvestment_nav numeric(20,12),
    is_reinvested_by_default bool DEFAULT false NOT NULL,
    is_supplemental bool DEFAULT false NOT NULL,
    fiscal_year int4,
    fiscal_period varchar(20),
    tax_year int4,
    form_1099_reportable bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_fund_distribution_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cafd_type CHECK (distribution_type IN (
        'INCOME','SHORT_TERM_CAPITAL_GAIN','LONG_TERM_CAPITAL_GAIN',
        'RETURN_OF_CAPITAL','SPECIAL','ORDINARY_DIVIDEND',
        'QUALIFIED_DIVIDEND','TAX_EXEMPT_INTEREST',
        'SECTION_199A_DIVIDEND','FOREIGN_TAX_CREDIT')),
    CONSTRAINT fk_cafd_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cafd_ca     ON mdm.ca_fund_distribution (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cafd_fsc    ON mdm.ca_fund_distribution (fund_share_class_id, ex_date);
CREATE INDEX IF NOT EXISTS idx_cafd_tenant ON mdm.ca_fund_distribution (tenant_id);

-- ── ca_fund_reorganization ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_fund_reorganization (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    surviving_fund_id uuid,
    surviving_share_class_id uuid,
    merging_fund_id uuid,
    merging_share_class_id uuid,
    reorganization_type varchar(30) NOT NULL,
    conversion_ratio numeric(20,12),
    conversion_date date NOT NULL,
    is_tax_free bool,
    is_shareholder_approved bool,
    shareholder_approval_date date,
    prospectus_effective_date date,
    sec_filing_ref varchar(100),
    regulatory_approval_ref varchar(100),
    description text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_fund_reorganization_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cafr_type CHECK (reorganization_type IN (
        'MERGER','CONSOLIDATION','REORGANIZATION','REBRANDING',
        'SHARE_CLASS_MERGER','FUND_ABSORPTION','TRUST_RESTRUCTURE')),
    CONSTRAINT fk_cafr_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cafr_ca     ON mdm.ca_fund_reorganization (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cafr_tenant ON mdm.ca_fund_reorganization (tenant_id);

-- ── ca_fund_liquidation ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_fund_liquidation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    fund_id uuid,
    fund_share_class_id uuid,
    liquidation_type varchar(30) NOT NULL,
    announcement_date date NOT NULL,
    cessation_of_sales_date date,
    cessation_of_dealing_date date,
    final_nav_date date,
    final_distribution_date date,
    final_distribution_per_share numeric(20,12),
    final_distribution_currency varchar(3),
    regulatory_filing_ref varchar(100),
    shareholder_notification_date date,
    description text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_fund_liquidation_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cafl_type CHECK (liquidation_type IN (
        'ORDERLY','FORCED','REGULATORY','VOLUNTARY','WIND_DOWN')),
    CONSTRAINT fk_cafl_ca FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cafl_ca     ON mdm.ca_fund_liquidation (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cafl_tenant ON mdm.ca_fund_liquidation (tenant_id);

-- ── ca_meeting (shareholder meeting) ───────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_meeting (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ca_event_id uuid NOT NULL,
    issuer_id uuid,
    security_id uuid,
    meeting_type varchar(30) NOT NULL,
    meeting_date date NOT NULL,
    meeting_time time,
    meeting_timezone_id uuid,
    meeting_location varchar(500),
    meeting_format varchar(20),
    record_date date NOT NULL,
    notice_date date,
    proxy_mailing_date date,
    voting_deadline_date date,
    voting_deadline_time time,
    voting_deadline_timezone_id uuid,
    is_contested bool DEFAULT false NOT NULL,
    meeting_status varchar(30),
    quorum_required_pct numeric(7,4),
    quorum_met bool,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_meeting_pkey PRIMARY KEY (id),
    CONSTRAINT chk_cam_type CHECK (meeting_type IN (
        'ANNUAL','SPECIAL','EXTRAORDINARY','COURT','CREDITOR',
        'BONDHOLDER','CLASS','COMBINED')),
    CONSTRAINT chk_cam_format CHECK (meeting_format IS NULL OR meeting_format IN (
        'IN_PERSON','VIRTUAL','HYBRID')),
    CONSTRAINT fk_cam_ca     FOREIGN KEY (ca_event_id) REFERENCES mdm.ca_event(id) ON DELETE CASCADE,
    CONSTRAINT fk_cam_issuer FOREIGN KEY (issuer_id)   REFERENCES edm.issuer_master(id),
    CONSTRAINT fk_cam_tz     FOREIGN KEY (meeting_timezone_id) REFERENCES mdm.time_zone(id),
    CONSTRAINT fk_cam_vtz    FOREIGN KEY (voting_deadline_timezone_id) REFERENCES mdm.time_zone(id)
);
CREATE INDEX IF NOT EXISTS idx_cam_ca     ON mdm.ca_meeting (ca_event_id);
CREATE INDEX IF NOT EXISTS idx_cam_date   ON mdm.ca_meeting (meeting_date);
CREATE INDEX IF NOT EXISTS idx_cam_tenant ON mdm.ca_meeting (tenant_id);

-- ── ca_agenda_item ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_agenda_item (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    meeting_id uuid NOT NULL,
    item_number varchar(20) NOT NULL,
    item_type varchar(50) NOT NULL,
    title varchar(500) NOT NULL,
    description text,
    management_recommendation varchar(30),
    board_recommendation varchar(30),
    requires_supermajority bool DEFAULT false NOT NULL,
    supermajority_threshold_pct numeric(7,4),
    is_contested bool DEFAULT false NOT NULL,
    is_shareholder_proposal bool DEFAULT false NOT NULL,
    proposal_sponsor varchar(500),
    proposal_category varchar(50),
    voting_result_for numeric(20,4),
    voting_result_against numeric(20,4),
    voting_result_abstain numeric(20,4),
    voting_result_withheld numeric(20,4),
    outcome varchar(30),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_agenda_item_pkey PRIMARY KEY (id),
    CONSTRAINT chk_caai_outcome CHECK (outcome IS NULL OR outcome IN (
        'PASSED','FAILED','WITHDRAWN','TABLED','PENDING')),
    CONSTRAINT chk_caai_category CHECK (proposal_category IS NULL OR proposal_category IN (
        'ENVIRONMENTAL','SOCIAL','GOVERNANCE','COMPENSATION',
        'POLITICAL','HUMAN_RIGHTS','ANIMAL_WELFARE')),
    CONSTRAINT fk_caai_meeting FOREIGN KEY (meeting_id) REFERENCES mdm.ca_meeting(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_caai_meeting ON mdm.ca_agenda_item (meeting_id);
CREATE INDEX IF NOT EXISTS idx_caai_type    ON mdm.ca_agenda_item (item_type);
CREATE INDEX IF NOT EXISTS idx_caai_tenant  ON mdm.ca_agenda_item (tenant_id);

-- ── ca_proxy_vote ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mdm.ca_proxy_vote (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    meeting_id uuid NOT NULL,
    agenda_item_id uuid NOT NULL,
    account_id uuid,
    security_id uuid,
    shares_held numeric(24,4),
    shares_voted numeric(24,4),
    vote_cast varchar(30) NOT NULL,
    vote_method varchar(30),
    voting_decision_source varchar(30),
    voting_policy_cd varchar(50),
    proxy_advisor_cd varchar(50),
    is_conflict_of_interest bool DEFAULT false NOT NULL,
    conflict_description text,
    conflict_mitigation text,
    is_share_lending_relevant bool DEFAULT false NOT NULL,
    recalled_shares numeric(24,4),
    vote_date date NOT NULL,
    vote_timestamp timestamptz,
    custodian_ref varchar(100),
    proxy_ref varchar(100),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT ca_proxy_vote_pkey PRIMARY KEY (id),
    CONSTRAINT chk_capv_vote CHECK (vote_cast IN (
        'FOR','AGAINST','ABSTAIN','WITHHOLD','NOT_VOTED',
        'SPLIT','BROKER_NON_VOTE')),
    CONSTRAINT fk_capv_meeting FOREIGN KEY (meeting_id)     REFERENCES mdm.ca_meeting(id) ON DELETE CASCADE,
    CONSTRAINT fk_capv_item    FOREIGN KEY (agenda_item_id) REFERENCES mdm.ca_agenda_item(id)
);
CREATE INDEX IF NOT EXISTS idx_capv_meeting  ON mdm.ca_proxy_vote (meeting_id);
CREATE INDEX IF NOT EXISTS idx_capv_item     ON mdm.ca_proxy_vote (agenda_item_id);
CREATE INDEX IF NOT EXISTS idx_capv_acct     ON mdm.ca_proxy_vote (account_id);
CREATE INDEX IF NOT EXISTS idx_capv_conflict ON mdm.ca_proxy_vote (is_conflict_of_interest) WHERE is_conflict_of_interest = true;
CREATE INDEX IF NOT EXISTS idx_capv_tenant   ON mdm.ca_proxy_vote (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
DO $rls$
DECLARE t text;
    tables text[] := ARRAY[
        'ca_fund_distribution','ca_fund_reorganization','ca_fund_liquidation',
        'ca_meeting','ca_agenda_item','ca_proxy_vote'
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

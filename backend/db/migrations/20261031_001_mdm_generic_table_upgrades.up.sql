-- 20261031_001_mdm_generic_table_upgrades.up.sql
-- Brings the ten generic mdm tables up to the capability level of their
-- domain-specific equivalents. All operations are ADD COLUMN IF NOT EXISTS.
--
-- Also expands the three thin domain anchors (portfolio, mandate,
-- portfolio_composite) that were stubs from the initial MDM bootstrap.
--
-- Notes on cross-schema references:
--   * mdm.issuer does not exist; canonical issuer table is edm.issuer_master.
--   * The source-system table is mdm.source_systems (plural) in the live DB.
--     Some legacy code and migrations reference mdm.source_system (singular).
--     Both are ALTERed conditionally below — whichever exists gets the upgrade.
--   * No new FKs are added here (matches the project convention of deferring
--     FK additions to a separate constraint pass). Columns are declared but
--     unconstrained; application layer enforces referential integrity.

DO $helper$
BEGIN
    RAISE NOTICE 'batch 4: starting generic table upgrades';
END $helper$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 1. mdm.match_rule — align with the domain-specific match rules
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='match_rule') THEN
        ALTER TABLE mdm.match_rule
            ADD COLUMN IF NOT EXISTS rule_cd              varchar(50),
            ADD COLUMN IF NOT EXISTS sub_type             varchar(50),
            ADD COLUMN IF NOT EXISTS priority             int4 DEFAULT 100,
            ADD COLUMN IF NOT EXISTS deterministic_keys   jsonb DEFAULT '[]'::jsonb,
            ADD COLUMN IF NOT EXISTS fuzzy_keys           jsonb DEFAULT '[]'::jsonb,
            ADD COLUMN IF NOT EXISTS threshold_no_match   numeric(5,4),
            ADD COLUMN IF NOT EXISTS version              varchar(20) DEFAULT '1.0',
            ADD COLUMN IF NOT EXISTS is_active_legacy     bool DEFAULT true;
        CREATE UNIQUE INDEX IF NOT EXISTS uq_match_rule_cd
            ON mdm.match_rule (tenant_id, rule_cd) WHERE rule_cd IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_match_rule_priority
            ON mdm.match_rule (priority, is_active);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 2. mdm.merge_log — add reversal + trace fields
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='merge_log') THEN
        ALTER TABLE mdm.merge_log
            ADD COLUMN IF NOT EXISTS match_candidate_id   uuid,
            ADD COLUMN IF NOT EXISTS reversible           bool DEFAULT true,
            ADD COLUMN IF NOT EXISTS reversal_data        jsonb,
            ADD COLUMN IF NOT EXISTS downstream_notified  bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS merged_by            uuid,
            ADD COLUMN IF NOT EXISTS merged_at            timestamptz DEFAULT CURRENT_TIMESTAMP;
        CREATE INDEX IF NOT EXISTS idx_merge_log_merged_at
            ON mdm.merge_log (merged_at DESC);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 3. mdm.survivorship_rule — align with issuer_survivorship_rule
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='survivorship_rule') THEN
        ALTER TABLE mdm.survivorship_rule
            ADD COLUMN IF NOT EXISTS field_group                 varchar(50),
            ADD COLUMN IF NOT EXISTS sub_type                    varchar(50),
            ADD COLUMN IF NOT EXISTS min_confidence              numeric(5,2),
            ADD COLUMN IF NOT EXISTS max_staleness_hours         int4,
            ADD COLUMN IF NOT EXISTS manual_override_allowed     bool DEFAULT true,
            ADD COLUMN IF NOT EXISTS source_priority_structured  jsonb;
        CREATE UNIQUE INDEX IF NOT EXISTS uq_survivorship_rule_structured
            ON mdm.survivorship_rule (
                tenant_id, entity_type,
                COALESCE(sub_type, ''),
                COALESCE(field_group, ''),
                attribute_name);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 4. mdm.change_request — full review state machine
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='change_request') THEN
        ALTER TABLE mdm.change_request
            ADD COLUMN IF NOT EXISTS request_ref       varchar(50),
            ADD COLUMN IF NOT EXISTS assigned_to       uuid,
            ADD COLUMN IF NOT EXISTS approved_by       uuid,
            ADD COLUMN IF NOT EXISTS approved_at       timestamptz,
            ADD COLUMN IF NOT EXISTS applied_at        timestamptz,
            ADD COLUMN IF NOT EXISTS rejection_reason  text,
            ADD COLUMN IF NOT EXISTS priority          varchar(20) DEFAULT 'MEDIUM',
            ADD COLUMN IF NOT EXISTS request_reason    text;
        CREATE UNIQUE INDEX IF NOT EXISTS uq_change_request_ref
            ON mdm.change_request (tenant_id, request_ref) WHERE request_ref IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_change_request_assigned
            ON mdm.change_request (assigned_to, status);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 5. mdm.source_systems / mdm.source_system — add operational metadata
--    (both variants covered; only the one that exists is altered)
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='source_systems') THEN
        ALTER TABLE mdm.source_systems
            ADD COLUMN IF NOT EXISTS last_sync_at       timestamptz,
            ADD COLUMN IF NOT EXISTS last_sync_status   varchar(20),
            ADD COLUMN IF NOT EXISTS contact_party_id   uuid,
            ADD COLUMN IF NOT EXISTS feed_type          varchar(30),
            ADD COLUMN IF NOT EXISTS coverage_scope     text[],
            ADD COLUMN IF NOT EXISTS sla_definition     jsonb DEFAULT '{}'::jsonb;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='source_system') THEN
        ALTER TABLE mdm.source_system
            ADD COLUMN IF NOT EXISTS last_sync_at       timestamptz,
            ADD COLUMN IF NOT EXISTS last_sync_status   varchar(20),
            ADD COLUMN IF NOT EXISTS contact_party_id   uuid,
            ADD COLUMN IF NOT EXISTS feed_type          varchar(30),
            ADD COLUMN IF NOT EXISTS coverage_scope     text[],
            ADD COLUMN IF NOT EXISTS sla_definition     jsonb DEFAULT '{}'::jsonb;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 6. mdm.steward — align with domain stewards (user_id-keyed)
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='steward') THEN
        ALTER TABLE mdm.steward
            ADD COLUMN IF NOT EXISTS user_id                     uuid,
            ADD COLUMN IF NOT EXISTS entity_type_scope           text[],
            ADD COLUMN IF NOT EXISTS can_override_survivorship   bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS can_merge                   bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS can_publish                 bool DEFAULT false;
        CREATE INDEX IF NOT EXISTS idx_steward_user_id
            ON mdm.steward (user_id) WHERE user_id IS NOT NULL;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 7. mdm.hierarchy — purpose, materialization metadata
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='hierarchy') THEN
        ALTER TABLE mdm.hierarchy
            ADD COLUMN IF NOT EXISTS hierarchy_purpose      varchar(30),
            ADD COLUMN IF NOT EXISTS is_inherited           bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS max_depth              int4,
            ADD COLUMN IF NOT EXISTS current_depth          int4,
            ADD COLUMN IF NOT EXISTS materialization_status varchar(20) DEFAULT 'STALE',
            ADD COLUMN IF NOT EXISTS last_materialized_at   timestamptz;
        CREATE INDEX IF NOT EXISTS idx_hierarchy_status
            ON mdm.hierarchy (materialization_status);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 8. mdm.hierarchy_closure — carry entity types + path
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='hierarchy_closure') THEN
        ALTER TABLE mdm.hierarchy_closure
            ADD COLUMN IF NOT EXISTS ancestor_entity_type   varchar(30),
            ADD COLUMN IF NOT EXISTS descendant_entity_type varchar(30),
            ADD COLUMN IF NOT EXISTS path                   text;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 9. mdm.entity_relationship — registry + ownership
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='entity_relationship') THEN
        ALTER TABLE mdm.entity_relationship
            ADD COLUMN IF NOT EXISTS relationship_sub_type varchar(50),
            ADD COLUMN IF NOT EXISTS ownership_pct         numeric(7,4),
            ADD COLUMN IF NOT EXISTS is_primary            bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS source_system_id      uuid,
            ADD COLUMN IF NOT EXISTS is_current            bool DEFAULT true;
        CREATE INDEX IF NOT EXISTS idx_entity_relationship_current
            ON mdm.entity_relationship (source_entity_type, source_entity_id, is_current);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 10. mdm.xref — authority + verification
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='xref') THEN
        ALTER TABLE mdm.xref
            ADD COLUMN IF NOT EXISTS authority_level   varchar(20) DEFAULT 'ADVISORY',
            ADD COLUMN IF NOT EXISTS is_verified       bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS verified_at       timestamptz,
            ADD COLUMN IF NOT EXISTS verified_by       uuid,
            ADD COLUMN IF NOT EXISTS confidence_score  numeric(5,2);
        CREATE INDEX IF NOT EXISTS idx_xref_authority
            ON mdm.xref (authority_level) WHERE authority_level <> 'ADVISORY';
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 11. mdm.dq_rule — structure (rule_cd, rule_type, attribute, weight)
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='dq_rule') THEN
        ALTER TABLE mdm.dq_rule
            ADD COLUMN IF NOT EXISTS rule_cd           varchar(50),
            ADD COLUMN IF NOT EXISTS rule_type         varchar(30),
            ADD COLUMN IF NOT EXISTS attribute_name    varchar(100),
            ADD COLUMN IF NOT EXISTS weight            numeric(5,2) DEFAULT 1.0,
            ADD COLUMN IF NOT EXISTS threshold         numeric(24,8),
            ADD COLUMN IF NOT EXISTS run_frequency     varchar(20);
        CREATE UNIQUE INDEX IF NOT EXISTS uq_dq_rule_cd
            ON mdm.dq_rule (tenant_id, rule_cd) WHERE rule_cd IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_dq_rule_type
            ON mdm.dq_rule (rule_type) WHERE rule_type IS NOT NULL;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 12. mdm.record_version — lineage + correlation
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='record_version') THEN
        ALTER TABLE mdm.record_version
            ADD COLUMN IF NOT EXISTS change_reason     varchar(500),
            ADD COLUMN IF NOT EXISTS change_request_id uuid,
            ADD COLUMN IF NOT EXISTS correlation_id    uuid;
        CREATE INDEX IF NOT EXISTS idx_record_version_correlation
            ON mdm.record_version (correlation_id) WHERE correlation_id IS NOT NULL;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 13. mdm.portfolio — expand from stub
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='portfolio') THEN
        ALTER TABLE mdm.portfolio
            ADD COLUMN IF NOT EXISTS base_currency           varchar(3),
            ADD COLUMN IF NOT EXISTS inception_date          date,
            ADD COLUMN IF NOT EXISTS termination_date        date,
            ADD COLUMN IF NOT EXISTS benchmark_id            uuid,
            ADD COLUMN IF NOT EXISTS mandate_id              uuid,
            ADD COLUMN IF NOT EXISTS client_group_id         uuid,
            ADD COLUMN IF NOT EXISTS party_id                uuid,
            ADD COLUMN IF NOT EXISTS strategy_cd             varchar(50),
            ADD COLUMN IF NOT EXISTS risk_profile            varchar(20),
            ADD COLUMN IF NOT EXISTS discretionary           bool DEFAULT true,
            ADD COLUMN IF NOT EXISTS regulatory_constraints  text,
            ADD COLUMN IF NOT EXISTS reporting_currency      varchar(3),
            ADD COLUMN IF NOT EXISTS source_system_id        uuid,
            ADD COLUMN IF NOT EXISTS golden_record_id        uuid,
            ADD COLUMN IF NOT EXISTS is_golden_record        bool DEFAULT true,
            ADD COLUMN IF NOT EXISTS dq_score                numeric(5,2),
            ADD COLUMN IF NOT EXISTS lifecycle_stage         varchar(20) DEFAULT 'ACTIVE';
        CREATE INDEX IF NOT EXISTS idx_portfolio_benchmark
            ON mdm.portfolio (benchmark_id) WHERE benchmark_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_portfolio_client_grp
            ON mdm.portfolio (client_group_id) WHERE client_group_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_portfolio_party
            ON mdm.portfolio (party_id) WHERE party_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_portfolio_status
            ON mdm.portfolio (status);
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 14. mdm.mandate — expand from stub
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='mandate') THEN
        ALTER TABLE mdm.mandate
            ADD COLUMN IF NOT EXISTS account_id             uuid,
            ADD COLUMN IF NOT EXISTS client_group_id        uuid,
            ADD COLUMN IF NOT EXISTS effective_from         date DEFAULT CURRENT_DATE,
            ADD COLUMN IF NOT EXISTS effective_to           date,
            ADD COLUMN IF NOT EXISTS investment_objective   text,
            ADD COLUMN IF NOT EXISTS time_horizon           varchar(30),
            ADD COLUMN IF NOT EXISTS liquidity_requirement  varchar(30),
            ADD COLUMN IF NOT EXISTS reporting_currency     varchar(3),
            ADD COLUMN IF NOT EXISTS fee_schedule_id        uuid,
            ADD COLUMN IF NOT EXISTS regulatory_constraints text,
            ADD COLUMN IF NOT EXISTS status_reason          varchar(500),
            ADD COLUMN IF NOT EXISTS is_discretionary       bool DEFAULT true,
            ADD COLUMN IF NOT EXISTS source_system_id       uuid;
        CREATE INDEX IF NOT EXISTS idx_mandate_client_grp
            ON mdm.mandate (client_group_id) WHERE client_group_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_mandate_account
            ON mdm.mandate (account_id) WHERE account_id IS NOT NULL;
    END IF;
END $do$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 15. mdm.portfolio_composite — expand from stub
-- ═══════════════════════════════════════════════════════════════════════════
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='portfolio_composite') THEN
        ALTER TABLE mdm.portfolio_composite
            ADD COLUMN IF NOT EXISTS composite_type         varchar(30),
            ADD COLUMN IF NOT EXISTS benchmark_id           uuid,
            ADD COLUMN IF NOT EXISTS parent_composite_id    uuid,
            ADD COLUMN IF NOT EXISTS inception_date         date,
            ADD COLUMN IF NOT EXISTS termination_date       date,
            ADD COLUMN IF NOT EXISTS base_currency          varchar(3),
            ADD COLUMN IF NOT EXISTS is_gips_compliant      bool DEFAULT false,
            ADD COLUMN IF NOT EXISTS gips_verification_date date,
            ADD COLUMN IF NOT EXISTS creation_date          date,
            ADD COLUMN IF NOT EXISTS definition             text,
            ADD COLUMN IF NOT EXISTS client_group_id        uuid;
        CREATE INDEX IF NOT EXISTS idx_pc_parent
            ON mdm.portfolio_composite (parent_composite_id) WHERE parent_composite_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_pc_benchmark
            ON mdm.portfolio_composite (benchmark_id) WHERE benchmark_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_pc_client_grp
            ON mdm.portfolio_composite (client_group_id) WHERE client_group_id IS NOT NULL;
    END IF;
END $do$;

DO $done$
BEGIN
    RAISE NOTICE 'batch 4: generic table upgrades complete';
END $done$;

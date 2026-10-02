-- 20261031_001_mdm_generic_table_upgrades.down.sql
-- Reverses only the columns added by the up migration. Existing columns
-- (that pre-dated batch 4) are untouched.
-- Drops happen in reverse order of the up migration.
-- All operations are DROP COLUMN IF EXISTS or DROP INDEX IF EXISTS.

-- 15. portfolio_composite
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='portfolio_composite') THEN
        DROP INDEX IF EXISTS mdm.idx_pc_parent;
        DROP INDEX IF EXISTS mdm.idx_pc_benchmark;
        DROP INDEX IF EXISTS mdm.idx_pc_client_grp;
        ALTER TABLE mdm.portfolio_composite
            DROP COLUMN IF EXISTS composite_type,
            DROP COLUMN IF EXISTS benchmark_id,
            DROP COLUMN IF EXISTS parent_composite_id,
            DROP COLUMN IF EXISTS inception_date,
            DROP COLUMN IF EXISTS termination_date,
            DROP COLUMN IF EXISTS base_currency,
            DROP COLUMN IF EXISTS is_gips_compliant,
            DROP COLUMN IF EXISTS gips_verification_date,
            DROP COLUMN IF EXISTS creation_date,
            DROP COLUMN IF EXISTS definition,
            DROP COLUMN IF EXISTS client_group_id;
    END IF;
END $do$;

-- 14. mandate
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='mandate') THEN
        DROP INDEX IF EXISTS mdm.idx_mandate_client_grp;
        DROP INDEX IF EXISTS mdm.idx_mandate_account;
        ALTER TABLE mdm.mandate
            DROP COLUMN IF EXISTS account_id,
            DROP COLUMN IF EXISTS client_group_id,
            DROP COLUMN IF EXISTS effective_from,
            DROP COLUMN IF EXISTS effective_to,
            DROP COLUMN IF EXISTS investment_objective,
            DROP COLUMN IF EXISTS time_horizon,
            DROP COLUMN IF EXISTS liquidity_requirement,
            DROP COLUMN IF EXISTS reporting_currency,
            DROP COLUMN IF EXISTS fee_schedule_id,
            DROP COLUMN IF EXISTS regulatory_constraints,
            DROP COLUMN IF EXISTS status_reason,
            DROP COLUMN IF EXISTS is_discretionary,
            DROP COLUMN IF EXISTS source_system_id;
    END IF;
END $do$;

-- 13. portfolio
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='portfolio') THEN
        DROP INDEX IF EXISTS mdm.idx_portfolio_benchmark;
        DROP INDEX IF EXISTS mdm.idx_portfolio_client_grp;
        DROP INDEX IF EXISTS mdm.idx_portfolio_party;
        DROP INDEX IF EXISTS mdm.idx_portfolio_status;
        ALTER TABLE mdm.portfolio
            DROP COLUMN IF EXISTS base_currency,
            DROP COLUMN IF EXISTS inception_date,
            DROP COLUMN IF EXISTS termination_date,
            DROP COLUMN IF EXISTS benchmark_id,
            DROP COLUMN IF EXISTS mandate_id,
            DROP COLUMN IF EXISTS client_group_id,
            DROP COLUMN IF EXISTS party_id,
            DROP COLUMN IF EXISTS strategy_cd,
            DROP COLUMN IF EXISTS risk_profile,
            DROP COLUMN IF EXISTS discretionary,
            DROP COLUMN IF EXISTS regulatory_constraints,
            DROP COLUMN IF EXISTS reporting_currency,
            DROP COLUMN IF EXISTS source_system_id,
            DROP COLUMN IF EXISTS golden_record_id,
            DROP COLUMN IF EXISTS is_golden_record,
            DROP COLUMN IF EXISTS dq_score,
            DROP COLUMN IF EXISTS lifecycle_stage;
    END IF;
END $do$;

-- 12. record_version
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='record_version') THEN
        DROP INDEX IF EXISTS mdm.idx_record_version_correlation;
        ALTER TABLE mdm.record_version
            DROP COLUMN IF EXISTS change_reason,
            DROP COLUMN IF EXISTS change_request_id,
            DROP COLUMN IF EXISTS correlation_id;
    END IF;
END $do$;

-- 11. dq_rule
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='dq_rule') THEN
        DROP INDEX IF EXISTS mdm.uq_dq_rule_cd;
        DROP INDEX IF EXISTS mdm.idx_dq_rule_type;
        ALTER TABLE mdm.dq_rule
            DROP COLUMN IF EXISTS rule_cd,
            DROP COLUMN IF EXISTS rule_type,
            DROP COLUMN IF EXISTS attribute_name,
            DROP COLUMN IF EXISTS weight,
            DROP COLUMN IF EXISTS threshold,
            DROP COLUMN IF EXISTS run_frequency;
    END IF;
END $do$;

-- 10. xref
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='xref') THEN
        DROP INDEX IF EXISTS mdm.idx_xref_authority;
        ALTER TABLE mdm.xref
            DROP COLUMN IF EXISTS authority_level,
            DROP COLUMN IF EXISTS is_verified,
            DROP COLUMN IF EXISTS verified_at,
            DROP COLUMN IF EXISTS verified_by,
            DROP COLUMN IF EXISTS confidence_score;
    END IF;
END $do$;

-- 9. entity_relationship
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='entity_relationship') THEN
        DROP INDEX IF EXISTS mdm.idx_entity_relationship_current;
        ALTER TABLE mdm.entity_relationship
            DROP COLUMN IF EXISTS relationship_sub_type,
            DROP COLUMN IF EXISTS ownership_pct,
            DROP COLUMN IF EXISTS is_primary,
            DROP COLUMN IF EXISTS source_system_id,
            DROP COLUMN IF EXISTS is_current;
    END IF;
END $do$;

-- 8. hierarchy_closure
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='hierarchy_closure') THEN
        ALTER TABLE mdm.hierarchy_closure
            DROP COLUMN IF EXISTS ancestor_entity_type,
            DROP COLUMN IF EXISTS descendant_entity_type,
            DROP COLUMN IF EXISTS path;
    END IF;
END $do$;

-- 7. hierarchy
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='hierarchy') THEN
        DROP INDEX IF EXISTS mdm.idx_hierarchy_status;
        ALTER TABLE mdm.hierarchy
            DROP COLUMN IF EXISTS hierarchy_purpose,
            DROP COLUMN IF EXISTS is_inherited,
            DROP COLUMN IF EXISTS max_depth,
            DROP COLUMN IF EXISTS current_depth,
            DROP COLUMN IF EXISTS materialization_status,
            DROP COLUMN IF EXISTS last_materialized_at;
    END IF;
END $do$;

-- 6. steward
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='steward') THEN
        DROP INDEX IF EXISTS mdm.idx_steward_user_id;
        ALTER TABLE mdm.steward
            DROP COLUMN IF EXISTS user_id,
            DROP COLUMN IF EXISTS entity_type_scope,
            DROP COLUMN IF EXISTS can_override_survivorship,
            DROP COLUMN IF EXISTS can_merge,
            DROP COLUMN IF EXISTS can_publish;
    END IF;
END $do$;

-- 5. source_systems / source_system
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='source_systems') THEN
        ALTER TABLE mdm.source_systems
            DROP COLUMN IF EXISTS last_sync_at,
            DROP COLUMN IF EXISTS last_sync_status,
            DROP COLUMN IF EXISTS contact_party_id,
            DROP COLUMN IF EXISTS feed_type,
            DROP COLUMN IF EXISTS coverage_scope,
            DROP COLUMN IF EXISTS sla_definition;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='source_system') THEN
        ALTER TABLE mdm.source_system
            DROP COLUMN IF EXISTS last_sync_at,
            DROP COLUMN IF EXISTS last_sync_status,
            DROP COLUMN IF EXISTS contact_party_id,
            DROP COLUMN IF EXISTS feed_type,
            DROP COLUMN IF EXISTS coverage_scope,
            DROP COLUMN IF EXISTS sla_definition;
    END IF;
END $do$;

-- 4. change_request
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='change_request') THEN
        DROP INDEX IF EXISTS mdm.uq_change_request_ref;
        DROP INDEX IF EXISTS mdm.idx_change_request_assigned;
        ALTER TABLE mdm.change_request
            DROP COLUMN IF EXISTS request_ref,
            DROP COLUMN IF EXISTS assigned_to,
            DROP COLUMN IF EXISTS approved_by,
            DROP COLUMN IF EXISTS approved_at,
            DROP COLUMN IF EXISTS applied_at,
            DROP COLUMN IF EXISTS rejection_reason,
            DROP COLUMN IF EXISTS priority,
            DROP COLUMN IF EXISTS request_reason;
    END IF;
END $do$;

-- 3. survivorship_rule
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='survivorship_rule') THEN
        DROP INDEX IF EXISTS mdm.uq_survivorship_rule_structured;
        ALTER TABLE mdm.survivorship_rule
            DROP COLUMN IF EXISTS field_group,
            DROP COLUMN IF EXISTS sub_type,
            DROP COLUMN IF EXISTS min_confidence,
            DROP COLUMN IF EXISTS max_staleness_hours,
            DROP COLUMN IF EXISTS manual_override_allowed,
            DROP COLUMN IF EXISTS source_priority_structured;
    END IF;
END $do$;

-- 2. merge_log
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='merge_log') THEN
        DROP INDEX IF EXISTS mdm.idx_merge_log_merged_at;
        ALTER TABLE mdm.merge_log
            DROP COLUMN IF EXISTS match_candidate_id,
            DROP COLUMN IF EXISTS reversible,
            DROP COLUMN IF EXISTS reversal_data,
            DROP COLUMN IF EXISTS downstream_notified,
            DROP COLUMN IF EXISTS merged_by,
            DROP COLUMN IF EXISTS merged_at;
    END IF;
END $do$;

-- 1. match_rule
DO $do$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='match_rule') THEN
        DROP INDEX IF EXISTS mdm.uq_match_rule_cd;
        DROP INDEX IF EXISTS mdm.idx_match_rule_priority;
        ALTER TABLE mdm.match_rule
            DROP COLUMN IF EXISTS rule_cd,
            DROP COLUMN IF EXISTS sub_type,
            DROP COLUMN IF EXISTS priority,
            DROP COLUMN IF EXISTS deterministic_keys,
            DROP COLUMN IF EXISTS fuzzy_keys,
            DROP COLUMN IF EXISTS threshold_no_match,
            DROP COLUMN IF EXISTS version,
            DROP COLUMN IF EXISTS is_active_legacy;
    END IF;
END $do$;

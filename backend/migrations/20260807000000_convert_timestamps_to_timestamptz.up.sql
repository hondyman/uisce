-- +migrate Up
-- Convert all timestamp columns to timestamptz for UTC storage
-- This migration ensures all datetime values are stored with timezone information

-- ============================================================================
-- totalddl.sql columns
-- ============================================================================

ALTER TABLE public.app_user ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE public.explorer_saved_query ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.explorer_saved_query ALTER COLUMN updated_at TYPE TIMESTAMPTZ;
ALTER TABLE public.explorer_saved_query ALTER COLUMN last_run_at TYPE TIMESTAMPTZ;

ALTER TABLE public.model_upgrade_audit ALTER COLUMN decided_at TYPE TIMESTAMPTZ;

ALTER TABLE public.policies ALTER COLUMN start_date TYPE TIMESTAMPTZ;
ALTER TABLE public.policies ALTER COLUMN end_date TYPE TIMESTAMPTZ;
ALTER TABLE public.policies ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.policies ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.rule_config_changelog ALTER COLUMN triggered_at TYPE TIMESTAMPTZ;

ALTER TABLE public.asset ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE public.broker_apis ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.broker_apis ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.broker_events ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.broker_events ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.customers ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.customers ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.event_subscriptions ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.event_subscriptions ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.orders ALTER COLUMN order_date TYPE TIMESTAMPTZ;
ALTER TABLE public.orders ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.orders ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.metadata_fields ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.metadata_fields ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.metadata_events ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE public.metadata_events ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE public.metadata_event_logs ALTER COLUMN execution_start TYPE TIMESTAMPTZ;
ALTER TABLE public.metadata_event_logs ALTER COLUMN execution_end TYPE TIMESTAMPTZ;
ALTER TABLE public.metadata_event_logs ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE public.metadata_event_versions ALTER COLUMN created_at TYPE TIMESTAMPTZ;

-- ============================================================================
-- sql/phase_3_23_schema.sql columns
-- ============================================================================

ALTER TABLE discovery_runs ALTER COLUMN started_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_runs ALTER COLUMN completed_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_runs ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_runs ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE discovery_candidates ALTER COLUMN discovered_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_candidates ALTER COLUMN approved_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_candidates ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE discovery_candidates ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE feature_catalog_mappings ALTER COLUMN mapped_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_catalog_mappings ALTER COLUMN deprecated_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_catalog_mappings ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_catalog_mappings ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE discovery_statistics ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE discovery_logs ALTER COLUMN timestamp TYPE TIMESTAMPTZ;
ALTER TABLE discovery_logs ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE discovery_audit ALTER COLUMN timestamp TYPE TIMESTAMPTZ;
ALTER TABLE discovery_audit ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE feature_metadata ALTER COLUMN last_computed_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_metadata ALTER COLUMN last_used_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_metadata ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE feature_metadata ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- ============================================================================
-- pkg/bp/bp_advanced_features_schema.sql columns
-- ============================================================================

ALTER TABLE bp_ai_models ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_ai_models ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_semantic_intents ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_semantic_intents ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_scoring_matrices ALTER COLUMN last_tuned_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_scoring_matrices ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_scoring_matrices ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_time_series_forecasts ALTER COLUMN forecast_timestamp TYPE TIMESTAMPTZ;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN last_retrained_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_adaptive_triggers ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_adaptive_triggers ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_resilience_policies ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_resilience_policies ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_tenant_branch_overrides ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_tenant_branch_overrides ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_branch_analytics_extended ALTER COLUMN metric_period TYPE TIMESTAMPTZ;
ALTER TABLE bp_branch_analytics_extended ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_branch_analytics_extended ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_collaborative_decisions ALTER COLUMN started_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN completed_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN timeout_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_geofence_rules ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_geofence_rules ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_blockchain_audit ALTER COLUMN event_timestamp TYPE TIMESTAMPTZ;
ALTER TABLE bp_blockchain_audit ALTER COLUMN last_verified_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_blockchain_audit ALTER COLUMN expiration_date TYPE TIMESTAMPTZ;
ALTER TABLE bp_blockchain_audit ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_blockchain_audit ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_nl_configurations ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_nl_configurations ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_resource_pools ALTER COLUMN last_scaled_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_resource_pools ALTER COLUMN last_load_check_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_resource_pools ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_resource_pools ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

ALTER TABLE bp_explainability_records ALTER COLUMN decision_timestamp TYPE TIMESTAMPTZ;
ALTER TABLE bp_explainability_records ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE bp_explainability_records ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- +migrate Down
-- Revert timestamptz columns back to timestamp (without timezone)

-- ============================================================================
-- totalddl.sql columns (revert)
-- ============================================================================

ALTER TABLE public.app_user ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE public.explorer_saved_query ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.explorer_saved_query ALTER COLUMN updated_at TYPE TIMESTAMP;
ALTER TABLE public.explorer_saved_query ALTER COLUMN last_run_at TYPE TIMESTAMP;

ALTER TABLE public.model_upgrade_audit ALTER COLUMN decided_at TYPE TIMESTAMP;

ALTER TABLE public.policies ALTER COLUMN start_date TYPE TIMESTAMP;
ALTER TABLE public.policies ALTER COLUMN end_date TYPE TIMESTAMP;
ALTER TABLE public.policies ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.policies ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.rule_config_changelog ALTER COLUMN triggered_at TYPE TIMESTAMP;

ALTER TABLE public.asset ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE public.broker_apis ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.broker_apis ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.broker_events ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.broker_events ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.customers ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.customers ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.event_subscriptions ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.event_subscriptions ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.orders ALTER COLUMN order_date TYPE TIMESTAMP;
ALTER TABLE public.orders ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.orders ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.metadata_fields ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.metadata_fields ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.metadata_events ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE public.metadata_events ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE public.metadata_event_logs ALTER COLUMN execution_start TYPE TIMESTAMP;
ALTER TABLE public.metadata_event_logs ALTER COLUMN execution_end TYPE TIMESTAMP;
ALTER TABLE public.metadata_event_logs ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE public.metadata_event_versions ALTER COLUMN created_at TYPE TIMESTAMP;

-- ============================================================================
-- sql/phase_3_23_schema.sql columns (revert)
-- ============================================================================

ALTER TABLE discovery_runs ALTER COLUMN started_at TYPE TIMESTAMP;
ALTER TABLE discovery_runs ALTER COLUMN completed_at TYPE TIMESTAMP;
ALTER TABLE discovery_runs ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE discovery_runs ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE discovery_candidates ALTER COLUMN discovered_at TYPE TIMESTAMP;
ALTER TABLE discovery_candidates ALTER COLUMN approved_at TYPE TIMESTAMP;
ALTER TABLE discovery_candidates ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE discovery_candidates ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE feature_catalog_mappings ALTER COLUMN mapped_at TYPE TIMESTAMP;
ALTER TABLE feature_catalog_mappings ALTER COLUMN deprecated_at TYPE TIMESTAMP;
ALTER TABLE feature_catalog_mappings ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE feature_catalog_mappings ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE discovery_statistics ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE discovery_logs ALTER COLUMN timestamp TYPE TIMESTAMP;
ALTER TABLE discovery_logs ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE discovery_audit ALTER COLUMN timestamp TYPE TIMESTAMP;
ALTER TABLE discovery_audit ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE feature_metadata ALTER COLUMN last_computed_at TYPE TIMESTAMP;
ALTER TABLE feature_metadata ALTER COLUMN last_used_at TYPE TIMESTAMP;
ALTER TABLE feature_metadata ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE feature_metadata ALTER COLUMN updated_at TYPE TIMESTAMP;

-- ============================================================================
-- pkg/bp/bp_advanced_features_schema.sql columns (revert)
-- ============================================================================

ALTER TABLE bp_ai_models ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_ai_models ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_semantic_intents ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_semantic_intents ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_scoring_matrices ALTER COLUMN last_tuned_at TYPE TIMESTAMP;
ALTER TABLE bp_scoring_matrices ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_scoring_matrices ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_time_series_forecasts ALTER COLUMN forecast_timestamp TYPE TIMESTAMP;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN last_retrained_at TYPE TIMESTAMP;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_time_series_forecasts ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_adaptive_triggers ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_adaptive_triggers ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_resilience_policies ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_resilience_policies ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_tenant_branch_overrides ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_tenant_branch_overrides ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_branch_analytics_extended ALTER COLUMN metric_period TYPE TIMESTAMP;
ALTER TABLE bp_branch_analytics_extended ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_branch_analytics_extended ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_collaborative_decisions ALTER COLUMN started_at TYPE TIMESTAMP;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN completed_at TYPE TIMESTAMP;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN timeout_at TYPE TIMESTAMP;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_collaborative_decisions ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_geofence_rules ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_geofence_rules ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_blockchain_audit ALTER COLUMN event_timestamp TYPE TIMESTAMP;
ALTER TABLE bp_blockchain_audit ALTER COLUMN last_verified_at TYPE TIMESTAMP;
ALTER TABLE bp_blockchain_audit ALTER COLUMN expiration_date TYPE TIMESTAMP;
ALTER TABLE bp_blockchain_audit ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_blockchain_audit ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_nl_configurations ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_nl_configurations ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_resource_pools ALTER COLUMN last_scaled_at TYPE TIMESTAMP;
ALTER TABLE bp_resource_pools ALTER COLUMN last_load_check_at TYPE TIMESTAMP;
ALTER TABLE bp_resource_pools ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_resource_pools ALTER COLUMN updated_at TYPE TIMESTAMP;

ALTER TABLE bp_explainability_records ALTER COLUMN decision_timestamp TYPE TIMESTAMP;
ALTER TABLE bp_explainability_records ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE bp_explainability_records ALTER COLUMN updated_at TYPE TIMESTAMP;

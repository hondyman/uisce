package queries

const (
	SetTenantGUC = `SELECT set_config('app.current_tenant', $1, false)`

	GetPortfolio = `
		SELECT id, tenant_id, name, aum, drift, last_rebalance::text, target_model, constraints
		FROM portfolios WHERE id = $1
	`

	GetPortfolioHoldings = `
		SELECT id, symbol, shares, current_price, cost_basis, purchase_date, tax_lot_id, sector
		FROM portfolio_holdings WHERE portfolio_id = $1
	`

	ListPortfolios = `
		SELECT p.id, p.tenant_id, p.name, p.aum, p.drift, p.last_rebalance::text,
		       p.target_model, p.constraints, p.rebalance_status,
		       COALESCE(p.risk_score, 0), COALESCE(p.alpha, 0),
		       COALESCE(p.sector_attribution, '{}'::jsonb), COALESCE(p.tax_saved, 0),
		       p.policy_document,
		       (SELECT COUNT(*) FROM portfolio_holdings h WHERE h.portfolio_id = p.id)
		FROM portfolios p
		WHERE p.tenant_id = $1
		ORDER BY p.aum DESC
	`

	ListRebalancePlans = `
		SELECT id, portfolio_id, timestamp::text, current_drift, expected_drift,
		       tax_savings, confidence, status, rationale, COALESCE(summary, ''), proposed_trades
		FROM rebalance_plans
		WHERE portfolio_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	UpdatePortfolioState = `
		UPDATE portfolios
		SET drift = $2,
			tax_saved = $3,
			last_rebalance = NOW(),
			rebalance_status = 'completed',
			updated_at = NOW()
		WHERE id = $1
	`

	InsertRebalancePlan = `
		INSERT INTO rebalance_plans (
			portfolio_id, timestamp, current_drift, expected_drift,
			tax_savings, confidence, status, rationale, proposed_trades, tax_analysis
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	UpdatePlanSummary = `
		UPDATE rebalance_plans SET summary = $2, updated_at = NOW() WHERE id = $1
	`

	InsertAuditLog = `
		INSERT INTO audit_logs (tenant_id, user_id, action, resource, resource_id, allowed)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	GetBacktestResults = `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE portfolio_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	GetBacktestByID = `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE id = $1
	`

	GetBacktestByRecommendationAndPortfolio = `
		SELECT id, recommendation_id, portfolio_id, simulation_type, start_date, end_date,
		       baseline_return, recommendation_return, alpha_generated, beta_adjusted_return,
		       sharpe_ratio_baseline, sharpe_ratio_recommended, max_drawdown_baseline, max_drawdown_recommended,
		       tax_savings_accumulated, transaction_costs, net_benefit, confidence, simulation_data, created_at
		FROM backtest_results
		WHERE recommendation_id = $1 AND portfolio_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	GetHierarchyRules = `
		SELECT id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at
		FROM entity_hierarchy_rules WHERE tenant_id=$1
	`

	GetHierarchyRuleByTypes = `
		SELECT id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at
		FROM entity_hierarchy_rules WHERE tenant_id=$1 AND parent_model_type=$2 AND child_model_type=$3
	`

	InsertHierarchyRule = `
		INSERT INTO entity_hierarchy_rules (id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, max_children, description, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET allowed=EXCLUDED.allowed, ownership_types=EXCLUDED.ownership_types, max_children=EXCLUDED.max_children, description=EXCLUDED.description, notes=EXCLUDED.notes, updated_at=EXCLUDED.updated_at
	`

	InsertHierarchyRuleNoConflict = `
		INSERT INTO entity_hierarchy_rules (id, tenant_id, parent_model_type, child_model_type, allowed, ownership_types, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO NOTHING
	`

	UpdateHierarchyRule = `
		UPDATE entity_hierarchy_rules SET allowed=$1, ownership_types=$2, max_children=$3, description=$4, notes=$5, updated_at=$6 WHERE id=$7 AND tenant_id=$8
	`

	DeleteHierarchyRule = `
		DELETE FROM entity_hierarchy_rules WHERE tenant_id=$1 AND parent_model_type=$2 AND child_model_type=$3
	`

	InsertEntityRelationship = `
		INSERT INTO entity_relationships (id, tenant_id, owner_id, owned_id, ownership_percentage, ownership_type, incepting_date, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	InsertHierarchyAuditLog = `
		INSERT INTO entity_hierarchy_audit_log (id, entity_id, tenant_id, action, created_by, parent_model_type, child_model_type, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`

	CountEntitiesByTenant = `SELECT count(*) FROM entities WHERE tenant_id=$1`

	ListWorkflowPolicies = `SELECT expression FROM core_policy WHERE tenant_id = $1 AND scope = 'workflow' AND type = 'authorization'`

	ListBusinessObjects = `
		SELECT id, tenant_id, name, display_name, description, icon,
		       metadata, created_at, updated_at
		FROM business_objects
		WHERE tenant_id = $1
		ORDER BY name
	`

	ListBOFieldsWithJoin = `
		SELECT f.id, f.business_object_id, f.name, f.label, f.type,
		       f.is_required, f.is_unique, f.enum_id, f.ref_object_id,
		       f.default_value, f.validation_json, f.visibility_json
		FROM bo_fields f
		JOIN business_objects bo ON f.business_object_id = bo.id
		WHERE bo.tenant_id = $1
		ORDER BY f.business_object_id, f.sequence
	`

	GetLayout = `
		SELECT id, tenant_id, bo_id, layout_name, layout_type, layout_description,
		       default_columns, mobile_layout, is_default_layout, is_active
		FROM page_layouts
		WHERE id = $1
	`

	GetBusinessObject = `
		SELECT id, tenant_id, bo_name, bo_description, entity_type,
		       allow_custom_fields, allow_field_deletion, is_system_bo, is_active
		FROM business_objects
		WHERE id = $1
	`

	ListBOFields = `
		SELECT id, bo_id, field_name, field_type, display_label, display_order,
		       section_name, help_text, placeholder_text, is_required, is_readonly,
		       is_searchable, is_sortable, max_length, min_value, max_value,
		       decimal_places, date_format, reference_bo_id, reference_display_field,
		       picklist_values, default_value, is_system_field, is_custom_field,
		       validation_rule_ids
		FROM bo_fields
		WHERE bo_id = $1
		ORDER BY display_order ASC
	`

	ListValidationRules = `
		SELECT id, tenant_id, rule_name, rule_description, rule_category, severity,
		       error_message, help_message, condition_type, condition_json,
		       execute_client_side, execute_server_side, run_on_blur, run_on_change,
		       run_on_submit, requires_database_call, is_active
		FROM validation_rules
		WHERE id = ANY($1) AND is_active = true
		ORDER BY rule_name ASC
	`

	GetValidationRule = `
		SELECT id, tenant_id, rule_name, rule_description, rule_category, severity,
		       error_message, help_message, condition_type, condition_json,
		       execute_client_side, execute_server_side, run_on_blur, run_on_change,
		       run_on_submit, requires_database_call, is_active
		FROM validation_rules
		WHERE id = $1
	`

	ListLayoutSections = `
		SELECT id, layout_id, section_order, section_title, section_description,
		       section_columns, is_collapsible, is_initially_collapsed, has_border,
		       background_color, is_visible, help_text, field_ids
		FROM layout_sections
		WHERE layout_id = $1
		ORDER BY section_order ASC
	`

	ListLayoutActions = `
		SELECT id, layout_id, action_order, action_label, action_type, action_icon,
		       requires_validation, requires_confirmation, confirmation_message,
		       triggers_bp_id, is_visible, is_enabled, button_style, button_size,
		       success_message, error_message, redirect_on_success
		FROM layout_actions
		WHERE layout_id = $1
		ORDER BY action_order ASC
	`

	DeleteBPSteps = `DELETE FROM bp_steps WHERE business_process_id = $1`

	InsertBPStep = `
		INSERT INTO bp_steps (
			id, business_process_id, step_order, step_type, step_name,
			assignee_role, description, duration_hours, status, config, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	GetBusinessProcess = `
		SELECT id, tenant_id, process_name, description, entity_type, status,
		       is_active, created_by, created_at, updated_by, updated_at,
		       total_duration_hours, version_number
		FROM business_processes
		WHERE id = $1 AND tenant_id = $2
	`

	ListBPSteps = `
		SELECT id, business_process_id, step_order, step_type, step_name,
		       assignee_role, description, duration_hours, status, config, created_at, updated_at
		FROM bp_steps
		WHERE business_process_id = $1
		ORDER BY step_order ASC
	`

	UpsertFormData = `
		INSERT INTO business_process_form_data (entity_id, form_data, status, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (entity_id)
		DO UPDATE SET
			form_data = EXCLUDED.form_data,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	ListBusinessProcesses = `
		SELECT id, tenant_id, process_name, description, entity_type, status,
		       is_active, created_by, created_at, updated_by, updated_at,
		       total_duration_hours, version_number
		FROM business_processes
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	CountBusinessProcesses = `SELECT COUNT(*) FROM business_processes WHERE tenant_id = $1`

	InsertBPExecution = `
		INSERT INTO bp_executions (
			id, tenant_id, business_process_id, entity_id, initiated_by,
			initiated_at, execution_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	UpdateBPExecutionStatus = `
		UPDATE bp_executions
		SET execution_status = $1, workflow_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	InsertAuditTrail = `
		INSERT INTO bp_audit_trail (
			id, tenant_id, business_process_id, action_type, actor_email,
			action_details, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
	`

	GetAuditTrail = `
		SELECT id, tenant_id, business_process_id, action_type, actor_email,
		       actor_role, action_details, timestamp, ip_address
		FROM bp_audit_trail
		WHERE tenant_id = $1 AND business_process_id = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	ArchiveBusinessProcess = `
		UPDATE business_processes
		SET status = 'archived', is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND tenant_id = $2
	`

	GetExecutionHistory = `
		SELECT id, tenant_id, business_process_id, workflow_id, entity_id,
		       initiated_by, initiated_at, completed_at, execution_status,
		       current_step_order, total_duration_minutes, error_message, metadata
		FROM bp_executions
		WHERE tenant_id = $1 AND business_process_id = $2
		ORDER BY initiated_at DESC
		LIMIT $3
	`

	SelectUnprocessedOutcomes = `
		SELECT
			workflow_id, routing_decision_id, branch_id, success,
			completion_time_minutes, expected_time_minutes,
			customer_satisfaction_score, first_time_resolution,
			cost_incurred, error_count, state_features
		FROM workflow_outcomes
		WHERE processed_for_training = false
		  AND completed_at >= NOW() - INTERVAL '1 hour'
		LIMIT $1
	`

	UpdateOutcomeProcessed = `
		UPDATE workflow_outcomes
		SET processed_for_training = true,
			rl_reward = $1,
			updated_at = NOW()
		WHERE workflow_id = $2
	`

	InsertRoutingDecision = `
		INSERT INTO routing_decisions (
			decision_id, workflow_id, tenant_id, datasource_id,
			selected_branch_id, confidence, reasoning, model_scores,
			execution_strategy, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	InsertWorkflowOutcome = `
		INSERT INTO workflow_outcomes (
			workflow_id, routing_decision_id, branch_id, success,
			completion_time_minutes, expected_time_minutes,
			customer_satisfaction_score, first_time_resolution,
			cost_incurred, error_count, state_features,
			created_at, processed_for_training
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), false)
	`

	GetDecisionHistory = `
		SELECT decision_id, selected_branch_id, confidence, reasoning, model_scores, created_at
		FROM routing_decisions
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	GetOutcomeStats = `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN success THEN 1 END) as success,
			COALESCE(AVG(customer_satisfaction_score), 0) as avg_satisfaction,
			COALESCE(AVG(cost_incurred), 0) as avg_cost
		FROM workflow_outcomes
		WHERE tenant_id = $1
		  AND created_at >= NOW() - INTERVAL '1 hour' * $2
	`

	ListBPTriggerEvents = `
		SELECT id, process_id, tenant_id, trigger_type, trigger_name, trigger_condition,
		       action_type, action_config, status, error_message, execution_id, created_at
		FROM bp_trigger_events
		WHERE process_id = $1 AND tenant_id = $2 AND status = 'pending'
		ORDER BY created_at ASC
	`

	UpdateTriggerEventCompleted = `
		UPDATE bp_trigger_events
		SET status = 'completed', execution_id = $1, updated_at = NOW()
		WHERE id = $2
	`

	UpdateTriggerEventFailed = `
		UPDATE bp_trigger_events
		SET status = 'failed', error_message = $1, updated_at = NOW()
		WHERE id = $2
	`

	GetBusinessProcessHeader = `
		SELECT id, tenant_id, process_name, description, is_active
		FROM business_processes
		WHERE id = $1 AND tenant_id = $2
	`

	ListBPStepsForTrigger = `
		SELECT id, process_id, step_order, step_type, step_name, description,
		       duration_hours, assignee_role, validation_rule_ids, condition_json, next_step_id
		FROM bp_steps
		WHERE process_id = $1 AND tenant_id = $2
		ORDER BY step_order ASC
	`

	InsertBranchExecution = `
		INSERT INTO bp_branch_executions (
			tenant_id, datasource_id, workflow_instance_id, step_id,
			branch_id, branch_label, selected_by, condition_evaluation,
			ml_model_score, started_at, completed_at, duration_ms,
			status, result_data, next_step_id, join_strategy,
			is_last_in_join, nesting_level, loop_iteration
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	InsertJoinConvergence = `
		INSERT INTO bp_join_convergences (
			tenant_id, workflow_instance_id, step_id, join_id,
			join_strategy, required_branches, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'waiting')
		RETURNING id
	`

	GetJoinConvergence = `
		SELECT completed_branches, required_branches FROM bp_join_convergences WHERE id = $1
	`

	UpsertBusinessProcess = `
		INSERT INTO business_processes (
			id, tenant_id, process_name, description, entity_type, status,
			is_active, created_by, created_at, total_duration_hours, version_number
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			process_name = $3,
			description = $4,
			entity_type = $5,
			status = $6,
			is_active = $7,
			updated_by = $8,
			updated_at = CURRENT_TIMESTAMP,
			total_duration_hours = $10,
			version_number = version_number + 1
	`

	InsertTriggerEvent = `
		INSERT INTO bp_trigger_events (id, tenant_id, trigger_type, trigger_name, source_system, payload, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
	`

	IncrementAIModelPredictions = `
		UPDATE bp_ai_models SET total_predictions = total_predictions + 1, updated_at = NOW()
		WHERE model_id = $1 AND tenant_id = $2
	`

	GetAIModelAccuracy = `
		SELECT last_accuracy, accuracy_threshold, min_accuracy_drop_threshold
		FROM bp_ai_models
		WHERE model_id = $1 AND tenant_id = $2
	`

	IncrementSemanticIntentMatch = `
		UPDATE bp_semantic_intents
		SET match_count = match_count + 1, avg_confidence = $1
		WHERE intent_id = $2 AND tenant_id = $3
	`

	IncrementScoringMatrix = `
		UPDATE bp_scoring_matrices
		SET evaluations_total = evaluations_total + 1, avg_score = $1
		WHERE id = $2 AND tenant_id = $3
	`

	IncrementAdaptiveTrigger = `
		UPDATE bp_adaptive_triggers
		SET trigger_count = trigger_count + 1
		WHERE step_id = $1 AND tenant_id = $2
	`

	GetResiliencePolicy = `
		SELECT retry_max_attempts, circuit_breaker_enabled, circuit_breaker_failure_threshold,
		       circuit_breaker_fallback_branch_id, total_retries
		FROM bp_resilience_policies
		WHERE step_id = $1 AND tenant_id = $2
		LIMIT 1
	`

	UpsertBranchAnalytics = `
		INSERT INTO bp_branch_analytics_extended
		(tenant_id, branch_id, branch_selection_count, avg_duration_ms, success_rate, anomaly_score, metric_period)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (tenant_id, branch_id, metric_period) DO UPDATE SET
			branch_selection_count = branch_selection_count + 1
	`

	GetBranchAnalytics = `
		SELECT branch_id, branch_selection_count, branch_completion_count, branch_abandonment_count,
		       avg_duration_ms, success_rate, anomaly_detected, anomaly_score
		FROM bp_branch_analytics_extended
		WHERE branch_id = $1 AND tenant_id = $2
		ORDER BY metric_period DESC
		LIMIT 1
	`

	IncrementCollaborativeVotes = `
		UPDATE bp_collaborative_decisions
		SET votes_received = votes_received + 1, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`

	GetGeofenceRule = `
		SELECT id, region_center_lat, region_center_lng, region_radius_km, target_branch_id, geofence_type
		FROM bp_geofence_rules
		WHERE step_id = $1 AND tenant_id = $2 AND is_active = TRUE
		LIMIT 1
	`

	InsertBlockchainAudit = `
		INSERT INTO bp_blockchain_audit
		(tenant_id, workflow_instance_id, event_type, event_hash, verification_status)
		VALUES ($1, $2, 'branch_decision', $3, 'verified')
	`

	GetNLConfiguration = `
		SELECT id, nl_query, generated_branching_config, field_validation_passed,
		       requires_human_approval, human_approval_status
		FROM bp_nl_configurations
		WHERE step_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	InsertExplainabilityRecord = `
		INSERT INTO bp_explainability_records
		(tenant_id, branch_execution_id, selected_branch_id, feature_importance,
		 natural_language_summary, decision_confidence)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	GetExplainability = `
		SELECT id, selected_branch_id, feature_importance, decision_path,
		       natural_language_summary, decision_confidence, alternative_paths
		FROM bp_explainability_records
		WHERE branch_execution_id = $1 AND tenant_id = $2
		LIMIT 1
	`

	IncrementAIModelPredictionsAlt = `
		UPDATE bp_ai_models
		SET predictions_count = predictions_count + 1, last_updated = NOW()
		WHERE model_id = $1 AND tenant_id = $2
	`

	UpsertSemanticIntent = `
		INSERT INTO bp_semantic_intents (intent_id, match_count, avg_confidence, tenant_id)
		VALUES ($1, 1, $2, $3)
		ON CONFLICT (intent_id, tenant_id) DO UPDATE SET
			match_count = match_count + 1,
			avg_confidence = ($2 + avg_confidence) / 2
	`

	UpsertScoringMatrix = `
		INSERT INTO bp_scoring_matrices (matrix_name, evaluations_total, avg_score, tenant_id)
		VALUES ($1, 1, $2, $3)
		ON CONFLICT (matrix_name, tenant_id) DO UPDATE SET
			evaluations_total = evaluations_total + 1,
			avg_score = ($2 + avg_score) / 2
	`

	UpsertAdaptiveTrigger = `
		INSERT INTO bp_adaptive_triggers (trigger_id, triggered_count, last_triggered_at, tenant_id)
		VALUES ($1, 1, NOW(), $2)
		ON CONFLICT (trigger_id, tenant_id) DO UPDATE SET
			triggered_count = triggered_count + 1,
			last_triggered_at = NOW()
	`

	InsertNLConfiguration = `
		INSERT INTO bp_nl_configurations (nl_query, intent_extraction, human_approval_status, tenant_id)
		VALUES ($1, $2, 'pending', $3)
		RETURNING config_id
	`

	InsertAdvancedExplainability = `
		INSERT INTO bp_explainability_records (branch_id, feature_importance, decision_path,
		       natural_language_summary, confidence_score, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING record_id
	`

	InsertBlockchainAuditAlt = `
		INSERT INTO bp_blockchain_audit (event_id, event_type, event_hash, network, tenant_id)
		VALUES ($1, 'branch_decision', $2, 'hyperledger_fabric', $3)
	`

	InsertCoreBO = `
		INSERT INTO core_bo (id, tenant_id, name, storage, version, status, fields, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	UpdateCoreBO = `
		UPDATE core_bo
		SET name = $1, storage = $2, version = $3, status = $4, fields = $5, metadata = $6
		WHERE id = $7 AND tenant_id = $8
	`

	DeprecateCoreBO = `UPDATE core_bo SET status = 'deprecated' WHERE id = $1`

	InsertSemanticQueryCache = `
		INSERT INTO semantic_query_cache_v2 (
			tenant_id, query_hash, query, result, result_rows, execution_time_ms, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, query_hash) DO UPDATE
		SET result = EXCLUDED.result,
		    result_rows = EXCLUDED.result_rows,
		    execution_time_ms = EXCLUDED.execution_time_ms,
		    expires_at = EXCLUDED.expires_at,
		    last_accessed_at = now(),
		    access_count = semantic_query_cache_v2.access_count + 1
	`

	UpdateSemanticQueryCacheAccess = `
		UPDATE semantic_query_cache_v2
		SET last_accessed_at = now(), access_count = access_count + 1
		WHERE id = $1
	`

	InsertAuditLedger = `
		INSERT INTO audit_ledger (id, tenant_id, transaction_type, actor_id, payload, previous_hash, hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	InsertHumanTask = `
		INSERT INTO human_tasks (workflow_id, run_id, task_token, view_definition_id, title, input_context, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'PENDING')
		RETURNING id
	`

	DeleteTenant = `DELETE FROM tenants WHERE id = $1`

	ValidateTenantIDs = `SELECT COUNT(*) FROM tenants WHERE id = ANY($1)`

	UnsuspendTenant = `UPDATE tenants SET is_suspended = false, updated_at = now() WHERE id = $1`

	DeleteAPIKey = `DELETE FROM api_keys WHERE id = $1`

	RevokeAPIKey = `UPDATE api_keys SET is_revoked = true, revoked_at = now() WHERE id = $1`

	DeleteReportTemplate = `DELETE FROM report_templates WHERE id = $1`

	InsertAPIKeyUsage = `
		INSERT INTO api_key_usage (api_key_id, user_id, tenant_id, path, method, region, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	DeleteAPIKeyUsage = `DELETE FROM api_key_usage WHERE api_key_id = $1`

	InsertLookupValue = `INSERT INTO lookup_values (lookup_id, tenant_id, value, label) VALUES ($1, $2, $3, $4)`

	InsertLookupValueWithParent = `INSERT INTO lookup_values (lookup_id, tenant_id, value, label, parent_id) VALUES ($1, $2, $3, $4, $5)`

	DeleteIPWhitelistEntry = `DELETE FROM ip_whitelist_entries WHERE id = $1`

	InsertReportTemplate = `
		INSERT INTO report_templates (
			id, tenant_id, template_name, description, category,
			layout_config, parameter_schema, is_active, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	UpdateReportTemplate = `
		UPDATE report_templates
		SET template_name = $1, description = $2, category = $3,
		    layout_config = $4, parameter_schema = $5, is_active = $6,
		    updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
	`
)

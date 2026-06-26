package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// TuningService handles the logic for self-tuning generation rules.
type TuningService struct {
	DB *sqlx.DB
}

// NewTuningService creates a new TuningService.
func NewTuningService(db *sqlx.DB) *TuningService {
	return &TuningService{DB: db}
}

// RuleMetrics holds the calculated global approval/rejection rates for a rule.
type RuleMetrics struct {
	RuleID        string  `db:"rule_id"`
	Total         int     `db:"total"`
	Approved      int     `db:"approved"`
	ApprovalRate  float64 `db:"approval_rate"`
	RejectionRate float64 `db:"rejection_rate"`
}

// RulePerformanceResponse is the top-level struct for the rule performance cockpit.
type RulePerformanceResponse struct {
	RuleID        string               `json:"rule_id"`
	SemanticTrend []SemanticTrendPoint `json:"semantic_trend"`
	RefreshTrend  []RefreshTrendPoint  `json:"refresh_trend"`
}

// SemanticTrendPoint represents a single data point for approval/rejection trends.
type SemanticTrendPoint struct {
	Period   string `json:"period" db:"period"`
	Approved int    `json:"approved" db:"approved"`
	Rejected int    `json:"rejected" db:"rejected"`
}

// RefreshTrendPoint represents a single data point for pre-aggregation refresh performance.
type RefreshTrendPoint struct {
	Timestamp  time.Time `json:"timestamp" db:"decided_at"`
	DurationMs *int64    `json:"duration_ms,omitempty" db:"refresh_duration_ms"`
	Rows       *int64    `json:"rows,omitempty" db:"refresh_row_count"`
	SizeBytes  *int64    `json:"size_bytes,omitempty" db:"refresh_size_bytes"`
}

// ModelRuleMetrics holds model-specific rejection rates.
type ModelRuleMetrics struct {
	RuleID        string  `db:"rule_id"`
	ModelName     string  `db:"model_name"`
	Total         int     `db:"total"`
	RejectionRate float64 `db:"rejection_rate"`
}

// GetTuningStatus retrieves the current configuration and metrics for all rules.
func (s *TuningService) GetTuningStatus(ctx context.Context) ([]models.RuleTuningStatus, error) {
	// 1. Get all global rule configs
	var configs []models.RuleConfig
	if err := s.DB.SelectContext(ctx, &configs, "SELECT * FROM rule_config ORDER BY rule_id"); err != nil {
		return nil, fmt.Errorf("failed to get rule configs: %w", err)
	}

	// 2. Get all model-specific overrides
	var overrides []models.RuleModelConfig
	if err := s.DB.SelectContext(ctx, &overrides, "SELECT * FROM rule_model_config ORDER BY rule_id, model_name"); err != nil {
		return nil, fmt.Errorf("failed to get rule model overrides: %w", err)
	}
	overridesByRule := make(map[string][]models.RuleModelConfig)
	for _, o := range overrides {
		overridesByRule[o.RuleID] = append(overridesByRule[o.RuleID], o)
	}

	// 3. Get global metrics
	metricsQuery := `
		SELECT
			rule_id,
			COALESCE(count(*) FILTER (WHERE decision = 'approved')::float / NULLIF(count(*), 0), 0) * 100 AS approval_rate,
			COALESCE(count(*) FILTER (WHERE decision = 'rejected')::float / NULLIF(count(*), 0), 0) * 100 AS rejection_rate
		FROM model_upgrade_audit
		WHERE decided_at > now() - interval '90 days' AND rule_id IS NOT NULL AND rule_id != ''
		GROUP BY rule_id
	`
	type ruleMetrics struct {
		RuleID        string  `db:"rule_id"`
		ApprovalRate  float64 `db:"approval_rate"`
		RejectionRate float64 `db:"rejection_rate"`
	}
	var metrics []ruleMetrics
	if err := s.DB.SelectContext(ctx, &metrics, metricsQuery); err != nil {
		return nil, fmt.Errorf("failed to get rule metrics: %w", err)
	}
	metricsByRule := make(map[string]ruleMetrics)
	for _, m := range metrics {
		metricsByRule[m.RuleID] = m
	}

	// 4. Combine everything
	var statuses []models.RuleTuningStatus
	for _, config := range configs {
		status := models.RuleTuningStatus{
			RuleConfig: config,
			Overrides:  overridesByRule[config.RuleID],
		}
		if m, ok := metricsByRule[config.RuleID]; ok {
			status.ApprovalRate = m.ApprovalRate
			status.RejectionRate = m.RejectionRate
		}
		if status.Overrides == nil {
			status.Overrides = []models.RuleModelConfig{} // Ensure not null for JSON marshalling
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetRulePerformance retrieves semantic and operational metrics for a single rule.
func (s *TuningService) GetRulePerformance(ctx context.Context, ruleID string) (*RulePerformanceResponse, error) {
	// 1. Get semantic trend (approvals/rejections over time)
	semanticQuery := `
		SELECT
			to_char(decided_at, 'YYYY-MM') as period,
			count(*) FILTER (WHERE decision = 'approved') AS approved,
			count(*) FILTER (WHERE decision = 'rejected') AS rejected
		FROM model_upgrade_audit
		WHERE rule_id = $1
		  AND (event_type IS NULL OR event_type != 'preagg')
		GROUP BY period
		ORDER BY period;
	`
	var semanticTrend []SemanticTrendPoint
	if err := s.DB.SelectContext(ctx, &semanticTrend, semanticQuery, ruleID); err != nil {
		return nil, fmt.Errorf("failed to get semantic trend for rule %s: %w", ruleID, err)
	}

	// 2. Get operational refresh trend
	refreshQuery := `
		SELECT
			decided_at,
			refresh_duration_ms,
			refresh_row_count,
			refresh_size_bytes
		FROM model_upgrade_audit
		WHERE rule_id = $1
		  AND event_type = 'preagg'
		  AND decision = 'refreshed'
		ORDER BY decided_at;
	`
	var refreshTrend []RefreshTrendPoint
	if err := s.DB.SelectContext(ctx, &refreshTrend, refreshQuery, ruleID); err != nil {
		return nil, fmt.Errorf("failed to get refresh trend for rule %s: %w", ruleID, err)
	}

	// Ensure empty slices are returned instead of null for JSON compatibility
	if semanticTrend == nil {
		semanticTrend = []SemanticTrendPoint{}
	}
	if refreshTrend == nil {
		refreshTrend = []RefreshTrendPoint{}
	}

	return &RulePerformanceResponse{
		RuleID:        ruleID,
		SemanticTrend: semanticTrend,
		RefreshTrend:  refreshTrend,
	}, nil
}

// GetChangelog retrieves the history of rule configuration changes.
func (s *TuningService) GetChangelog(ctx context.Context, ruleID, scope string) ([]models.RuleConfigChangelog, error) {
	query := "SELECT * FROM rule_config_changelog"
	var conditions []string
	var args []interface{}

	if ruleID != "" {
		conditions = append(conditions, fmt.Sprintf("rule_id = $%d", len(args)+1))
		args = append(args, ruleID)
	}
	if scope != "" {
		conditions = append(conditions, fmt.Sprintf("scope LIKE $%d", len(args)+1))
		args = append(args, scope+"%") // Use LIKE for 'model:%'
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY triggered_at DESC"

	var changelog []models.RuleConfigChangelog
	if err := s.DB.SelectContext(ctx, &changelog, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get tuning changelog: %w", err)
	}
	return changelog, nil
}

// generateTuningProposals is the core read-only algorithm for generating proposals.
func (s *TuningService) generateTuningProposals(ctx context.Context, lookbackDays int, thresholds models.TuningThresholds, ruleIDs []string, scope string) ([]*models.TuningProposal, error) {
	// 1. Fetch all current configurations
	var allGlobalConfigs []models.RuleConfig
	if err := s.DB.SelectContext(ctx, &allGlobalConfigs, "SELECT * FROM rule_config"); err != nil {
		return nil, fmt.Errorf("failed to fetch global configs: %w", err)
	}
	globalConfigMap := make(map[string]models.RuleConfig)
	for _, cfg := range allGlobalConfigs {
		globalConfigMap[cfg.RuleID] = cfg
	}

	// 2. Fetch metrics with dynamic filtering
	query := `
        SELECT rule_id, count(*) AS total, count(*) FILTER (WHERE decision = 'approved') AS approved,
               (count(*) FILTER (WHERE decision = 'approved'))::float / NULLIF(count(*), 0) AS approval_rate,
               (count(*) FILTER (WHERE decision = 'rejected'))::float / NULLIF(count(*), 0) AS rejection_rate
        FROM model_upgrade_audit 
        WHERE decided_at > now() - ($1 * interval '1 day') AND rule_id IS NOT NULL AND rule_id != ''`
	args := []interface{}{lookbackDays}

	if len(ruleIDs) > 0 {
		query += fmt.Sprintf(" AND rule_id = ANY($%d)", len(args)+1)
		args = append(args, pq.Array(ruleIDs))
	}

	if scope != "" && scope != "global" {
		if strings.HasPrefix(scope, "model:") {
			modelName := strings.TrimPrefix(scope, "model:")
			query += fmt.Sprintf(" AND model_name = $%d", len(args)+1)
			args = append(args, modelName)
		}
	}

	query += " GROUP BY rule_id"

	var globalMetrics []RuleMetrics
	if err := s.DB.SelectContext(ctx, &globalMetrics, query, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch global rule metrics for simulation: %w", err)
	}

	var proposals []*models.TuningProposal
	for _, gm := range globalMetrics {
		oldCfg, ok := globalConfigMap[gm.RuleID]
		if !ok {
			continue
		}

		// Rule: High rejection rate -> decrease aggressiveness
		if gm.RejectionRate > thresholds.RejectRateDisable && gm.Total > 10 {
			newAgg := math.Max(0, oldCfg.Aggressiveness-0.2)
			if newAgg != oldCfg.Aggressiveness {
				proposals = append(proposals, &models.TuningProposal{
					RuleID:                 gm.RuleID,
					Scope:                  "global",
					CurrentAggressiveness:  oldCfg.Aggressiveness,
					ProposedAggressiveness: newAgg,
					CurrentAutoAccept:      oldCfg.AutoAcceptDefault,
					ProposedAutoAccept:     oldCfg.AutoAcceptDefault,
					Reason:                 fmt.Sprintf("Rejection rate %.2f > threshold %.2f", gm.RejectionRate, thresholds.RejectRateDisable),
					Metrics:                models.ProposalMetrics{ApprovalRate: gm.ApprovalRate, RejectionRate: gm.RejectionRate, TotalChanges: gm.Total},
				})
			}
		}

		// Rule: High approval rate -> enable auto-accept
		if gm.ApprovalRate > thresholds.ApproveRateAutoAccept && gm.Total > 20 {
			if !oldCfg.AutoAcceptDefault {
				proposals = append(proposals, &models.TuningProposal{
					RuleID:                 gm.RuleID,
					Scope:                  "global",
					CurrentAggressiveness:  oldCfg.Aggressiveness,
					ProposedAggressiveness: oldCfg.Aggressiveness,
					CurrentAutoAccept:      false,
					ProposedAutoAccept:     true,
					Reason:                 fmt.Sprintf("Approval rate %.2f > threshold %.2f", gm.ApprovalRate, thresholds.ApproveRateAutoAccept),
					Metrics:                models.ProposalMetrics{ApprovalRate: gm.ApprovalRate, RejectionRate: gm.RejectionRate, TotalChanges: gm.Total},
				})
			}
		}
	}

	// NOTE: Model-specific proposal logic would be added here, similar to the global logic.

	return proposals, nil
}

// SimulateTuning runs the tuning algorithm in dry-run mode to show proposed changes.
func (s *TuningService) SimulateTuning(ctx context.Context, req models.TuningSimulationRequest) (*models.TuningSimulationResponse, error) {
	// Set defaults for the simulation
	if req.LookbackDays <= 0 {
		req.LookbackDays = 90
	}
	thresholds := models.TuningThresholds{
		RejectRateDisable:     0.7,
		ApproveRateAutoAccept: 0.9,
	}
	if req.Thresholds != nil {
		if req.Thresholds.RejectRateDisable > 0 {
			thresholds.RejectRateDisable = req.Thresholds.RejectRateDisable
		}
		if req.Thresholds.ApproveRateAutoAccept > 0 {
			thresholds.ApproveRateAutoAccept = req.Thresholds.ApproveRateAutoAccept
		}
	}

	// The core logic is now in a separate function to be shared with RunTuning.
	// This is a read-only operation, so we don't need a transaction.
	proposals, err := s.generateTuningProposals(ctx, req.LookbackDays, thresholds, req.RuleIDs, req.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tuning proposals: %w", err)
	}

	// Enrich proposals with impact preview and optional side-by-side diffs
	for _, p := range proposals {
		impact, err := s.getImpactPreview(ctx, p.RuleID, p.Scope)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: could not get impact preview for rule %s: %v", p.RuleID, err)
		}
		p.ImpactPreview = impact

		if req.WithPreview {
			// In a real implementation, this would call the generation service twice:
			// once with the current config and once with the proposed config.
			// For this example, we'll return mock data to demonstrate the structure.
			p.SideBySide = s.getSideBySidePreview(p)
		}
	}

	response := &models.TuningSimulationResponse{
		SimulationID: fmt.Sprintf("sim-%s", time.Now().UTC().Format("20060102-150405")),
		Proposals:    proposals,
	}

	return response, nil
}

// getImpactPreview queries audit data to estimate the impact of a rule change.
func (s *TuningService) getImpactPreview(ctx context.Context, ruleID, scope string) (*models.ImpactPreview, error) {
	baseQuery := "FROM model_upgrade_audit WHERE rule_id = $1"
	args := []interface{}{ruleID}

	if scope != "" && scope != "global" {
		if strings.HasPrefix(scope, "model:") {
			modelName := strings.TrimPrefix(scope, "model:")
			baseQuery += " AND model_name = $2"
			args = append(args, modelName)
		}
		// Datasource scope is not supported as we don't have datasource_id in the audit table.
	}

	countQuery := "SELECT COUNT(DISTINCT model_name), COUNT(DISTINCT field_path) " + baseQuery
	var modelCount, fieldCount int
	err := s.DB.QueryRowContext(ctx, countQuery, args...).Scan(&modelCount, &fieldCount)
	if err != nil {
		return nil, err
	}

	namesQuery := "SELECT DISTINCT model_name " + baseQuery + " LIMIT 5"
	var modelNames []string
	err = s.DB.SelectContext(ctx, &modelNames, namesQuery, args...)
	if err != nil {
		return nil, err
	}

	return &models.ImpactPreview{
		ModelsAffected: modelNames,
		FieldsAffected: fieldCount,
	}, nil
}

// getSideBySidePreview is a placeholder for a complex generation comparison.
func (s *TuningService) getSideBySidePreview(p *models.TuningProposal) []models.SideBySidePreview {
	// This is a mock of the generation process, but it demonstrates the diffing and annotation.
	// A real implementation would call the full model generation logic twice with different configs.
	modelName := "orders"
	if strings.Contains(p.Scope, "model:") {
		modelName = strings.Split(p.Scope, ":")[1]
	}

	// --- Create a "before" state ---
	beforeConfig := models.ResolvedModelConfig{
		ModelKey: modelName,
		Cubes: []cube.Cube{
			{
				Name:       modelName,
				Dimensions: map[string]map[string]any{"order_date": {"sql": "order_date", "type": "time"}},
				Measures:   map[string]map[string]any{"total_revenue": {"sql": "SUM(revenue)", "type": "sum"}},
			},
		},
	}

	// --- Create an "after" state that reflects the proposed change ---
	afterConfig := beforeConfig // Start with the same base
	// Deep copy to avoid modifying the original
	afterConfig.Cubes = make([]cube.Cube, len(beforeConfig.Cubes))
	copy(afterConfig.Cubes, beforeConfig.Cubes)
	afterConfig.Cubes[0].Measures = make(map[string]map[string]any)
	for k, v := range beforeConfig.Cubes[0].Measures {
		afterConfig.Cubes[0].Measures[k] = v
	}
	afterConfig.Cubes[0].Dimensions = make(map[string]map[string]any)
	for k, v := range beforeConfig.Cubes[0].Dimensions {
		afterConfig.Cubes[0].Dimensions[k] = v
	}

	// Simulate the rule firing to add a pre-aggregation
	switch p.RuleID {
	case "auto_daily_rollup":
		if afterConfig.Cubes[0].PreAggregations == nil {
			afterConfig.Cubes[0].PreAggregations = make(map[string]map[string]any)
		}
		afterConfig.Cubes[0].PreAggregations["orders_daily_rollup"] = map[string]any{
			"type":          "rollup",
			"timeDimension": "order_date",
			"granularity":   "day",
			"meta":          models.ExplainMeta(p.RuleID, modelName, "order_date"),
		}
	case "auto_materialized_view":
		if afterConfig.Cubes[0].PreAggregations == nil {
			afterConfig.Cubes[0].PreAggregations = make(map[string]map[string]any)
		}
		afterConfig.Cubes[0].PreAggregations["main"] = map[string]any{
			"type": "original_sql",
			"meta": models.ExplainMeta(p.RuleID, modelName, "cube_sql"),
		}
	case "auto_rollup_join":
		if afterConfig.Cubes[0].PreAggregations == nil {
			afterConfig.Cubes[0].PreAggregations = make(map[string]map[string]any)
		}
		afterConfig.Cubes[0].PreAggregations["orders_with_users"] = map[string]any{
			"type":    "rollup_join",
			"rollups": []string{"users.users_rollup", "CUBE.orders_rollup"},
			"meta":    models.ExplainMeta(p.RuleID, modelName, "join:users"),
		}
	}

	// --- Generate YAML and Annotations ---
	beforeBytes, _ := json.MarshalIndent(beforeConfig, "", "  ")
	afterBytes, _ := json.MarshalIndent(afterConfig, "", "  ")

	annotations := GenerateAnnotatedDiff(beforeConfig, afterConfig)

	return []models.SideBySidePreview{
		{
			Model:       modelName,
			BeforeYAML:  string(beforeBytes),
			AfterYAML:   string(afterBytes),
			Annotations: annotations,
		},
	}
}

// RunTuning executes the self-tuning algorithm.
func (s *TuningService) RunTuning(ctx context.Context) (string, error) {
	logging.GetLogger().Sugar().Info("Starting self-tuning process...")

	// 1. Get global metrics for all rules
	globalMetricsQuery := `
        SELECT
            rule_id,
            count(*) AS total,
            count(*) FILTER (WHERE decision = 'approved') AS approved,
            (count(*) FILTER (WHERE decision = 'approved'))::float / NULLIF(count(*), 0) AS approval_rate,
            (count(*) FILTER (WHERE decision = 'rejected'))::float / NULLIF(count(*), 0) AS rejection_rate
        FROM model_upgrade_audit
        WHERE decided_at > now() - interval '90 days' AND rule_id IS NOT NULL AND rule_id != '' AND model_name IS NOT NULL AND model_name != ''
        GROUP BY rule_id
    `
	var globalMetrics []RuleMetrics
	if err := s.DB.SelectContext(ctx, &globalMetrics, globalMetricsQuery); err != nil {
		return "", fmt.Errorf("failed to fetch global rule metrics: %w", err)
	}

	// 2. Get model-specific metrics
	modelMetricsQuery := `
        SELECT
            rule_id,
            model_name,
            count(*) as total,
            (count(*) FILTER (WHERE decision = 'rejected'))::float / NULLIF(count(*), 0) AS rejection_rate
        FROM model_upgrade_audit
        WHERE decided_at > now() - interval '90 days' AND rule_id IS NOT NULL AND rule_id != '' AND model_name IS NOT NULL AND model_name != ''
        GROUP BY rule_id, model_name
    `
	var modelMetrics []ModelRuleMetrics
	if err := s.DB.SelectContext(ctx, &modelMetrics, modelMetricsQuery); err != nil {
		return "", fmt.Errorf("failed to fetch model-specific metrics: %w", err)
	}

	if len(globalMetrics) == 0 && len(modelMetrics) == 0 {
		logging.GetLogger().Sugar().Info("No audit data found in the last 90 days. Skipping tuning.")
		return "No audit data to process.", nil
	}

	tx, err := s.DB.Beginx()
	if err != nil {
		return "", fmt.Errorf("failed to begin tuning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var changes []string
	triggeredBy := "auto_tuning"

	// Process global metrics
	for _, gm := range globalMetrics {
		var oldCfg models.RuleConfig
		if err := tx.GetContext(ctx, &oldCfg, "SELECT * FROM rule_config WHERE rule_id = $1 FOR UPDATE", gm.RuleID); err != nil {
			logging.GetLogger().Sugar().Warnf("Tuning: could not get config for rule %s, skipping: %v", gm.RuleID, err)
			continue
		}

		if gm.RejectionRate > 0.7 && gm.Total > 10 {
			newAgg := math.Max(0, oldCfg.Aggressiveness-0.2)
			if newAgg != oldCfg.Aggressiveness {
				_, err := tx.ExecContext(ctx, "UPDATE rule_config SET aggressiveness = $1, last_tuned = now() WHERE rule_id = $2", newAgg, gm.RuleID)
				if err != nil {
					return "", fmt.Errorf("failed to update aggressiveness for %s: %w", gm.RuleID, err)
				}
				reason := fmt.Sprintf("Global rejection rate of %.0f%% exceeded threshold of 70%%", gm.RejectionRate*100)
				if err := s.logChange(ctx, tx, gm.RuleID, &oldCfg.Aggressiveness, &newAgg, nil, nil, "global", reason, triggeredBy); err != nil {
					return "", err
				}
				changes = append(changes, fmt.Sprintf("Decreased global aggressiveness for '%s' to %.2f", gm.RuleID, newAgg))
			}
		} else if gm.ApprovalRate > 0.95 && gm.Total > 20 {
			if !oldCfg.AutoAcceptDefault {
				_, err := tx.ExecContext(ctx, "UPDATE rule_config SET auto_accept_default = true, last_tuned = now() WHERE rule_id = $1", gm.RuleID)
				if err != nil {
					return "", fmt.Errorf("failed to enable auto-accept for %s: %w", gm.RuleID, err)
				}
				newAutoAccept := true
				reason := fmt.Sprintf("Global approval rate of %.0f%% exceeded threshold of 95%%", gm.ApprovalRate*100)
				if err := s.logChange(ctx, tx, gm.RuleID, nil, nil, &oldCfg.AutoAcceptDefault, &newAutoAccept, "global", reason, triggeredBy); err != nil {
					return "", err
				}
				changes = append(changes, fmt.Sprintf("Enabled auto-accept for '%s'", gm.RuleID))
			}
		}
	}

	// Process model-specific overrides independently as a "safety valve"
	for _, mm := range modelMetrics {
		if mm.RejectionRate > 0.8 && mm.Total > 5 {
			var oldCfg models.RuleModelConfig
			err := tx.GetContext(ctx, &oldCfg, "SELECT * FROM rule_model_config WHERE rule_id = $1 AND model_name = $2 FOR UPDATE", mm.RuleID, mm.ModelName)
			if err != nil && err != sql.ErrNoRows {
				return "", fmt.Errorf("failed to get model config for %s/%s: %w", mm.RuleID, mm.ModelName, err)
			}

			if err == sql.ErrNoRows || oldCfg.Enabled {
				query := `INSERT INTO rule_model_config (rule_id, model_name, enabled, last_tuned) VALUES ($1, $2, false, now()) ON CONFLICT (rule_id, model_name) DO UPDATE SET enabled = false, last_tuned = now()`
				if _, err := tx.ExecContext(ctx, query, mm.RuleID, mm.ModelName); err != nil {
					return "", fmt.Errorf("failed to disable rule %s for model %s: %w", mm.RuleID, mm.ModelName, err)
				}

				oldEnabled := err == sql.ErrNoRows || oldCfg.Enabled
				newEnabled := false
				reason := fmt.Sprintf("Model-specific rejection rate of %.0f%% exceeded threshold of 80%%", mm.RejectionRate*100)
				scope := fmt.Sprintf("model:%s", mm.ModelName)
				if err := s.logChange(ctx, tx, mm.RuleID, nil, nil, &oldEnabled, &newEnabled, scope, reason, triggeredBy); err != nil {
					return "", err
				}
				changes = append(changes, fmt.Sprintf("Disabled rule '%s' for model '%s'", mm.RuleID, mm.ModelName))
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit tuning changes: %w", err)
	}

	summary := fmt.Sprintf("Tuning complete. Made %d adjustments.", len(changes))
	logging.GetLogger().Sugar().Info(summary)
	for _, change := range changes {
		logging.GetLogger().Sugar().Infof(" - %s", change)
	}
	return summary, nil
}

// logChange inserts a record into the rule_config_changelog.
func (s *TuningService) logChange(ctx context.Context, tx *sqlx.Tx, ruleID string, oldAgg, newAgg *float64, oldAuto, newAuto *bool, scope, reason, triggeredBy string) error {
	changelog := models.RuleConfigChangelog{
		ID:                uuid.New(),
		RuleID:            ruleID,
		OldAggressiveness: oldAgg,
		NewAggressiveness: newAgg,
		OldAutoAccept:     oldAuto,
		NewAutoAccept:     newAuto,
		Scope:             scope,
		Reason:            reason,
		TriggeredBy:       triggeredBy,
		TriggeredAt:       time.Now().UTC(),
	}
	query := `
		INSERT INTO rule_config_changelog (id, rule_id, old_aggressiveness, new_aggressiveness, old_auto_accept, new_auto_accept, scope, reason, triggered_by, triggered_at)
		VALUES (:id, :rule_id, :old_aggressiveness, :new_aggressiveness, :old_auto_accept, :new_auto_accept, :scope, :reason, :triggered_by, :triggered_at)
	`
	_, err := tx.NamedExecContext(ctx, query, &changelog)
	return err
}

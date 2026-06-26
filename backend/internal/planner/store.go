package planner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Store handles persistence of planner decisions
type Store struct {
	db *sql.DB
}

// NewStore creates a new planner store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveDecision persists a planner decision to the database
func (s *Store) SaveDecision(ctx context.Context, req *QueryRequest, plan *QueryPlan, regionHealthSnapshot interface{}) error {
	rawReq, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	rawPlan, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}

	var healthSnapshotJSON json.RawMessage
	if regionHealthSnapshot != nil {
		jsb, err := json.Marshal(regionHealthSnapshot)
		if err != nil {
			return fmt.Errorf("marshal region health: %w", err)
		}
		healthSnapshotJSON = jsb
	}

	query := `
		INSERT INTO planner_decisions (
			plan_id, tenant_id, query_type, semantic_target, selected_regions,
			plan_type, estimated_cost, estimated_latency_ms, degradation_strategy,
			explain, raw_request, raw_plan, region_health_snapshot
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	degradationJSON, err := json.Marshal(plan.DegradationStrategy)
	if err != nil {
		return fmt.Errorf("marshal degradation: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		plan.PlanID,
		req.TenantID,
		req.QueryType,
		req.SemanticTarget,
		pq.Array(plan.SelectedRegions),
		plan.PlanType,
		plan.EstimatedCost,
		plan.EstimatedLatencyMS,
		degradationJSON,
		plan.Explain,
		rawReq,
		rawPlan,
		healthSnapshotJSON,
	)

	return err
}

// UpdateDecisionExecution updates a decision with execution results
func (s *Store) UpdateDecisionExecution(ctx context.Context, planID string, actualLatencyMS float64, actualCost float64, status string, errMsg string) error {
	query := `
		UPDATE planner_decisions
		SET executed_at = $1, actual_latency_ms = $2, actual_cost = $3, execution_status = $4, execution_error = $5
		WHERE plan_id = $6
	`

	_, err := s.db.ExecContext(ctx, query,
		time.Now(),
		actualLatencyMS,
		actualCost,
		status,
		errMsg,
		planID,
	)

	return err
}

// GetDecision retrieves a planner decision by ID
func (s *Store) GetDecision(ctx context.Context, planID string) (*PlannerDecision, error) {
	query := `
		SELECT
			plan_id, created_at, tenant_id, query_type, semantic_target,
			selected_regions, plan_type, estimated_cost, estimated_latency_ms,
			degradation_strategy, explain, raw_request, raw_plan,
			executed_at, actual_latency_ms, actual_cost, execution_status, execution_error,
			region_health_snapshot
		FROM planner_decisions
		WHERE plan_id = $1
	`

	var decision PlannerDecision
	err := s.db.QueryRowContext(ctx, query, planID).Scan(
		&decision.PlanID,
		&decision.CreatedAt,
		&decision.TenantID,
		&decision.QueryType,
		&decision.SemanticTarget,
		pq.Array(&decision.SelectedRegions),
		&decision.PlanType,
		&decision.EstimatedCost,
		&decision.EstimatedLatencyMS,
		&decision.DegradationStrategy,
		&decision.Explain,
		&decision.RawRequest,
		&decision.RawPlan,
		&decision.ExecutedAt,
		&decision.ActualLatencyMS,
		&decision.ActualCost,
		&decision.ExecutionStatus,
		&decision.ExecutionError,
		&decision.RegionHealthSnapshot,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &decision, err
}

// GetDecisionsForTarget retrieves all planner decisions for a semantic target
func (s *Store) GetDecisionsForTarget(ctx context.Context, semanticTarget string, limit int) ([]PlannerDecision, error) {
	query := `
		SELECT
			plan_id, created_at, tenant_id, query_type, semantic_target,
			selected_regions, plan_type, estimated_cost, estimated_latency_ms,
			degradation_strategy, explain, raw_request, raw_plan,
			executed_at, actual_latency_ms, actual_cost, execution_status, execution_error,
			region_health_snapshot
		FROM planner_decisions
		WHERE semantic_target = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, semanticTarget, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []PlannerDecision
	for rows.Next() {
		var d PlannerDecision
		err := rows.Scan(
			&d.PlanID,
			&d.CreatedAt,
			&d.TenantID,
			&d.QueryType,
			&d.SemanticTarget,
			pq.Array(&d.SelectedRegions),
			&d.PlanType,
			&d.EstimatedCost,
			&d.EstimatedLatencyMS,
			&d.DegradationStrategy,
			&d.Explain,
			&d.RawRequest,
			&d.RawPlan,
			&d.ExecutedAt,
			&d.ActualLatencyMS,
			&d.ActualCost,
			&d.ExecutionStatus,
			&d.ExecutionError,
			&d.RegionHealthSnapshot,
		)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}

	return decisions, rows.Err()
}

// GetSLOCompliance returns planner SLO metrics for a query type
type SLOCompliance struct {
	MetricName         string  `db:"metric_name"`
	QueryCount         int     `db:"query_count"`
	LatencyErrorAvgPct float64 `db:"latency_error_avg_pct"`
	SuccessRate        float64 `db:"success_rate"`
}

func (s *Store) GetSLOCompliance(ctx context.Context, queryType string, hoursBack int) (*SLOCompliance, error) {
	query := `
		SELECT
			'latency_estimation' AS metric_name,
			COUNT(*)::INTEGER,
			AVG(latency_error_pct),
			COUNT(CASE WHEN execution_status = 'success' THEN 1 END)::DOUBLE PRECISION * 100.0 / COUNT(*)
		FROM planner_metrics
		WHERE query_type = $1 AND ts > now() - ($2 || ' hours')::INTERVAL
	`

	var slo SLOCompliance
	err := s.db.QueryRowContext(ctx, query, queryType, hoursBack).Scan(
		&slo.MetricName,
		&slo.QueryCount,
		&slo.LatencyErrorAvgPct,
		&slo.SuccessRate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &slo, err
}

// GetRegionPerformance retrieves current region performance metrics
func (s *Store) GetRegionPerformance(ctx context.Context, region string) (*RegionPerformance, error) {
	query := `
		SELECT
			region, last_updated, is_healthy, latency_ms_p50, latency_ms_p95,
			latency_ms_p99, error_rate, active_features, materialization_freshness_pct,
			cache_hit_rate
		FROM planner_region_performance
		WHERE region = $1
	`

	var perf RegionPerformance
	err := s.db.QueryRowContext(ctx, query, region).Scan(
		&perf.Region,
		&perf.LastUpdated,
		&perf.IsHealthy,
		&perf.LatencyP50MS,
		&perf.LatencyP95MS,
		&perf.LatencyP99MS,
		&perf.ErrorRate,
		&perf.ActiveFeatures,
		&perf.MaterializationFreshnessPercent,
		&perf.CacheHitRate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &perf, err
}

// ListAllRegionPerformance retrieves performance for all regions
func (s *Store) ListAllRegionPerformance(ctx context.Context) (map[string]*RegionPerformance, error) {
	query := `
		SELECT
			region, last_updated, is_healthy, latency_ms_p50, latency_ms_p95,
			latency_ms_p99, error_rate, active_features, materialization_freshness_pct,
			cache_hit_rate
		FROM planner_region_performance
		ORDER BY region
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := make(map[string]*RegionPerformance)
	for rows.Next() {
		var perf RegionPerformance
		err := rows.Scan(
			&perf.Region,
			&perf.LastUpdated,
			&perf.IsHealthy,
			&perf.LatencyP50MS,
			&perf.LatencyP95MS,
			&perf.LatencyP99MS,
			&perf.ErrorRate,
			&perf.ActiveFeatures,
			&perf.MaterializationFreshnessPercent,
			&perf.CacheHitRate,
		)
		if err != nil {
			return nil, err
		}
		regions[perf.Region] = &perf
	}

	return regions, rows.Err()
}

// GetFeaturePlannerConfig retrieves planner config for a feature
func (s *Store) GetFeaturePlannerConfig(ctx context.Context, featureID string) (*FeaturePlannerConfig, error) {
	query := `
		SELECT
			feature_id, preferred_regions, disallowed_regions, default_consistency,
			default_freshness, interactive_latency_budget_ms, batch_latency_budget_ms,
			use_cache_if_stale, max_cache_staleness, created_at, updated_at
		FROM planner_feature_config
		WHERE feature_id = $1
	`

	var config FeaturePlannerConfig
	err := s.db.QueryRowContext(ctx, query, featureID).Scan(
		&config.FeatureID,
		pq.Array(&config.PreferredRegions),
		pq.Array(&config.DisallowedRegions),
		&config.DefaultConsistency,
		&config.DefaultFreshness,
		&config.InteractiveLatencyBudgetMS,
		&config.BatchLatencyBudgetMS,
		&config.UseCacheIfStale,
		&config.MaxCacheStaleness,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &config, err
}

// SaveFeaturePlannerConfig saves or updates planner config for a feature
func (s *Store) SaveFeaturePlannerConfig(ctx context.Context, config *FeaturePlannerConfig) error {
	query := `
		INSERT INTO planner_feature_config (
			feature_id, preferred_regions, disallowed_regions, default_consistency,
			default_freshness, interactive_latency_budget_ms, batch_latency_budget_ms,
			use_cache_if_stale, max_cache_staleness
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (feature_id) DO UPDATE SET
			preferred_regions = EXCLUDED.preferred_regions,
			disallowed_regions = EXCLUDED.disallowed_regions,
			default_consistency = EXCLUDED.default_consistency,
			default_freshness = EXCLUDED.default_freshness,
			interactive_latency_budget_ms = EXCLUDED.interactive_latency_budget_ms,
			batch_latency_budget_ms = EXCLUDED.batch_latency_budget_ms,
			use_cache_if_stale = EXCLUDED.use_cache_if_stale,
			max_cache_staleness = EXCLUDED.max_cache_staleness,
			updated_at = now()
	`

	_, err := s.db.ExecContext(ctx, query,
		config.FeatureID,
		pq.Array(config.PreferredRegions),
		pq.Array(config.DisallowedRegions),
		config.DefaultConsistency,
		config.DefaultFreshness,
		config.InteractiveLatencyBudgetMS,
		config.BatchLatencyBudgetMS,
		config.UseCacheIfStale,
		config.MaxCacheStaleness,
	)

	return err
}

// GetRecentDecisions retrieves recent planner decisions (for dashboards)
func (s *Store) GetRecentDecisions(ctx context.Context, limit int, offset int) ([]PlannerDecision, error) {
	query := `
		SELECT
			plan_id, created_at, tenant_id, query_type, semantic_target,
			selected_regions, plan_type, estimated_cost, estimated_latency_ms,
			degradation_strategy, explain, raw_request, raw_plan,
			executed_at, actual_latency_ms, actual_cost, execution_status, execution_error,
			region_health_snapshot
		FROM planner_decisions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []PlannerDecision
	for rows.Next() {
		var d PlannerDecision
		err := rows.Scan(
			&d.PlanID,
			&d.CreatedAt,
			&d.TenantID,
			&d.QueryType,
			&d.SemanticTarget,
			pq.Array(&d.SelectedRegions),
			&d.PlanType,
			&d.EstimatedCost,
			&d.EstimatedLatencyMS,
			&d.DegradationStrategy,
			&d.Explain,
			&d.RawRequest,
			&d.RawPlan,
			&d.ExecutedAt,
			&d.ActualLatencyMS,
			&d.ActualCost,
			&d.ExecutionStatus,
			&d.ExecutionError,
			&d.RegionHealthSnapshot,
		)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}

	return decisions, rows.Err()
}

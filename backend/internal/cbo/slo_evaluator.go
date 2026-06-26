package cbo

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SLOEvaluator evaluates SLOs against telemetry and handles violations
type SLOEvaluator struct {
	db             *sqlx.DB
	sloProvider    *DBSLOProvider
	tuningProvider *DBASOTuningProvider
}

// NewSLOEvaluator creates a new SLO evaluator
func NewSLOEvaluator(db *sqlx.DB, sloProvider *DBSLOProvider, tuningProvider *DBASOTuningProvider) *SLOEvaluator {
	return &SLOEvaluator{
		db:             db,
		sloProvider:    sloProvider,
		tuningProvider: tuningProvider,
	}
}

// EvaluateAll evaluates all active SLOs for an environment
func (e *SLOEvaluator) EvaluateAll(ctx context.Context, env string) ([]SLOEvaluation, error) {
	// Get all active SLOs
	slos, err := e.sloProvider.ListSLOs(ctx, env, nil, "", "")
	if err != nil {
		return nil, err
	}

	var evaluations []SLOEvaluation
	for _, slo := range slos {
		if !slo.Enabled {
			continue
		}

		eval, err := e.Evaluate(ctx, &slo)
		if err != nil {
			log.Printf("[slo_evaluator] Error evaluating SLO %s: %v", slo.ID, err)
			continue
		}

		evaluations = append(evaluations, *eval)

		// Handle violations
		if eval.Status == "violated" {
			if err := e.handleViolation(ctx, &slo, eval); err != nil {
				log.Printf("[slo_evaluator] Error handling violation for SLO %s: %v", slo.ID, err)
			}
		}
	}

	return evaluations, nil
}

// Evaluate evaluates a single SLO
func (e *SLOEvaluator) Evaluate(ctx context.Context, slo *SLODefinition) (*SLOEvaluation, error) {
	windowEnd := time.Now()
	windowStart := e.calculateWindowStart(windowEnd, slo.TimeWindow)

	// Measure current value based on SLO type
	measuredValue, err := e.measureValue(ctx, slo, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}

	// Determine status
	status := e.determineStatus(slo.SLOType, slo.Target, measuredValue)

	// Calculate delta
	var deltaPercent *float64
	if slo.Target > 0 {
		d := ((measuredValue - slo.Target) / slo.Target) * 100
		deltaPercent = &d
	}

	// Create evaluation record
	eval := &SLOEvaluation{
		ID:            uuid.New(),
		SLOID:         slo.ID,
		Env:           slo.Env,
		TenantID:      slo.TenantID,
		ScopeType:     slo.ScopeType,
		ScopeID:       slo.ScopeID,
		WindowStart:   windowStart.Format(time.RFC3339),
		WindowEnd:     windowEnd.Format(time.RFC3339),
		MeasuredValue: measuredValue,
		TargetValue:   slo.Target,
		Status:        status,
		DeltaPercent:  deltaPercent,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	// Store evaluation
	if err := e.storeEvaluation(ctx, eval); err != nil {
		log.Printf("[slo_evaluator] Error storing evaluation: %v", err)
	}

	return eval, nil
}

// measureValue measures the current value for an SLO
func (e *SLOEvaluator) measureValue(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	switch slo.SLOType {
	case "latency":
		return e.measureLatency(ctx, slo, windowStart, windowEnd)
	case "freshness":
		return e.measureFreshness(ctx, slo)
	case "error_rate":
		if slo.ScopeType == "page" {
			return e.measurePageErrorRate(ctx, slo, windowStart, windowEnd)
		}
		if slo.ScopeType == "api" {
			return e.measureApiErrorRate(ctx, slo, windowStart, windowEnd)
		}
		return e.measureErrorRate(ctx, slo, windowStart, windowEnd)
	case "preagg_hit_rate":
		return e.measurePreAggHitRate(ctx, slo, windowStart, windowEnd)
	case "entitlement_latency":
		return e.measureEntitlementLatency(ctx, slo, windowStart, windowEnd)
	default:
		return 0, nil
	}
}

// measureLatency measures p95 latency for a BO
func (e *SLOEvaluator) measureLatency(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	if slo.ScopeType == "page" {
		return e.measurePageLatency(ctx, slo, windowStart, windowEnd)
	}
	if slo.ScopeType == "api" {
		return e.measureApiLatency(ctx, slo, windowStart, windowEnd)
	}
	query := `
		SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY actual_latency_ms), 0)
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND created_at >= $4
		  AND created_at <= $5
	`

	var p95 float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&p95)
	if err != nil {
		return 0, err
	}

	return p95, nil
}

func (e *SLOEvaluator) measurePageLatency(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY actual_latency_ms), 0)
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND page_slug = $3
		  AND created_at >= $4
		  AND created_at <= $5
	`

	var p95 float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&p95)
	if err != nil {
		return 0, err
	}

	return p95, nil
}

func (e *SLOEvaluator) measurePageErrorRate(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(
			SUM(CASE WHEN success = false THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0),
			0
		)
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND page_slug = $3
		  AND created_at >= $4
		  AND created_at <= $5
	`

	var errorRate float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&errorRate)
	if err != nil {
		return 0, err
	}

	return errorRate, nil
}

// measureApiLatency measures p95 latency for an API endpoint from api_telemetry
func (e *SLOEvaluator) measureApiLatency(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0)
		FROM api_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND api_id = $3
		  AND requested_at >= $4
		  AND requested_at <= $5
	`
	var p95 float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&p95)
	return p95, err
}

// measureApiErrorRate measures error rate for an API endpoint from api_telemetry
func (e *SLOEvaluator) measureApiErrorRate(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(
			CAST(COUNT(CASE WHEN status_code >= 400 THEN 1 END) AS FLOAT) / 
			NULLIF(COUNT(*), 0),
			0
		)
		FROM api_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND api_id = $3
		  AND requested_at >= $4
		  AND requested_at <= $5
	`
	var rate float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&rate)
	return rate, err
}

// measureFreshness measures max freshness lag for a pre-agg
func (e *SLOEvaluator) measureFreshness(ctx context.Context, slo *SLODefinition) (float64, error) {
	// For pre-agg scopes, check the actual pre-agg table
	query := `
		SELECT COALESCE(EXTRACT(EPOCH FROM (NOW() - MAX(updated_at))), 0)
		FROM catalog_node
		WHERE node_name = $1
		  AND tenant_id = $2
	`

	var lagSec float64
	err := e.db.QueryRowxContext(ctx, query, slo.ScopeID, slo.TenantID).Scan(&lagSec)
	if err != nil {
		return 0, err
	}

	return lagSec, nil
}

// measureErrorRate measures error rate for a BO
func (e *SLOEvaluator) measureErrorRate(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(
			SUM(CASE WHEN success = false THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0),
			0
		)
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND created_at >= $4
		  AND created_at <= $5
	`

	var errorRate float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&errorRate)
	if err != nil {
		return 0, err
	}

	return errorRate, nil
}

// measurePreAggHitRate measures pre-agg hit rate for a BO
func (e *SLOEvaluator) measurePreAggHitRate(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	query := `
		SELECT COALESCE(
			SUM(CASE WHEN plan_type = 'preagg' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0),
			0
		)
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND created_at >= $4
		  AND created_at <= $5
	`

	var hitRate float64
	err := e.db.QueryRowxContext(ctx, query, slo.Env, slo.TenantID, slo.ScopeID, windowStart, windowEnd).Scan(&hitRate)
	if err != nil {
		return 0, err
	}

	return hitRate, nil
}

// measureEntitlementLatency measures entitlement processing latency
func (e *SLOEvaluator) measureEntitlementLatency(ctx context.Context, slo *SLODefinition, windowStart, windowEnd time.Time) (float64, error) {
	// For entitlement latency, we'd need separate telemetry
	// For now, return 0 as placeholder
	return 0, nil
}

// determineStatus determines if an SLO is met or violated
func (e *SLOEvaluator) determineStatus(sloType string, target, measured float64) string {
	switch sloType {
	case "latency", "freshness", "entitlement_latency":
		// These are "lower is better" - violated if measured > target
		if measured > target {
			return "violated"
		}
	case "error_rate":
		// Error rate is "lower is better" - violated if measured > target
		if measured > target {
			return "violated"
		}
	case "preagg_hit_rate":
		// Hit rate is "higher is better" - violated if measured < target
		if measured < target {
			return "violated"
		}
	default:
		return "unknown"
	}
	return "met"
}

// calculateWindowStart calculates the window start time
func (e *SLOEvaluator) calculateWindowStart(windowEnd time.Time, window string) time.Time {
	switch window {
	case "1d":
		return windowEnd.Add(-24 * time.Hour)
	case "7d":
		return windowEnd.Add(-7 * 24 * time.Hour)
	case "30d":
		return windowEnd.Add(-30 * 24 * time.Hour)
	case "90d":
		return windowEnd.Add(-90 * 24 * time.Hour)
	default:
		return windowEnd.Add(-7 * 24 * time.Hour)
	}
}

// storeEvaluation stores an evaluation result
func (e *SLOEvaluator) storeEvaluation(ctx context.Context, eval *SLOEvaluation) error {
	query := `
		INSERT INTO semantic_slo_evaluations (
			id, slo_id, env, tenant_id, scope_type, scope_id,
			window_start, window_end, measured_value, target_value, status, delta_percent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := e.db.ExecContext(ctx, query,
		eval.ID, eval.SLOID, eval.Env, eval.TenantID, eval.ScopeType, eval.ScopeID,
		eval.WindowStart, eval.WindowEnd, eval.MeasuredValue, eval.TargetValue, eval.Status, eval.DeltaPercent,
	)
	return err
}

// handleViolation handles an SLO violation
func (e *SLOEvaluator) handleViolation(ctx context.Context, slo *SLODefinition, eval *SLOEvaluation) error {
	log.Printf("[slo_evaluator] SLO violation detected: scope=%s/%s, type=%s, target=%.2f, actual=%.2f",
		slo.ScopeType, slo.ScopeID, slo.SLOType, slo.Target, eval.MeasuredValue)

	// Create violation record
	violation := &SLOViolation{
		ID:           uuid.New(),
		SLOID:        slo.ID,
		EvaluationID: &eval.ID,
		Env:          slo.Env,
		TenantID:     slo.TenantID,
		ScopeType:    slo.ScopeType,
		ScopeID:      slo.ScopeID,
		SLOType:      slo.SLOType,
		TargetValue:  slo.Target,
		ActualValue:  eval.MeasuredValue,
		Severity:     e.calculateSeverity(slo.SLOType, slo.Target, eval.MeasuredValue),
		CreatedAt:    time.Now().Format(time.RFC3339),
	}

	// Store violation
	if err := e.storeViolation(ctx, violation); err != nil {
		return err
	}

	// Trigger ASO tuning adjustment
	if e.tuningProvider != nil {
		if err := e.tuningProvider.HandleSLOViolation(ctx, violation); err != nil {
			log.Printf("[slo_evaluator] Error adjusting ASO tuning: %v", err)
		}
	}

	return nil
}

// calculateSeverity calculates violation severity
func (e *SLOEvaluator) calculateSeverity(sloType string, target, actual float64) string {
	if target == 0 {
		return "warning"
	}

	ratio := actual / target

	// For "lower is better" metrics
	if sloType == "latency" || sloType == "freshness" || sloType == "error_rate" || sloType == "entitlement_latency" {
		if ratio > 2.0 {
			return "critical"
		} else if ratio > 1.5 {
			return "warning"
		}
	}

	// For "higher is better" metrics
	if sloType == "preagg_hit_rate" {
		if ratio < 0.5 {
			return "critical"
		} else if ratio < 0.75 {
			return "warning"
		}
	}

	return "info"
}

// storeViolation stores a violation record
func (e *SLOEvaluator) storeViolation(ctx context.Context, v *SLOViolation) error {
	query := `
		INSERT INTO semantic_slo_violations (
			id, slo_id, evaluation_id, env, tenant_id, scope_type, scope_id,
			slo_type, target_value, actual_value, severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := e.db.ExecContext(ctx, query,
		v.ID, v.SLOID, v.EvaluationID, v.Env, v.TenantID, v.ScopeType, v.ScopeID,
		v.SLOType, v.TargetValue, v.ActualValue, v.Severity,
	)
	return err
}

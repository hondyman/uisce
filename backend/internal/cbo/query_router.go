package cbo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// QueryRouter routes semantic queries to optimal execution paths
type QueryRouter struct {
	db            *sqlx.DB
	costEstimator *CostEstimator
}

// NewQueryRouter creates a new query router
func NewQueryRouter(
	db *sqlx.DB,
	costEstimator *CostEstimator,
) *QueryRouter {
	return &QueryRouter{
		db:            db,
		costEstimator: costEstimator,
	}
}

// Route determines the optimal execution path for a query
func (r *QueryRouter) Route(ctx context.Context, query *SemanticQuery) (*QueryPlan, error) {
	startTime := time.Now()
	queryHash := HashQuery(query)

	plan := &QueryPlan{
		QueryID:   queryHash,
		CreatedAt: startTime,
	}

	// Step 1: Check for pre-aggregation coverage
	if match := r.checkPreAgg(ctx, query); match != nil {
		directCost, _, _ := r.costEstimator.EstimateCost(ctx, query)
		preAggCost := directCost * (1.0 - match.CoverageScore) * 0.1 // Much cheaper with pre-agg

		plan.PreAggMatch = match
		plan.ExecutionPath = PathPreAgg
		plan.EstimatedCost = preAggCost
		plan.Explanation = fmt.Sprintf(
			"Using pre-aggregation '%s' (%.0f%% coverage, freshness: %s)",
			match.PreAggName, match.CoverageScore*100, match.Freshness,
		)
		return plan, nil
	}

	// Step 2: Estimate direct execution cost
	directCost, factors, err := r.costEstimator.EstimateCost(ctx, query)
	if err != nil {
		log.Printf("Cost estimation failed: %v", err)
		directCost = 1000 // Default high cost
	}

	plan.EstimatedCost = directCost
	plan.ExecutionPath = PathDirect

	// Step 3: Determine if materialized view would be better
	if r.shouldUseMaterialized(query, factors) {
		plan.ExecutionPath = PathMaterialized
		plan.EstimatedCost = directCost * 0.5
		plan.Explanation = "Query matches materialized view pattern"
	} else {
		plan.Explanation = fmt.Sprintf(
			"Direct execution (scan ~%.0f rows, join complexity: %d, selectivity: %.2f%%)",
			float64(factors.DataVolume)*factors.FilterSelectivity,
			factors.JoinComplexity,
			factors.FilterSelectivity*100,
		)
	}

	return plan, nil
}

// PreAggNode represents a pre-aggregation from the catalog
type PreAggNode struct {
	ID         uuid.UUID       `db:"id"`
	Name       string          `db:"node_name"`
	Config     json.RawMessage `db:"config"`
	Properties json.RawMessage `db:"properties"`
}

// PreAggConfigDB represents the config structure in the database
type PreAggConfigDB struct {
	Terms        []string `json:"terms"`
	Calculations []string `json:"calculations"`
	GroupBy      []string `json:"group_by"`
	Dimensions   []string `json:"dimensions"`
	Measures     []string `json:"measures"`
}

// checkPreAgg checks for pre-aggregation coverage
func (r *QueryRouter) checkPreAgg(ctx context.Context, query *SemanticQuery) *PreAggMatch {
	// Get pre-aggregations from catalog
	var preAggs []PreAggNode
	err := r.db.SelectContext(ctx, &preAggs, `
		SELECT n.id, n.node_name, n.config, n.properties
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.tenant_id = $1
	`, query.TenantID)

	if err != nil || len(preAggs) == 0 {
		return nil
	}

	// Find best matching pre-aggregation
	var bestMatch *PreAggMatch
	var bestScore float64

	for _, pa := range preAggs {
		score := r.calculatePreAggCoverage(query, &pa)
		if score > bestScore && score >= 0.5 { // Require at least 50% coverage
			bestScore = score
			bestMatch = &PreAggMatch{
				PreAggID:      pa.ID,
				PreAggName:    pa.Name,
				CoverageScore: score,
				Freshness:     "unknown",
			}
		}
	}

	return bestMatch
}

// calculatePreAggCoverage calculates how well a pre-aggregation covers a query
func (r *QueryRouter) calculatePreAggCoverage(query *SemanticQuery, preAgg *PreAggNode) float64 {
	var cfg PreAggConfigDB
	if err := json.Unmarshal(preAgg.Config, &cfg); err != nil {
		return 0
	}

	// Use dimensions and measures from config, falling back to group_by and calculations
	dimensions := cfg.Dimensions
	if len(dimensions) == 0 {
		dimensions = cfg.GroupBy
	}
	measures := cfg.Measures
	if len(measures) == 0 {
		measures = cfg.Calculations
	}

	// Check dimension coverage
	dimCovered := 0
	for _, dim := range query.Dimensions {
		for _, paDim := range dimensions {
			if dim == paDim {
				dimCovered++
				break
			}
		}
	}

	// Check measure coverage
	measureCovered := 0
	for _, measure := range query.Measures {
		for _, paMeasure := range measures {
			if measure == paMeasure {
				measureCovered++
				break
			}
		}
	}

	// Calculate coverage score
	dimScore := 0.0
	if len(query.Dimensions) > 0 {
		dimScore = float64(dimCovered) / float64(len(query.Dimensions))
	} else {
		dimScore = 1.0
	}

	measureScore := 0.0
	if len(query.Measures) > 0 {
		measureScore = float64(measureCovered) / float64(len(query.Measures))
	} else {
		measureScore = 1.0
	}

	// Weight measures more heavily (0.6) vs dimensions (0.4)
	return dimScore*0.4 + measureScore*0.6
}

// shouldUseMaterialized determines if a materialized view is appropriate
func (r *QueryRouter) shouldUseMaterialized(query *SemanticQuery, factors *CostFactors) bool {
	// Use materialized view if:
	// 1. High data volume
	// 2. Low selectivity (scanning lots of data)
	// 3. Complex joins
	// 4. Query allows for slight staleness (not real-time)

	if query.Freshness == "realtime" {
		return false
	}

	if factors.DataVolume > 1000000 && factors.FilterSelectivity > 0.1 {
		return true
	}

	if factors.JoinComplexity >= 3 {
		return true
	}

	return false
}

// RecordExecution records query execution for workload analysis
func (r *QueryRouter) RecordExecution(ctx context.Context, plan *QueryPlan, actualDuration time.Duration, tenantID uuid.UUID) error {
	query := `
		INSERT INTO cbo_query_log (
			tenant_id, query_hash, query_pattern, execution_path,
			estimated_cost, actual_duration_ms, cache_hit, preagg_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var preAggID *uuid.UUID
	if plan.PreAggMatch != nil {
		preAggID = &plan.PreAggMatch.PreAggID
	}

	_, err := r.db.ExecContext(ctx, query,
		tenantID,
		plan.QueryID,
		"", // TODO: Extract pattern
		string(plan.ExecutionPath),
		plan.EstimatedCost,
		actualDuration.Milliseconds(),
		plan.CacheHit,
		preAggID,
	)

	return err
}

// GetStats returns CBO performance statistics
func (r *QueryRouter) GetStats(ctx context.Context, tenantID uuid.UUID) (*CBOStats, error) {
	stats := &CBOStats{}

	// Get total queries and cache hit rate
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) as cache_hit_rate,
			COALESCE(SUM(CASE WHEN preagg_used IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) as preagg_hit_rate,
			COALESCE(AVG(actual_duration_ms), 0) as avg_query_time
		FROM cbo_query_log
		WHERE created_at > NOW() - INTERVAL '24 hours'
	`

	err := r.db.QueryRowxContext(ctx, query).Scan(
		&stats.TotalQueries,
		&stats.CacheHitRate,
		&stats.PreAggHitRate,
		&stats.AvgQueryTime,
	)
	if err != nil {
		return stats, nil // Return empty stats on error
	}

	return stats, nil
}

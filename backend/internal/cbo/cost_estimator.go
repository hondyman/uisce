package cbo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/jmoiron/sqlx"
)

// CostEstimator estimates query execution costs
type CostEstimator struct {
	db         *sqlx.DB
	statsCache map[string]*TableStats
}

// NewCostEstimator creates a new cost estimator
func NewCostEstimator(db *sqlx.DB) *CostEstimator {
	return &CostEstimator{
		db:         db,
		statsCache: make(map[string]*TableStats),
	}
}

// EstimateCost estimates the cost of executing a query
func (e *CostEstimator) EstimateCost(ctx context.Context, query *SemanticQuery) (float64, *CostFactors, error) {
	factors := &CostFactors{}

	// Get table statistics
	stats, err := e.GetTableStats(ctx, query.BOName)
	if err != nil {
		// Use default estimates if stats unavailable
		factors.DataVolume = 100000 // Default 100k rows
	} else {
		factors.DataVolume = stats.RowCount
	}

	// Calculate join complexity
	factors.JoinComplexity = e.estimateJoinComplexity(query)

	// Calculate filter selectivity
	factors.FilterSelectivity = e.estimateFilterSelectivity(query.Filters, stats)

	// Calculate aggregation cost
	factors.AggregationCost = e.estimateAggregationCost(query)

	// Calculate total cost using a simple cost model
	cost := e.calculateTotalCost(factors)

	return cost, factors, nil
}

// GetTableStats retrieves or refreshes table statistics
func (e *CostEstimator) GetTableStats(ctx context.Context, tableName string) (*TableStats, error) {
	// Check cache first
	if stats, ok := e.statsCache[tableName]; ok {
		return stats, nil
	}

	// Try to get from cbo_table_stats
	var stats TableStats
	query := `
		SELECT table_name, row_count, avg_row_size, analyzed_at
		FROM cbo_table_stats
		WHERE table_name = $1
		ORDER BY analyzed_at DESC
		LIMIT 1
	`
	err := e.db.QueryRowxContext(ctx, query, tableName).Scan(
		&stats.TableName, &stats.RowCount, &stats.AvgRowSize, &stats.LastAnalyzed,
	)
	if err != nil {
		// Fallback: try to estimate from pg_class
		pgQuery := `
			SELECT relname, reltuples::bigint, (pg_relation_size(oid) / NULLIF(reltuples, 0))::int
			FROM pg_class
			WHERE relname = $1
		`
		var relTuples int64
		var avgSize int64
		err2 := e.db.QueryRowxContext(ctx, pgQuery, tableName).Scan(&stats.TableName, &relTuples, &avgSize)
		if err2 != nil {
			return nil, fmt.Errorf("no statistics available for table %s", tableName)
		}
		stats.RowCount = relTuples
		stats.AvgRowSize = avgSize
	}

	// Cache the stats
	e.statsCache[tableName] = &stats
	return &stats, nil
}

// RefreshTableStats updates statistics for a table
func (e *CostEstimator) RefreshTableStats(ctx context.Context, tableName string) error {
	// Get current stats from pg_class
	var rowCount int64
	var avgSize int64
	query := `
		SELECT reltuples::bigint, 
		       COALESCE((pg_relation_size(oid) / NULLIF(reltuples, 0))::int, 100)
		FROM pg_class
		WHERE relname = $1
	`
	err := e.db.QueryRowxContext(ctx, query, tableName).Scan(&rowCount, &avgSize)
	if err != nil {
		return fmt.Errorf("failed to get table stats: %w", err)
	}

	// Upsert into cbo_table_stats
	upsertQuery := `
		INSERT INTO cbo_table_stats (tenant_id, table_name, row_count, avg_row_size, analyzed_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tenant_id, table_name) DO UPDATE SET
			row_count = EXCLUDED.row_count,
			avg_row_size = EXCLUDED.avg_row_size,
			analyzed_at = NOW()
	`
	// Use a system tenant for system-wide stats
	_, err = e.db.ExecContext(ctx, upsertQuery, "00000000-0000-0000-0000-000000000000", tableName, rowCount, avgSize)
	if err != nil {
		return fmt.Errorf("failed to save table stats: %w", err)
	}

	// Invalidate cache
	delete(e.statsCache, tableName)

	return nil
}

// estimateJoinComplexity estimates the complexity of joins in a query
func (e *CostEstimator) estimateJoinComplexity(query *SemanticQuery) int {
	// Simple heuristic: each dimension beyond the first adds complexity
	complexity := 0
	if len(query.Dimensions) > 1 {
		complexity = len(query.Dimensions) - 1
	}
	// Measures can also require joins
	complexity += len(query.Measures) / 3
	return complexity
}

// estimateFilterSelectivity estimates how selective filters are
func (e *CostEstimator) estimateFilterSelectivity(filters []QueryFilter, stats *TableStats) float64 {
	if len(filters) == 0 {
		return 1.0 // No filters = scan everything
	}

	selectivity := 1.0
	for _, filter := range filters {
		// Use column stats if available
		if stats != nil && stats.ColumnStats != nil {
			if colStats, ok := stats.ColumnStats[filter.Field]; ok {
				// Estimate selectivity based on operator and distinct count
				switch filter.Operator {
				case "eq":
					if colStats.DistinctCount > 0 {
						selectivity *= 1.0 / float64(colStats.DistinctCount)
					} else {
						selectivity *= 0.1
					}
				case "in":
					// Depends on number of values
					selectivity *= 0.2
				case "between", "gt", "gte", "lt", "lte":
					selectivity *= 0.3
				case "like":
					if strings.HasPrefix(filter.Value.(string), "%") {
						selectivity *= 0.5 // Leading wildcard is expensive
					} else {
						selectivity *= 0.2
					}
				default:
					selectivity *= 0.5
				}
				continue
			}
		}

		// Default selectivity estimates
		switch filter.Operator {
		case "eq":
			selectivity *= 0.1
		case "ne":
			selectivity *= 0.9
		case "in":
			selectivity *= 0.2
		case "between":
			selectivity *= 0.25
		case "gt", "gte", "lt", "lte":
			selectivity *= 0.33
		case "like":
			selectivity *= 0.25
		default:
			selectivity *= 0.5
		}
	}

	return math.Max(0.001, selectivity) // At least 0.1% selectivity
}

// estimateAggregationCost estimates the cost of aggregations
func (e *CostEstimator) estimateAggregationCost(query *SemanticQuery) float64 {
	cost := 0.0

	// Each measure adds aggregation cost
	for _, measure := range query.Measures {
		switch {
		case strings.Contains(strings.ToLower(measure), "count"):
			cost += 0.1
		case strings.Contains(strings.ToLower(measure), "sum"), strings.Contains(strings.ToLower(measure), "avg"):
			cost += 0.2
		case strings.Contains(strings.ToLower(measure), "min"), strings.Contains(strings.ToLower(measure), "max"):
			cost += 0.15
		case strings.Contains(strings.ToLower(measure), "distinct"):
			cost += 0.5 // Distinct is expensive
		case strings.Contains(strings.ToLower(measure), "percentile"), strings.Contains(strings.ToLower(measure), "median"):
			cost += 1.0 // Very expensive
		default:
			cost += 0.2
		}
	}

	// Group by cardinality affects cost
	cost += float64(len(query.GroupBy)) * 0.1

	return cost
}

// calculateTotalCost combines factors into a single cost estimate
func (e *CostEstimator) calculateTotalCost(factors *CostFactors) float64 {
	// Base cost from data volume
	baseCost := float64(factors.DataVolume) * factors.FilterSelectivity

	// Join cost multiplier (exponential with complexity)
	joinMultiplier := math.Pow(1.5, float64(factors.JoinComplexity))

	// Aggregation overhead
	aggOverhead := 1.0 + factors.AggregationCost

	// Resource pressure adjustment
	pressureAdjustment := 1.0 + factors.ResourcePressure

	totalCost := baseCost * joinMultiplier * aggOverhead * pressureAdjustment

	return totalCost
}

// HashQuery generates a hash for a semantic query (for caching and pattern matching)
func HashQuery(query *SemanticQuery) string {
	// Create a canonical representation
	canonical := fmt.Sprintf(
		"bo:%s|dims:%v|measures:%v|groupby:%v|filters:%d",
		query.BOID.String(),
		query.Dimensions,
		query.Measures,
		query.GroupBy,
		len(query.Filters),
	)
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:8]) // First 8 bytes
}

// ExtractPattern extracts a query pattern (removing specific filter values)
func ExtractPattern(query *SemanticQuery) string {
	filterTypes := make([]string, len(query.Filters))
	for i, f := range query.Filters {
		filterTypes[i] = fmt.Sprintf("%s:%s", f.Field, f.Operator)
	}

	return fmt.Sprintf(
		"dims[%s]|measures[%s]|filters[%s]|groupby[%s]",
		strings.Join(query.Dimensions, ","),
		strings.Join(query.Measures, ","),
		strings.Join(filterTypes, ","),
		strings.Join(query.GroupBy, ","),
	)
}

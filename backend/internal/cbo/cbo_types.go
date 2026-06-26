package cbo

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionPath represents how a query will be executed
type ExecutionPath string

const (
	// PathDirect executes against source tables directly
	PathDirect ExecutionPath = "direct"
	// PathPreAgg uses a pre-aggregation table
	PathPreAgg ExecutionPath = "preagg"
	// PathCache uses cached SQL results
	PathCache ExecutionPath = "cache"
	// PathMaterialized uses a real-time materialized view
	PathMaterialized ExecutionPath = "materialized"
)

// QueryPlan represents the execution plan for a semantic query
type QueryPlan struct {
	QueryID       string        `json:"query_id"`
	OriginalSQL   string        `json:"original_sql"`
	OptimizedSQL  string        `json:"optimized_sql,omitempty"`
	EstimatedCost float64       `json:"estimated_cost"`
	ExecutionPath ExecutionPath `json:"execution_path"`
	CacheHit      bool          `json:"cache_hit"`
	PreAggMatch   *PreAggMatch  `json:"preagg_match,omitempty"`
	Explanation   string        `json:"explanation"`
	CreatedAt     time.Time     `json:"created_at"`
}

// PreAggMatch represents a match with a pre-aggregation table
type PreAggMatch struct {
	PreAggID      uuid.UUID `json:"preagg_id"`
	PreAggName    string    `json:"preagg_name"`
	CoverageScore float64   `json:"coverage_score"` // 0-1, how well it covers the query
	Freshness     string    `json:"freshness"`      // e.g., "5m", "1h"
}

// CostFactors represents factors that affect query cost
type CostFactors struct {
	DataVolume        int64         `json:"data_volume"`        // Estimated rows to scan
	JoinComplexity    int           `json:"join_complexity"`    // Number and type of joins
	FilterSelectivity float64       `json:"filter_selectivity"` // 0-1, how selective filters are
	AggregationCost   float64       `json:"aggregation_cost"`   // Cost of aggregations
	Freshness         time.Duration `json:"freshness"`          // Required data freshness
	ResourcePressure  float64       `json:"resource_pressure"`  // Current system load 0-1
}

// TableStats contains statistics for cost estimation
type TableStats struct {
	TableName    string                  `json:"table_name"`
	RowCount     int64                   `json:"row_count"`
	AvgRowSize   int64                   `json:"avg_row_size"`
	LastAnalyzed time.Time               `json:"last_analyzed"`
	ColumnStats  map[string]*ColumnStats `json:"column_stats"`
}

// ColumnStats contains column-level statistics
type ColumnStats struct {
	ColumnName    string  `json:"column_name"`
	NullFraction  float64 `json:"null_fraction"`
	DistinctCount int64   `json:"distinct_count"`
	AvgWidth      int     `json:"avg_width"`
	MinValue      string  `json:"min_value,omitempty"`
	MaxValue      string  `json:"max_value,omitempty"`
}

// SemanticQuery represents a query against the semantic layer
type SemanticQuery struct {
	BOID         uuid.UUID     `json:"bo_id"`
	BOName       string        `json:"bo_name"`
	Dimensions   []string      `json:"dimensions"`
	Measures     []string      `json:"measures"`
	Filters      []QueryFilter `json:"filters"`
	GroupBy      []string      `json:"group_by"`
	OrderBy      []string      `json:"order_by"`
	Limit        int           `json:"limit"`
	Freshness    string        `json:"freshness"` // "realtime", "near-realtime", "batch"
	TenantID     uuid.UUID     `json:"tenant_id"`
	DatasourceID uuid.UUID     `json:"datasource_id"`
}

// QueryFilter represents a filter condition
type QueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, gte, lt, lte, in, between, like
	Value    interface{} `json:"value"`
}

// QueryRecord stores executed query information for workload analysis
type QueryRecord struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	TenantID       uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	QueryHash      string        `json:"query_hash" db:"query_hash"`
	QueryPattern   string        `json:"query_pattern" db:"query_pattern"`
	ExecutionPath  ExecutionPath `json:"execution_path" db:"execution_path"`
	EstimatedCost  float64       `json:"estimated_cost" db:"estimated_cost"`
	ActualDuration int           `json:"actual_duration_ms" db:"actual_duration_ms"`
	CacheHit       bool          `json:"cache_hit" db:"cache_hit"`
	PreAggUsed     *uuid.UUID    `json:"preagg_used,omitempty" db:"preagg_used"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
}

// QueryPattern represents a frequently occurring query pattern
type QueryPattern struct {
	Pattern     string    `json:"pattern"`
	Frequency   int       `json:"frequency"`
	AvgDuration float64   `json:"avg_duration_ms"`
	AvgCost     float64   `json:"avg_cost"`
	LastSeen    time.Time `json:"last_seen"`
	Dimensions  []string  `json:"dimensions"`
	Measures    []string  `json:"measures"`
	Optimizable bool      `json:"optimizable"`
}

// Recommendation is a suggestion from the CBO
type Recommendation struct {
	Type        string    `json:"type"`     // "create_preagg", "add_index", "partition_table"
	Priority    string    `json:"priority"` // "critical", "high", "medium", "low"
	Description string    `json:"description"`
	Impact      string    `json:"impact"` // Expected improvement
	SQLHint     string    `json:"sql_hint,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CBOStats contains statistics about the CBO performance
type CBOStats struct {
	TotalQueries           int64          `json:"total_queries"`
	CacheHitRate           float64        `json:"cache_hit_rate"`
	PreAggHitRate          float64        `json:"preagg_hit_rate"`
	AvgPlanTime            float64        `json:"avg_plan_time_ms"`
	AvgQueryTime           float64        `json:"avg_query_time_ms"`
	CostSavings            float64        `json:"cost_savings_pct"`
	TopPatterns            []QueryPattern `json:"top_patterns"`
	PendingRecommendations int            `json:"pending_recommendations"`
}

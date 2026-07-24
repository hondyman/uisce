package models

import (
	"time"

	"github.com/google/uuid"
)

// QueryTelemetry represents a single query telemetry event.
type QueryTelemetry struct {
	ID               uuid.UUID `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	BOName           string    `json:"bo_name" db:"bo_name"`
	CubeQueryID      *string   `json:"cube_query_id,omitempty" db:"cube_query_id"`
	StarRocksQueryID *string   `json:"starrocks_query_id,omitempty" db:"starrocks_query_id"`
	StartedAt        time.Time `json:"started_at" db:"started_at"`
	DurationMs       int       `json:"duration_ms" db:"duration_ms"`
	RowsScanned      *int64    `json:"rows_scanned,omitempty" db:"rows_scanned"`
	BytesScanned     *int64    `json:"bytes_scanned,omitempty" db:"bytes_scanned"`
	RowsReturned     *int64    `json:"rows_returned,omitempty" db:"rows_returned"`
	Status           string    `json:"status" db:"status"`
	ErrorMessage     *string   `json:"error_message,omitempty" db:"error_message"`
	GroupByTerms     []string  `json:"group_by_terms,omitempty"`
	Measures         []string  `json:"measures,omitempty"`
	Filters          []Filter  `json:"filters,omitempty"`
	Source           string    `json:"source" db:"source"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	// Pre-aggregation tracking
	PreAggID  *string `json:"preagg_id,omitempty" db:"preagg_id"`
	PreAggHit bool    `json:"preagg_hit" db:"preagg_hit"`
}

// Filter represents a filter expression in telemetry.
type Filter struct {
	Term     string `json:"term"`
	Operator string `json:"op"`
	Value    string `json:"value,omitempty"`
}

// TelemetryIngestionRequest is the API payload for ingesting telemetry.
type TelemetryIngestionRequest struct {
	TenantID         string   `json:"tenant_id"`
	BOName           string   `json:"bo_name"`
	CubeQueryID      *string  `json:"cube_query_id,omitempty"`
	StarRocksQueryID *string  `json:"starrocks_query_id,omitempty"`
	DurationMs       int      `json:"duration_ms"`
	RowsScanned      *int64   `json:"rows_scanned,omitempty"`
	BytesScanned     *int64   `json:"bytes_scanned,omitempty"`
	RowsReturned     *int64   `json:"rows_returned,omitempty"`
	GroupByTerms     []string `json:"group_by_terms,omitempty"`
	Measures         []string `json:"measures,omitempty"`
	Filters          []Filter `json:"filters,omitempty"`
	Status           string   `json:"status"`
	ErrorMessage     *string  `json:"error_message,omitempty"`
	Source           string   `json:"source,omitempty"`
	// Pre-aggregation tracking
	PreAggID  *string `json:"preagg_id,omitempty"`
	PreAggHit bool    `json:"preagg_hit,omitempty"`
}

// BOWorkloadProfile represents aggregated workload metrics for a BO.
type BOWorkloadProfile struct {
	TenantID       string  `json:"tenant_id"`
	BOName         string  `json:"bo_name"`
	TotalQueries   int     `json:"total_queries"`
	SlowQueries    int     `json:"slow_queries"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	P95DurationMs  float64 `json:"p95_duration_ms"`
	AvgRowsScanned float64 `json:"avg_rows_scanned"`
	P95RowsScanned float64 `json:"p95_rows_scanned"`

	TopGroupBys []GroupByProfile `json:"top_group_bys"`
	TopMeasures []MeasureProfile `json:"top_measures"`
	TopFilters  []FilterProfile  `json:"top_filters"`
}

// GroupByProfile represents usage stats for a specific grain.
type GroupByProfile struct {
	Terms          []string `json:"terms"`
	QueryCount     int      `json:"query_count"`
	AvgDurationMs  float64  `json:"avg_duration_ms"`
	P95DurationMs  float64  `json:"p95_duration_ms"`
	AvgRowsScanned float64  `json:"avg_rows_scanned"`
}

// MeasureProfile represents usage stats for a specific measure.
type MeasureProfile struct {
	Name          string  `json:"name"`
	QueryCount    int     `json:"query_count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

// FilterProfile represents usage stats for a specific filter.
type FilterProfile struct {
	Term          string  `json:"term"`
	Operator      string  `json:"operator"`
	QueryCount    int     `json:"query_count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

// PreAggCostEstimate contains cost/benefit estimates for a pre-aggregation.
type PreAggCostEstimate struct {
	TenantID               string   `json:"tenant_id"`
	BOName                 string   `json:"bo_name"`
	Grain                  []string `json:"grain"`
	Measures               []string `json:"measures"`
	EstimatedQueriesPerDay int      `json:"estimated_queries_per_day"`
	AvgDurationMs          float64  `json:"avg_duration_ms"`
	P95DurationMs          float64  `json:"p95_duration_ms"`
	AvgRowsScanned         float64  `json:"avg_rows_scanned"`
	EstimatedSpeedupFactor float64  `json:"estimated_speedup_factor"`
	EstimatedStorageBytes  int64    `json:"estimated_storage_bytes"`
	EstimatedBuildCost     float64  `json:"estimated_build_cost"`
	EstimatedRefreshCost   float64  `json:"estimated_refresh_cost"`
	Score                  float64  `json:"score"`
}

// PreAggRecommendation represents a recommendation for a pre-aggregation.
type PreAggRecommendation struct {
	TenantID           string             `json:"tenant_id"`
	BOName             string             `json:"bo_name"`
	Grain              []string           `json:"grain"`
	Measures           []string           `json:"measures"`
	SuggestedFilters   []Filter           `json:"suggested_filters,omitempty"`
	CostEstimate       PreAggCostEstimate `json:"cost_estimate"`
	ExistingPreAggID   *uuid.UUID         `json:"existing_pre_agg_id,omitempty"`
	ExistingPreAggName *string            `json:"existing_pre_agg_name,omitempty"`
	RecommendationType string             `json:"recommendation_type"` // new, tune_refresh, expand_measures, retire
}

// BOAdvisorResponse is the API response for BO-level advisor.
type BOAdvisorResponse struct {
	Workload                *BOWorkloadProfile     `json:"workload"`
	Recommendations         []PreAggRecommendation `json:"recommendations"`
	ExistingPreAggregations []PreAggDescriptor     `json:"existing_pre_aggregations"`
}

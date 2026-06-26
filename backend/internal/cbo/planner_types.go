package cbo

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
)

// QuerySLO defines SLO constraints for a query
type QuerySLO struct {
	MaxP95LatencyMs    *float64 `json:"max_p95_latency_ms,omitempty"`
	MaxFreshnessLagSec *float64 `json:"max_freshness_lag_sec,omitempty"`
	MaxErrorRate       *float64 `json:"max_error_rate,omitempty"`
}

// PlanContext provides all context needed to plan a query
type PlanContext struct {
	Env           string                 `json:"env"`
	TenantID      *uuid.UUID             `json:"tenant_id,omitempty"`
	DatasourceID  *uuid.UUID             `json:"datasource_id,omitempty"`
	BOName        string                 `json:"bo_name"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	GroupBy       []string               `json:"group_by,omitempty"`
	Measures      []string               `json:"measures,omitempty"`
	Dimensions    []string               `json:"dimensions,omitempty"`
	CurrentUserID string                 `json:"current_user_id,omitempty"`
	PageSlug      string                 `json:"page_slug,omitempty"`
	ApiID         string                 `json:"api_id,omitempty"`
	SLO           *QuerySLO              `json:"slo,omitempty"`
	RequestedAt   time.Time              `json:"requested_at"`

	// Region — required for all runtime operations per region-spec
	Region string `json:"region,omitempty"`

	// Snapshot provides an optional semantic snapshot for the planner to consult
	Snapshot *audit.SemanticSnapshot `json:"snapshot,omitempty"`
}

// PlanCost represents the estimated cost of a query plan
type PlanCost struct {
	EstimatedLatencyMs   float64 `json:"estimated_latency_ms"`
	EstimatedScanBytes   float64 `json:"estimated_scan_bytes"`
	EstimatedComputeCost float64 `json:"estimated_compute_cost"`
	EntitlementCostMs    float64 `json:"entitlement_cost_ms"`
	FreshnessLagSec      float64 `json:"freshness_lag_sec"`
}

// TotalCost returns a combined cost score for comparison
func (c PlanCost) TotalCost() float64 {
	// Weight latency highest, then compute, then entitlement overhead
	return c.EstimatedLatencyMs + (c.EstimatedComputeCost * 10) + c.EntitlementCostMs
}

// PlanType represents the type of execution plan
type PlanType string

const (
	PlanTypeBase   PlanType = "base"
	PlanTypePreAgg PlanType = "preagg"
	PlanTypeHybrid PlanType = "hybrid"
	PlanTypeCached PlanType = "cached"
)

// EntitlementStrategy represents how entitlements are applied
type EntitlementStrategy string

const (
	EntitlementJoin      EntitlementStrategy = "join"      // Join with entitlement table
	EntitlementPrefilter EntitlementStrategy = "prefilter" // Pre-filter using cached entitlements
	EntitlementNone      EntitlementStrategy = "none"      // No entitlement filtering needed
	EntitlementInline    EntitlementStrategy = "inline"    // Inline entitlement predicates
)

// PlannedQuery represents a fully planned query ready for execution
type PlannedQuery struct {
	SQL                 string              `json:"sql"`
	PlanType            PlanType            `json:"plan_type"`
	PreAggName          *string             `json:"preagg_name,omitempty"`
	PreAggID            *uuid.UUID          `json:"preagg_id,omitempty"`
	EntitlementStrategy EntitlementStrategy `json:"entitlement_strategy"`
	Cost                PlanCost            `json:"cost"`
	CandidatesEvaluated int                 `json:"candidates_evaluated"`
	SLOSatisfied        bool                `json:"slo_satisfied"`
	PlanningTimeMs      float64             `json:"planning_time_ms"`
}

// UsePreAgg returns true if this plan uses a pre-aggregation (should route to StarRocks)
func (p PlannedQuery) UsePreAgg() bool {
	return p.PlanType == PlanTypePreAgg && p.PreAggID != nil
}

// QueryPlanMetadata is returned alongside SQL for telemetry
type QueryPlanMetadata struct {
	PlanType            PlanType            `json:"plan_type"`
	PreAggName          *string             `json:"preagg_name,omitempty"`
	EntitlementStrategy EntitlementStrategy `json:"entitlement_strategy"`
	Cost                PlanCost            `json:"cost"`
	CandidatesEvaluated int                 `json:"candidates_evaluated"`
	PlanningTimeMs      float64             `json:"planning_time_ms"`
}

// PreAggDescriptor describes a pre-aggregation for planning
type PreAggDescriptor struct {
	ID                  uuid.UUID         `json:"id"`
	Name                string            `json:"name"`
	BOName              string            `json:"bo_name"`
	Dimensions          []string          `json:"dimensions"`
	Measures            []string          `json:"measures"`
	Filters             []string          `json:"filters,omitempty"`
	Grain               string            `json:"grain,omitempty"` // e.g., "daily", "hourly"
	RefreshFrequencySec int               `json:"refresh_frequency_sec"`
	LastRefreshAt       *time.Time        `json:"last_refresh_at,omitempty"`
	AvgSpeedup          float64           `json:"avg_speedup"`
	StorageBytes        int64             `json:"storage_bytes"`
	HitRate             float64           `json:"hit_rate"`
	TargetTable         string            `json:"target_table"`
	Region              string            `json:"region,omitempty"`
	Properties          map[string]string `json:"properties,omitempty"`
}

// BOFeatures represents telemetry-derived features for a BO
type BOFeatures struct {
	BOName        string    `json:"bo_name"`
	Env           string    `json:"env"`
	TenantID      *string   `json:"tenant_id,omitempty"`
	P50LatencyMs  float64   `json:"p50_latency_ms"`
	P95LatencyMs  float64   `json:"p95_latency_ms"`
	P99LatencyMs  float64   `json:"p99_latency_ms"`
	AvgScanBytes  float64   `json:"avg_scan_bytes"`
	QueryCount    int64     `json:"query_count"`
	ErrorRate     float64   `json:"error_rate"`
	CacheHitRate  float64   `json:"cache_hit_rate"`
	PreAggHitRate float64   `json:"preagg_hit_rate"`
	LastQueryAt   time.Time `json:"last_query_at"`
	Window        string    `json:"window"` // e.g., "7d", "30d"
}

// PreAggFeatures represents telemetry-derived features for a pre-agg
type PreAggFeatures struct {
	PreAggName          string    `json:"preagg_name"`
	Env                 string    `json:"env"`
	TenantID            *string   `json:"tenant_id,omitempty"`
	AvgSpeedup          float64   `json:"avg_speedup"`
	HitCount            int64     `json:"hit_count"`
	MissCount           int64     `json:"miss_count"`
	HitRate             float64   `json:"hit_rate"`
	StorageBytes        int64     `json:"storage_bytes"`
	RefreshFrequencySec int       `json:"refresh_frequency_sec"`
	LastRefreshAt       time.Time `json:"last_refresh_at"`
	AvgFreshnessLagSec  float64   `json:"avg_freshness_lag_sec"`
	Window              string    `json:"window"`
}

// CandidatePlan represents one possible execution plan before cost estimation
type CandidatePlan struct {
	SQL                 string              `json:"sql"`
	PlanType            PlanType            `json:"plan_type"`
	PreAgg              *PreAggDescriptor   `json:"preagg,omitempty"`
	EntitlementStrategy EntitlementStrategy `json:"entitlement_strategy"`
}

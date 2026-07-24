package planner

import (
	"encoding/json"
	"time"
)

// QueryRequest represents a semantic query request
type QueryRequest struct {
	TenantID             string                 `json:"tenant_id,omitempty"`
	RegionHint           string                 `json:"region_hint,omitempty"`
	QueryType            string                 `json:"query_type"` // feature|metric|ts|drift|importance|discovery
	SemanticTarget       string                 `json:"semantic_target"`
	TimeRange            *TimeRange             `json:"time_range,omitempty"`
	FreshnessRequirement string                 `json:"freshness_requirement,omitempty"` // e.g., "5m", "1h"
	ConsistencyLevel     string                 `json:"consistency_level"`               // strong|eventual|region_preferred
	Priority             string                 `json:"priority"`                        // interactive|batch|background
	Extra                map[string]interface{} `json:"extra,omitempty"`
}

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// EngineRoute describes how to reach a specific engine for a query execution
type EngineRoute struct {
	EngineType string `json:"engine_type"` // trino|ts_service|drift_service|discovery_service
	Region     string `json:"region"`
	Endpoint   string `json:"endpoint"`
	Catalog    string `json:"catalog,omitempty"`
	Table      string `json:"table,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// DegradationStrategy defines how to handle partial failures
type DegradationStrategy struct {
	Mode            string   `json:"mode"` // fail_fast|partial_results|fallback_region|use_cache
	FallbackRegions []string `json:"fallback_regions,omitempty"`
	MaxStaleness    string   `json:"max_staleness,omitempty"` // e.g., "15m"
}

// QueryPlan represents the planner's decision on how to execute a query
type QueryPlan struct {
	PlanID               string              `json:"plan_id"`
	PlanType             string              `json:"plan_type"` // single_region|multi_region_fanout|global_federated
	SelectedRegions      []string            `json:"selected_regions"`
	EngineRoutes         []EngineRoute       `json:"engine_routes"`
	EstimatedCost        float64             `json:"estimated_cost"`
	EstimatedLatencyMS   float64             `json:"estimated_latency_ms"`
	DegradationStrategy  DegradationStrategy `json:"degradation_strategy"`
	Explain              string              `json:"explain"`
	RegionHealthSnapshot json.RawMessage     `json:"region_health_snapshot,omitempty"`
}

// ExplainPlan is the detailed explanation returned to the UI
type ExplainPlan struct {
	PlanID  string              `json:"plan_id"`
	Summary ExplanationSummary  `json:"summary"`
	Routing ExplanationRouting  `json:"routing"`
	Engines []ExplanationEngine `json:"engines"`
	Explain ExplainDetails      `json:"explain"`
}

type ExplanationSummary struct {
	PlanType  string   `json:"plan_type"`
	Regions   []string `json:"regions"`
	LatencyMS float64  `json:"latency_ms"`
	Cost      float64  `json:"cost"`
	Degraded  bool     `json:"degraded"`
}

type ExplanationRouting struct {
	SelectedRegions   []string `json:"selected_regions"`
	FallbackRegions   []string `json:"fallback_regions,omitempty"`
	Consistency       string   `json:"consistency"`
	FreshnessRequired string   `json:"freshness_requirement,omitempty"`
}

type ExplanationEngine struct {
	EngineType string `json:"engine_type"`
	Region     string `json:"region"`
	Endpoint   string `json:"endpoint"`
	Catalog    string `json:"catalog,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type ExplainDetails struct {
	DecisionText              string `json:"decision_text"`
	RegionSelectionReason     string `json:"region_selection_reason"`
	EngineSelectionReason     string `json:"engine_selection_reason"`
	LatencyEstimateReason     string `json:"latency_estimate_reason"`
	CostEstimateReason        string `json:"cost_estimate_reason"`
	DegradationStrategyReason string `json:"degradation_strategy_reason,omitempty"`
}

// PlannerDecision is the persisted decision record
type PlannerDecision struct {
	PlanID               string          `db:"plan_id"`
	CreatedAt            time.Time       `db:"created_at"`
	TenantID             string          `db:"tenant_id"`
	QueryType            string          `db:"query_type"`
	SemanticTarget       string          `db:"semantic_target"`
	SelectedRegions      []string        `db:"selected_regions"`
	PlanType             string          `db:"plan_type"`
	EstimatedCost        float64         `db:"estimated_cost"`
	EstimatedLatencyMS   float64         `db:"estimated_latency_ms"`
	DegradationStrategy  json.RawMessage `db:"degradation_strategy"`
	Explain              string          `db:"explain"`
	RawRequest           json.RawMessage `db:"raw_request"`
	RawPlan              json.RawMessage `db:"raw_plan"`
	ExecutedAt           *time.Time      `db:"executed_at"`
	ActualLatencyMS      *float64        `db:"actual_latency_ms"`
	ActualCost           *float64        `db:"actual_cost"`
	ExecutionStatus      string          `db:"execution_status"`
	ExecutionError       *string         `db:"execution_error"`
	RegionHealthSnapshot json.RawMessage `db:"region_health_snapshot"`
}

// RegionPerformance describes current region health
type RegionPerformance struct {
	Region                          string    `db:"region"`
	LastUpdated                     time.Time `db:"last_updated"`
	IsHealthy                       bool      `db:"is_healthy"`
	LatencyP50MS                    *float64  `db:"latency_ms_p50"`
	LatencyP95MS                    *float64  `db:"latency_ms_p95"`
	LatencyP99MS                    *float64  `db:"latency_ms_p99"`
	ErrorRate                       *float64  `db:"error_rate"`
	ActiveFeatures                  int       `db:"active_features"`
	MaterializationFreshnessPercent *float64  `db:"materialization_freshness_pct"`
	CacheHitRate                    *float64  `db:"cache_hit_rate"`
}

// PlannerMetric tracks planner decision accuracy
type PlannerMetric struct {
	ID                 int       `db:"id"`
	Ts                 time.Time `db:"ts"`
	QueryType          string    `db:"query_type"`
	PlanType           string    `db:"plan_type"`
	EstimatedLatencyMS *float64  `db:"estimated_latency_ms"`
	ActualLatencyMS    *float64  `db:"actual_latency_ms"`
	LatencyErrorPct    *float64  `db:"latency_error_pct"`
	EstimatedCost      *float64  `db:"estimated_cost"`
	ActualCost         *float64  `db:"actual_cost"`
	RegionsUsed        int       `db:"regions_used"`
	ExecutionStatus    string    `db:"execution_status"`
	Degraded           bool      `db:"degraded"`
}

// FeaturePlannerConfig describes planner preferences for a feature
type FeaturePlannerConfig struct {
	FeatureID                  string    `db:"feature_id"`
	PreferredRegions           []string  `db:"preferred_regions"`
	DisallowedRegions          []string  `db:"disallowed_regions"`
	DefaultConsistency         string    `db:"default_consistency"`
	DefaultFreshness           string    `db:"default_freshness"`
	InteractiveLatencyBudgetMS int       `db:"interactive_latency_budget_ms"`
	BatchLatencyBudgetMS       int       `db:"batch_latency_budget_ms"`
	UseCacheIfStale            bool      `db:"use_cache_if_stale"`
	MaxCacheStaleness          string    `db:"max_cache_staleness"`
	CreatedAt                  time.Time `db:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at"`
}

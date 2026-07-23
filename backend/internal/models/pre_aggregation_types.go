package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PreAggregation node properties (stored in catalog_node.properties)
type PreAggProperties struct {
	BOName                 string `json:"bo_name"`
	TenantID               string `json:"tenant_id"`
	Dialect                string `json:"dialect"`          // "starrocks"
	RefreshStrategy        string `json:"refresh_strategy"` // "manual", "interval", "incremental"
	RefreshIntervalMinutes int    `json:"refresh_interval_minutes,omitempty"`
	GovernanceStatus       string `json:"governance_status,omitempty"` // "draft", "review", "published", "deprecated"
	TargetDatabase         string `json:"target_database"`             // e.g., "tenant_{tenant_id}"

	// Lifecycle fields
	LifecycleStatus      string     `json:"lifecycle_status,omitempty"` // idle, materializing, active, refreshing, stale, failed
	LastMaterializedAt   *time.Time `json:"last_materialized_at,omitempty"`
	LastRefreshedAt      *time.Time `json:"last_refreshed_at,omitempty"`
	LastRefreshStatus    string     `json:"last_refresh_status,omitempty"` // success, failed
	LastRefreshError     string     `json:"last_refresh_error,omitempty"`
	NextScheduledRefresh *time.Time `json:"next_scheduled_refresh,omitempty"`
	RowCount             *int64     `json:"row_count,omitempty"`
	SizeBytes            *int64     `json:"size_bytes,omitempty"`
	// Usage tracking (persisted snapshot)
	UsageCount            int64   `json:"usage_count,omitempty"`
	AvgLatencyReductionMs float64 `json:"avg_latency_reduction_ms,omitempty"`
}

// Lifecycle status constants
const (
	LifecycleIdle          = "idle"
	LifecycleMaterializing = "materializing"
	LifecycleActive        = "active"
	LifecycleRefreshing    = "refreshing"
	LifecycleStale         = "stale"
	LifecycleFailed        = "failed"
)

// PreAggregation node config (stored in catalog_node.config)
type PreAggConfig struct {
	Terms           []string              `json:"terms"`        // SemanticTerm names to include as dimensions
	Calculations    []string              `json:"calculations"` // CalculationTerm names to include as measures
	Filters         []PreAggFilter        `json:"filters,omitempty"`
	GroupBy         []string              `json:"group_by"` // Grain columns
	OrderBy         []string              `json:"order_by,omitempty"`
	Materialization MaterializationConfig `json:"materialization"`
}

type PreAggFilter struct {
	Expression string `json:"expression"`
}

type MaterializationConfig struct {
	Type                  string `json:"type"`        // "materialized_view" or "table"
	TargetName            string `json:"target_name"` // Physical StarRocks object name
	IncrementalColumn     string `json:"incremental_column,omitempty"`
	IncrementalWindowDays int    `json:"incremental_window_days,omitempty"`
	PartitionBy           string `json:"partition_by,omitempty"`
}

// API Request/Response types

type UpsertPreAggRequest struct {
	TenantID               string                `json:"tenant_id"`
	BOName                 string                `json:"bo_name"`
	Name                   string                `json:"name"` // pre-agg node_name
	Description            string                `json:"description,omitempty"`
	Terms                  []string              `json:"terms"`
	Calculations           []string              `json:"calculations"`
	GroupBy                []string              `json:"group_by"`
	Filters                []PreAggFilter        `json:"filters,omitempty"`
	Materialization        MaterializationConfig `json:"materialization"`
	RefreshStrategy        string                `json:"refresh_strategy"`
	RefreshIntervalMinutes int                   `json:"refresh_interval_minutes,omitempty"`
}

// PreAggStatus represents the lifecycle status of a pre-aggregation.
type PreAggStatus string

const (
	PreAggStatusDraft    PreAggStatus = "draft"
	PreAggStatusActive   PreAggStatus = "active"
	PreAggStatusDisabled PreAggStatus = "disabled"
	PreAggStatusError    PreAggStatus = "error"
)

type PreAggDescriptor struct {
	ID                     uuid.UUID    `json:"id"`
	TenantID               string       `json:"tenant_id"`
	BOName                 string       `json:"bo_name"`
	Datasource             string       `json:"datasource,omitempty"` // Semantic datasource name
	Name                   string       `json:"name"`
	Description            string       `json:"description,omitempty"`
	TargetDatabase         string       `json:"target_database"`
	TargetName             string       `json:"target_name"`
	Dialect                string       `json:"dialect"`
	RefreshStrategy        string       `json:"refresh_strategy"`
	RefreshIntervalMinutes int          `json:"refresh_interval_minutes,omitempty"`
	GovernanceStatus       string       `json:"governance_status"`
	Status                 PreAggStatus `json:"status"` // draft, active, disabled, error
	CreatedAt              time.Time    `json:"created_at"`
	CreatedBy              string       `json:"created_by,omitempty"`
	UpdatedAt              time.Time    `json:"updated_at"`

	// Pattern matching fields
	PatternFingerprint string   `json:"pattern_fingerprint,omitempty"` // SQL/semantic fingerprint
	GroupBy            []string `json:"group_by,omitempty"`            // Group-by columns
	Measures           []string `json:"measures,omitempty"`            // Measure names/aliases
	FiltersSupported   []string `json:"filters_supported,omitempty"`   // Allowed filter fields

	// Storage references
	IcebergTable string `json:"iceberg_table,omitempty"` // Iceberg rollup table name
	StarRocksMV  string `json:"starrocks_mv,omitempty"`  // StarRocks MV name

	// Lifecycle fields
	LifecycleStatus      string     `json:"lifecycle_status"`
	LastMaterializedAt   *time.Time `json:"last_materialized_at,omitempty"`
	LastRefreshedAt      *time.Time `json:"last_refreshed_at,omitempty"`
	LastRefreshStatus    string     `json:"last_refresh_status,omitempty"`
	LastRefreshError     string     `json:"last_refresh_error,omitempty"`
	NextScheduledRefresh *time.Time `json:"next_scheduled_refresh,omitempty"`
	RowCount             *int64     `json:"row_count,omitempty"`
	SizeBytes            *int64     `json:"size_bytes,omitempty"`

	// Usage tracking
	UsageCount            int64   `json:"usage_count"`              // Number of queries using this pre-agg
	AvgLatencyReductionMs float64 `json:"avg_latency_reduction_ms"` // Average latency benefit
}

// PreAggStats contains statistics from StarRocks
type PreAggStats struct {
	RowCount  int64 `json:"row_count"`
	SizeBytes int64 `json:"size_bytes"`
}

// PreAggSuggestion represents a suggested pre-aggregation from the suggestion engine.
type PreAggSuggestion struct {
	TenantID    string    `json:"tenant_id"`
	Datasource  string    `json:"datasource"`
	Fingerprint string    `json:"fingerprint"` // SQL/semantic fingerprint
	GroupBy     []string  `json:"group_by"`    // Suggested group-by columns
	Filters     []string  `json:"filters"`     // Suggested filter columns
	Measures    []string  `json:"measures"`    // Suggested measures
	AvgLatency  float64   `json:"avg_latency_ms"`
	AvgRows     float64   `json:"avg_rows"`
	Freq        int64     `json:"freq"`  // Query frequency in observation window
	Score       float64   `json:"score"` // Suggestion score (higher = more impactful)
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// Cube Schema types (for dynamic cube generation)

type CubeSchema struct {
	Cubes []CubeDefinition `json:"cubes"`
}

type CubeDefinition struct {
	Name       string                   `json:"name"`
	SQL        string                   `json:"sql"`
	Measures   map[string]CubeMeasure   `json:"measures"`
	Dimensions map[string]CubeDimension `json:"dimensions"`
}

type CubeMeasure struct {
	SQL  string `json:"sql"`
	Type string `json:"type"` // "number", "sum", "avg", etc.
}

type CubeDimension struct {
	SQL  string `json:"sql"`
	Type string `json:"type"` // "string", "time", "number"
}

// Helper to unmarshal PreAggProperties from catalog_node.properties
func ParsePreAggProperties(raw json.RawMessage) (*PreAggProperties, error) {
	var props PreAggProperties
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// Helper to unmarshal PreAggConfig from catalog_node.config
func ParsePreAggConfig(raw json.RawMessage) (*PreAggConfig, error) {
	var cfg PreAggConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

package aso

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Policy Types
// ============================================================================

// ASOPolicy defines optimization behavior for an environment/tenant
type ASOPolicy struct {
	ID       uuid.UUID  `json:"id" db:"id"`
	Env      string     `json:"env" db:"env"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"` // nil = core policy

	// Policy state
	Enabled bool    `json:"enabled" db:"enabled"`
	Mode    ASOMode `json:"mode" db:"mode"`

	// Thresholds and limits
	MaxNewPreAggsPerDay   int     `json:"max_new_preaggs_per_day" db:"max_new_preaggs_per_day"`
	MaxChangesPerDay      int     `json:"max_changes_per_day" db:"max_changes_per_day"`
	MinScoreForNewPreAgg  float64 `json:"min_score_for_new_preagg" db:"min_score_for_new_preagg"`
	MinUsageForRetirement int     `json:"min_usage_for_retirement" db:"min_usage_for_retirement"`
	HotPathThresholdMs    int     `json:"hot_path_threshold_ms" db:"hot_path_threshold_ms"`
	LookbackWindowSeconds int     `json:"lookback_window_seconds" db:"lookback_window_seconds"`

	// Pre-warm settings
	PrewarmEnabled         bool `json:"prewarm_enabled" db:"prewarm_enabled"`
	PrewarmLeadTimeMinutes int  `json:"prewarm_lead_time_minutes" db:"prewarm_lead_time_minutes"`

	// Audit
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	UpdatedBy string    `json:"updated_by" db:"updated_by"`
}

// LookbackWindow returns the lookback window as a duration
func (p *ASOPolicy) LookbackWindow() time.Duration {
	return time.Duration(p.LookbackWindowSeconds) * time.Second
}

// ASOMode defines the optimization behavior
type ASOMode string

const (
	ASOModAdvisory   ASOMode = "advisory"   // Only recommend, never apply
	ASOModeAutoTune  ASOMode = "auto_tune"  // Can tune existing pre-aggs (refresh intervals)
	ASOModeAutoApply ASOMode = "auto_apply" // Can create/tune/retire within policy limits
)

// ============================================================================
// Optimization Types
// ============================================================================

// ASOOptimization represents a proposed or applied optimization
type ASOOptimization struct {
	ID       uuid.UUID  `json:"id" db:"id"`
	Env      string     `json:"env" db:"env"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Scope    ASOScope   `json:"scope" db:"scope"`

	// Optimization type and target
	OptimizationType OptimizationType `json:"optimization_type" db:"optimization_type"`
	TargetType       TargetType       `json:"target_type" db:"target_type"`
	TargetID         uuid.UUID        `json:"target_id" db:"target_id"`
	TargetName       string           `json:"target_name" db:"target_name"`

	// Status
	Status OptimizationStatus `json:"status" db:"status"`
	Mode   string             `json:"mode" db:"mode"` // "advisory" or "auto"

	// Scoring and reasoning
	Score   float64         `json:"score" db:"score"`
	Reason  string          `json:"reason" db:"reason"`
	Details json.RawMessage `json:"details" db:"details"`

	// Workload evidence
	WorkloadWindowDays int      `json:"workload_window_days" db:"workload_window_days"`
	QueriesPerDay      *float64 `json:"queries_per_day,omitempty" db:"queries_per_day"`
	AvgLatencyMs       *float64 `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	P95LatencyMs       *float64 `json:"p95_latency_ms,omitempty" db:"p95_latency_ms"`
	AvgRowsScanned     *int64   `json:"avg_rows_scanned,omitempty" db:"avg_rows_scanned"`

	// Policy reference
	PolicyID *uuid.UUID `json:"policy_id,omitempty" db:"policy_id"`

	// Lifecycle
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	CreatedBy       string     `json:"created_by" db:"created_by"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	ApprovedBy      *string    `json:"approved_by,omitempty" db:"approved_by"`
	AppliedAt       *time.Time `json:"applied_at,omitempty" db:"applied_at"`
	AppliedBy       *string    `json:"applied_by,omitempty" db:"applied_by"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty" db:"rejected_at"`
	RejectedBy      *string    `json:"rejected_by,omitempty" db:"rejected_by"`
	RejectionReason *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`

	// For rollback
	BeforeConfig json.RawMessage `json:"before_config,omitempty" db:"before_config"`
	AfterConfig  json.RawMessage `json:"after_config,omitempty" db:"after_config"`
}

// ASOScope indicates whether optimization targets core or tenant assets
type ASOScope string

const (
	ASOScopeCore   ASOScope = "core"
	ASOScopeTenant ASOScope = "tenant"
)

// OptimizationType defines what kind of optimization this is
type OptimizationType string

const (
	OptTypeTuneRefresh    OptimizationType = "tune_refresh"
	OptTypeTuneDefinition OptimizationType = "tune_definition"
	OptTypeCreatePreAgg   OptimizationType = "create_preagg"
	OptTypeRetireAsset    OptimizationType = "retire_asset"
	OptTypePrewarm        OptimizationType = "prewarm"
)

// TargetType defines what asset type is being optimized
type TargetType string

const (
	TargetTypePreAgg TargetType = "preagg"
	TargetTypeBO     TargetType = "bo"
	TargetTypeCalc   TargetType = "calc"
	TargetTypeTerm   TargetType = "term"
)

// OptimizationStatus tracks the lifecycle of an optimization
type OptimizationStatus string

const (
	OptStatusProposed   OptimizationStatus = "proposed"
	OptStatusApproved   OptimizationStatus = "approved"
	OptStatusApplied    OptimizationStatus = "applied"
	OptStatusRejected   OptimizationStatus = "rejected"
	OptStatusFailed     OptimizationStatus = "failed"
	OptStatusSuperseded OptimizationStatus = "superseded"
)

// ============================================================================
// Optimization Details (per type)
// ============================================================================

// TuneRefreshDetails contains before/after for refresh interval tuning
type TuneRefreshDetails struct {
	CurrentRefreshInterval  string  `json:"current_refresh_interval"`
	ProposedRefreshInterval string  `json:"proposed_refresh_interval"`
	QueriesPerDay           float64 `json:"queries_per_day"`
	AvgLatencyPreAggMs      float64 `json:"avg_latency_preagg_ms"`
	AvgLatencyBaseMs        float64 `json:"avg_latency_base_ms"`
	LastRefreshAge          string  `json:"last_refresh_age"`
	StalenessRisk           string  `json:"staleness_risk"` // low, medium, high
}

// CreatePreAggDetails contains proposal for new pre-agg creation
type CreatePreAggDetails struct {
	BOName   string   `json:"bo_name"`
	Grain    []string `json:"grain"`
	Measures []string `json:"measures"`
	Filters  []string `json:"filters,omitempty"`

	CostEstimate struct {
		EstimatedQueriesPerDay float64 `json:"estimated_queries_per_day"`
		AvgDurationMs          float64 `json:"avg_duration_ms"`
		P95DurationMs          float64 `json:"p95_duration_ms"`
		AvgRowsScanned         int64   `json:"avg_rows_scanned"`
		EstimatedSpeedupFactor float64 `json:"estimated_speedup_factor"`
		EstimatedStorageBytes  int64   `json:"estimated_storage_bytes"`
		EstimatedBuildCost     float64 `json:"estimated_build_cost"`
		EstimatedRefreshCost   float64 `json:"estimated_refresh_cost"`
	} `json:"cost_estimate"`

	CoreVsTenant struct {
		Scope      string `json:"scope"` // core or tenant
		CoreBOID   string `json:"core_bo_id,omitempty"`
		TenantBOID string `json:"tenant_bo_id,omitempty"`
	} `json:"core_vs_tenant"`
}

// RetireAssetDetails contains evidence for asset retirement
type RetireAssetDetails struct {
	QueriesLast30Days int64      `json:"queries_last_30_days"`
	QueriesLast90Days int64      `json:"queries_last_90_days"`
	RefreshCostMs     int64      `json:"refresh_cost_ms"`
	StorageBytes      int64      `json:"storage_bytes"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
}

// PrewarmDetails contains pre-warm schedule proposal
type PrewarmDetails struct {
	CurrentSchedule  *string `json:"current_schedule,omitempty"`
	ProposedSchedule string  `json:"proposed_schedule"` // cron expression
	PeakWindow       struct {
		DayOfWeek string `json:"day_of_week"` // Mon-Fri, etc.
		Time      string `json:"time"`        // 09:00
		Timezone  string `json:"timezone"`    // America/New_York
	} `json:"peak_window"`
	AvgQueriesInPeak float64 `json:"avg_queries_in_peak"`
}

// ============================================================================
// Validation Result
// ============================================================================

// ASOValidationResult is returned during promotion pipeline validation
type ASOValidationResult struct {
	Valid       bool                 `json:"valid"`
	Warnings    []ASOValidationIssue `json:"warnings,omitempty"`
	Errors      []ASOValidationIssue `json:"errors,omitempty"`
	Suggestions []ASOOptimization    `json:"suggestions,omitempty"`
}

// ASOValidationIssue describes a warning or error
type ASOValidationIssue struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	TargetType string `json:"target_type,omitempty"`
	TargetName string `json:"target_name,omitempty"`
	Severity   string `json:"severity"` // warning, error
	Suggestion string `json:"suggestion,omitempty"`
}

// ============================================================================
// Summary Types (for UI)
// ============================================================================

// ASOSummary provides dashboard-level metrics
type ASOSummary struct {
	Env                    string     `json:"env"`
	PolicyEnabled          bool       `json:"policy_enabled"`
	PolicyMode             ASOMode    `json:"policy_mode"`
	OptimizationsToday     int        `json:"optimizations_today"`
	OptimizationsPending   int        `json:"optimizations_pending"`
	OptimizationsApplied7d int        `json:"optimizations_applied_7d"`
	HotPathsDetected       int        `json:"hot_paths_detected"`
	RetirementCandidates   int        `json:"retirement_candidates"`
	LastEvaluatedAt        *time.Time `json:"last_evaluated_at,omitempty"`
}

// ============================================================================
// Daily Stats
// ============================================================================

// ASODailyStats tracks daily optimization activity for rate limiting
type ASODailyStats struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	Env                   string     `json:"env" db:"env"`
	TenantID              *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	StatDate              time.Time  `json:"stat_date" db:"stat_date"`
	PreAggsCreated        int        `json:"preaggs_created" db:"preaggs_created"`
	ChangesApplied        int        `json:"changes_applied" db:"changes_applied"`
	OptimizationsProposed int        `json:"optimizations_proposed" db:"optimizations_proposed"`
	OptimizationsRejected int        `json:"optimizations_rejected" db:"optimizations_rejected"`
}

// ============================================================================
// Workload Profile (for optimization decisions)
// ============================================================================

// WorkloadProfile contains aggregated workload data for a BO/tenant
type WorkloadProfile struct {
	TenantID   string    `json:"tenant_id"`
	BOName     string    `json:"bo_name"`
	BOID       uuid.UUID `json:"bo_id"`
	WindowDays int       `json:"window_days"`

	// Query metrics
	TotalQueries   int64   `json:"total_queries"`
	QueriesPerDay  float64 `json:"queries_per_day"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	P95DurationMs  float64 `json:"p95_duration_ms"`
	AvgRowsScanned int64   `json:"avg_rows_scanned"`

	// Hot paths
	HotGrains   []string `json:"hot_grains"`
	HotMeasures []string `json:"hot_measures"`

	// Pre-agg coverage
	PreAggHitRate     float64 `json:"preagg_hit_rate"`
	PreAggMissRate    float64 `json:"preagg_miss_rate"`
	PreAggMissQueries int64   `json:"preagg_miss_queries"`

	// Time patterns
	PeakHours      []int `json:"peak_hours"`        // Hours of day (0-23)
	PeakDaysOfWeek []int `json:"peak_days_of_week"` // 0=Sun, 6=Sat
}

package ops

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditLogFilters defines filtering options for audit log retrieval
type AuditLogFilters struct {
	UserID     *string
	ActionType *string
	Status     *string
	StartTime  *time.Time
	EndTime    *time.Time
	IncidentID *uuid.UUID
}

// RegionConfig represents a geographic region configuration
type RegionConfig struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RegionCode  string    `json:"region_code" db:"region_code"` // e.g., "us-east-1", "eu-west-1"
	RegionName  string    `json:"region_name" db:"region_name"` // e.g., "US East (N. Virginia)"
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RegionRouting maps a tenant to region-specific service endpoints
type RegionRouting struct {
	ID                uuid.UUID `json:"id" db:"id"`
	TenantID          uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Region            string    `json:"region" db:"region"`
	StarRocksCluster  *string   `json:"starrocks_cluster,omitempty" db:"starrocks_cluster"`
	RedpandaBroker    *string   `json:"redpanda_broker,omitempty" db:"redpanda_broker"`
	TemporalNamespace *string   `json:"temporal_namespace,omitempty" db:"temporal_namespace"`
	OpsWorkerPool     *string   `json:"ops_worker_pool,omitempty" db:"ops_worker_pool"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// RegionalMetrics tracks performance metrics per region
type RegionalMetrics struct {
	ID            uuid.UUID          `json:"id" db:"id"`
	Region        string             `json:"region" db:"region"`
	ErrorRate     float64            `json:"error_rate" db:"error_rate"` // Percentage (0-100)
	P50Latency    int                `json:"p50_latency_ms" db:"p50_latency_ms"`
	P95Latency    int                `json:"p95_latency_ms" db:"p95_latency_ms"`
	P99Latency    int                `json:"p99_latency_ms" db:"p99_latency_ms"`
	Availability  float64            `json:"availability_pct" db:"availability_pct"` // Percentage (0-100)
	RequestCount  int64              `json:"request_count" db:"request_count"`
	IncidentCount int                `json:"incident_count" db:"incident_count"` // Last 24h
	Components    map[string]float64 `json:"components" db:"components"`         // JSON: composite scores
	ComputedAt    time.Time          `json:"computed_at" db:"computed_at"`
	CreatedAt     time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" db:"updated_at"`
}

// RegionalHealth tracks regional health scores
type RegionalHealth struct {
	ID         uuid.UUID          `json:"id" db:"id"`
	Region     string             `json:"region" db:"region"`
	Score      int                `json:"health_score" db:"health_score"` // 0-100
	Status     string             `json:"status" db:"status"`             // "healthy", "degraded", "critical"
	ComputedAt time.Time          `json:"computed_at" db:"computed_at"`
	UpdatedAt  time.Time          `json:"updated_at" db:"updated_at"`
	Components map[string]float64 `json:"components,omitempty"` // Not persisted
}

// RegionalSLA defines Service Level Agreements per region
type RegionalSLA struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Region          string    `json:"region" db:"region"`
	AvailabilitySLA float64   `json:"availability_sla_pct" db:"availability_sla_pct"` // Target availability (e.g., 99.9)
	P95LatencySLA   int       `json:"p95_latency_sla_ms" db:"p95_latency_sla_ms"`     // Target p95 latency in ms
	ErrorRateSLA    float64   `json:"error_rate_sla_pct" db:"error_rate_sla_pct"`     // Max acceptable error rate
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// RegionalSLAStatus tracks SLA compliance
type RegionalSLAStatus struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Region          string    `json:"region" db:"region"`
	SLAID           uuid.UUID `json:"sla_id" db:"sla_id"`
	AvailabilityMet bool      `json:"availability_met" db:"availability_met"`
	LatencyMet      bool      `json:"latency_met" db:"latency_met"`
	ErrorRateMet    bool      `json:"error_rate_met" db:"error_rate_met"`
	CompliancePct   float64   `json:"compliance_pct" db:"compliance_pct"` // 0-100
	CheckedAt       time.Time `json:"checked_at" db:"checked_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Phase 3.9: Region-Aware API types

// RegionalIncidentCount tracks incident count per region in a time window
type RegionalIncidentCount struct {
	Region string `json:"region" db:"region"`
	Count  int    `json:"count" db:"count"`
}

// RegionSummary aggregates regional health, SLA, metrics, and incidents
type RegionSummary struct {
	Region           string    `json:"region"`
	HealthScore      int       `json:"health_score"`
	HealthStatus     string    `json:"health_status"`
	SLACompliance    float64   `json:"sla_compliance"`
	ErrorRate        float64   `json:"error_rate"`
	LatencyP95Ms     float64   `json:"latency_p95_ms"`
	Availability     float64   `json:"availability"`
	IncidentCount24h int       `json:"incident_count_24h"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// RegionDetail provides comprehensive drill-down data for a specific region
type RegionDetail struct {
	Region           string              `json:"region"`
	Metrics          *RegionalMetrics    `json:"metrics"`
	Health           *RegionalHealth     `json:"health"`
	SLA              *RegionalSLA        `json:"sla"`
	SLAStatusHistory []RegionalSLAStatus `json:"sla_status_history"`
	RecentIncidents  []Incident          `json:"recent_incidents"`
	RecentOpsEvents  []Event             `json:"recent_ops_events"`
	RecentActions    []AuditLog          `json:"recent_actions"`
	RecentRCASummary interface{}         `json:"recent_rca_summaries"` // RCAResult from RCA engine
}

// RCASummary is a summary of RCA results (placeholder until RCAResult is formalized)
type RCASummary struct {
	ID                    uuid.UUID `json:"id,omitempty"`
	IncidentID            uuid.UUID `json:"incident_id,omitempty" db:"incident_id"`
	Region                string    `json:"region,omitempty" db:"region"`
	SuspectedRootCause    string    `json:"suspected_root_cause,omitempty"`
	ConfidenceScore       float64   `json:"confidence_score" db:"confidence_score"`
	AffectedServicesCount int       `json:"affected_services_count"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// Phase 3.10: Failover Policies & Automated Regional Failover

// FailoverPolicy defines conditions and targets for automatic regional failover
type FailoverPolicy struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	TenantID           uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name               string    `json:"name" db:"name"`                                 // e.g., "us-east-1 → us-west-2"
	SourceRegion       string    `json:"source_region" db:"source_region"`               // Region that may fail
	TargetRegions      string    `json:"target_regions" db:"target_regions"`             // JSON list of failover targets
	TriggerHealthScore *int      `json:"trigger_health_score" db:"trigger_health_score"` // Health threshold (0-100) to trigger failover
	TriggerErrorRate   *float64  `json:"trigger_error_rate" db:"trigger_error_rate"`     // Error rate % to trigger failover
	TriggerLatency     *int      `json:"trigger_latency_ms" db:"trigger_latency_ms"`     // Latency ms to trigger failover
	IsAutomatic        bool      `json:"is_automatic" db:"is_automatic"`                 // True = automatic, False = manual only
	CooldownMinutes    int       `json:"cooldown_minutes" db:"cooldown_minutes"`         // Prevent thrashing (default 30min)
	Priority           int       `json:"priority" db:"priority"`                         // Lower = higher priority
	IsEnabled          bool      `json:"is_enabled" db:"is_enabled"`                     // Can be disabled
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// FailoverEvent tracks failover incidents and their outcomes
type FailoverEvent struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	IncidentID     uuid.UUID  `json:"incident_id" db:"incident_id"`
	PolicyID       uuid.UUID  `json:"policy_id" db:"policy_id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SourceRegion   string     `json:"source_region" db:"source_region"`
	TargetRegion   string     `json:"target_region" db:"target_region"`
	TriggerReason  string     `json:"trigger_reason" db:"trigger_reason"`   // "health_score", "error_rate", "latency", "manual"
	TriggerValue   float64    `json:"trigger_value" db:"trigger_value"`     // The actual value that triggered
	Status         string     `json:"status" db:"status"`                   // "pending", "in_progress", "success", "failed", "rolled_back"
	RollbackNeeded *bool      `json:"rollback_needed" db:"rollback_needed"` // True if rollback executed
	ErrorMsg       *string    `json:"error_msg" db:"error_msg"`
	TriggeredAt    time.Time  `json:"triggered_at" db:"triggered_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// FailoverMetrics tracks failover success rates and performance
type FailoverMetrics struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	PolicyID        uuid.UUID  `json:"policy_id" db:"policy_id"`
	TotalFailovers  int        `json:"total_failovers" db:"total_failovers"`
	SuccessfulCount int        `json:"successful_count" db:"successful_count"`
	FailedCount     int        `json:"failed_count" db:"failed_count"`
	AvgDurationMs   int64      `json:"avg_duration_ms" db:"avg_duration_ms"`
	LastFailoverAt  *time.Time `json:"last_failover_at" db:"last_failover_at"`
	SuccessRatePct  float64    `json:"success_rate_pct" db:"success_rate_pct"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// ========== Phase 3.11: Failover Chain Orchestration ==========

// FailoverChain defines a sequence of failover policies to execute
// If primary fails, move to next target; if that fails, continue down the chain
type FailoverChain struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	TenantID           uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name               string    `json:"name" db:"name"`                                 // e.g., "us-east-1 primary → us-west-2 → eu-west-1"
	SourceRegion       string    `json:"source_region" db:"source_region"`               // The region that may fail
	ChainTargets       string    `json:"chain_targets" db:"chain_targets"`               // JSON: ordered list of [region1, region2, region3, ...]
	TriggerHealthScore *int      `json:"trigger_health_score" db:"trigger_health_score"` // Condition to initiate chain
	TriggerErrorRate   *float64  `json:"trigger_error_rate" db:"trigger_error_rate"`
	TriggerLatency     *int      `json:"trigger_latency_ms" db:"trigger_latency_ms"`
	MaxChainDepth      int       `json:"max_chain_depth" db:"max_chain_depth"`   // Prevent infinite chains (default 3)
	CooldownMinutes    int       `json:"cooldown_minutes" db:"cooldown_minutes"` // Cooldown between chain steps
	Priority           int       `json:"priority" db:"priority"`                 // Lower = higher priority
	IsEnabled          bool      `json:"is_enabled" db:"is_enabled"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// FailoverChainExecution tracks the execution of a failover chain
type FailoverChainExecution struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ChainID        uuid.UUID  `json:"chain_id" db:"chain_id"`
	IncidentID     uuid.UUID  `json:"incident_id" db:"incident_id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SourceRegion   string     `json:"source_region" db:"source_region"`
	CurrentStep    int        `json:"current_step" db:"current_step"`       // Which target in chain (0-indexed)
	CurrentTarget  string     `json:"current_target" db:"current_target"`   // Current failover target
	PreviousTarget *string    `json:"previous_target" db:"previous_target"` // Where we failed from
	Status         string     `json:"status" db:"status"`                   // "pending", "in_progress", "success", "failed", "exhausted"
	StepsExecuted  []string   `json:"steps_executed" db:"steps_executed"`   // JSON: [region1, region2, ...]
	FailureReasons []string   `json:"failure_reasons" db:"failure_reasons"` // JSON: error messages per step
	TriggeredAt    time.Time  `json:"triggered_at" db:"triggered_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// FailoverChainMetrics aggregates chain execution statistics
type FailoverChainMetrics struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	ChainID             uuid.UUID  `json:"chain_id" db:"chain_id"`
	TotalExecutions     int        `json:"total_executions" db:"total_executions"`
	SuccessfulCount     int        `json:"successful_count" db:"successful_count"`           // Resolved at step 1
	PartialSuccessCount int        `json:"partial_success_count" db:"partial_success_count"` // Resolved at step 2+
	FailedCount         int        `json:"failed_count" db:"failed_count"`                   // All steps exhausted
	AvgStepsNeeded      float64    `json:"avg_steps_needed" db:"avg_steps_needed"`           // Average steps before resolution
	AvgDurationMs       int64      `json:"avg_duration_ms" db:"avg_duration_ms"`
	LastExecutionAt     *time.Time `json:"last_execution_at" db:"last_execution_at"`
	SuccessRatePct      float64    `json:"success_rate_pct" db:"success_rate_pct"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// ========== Phase 3.12: Multi-Tenant & Priority Failover ==========

// FailoverChainState tracks per-tenant chain execution state and cooldown
type FailoverChainState struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	ChainID             uuid.UUID  `json:"chain_id" db:"chain_id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	LastExecutedAt      *time.Time `json:"last_executed_at" db:"last_executed_at"`         // Cooldown: don't execute if < cooldown
	NextEligibleAt      *time.Time `json:"next_eligible_at" db:"next_eligible_at"`         // When chain becomes eligible again
	CurrentStepIndex    int        `json:"current_step_index" db:"current_step_index"`     // Where in cascade we are (0-indexed)
	IsExecuting         bool       `json:"is_executing" db:"is_executing"`                 // Lock: prevent concurrent executions
	ExecutionLockAt     *time.Time `json:"execution_lock_at" db:"execution_lock_at"`       // Timeout for lock
	LastError           *string    `json:"last_error" db:"last_error"`                     // Last failure reason
	ConsecutiveFailures int        `json:"consecutive_failures" db:"consecutive_failures"` // Track failure streak
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// FailoverChainConflict tracks conflicts between chains for same tenant
type FailoverChainConflict struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ChainID1       uuid.UUID  `json:"chain_id_1" db:"chain_id_1"`       // First chain
	ChainID2       uuid.UUID  `json:"chain_id_2" db:"chain_id_2"`       // Second chain (conflicting)
	ConflictType   string     `json:"conflict_type" db:"conflict_type"` // "same_target", "overlapping_targets", "incompatible"
	SourceRegion1  string     `json:"source_region_1" db:"source_region_1"`
	SourceRegion2  string     `json:"source_region_2" db:"source_region_2"`
	SharedTargets  string     `json:"shared_targets" db:"shared_targets"`   // JSON: [region1, region2, ...]
	ResolutionRule string     `json:"resolution_rule" db:"resolution_rule"` // "priority", "first_win", "serial_execute"
	IsResolved     bool       `json:"is_resolved" db:"is_resolved"`
	ResolvedAt     *time.Time `json:"resolved_at" db:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// ChainExecutionMetricsAdvanced tracks percentile metrics for SLA compliance
type ChainExecutionMetricsAdvanced struct {
	ID                uuid.UUID `json:"id" db:"id"`
	ChainID           uuid.UUID `json:"chain_id" db:"chain_id"`
	TotalExecutions   int       `json:"total_executions" db:"total_executions"`
	P50DurationMs     int64     `json:"p50_duration_ms" db:"p50_duration_ms"` // 50th percentile
	P75DurationMs     int64     `json:"p75_duration_ms" db:"p75_duration_ms"` // 75th percentile
	P95DurationMs     int64     `json:"p95_duration_ms" db:"p95_duration_ms"` // 95th percentile
	P99DurationMs     int64     `json:"p99_duration_ms" db:"p99_duration_ms"` // 99th percentile
	MaxDurationMs     int64     `json:"max_duration_ms" db:"max_duration_ms"`
	MinDurationMs     int64     `json:"min_duration_ms" db:"min_duration_ms"`
	StdDevDurationMs  float64   `json:"std_dev_duration_ms" db:"std_dev_duration_ms"` // Standard deviation
	SuccessRate99th   float64   `json:"success_rate_99th" db:"success_rate_99th"`     // Over 99 executions
	AvgStepsNeeded    float64   `json:"avg_steps_needed" db:"avg_steps_needed"`
	P95StepsNeeded    int       `json:"p95_steps_needed" db:"p95_steps_needed"`       // 95th percentile steps
	MostCommonFailure *string   `json:"most_common_failure" db:"most_common_failure"` // Most frequent error
	SLACompliance     float64   `json:"sla_compliance" db:"sla_compliance"`           // % execution within SLA target
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// ChainPriorityExecution tracks priority-based chain execution queue
type ChainPriorityExecution struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	IncidentID      uuid.UUID  `json:"incident_id" db:"incident_id"`
	ChainsToExecute string     `json:"chains_to_execute" db:"chains_to_execute"` // JSON: [{id, priority, order}, ...]
	ExecutionOrder  string     `json:"execution_order" db:"execution_order"`     // JSON: [chain_id_1, chain_id_2, ...]
	CurrentChainIdx int        `json:"current_chain_idx" db:"current_chain_idx"` // Index in execution order
	Status          string     `json:"status" db:"status"`                       // "pending", "in_progress", "completed", "failed"
	CompletedChains string     `json:"completed_chains" db:"completed_chains"`   // JSON: [chain_id, ...]
	FailedChains    string     `json:"failed_chains" db:"failed_chains"`         // JSON: [chain_id, ...]
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// Phase 3.14: Analytics & Trends Types

// SLAComplianceTrend tracks SLA compliance metrics over time
type SLAComplianceTrend struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ChainID          uuid.UUID `json:"chain_id" db:"chain_id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ComplianceScore  float64   `json:"compliance_score" db:"compliance_score"`     // 0-100
	SuccessRateTrend float64   `json:"success_rate_trend" db:"success_rate_trend"` // % change from previous period
	LatencyTrend     float64   `json:"latency_trend" db:"latency_trend"`           // % change in P95 latency
	Percentile99     float64   `json:"percentile_99" db:"percentile_99"`           // 99th percentile compliance
	Status           string    `json:"status" db:"status"`                         // "improving", "stable", "degrading"
	ReportedAt       time.Time `json:"reported_at" db:"reported_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ConflictResolutionTrend tracks conflict resolution patterns
type ConflictResolutionTrend struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	TotalConflicts  int       `json:"total_conflicts" db:"total_conflicts"`     // Over period
	ResolvedCount   int       `json:"resolved_count" db:"resolved_count"`       // Successfully resolved
	FailedCount     int       `json:"failed_count" db:"failed_count"`           // Resolution failed
	ResolutionRate  float64   `json:"resolution_rate" db:"resolution_rate"`     // % resolved
	AvgResolutionMs int64     `json:"avg_resolution_ms" db:"avg_resolution_ms"` // Time to resolve
	MostCommonRule  string    `json:"most_common_rule" db:"most_common_rule"`   // "priority", "first_win", etc.
	PeriodStart     time.Time `json:"period_start" db:"period_start"`
	PeriodEnd       time.Time `json:"period_end" db:"period_end"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ChainExecutionStats aggregates execution statistics
type ChainExecutionStats struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	ChainID              uuid.UUID  `json:"chain_id" db:"chain_id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	TotalExecutions      int        `json:"total_executions" db:"total_executions"`
	SuccessfulExecutions int        `json:"successful_executions" db:"successful_executions"`
	FailedExecutions     int        `json:"failed_executions" db:"failed_executions"`
	SuccessRatePct       float64    `json:"success_rate_pct" db:"success_rate_pct"` // 0-100
	AvgExecutionMs       int64      `json:"avg_execution_ms" db:"avg_execution_ms"`
	MaxExecutionMs       int64      `json:"max_execution_ms" db:"max_execution_ms"`
	MinExecutionMs       int64      `json:"min_execution_ms" db:"min_execution_ms"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty" db:"last_success_at"`
	LastFailureAt        *time.Time `json:"last_failure_at,omitempty" db:"last_failure_at"`
	PeriodStart          time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time  `json:"period_end" db:"period_end"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// ChainFilterCriteria represents advanced search/filtering options
type ChainFilterCriteria struct {
	TenantID         *uuid.UUID
	SourceRegion     *string
	MinSLACompliance *float64 // 0-100
	MaxSLACompliance *float64
	Status           *string // "pending", "in_progress", "completed", "failed"
	IsEnabled        *bool
	MinSuccessRate   *float64
	MinP95LatencyMs  *int64
	HasConflicts     *bool
	CreatedAfter     *time.Time
	CreatedBefore    *time.Time
	SortBy           string // "sla_compliance", "success_rate", "created_at"
	SortOrder        string // "asc", "desc"
	Limit            int
	Offset           int
}

// BatchConflictResolution represents batch conflict resolution operation
type BatchConflictResolution struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ConflictIDs    string     `json:"conflict_ids" db:"conflict_ids"`       // JSON: [id1, id2, ...]
	ResolutionRule string     `json:"resolution_rule" db:"resolution_rule"` // Apply to all
	Status         string     `json:"status" db:"status"`                   // "pending", "in_progress", "completed", "failed"
	TotalConflicts int        `json:"total_conflicts" db:"total_conflicts"`
	ResolvedCount  int        `json:"resolved_count" db:"resolved_count"`
	FailedCount    int        `json:"failed_count" db:"failed_count"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty" db:"executed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// ChainHealthReport aggregates chain health for monitoring
type ChainHealthReport struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	ChainID             uuid.UUID `json:"chain_id" db:"chain_id"`
	TenantID            uuid.UUID `json:"tenant_id" db:"tenant_id"`
	OverallHealth       int       `json:"overall_health" db:"overall_health"`               // 0-100
	LastExecutionStatus string    `json:"last_execution_status" db:"last_execution_status"` // "success", "failure", "running"
	ConsecutiveFailures int       `json:"consecutive_failures" db:"consecutive_failures"`
	IsHealthy           bool      `json:"is_healthy" db:"is_healthy"`
	RecommendedAction   string    `json:"recommended_action" db:"recommended_action"` // "investigate", "retry", "disable", "none"
	ReportedAt          time.Time `json:"reported_at" db:"reported_at"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// Store defines all data access operations for the ops system
type Store interface {
	// Alerts
	ListAlerts(ctx context.Context, enabled *bool) ([]Alert, error)
	GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error)
	CreateAlert(ctx context.Context, alert Alert) (*Alert, error)
	UpdateAlert(ctx context.Context, id uuid.UUID, alert Alert) error
	DeleteAlert(ctx context.Context, id uuid.UUID) error
	InsertAlertEvent(ctx context.Context, event AlertEvent) error
	GetAlertEvents(ctx context.Context, alertID uuid.UUID, limit int) ([]AlertEvent, error)

	// Error Fingerprints
	GetOrCreateFingerprint(ctx context.Context, fingerprint, path string, statusCode int, sample string) (*ErrorFingerprint, error)
	UpdateFingerprintCount(ctx context.Context, fingerprintID uuid.UUID, newCount int64) error
	InsertErrorEvent(ctx context.Context, event ErrorEvent) error
	ListFingerprints(ctx context.Context, limit int) ([]ErrorFingerprint, error)
	GetFingerprintEvents(ctx context.Context, fingerprintID uuid.UUID, limit int) ([]ErrorEvent, error)

	// Health Scores
	UpsertTenantHealth(ctx context.Context, health TenantHealth) error
	GetTenantHealth(ctx context.Context, tenantID uuid.UUID) (*TenantHealth, error)
	GetTenantHealths(ctx context.Context, limit int) ([]TenantHealth, error)

	UpsertEndpointHealth(ctx context.Context, health EndpointHealth) error
	GetEndpointHealth(ctx context.Context, endpoint string) (*EndpointHealth, error)
	GetEndpointHealths(ctx context.Context, limit int) ([]EndpointHealth, error)

	// Heatmap data
	InsertHeatmapBucket(ctx context.Context, bucketTime time.Time, dimensionType, dimensionValue string, p50, p95, p99 int, requestCount int) error
	GetHeatmapData(ctx context.Context, dimensionType, dimensionValue string, bucketSize time.Duration, window time.Duration) ([]HeatmapSeriesPoint, error)
	GetHeatmapSeries(ctx context.Context, dimensionType string, limit int, bucketSize time.Duration, window time.Duration) ([]HeatmapSeries, error)

	// Metrics (for evaluating alerts)
	GetMetricValue(ctx context.Context, metric, scope string, since time.Time) (float64, error)
	GetTenantMetrics(ctx context.Context, tenantID uuid.UUID, since time.Time) (*TenantMetrics, error)
	GetEndpointMetrics(ctx context.Context, endpoint string, since time.Time) (*EndpointMetrics, error)
	GetGlobalMetrics(ctx context.Context, since time.Time) (*TenantMetrics, error)

	// Timeline and Incident Management
	InsertEvent(ctx context.Context, e Event) error
	ListEvents(ctx context.Context, since time.Time, limit int) ([]Event, error)
	GetIncident(ctx context.Context, id uuid.UUID) (*Incident, []Event, error)
	UpsertIncidentForEvent(ctx context.Context, e Event) (*Incident, error)
	CloseIncident(ctx context.Context, id uuid.UUID, summary, rootCause *string) error

	// Action History
	InsertActionHistory(ctx context.Context, history ActionHistory) error
	UpdateActionHistory(ctx context.Context, id uuid.UUID, status string, result []byte, errorMsg *string) error
	GetActionHistory(ctx context.Context, id uuid.UUID) (*ActionHistory, error)
	ListIncidentActions(ctx context.Context, incidentID uuid.UUID, limit int) ([]ActionHistory, error)

	// Audit Log (Phase 2.4c)
	InsertAuditLog(ctx context.Context, auditLog *AuditLog) error
	GetAuditLog(ctx context.Context, id uuid.UUID) (*AuditLog, error)
	ListAuditLogs(ctx context.Context, filters AuditLogFilters, limit int, offset int) ([]AuditLog, error)
	ListIncidentAuditLogs(ctx context.Context, incidentID uuid.UUID, limit int) ([]AuditLog, error)

	// Region Metadata (Phase 3.1)
	GetRegionConfig(ctx context.Context, regionCode string) (*RegionConfig, error)
	ListRegionConfigs(ctx context.Context, activeOnly bool) ([]RegionConfig, error)
	InsertRegionRouting(ctx context.Context, routing *RegionRouting) error
	GetRegionRouting(ctx context.Context, tenantID uuid.UUID, region string) (*RegionRouting, error)
	ListRegionRoutings(ctx context.Context, tenantID uuid.UUID) ([]RegionRouting, error)

	// Regional Metrics & SLA (Phase 3.5)
	UpsertRegionalMetrics(ctx context.Context, metrics *RegionalMetrics) error
	GetRegionalMetrics(ctx context.Context, region string) (*RegionalMetrics, error)
	ListRegionalMetrics(ctx context.Context, limit int) ([]RegionalMetrics, error)

	UpsertRegionalHealth(ctx context.Context, health *RegionalHealth) error
	GetRegionalHealth(ctx context.Context, region string) (*RegionalHealth, error)
	ListRegionalHealth(ctx context.Context, limit int) ([]RegionalHealth, error)

	UpsertRegionalSLA(ctx context.Context, sla *RegionalSLA) error
	GetRegionalSLA(ctx context.Context, region string) (*RegionalSLA, error)
	ListRegionalSLAs(ctx context.Context, limit int) ([]RegionalSLA, error)

	InsertRegionalSLAStatus(ctx context.Context, status *RegionalSLAStatus) error
	GetRegionalSLAStatus(ctx context.Context, region string) (*RegionalSLAStatus, error)
	ListRegionalSLAStatuses(ctx context.Context, region string, limit int) ([]RegionalSLAStatus, error)

	// Phase 3.9: Region-Aware API Layer
	ListIncidents(ctx context.Context, limit int) ([]Incident, error)
	ListIncidentsByRegion(ctx context.Context, region string, limit int) ([]Incident, error)
	ListLatestRegionalSLAStatuses(ctx context.Context) ([]RegionalSLAStatus, error)
	ListRegionalIncidentCounts(ctx context.Context, since, until time.Time) ([]RegionalIncidentCount, error)
	ListOpsEventsByRegion(ctx context.Context, region string, limit int) ([]Event, error)
	ListAuditLogsByRegion(ctx context.Context, region string, limit int) ([]AuditLog, error)

	// Phase 3.10: Failover Policies & Automated Regional Failover
	InsertFailoverPolicy(ctx context.Context, policy *FailoverPolicy) error
	GetFailoverPolicy(ctx context.Context, id uuid.UUID) (*FailoverPolicy, error)
	ListFailoverPolicies(ctx context.Context, tenantID uuid.UUID) ([]FailoverPolicy, error)
	UpdateFailoverPolicy(ctx context.Context, id uuid.UUID, policy *FailoverPolicy) error
	DeleteFailoverPolicy(ctx context.Context, id uuid.UUID) error

	InsertFailoverEvent(ctx context.Context, event *FailoverEvent) error
	UpdateFailoverEvent(ctx context.Context, id uuid.UUID, status string, errorMsg *string, completedAt *time.Time) error
	ListFailoverEvents(ctx context.Context, policyID uuid.UUID, limit int) ([]FailoverEvent, error)
	ListIncidentFailoverEvents(ctx context.Context, incidentID uuid.UUID) ([]FailoverEvent, error)

	UpsertFailoverMetrics(ctx context.Context, metrics *FailoverMetrics) error
	GetFailoverMetrics(ctx context.Context, policyID uuid.UUID) (*FailoverMetrics, error)

	// Phase 3.11: Failover Chain Orchestration
	InsertFailoverChain(ctx context.Context, chain *FailoverChain) error
	GetFailoverChain(ctx context.Context, id uuid.UUID) (*FailoverChain, error)
	ListFailoverChains(ctx context.Context, tenantID uuid.UUID) ([]FailoverChain, error)
	UpdateFailoverChain(ctx context.Context, id uuid.UUID, chain *FailoverChain) error
	DeleteFailoverChain(ctx context.Context, id uuid.UUID) error

	InsertFailoverChainExecution(ctx context.Context, execution *FailoverChainExecution) error
	UpdateFailoverChainExecution(ctx context.Context, id uuid.UUID, status string, stepsExecuted []string, failureReasons []string, completedAt *time.Time) error
	ListFailoverChainExecutions(ctx context.Context, chainID uuid.UUID, limit int) ([]FailoverChainExecution, error)
	ListIncidentChainExecutions(ctx context.Context, incidentID uuid.UUID) ([]FailoverChainExecution, error)

	UpsertFailoverChainMetrics(ctx context.Context, metrics *FailoverChainMetrics) error
	GetFailoverChainMetrics(ctx context.Context, chainID uuid.UUID) (*FailoverChainMetrics, error)

	// Phase 3.12: Multi-Tenant & Priority Failover
	// FailoverChainState (multi-tenant isolation)
	InsertFailoverChainState(ctx context.Context, state *FailoverChainState) error
	UpdateFailoverChainState(ctx context.Context, id uuid.UUID, state *FailoverChainState) error
	GetFailoverChainState(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) (*FailoverChainState, error)
	ListFailoverChainStates(ctx context.Context, tenantID uuid.UUID) ([]FailoverChainState, error)
	LockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID, lockDurationMs int) error
	UnlockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) error

	// FailoverChainConflict (cross-chain coordination)
	InsertFailoverChainConflict(ctx context.Context, conflict *FailoverChainConflict) error
	ListFailoverChainConflicts(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]FailoverChainConflict, error)
	UpdateConflictResolution(ctx context.Context, conflictID uuid.UUID, resolved bool, rule string) error
	GetConflictingChains(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]uuid.UUID, error)

	// ChainExecutionMetricsAdvanced (SLA tracking)
	UpsertChainExecutionMetricsAdvanced(ctx context.Context, metrics *ChainExecutionMetricsAdvanced) error
	GetChainExecutionMetricsAdvanced(ctx context.Context, chainID uuid.UUID) (*ChainExecutionMetricsAdvanced, error)
	ListChainsSortedBySLACompliance(ctx context.Context, tenantID uuid.UUID) ([]ChainExecutionMetricsAdvanced, error)

	// ChainPriorityExecution (queue management)
	InsertChainPriorityExecution(ctx context.Context, execution *ChainPriorityExecution) error
	UpdateChainPriorityExecution(ctx context.Context, id uuid.UUID, currentIdx int, status string, completedChains []string, failedChains []string) error
	GetChainPriorityExecution(ctx context.Context, id uuid.UUID) (*ChainPriorityExecution, error)
	ListPendingChainQueues(ctx context.Context, tenantID uuid.UUID) ([]ChainPriorityExecution, error)

	// Phase 3.14: Analytics & Trends (8 methods)
	// SLA Compliance Trends
	UpsertSLAComplianceTrend(ctx context.Context, trend *SLAComplianceTrend) error
	ListSLAComplianceTrends(ctx context.Context, tenantID uuid.UUID, limit int) ([]SLAComplianceTrend, error)

	// Conflict Resolution Trends
	UpsertConflictResolutionTrend(ctx context.Context, trend *ConflictResolutionTrend) error
	GetConflictResolutionTrend(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*ConflictResolutionTrend, error)

	// Chain Execution Stats
	UpsertChainExecutionStats(ctx context.Context, stats *ChainExecutionStats) error
	GetChainExecutionStats(ctx context.Context, chainID uuid.UUID) (*ChainExecutionStats, error)

	// Chain Health Reports
	UpsertChainHealthReport(ctx context.Context, report *ChainHealthReport) error
	GetChainHealthReport(ctx context.Context, chainID uuid.UUID) (*ChainHealthReport, error)

	// Advanced Filtering & Search
	ListChainsByFilter(ctx context.Context, criteria *ChainFilterCriteria) ([]FailoverChain, error)
	SearchChains(ctx context.Context, tenantID uuid.UUID, searchTerm string, limit int) ([]FailoverChain, error)

	// Batch Operations
	InsertBatchConflictResolution(ctx context.Context, batch *BatchConflictResolution) error
	UpdateBatchConflictResolution(ctx context.Context, id uuid.UUID, resolvedCount int, failedCount int, status string) error
	GetBatchConflictResolution(ctx context.Context, id uuid.UUID) (*BatchConflictResolution, error)
}

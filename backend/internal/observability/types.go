package observability

import (
	"time"

	"github.com/google/uuid"
)

// Metric represents a single metric data point
type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	TenantID  uuid.UUID         `json:"tenant_id"`
}

// TimeSeries represents a series of metric values over time
type TimeSeries struct {
	Name       string            `json:"name"`
	Labels     map[string]string `json:"labels"`
	DataPoints []DataPoint       `json:"data_points"`
}

// DataPoint represents a single time-value pair
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricQuery represents a query for metrics
type MetricQuery struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels,omitempty"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Step      string            `json:"step"`                // e.g., "1m", "5m", "1h"
	Aggregate string            `json:"aggregate,omitempty"` // avg, sum, min, max, count
}

// SLO represents a Service Level Objective
type SLO struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	TenantID    uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Target      float64     `json:"target" db:"target"`      // e.g., 99.9
	Window      string      `json:"window" db:"time_window"` // e.g., "7d", "30d"
	MetricQuery string      `json:"metric_query" db:"metric_query"`
	AlertRules  []AlertRule `json:"alert_rules,omitempty"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// SLOStatus represents the current status of an SLO
type SLOStatus struct {
	SLOID           uuid.UUID `json:"slo_id"`
	SLOName         string    `json:"slo_name"`
	CurrentValue    float64   `json:"current_value"`
	Target          float64   `json:"target"`
	BudgetTotal     float64   `json:"budget_total"`
	BudgetConsumed  float64   `json:"budget_consumed"`
	BudgetRemaining float64   `json:"budget_remaining"`
	Status          string    `json:"status"` // healthy, degraded, breached
	WindowStart     time.Time `json:"window_start"`
	WindowEnd       time.Time `json:"window_end"`
	LastEvaluated   time.Time `json:"last_evaluated"`
}

// AlertRule defines when to trigger alerts for an SLO
type AlertRule struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	SLOID     uuid.UUID `json:"slo_id" db:"slo_id"`
	Name      string    `json:"name" db:"name"`
	Condition string    `json:"condition" db:"condition"` // e.g., "error_rate > 0.01"
	Severity  string    `json:"severity" db:"severity"`   // critical, warning, info
	Channels  []string  `json:"channels"`                 // slack, email, webhook
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Alert represents a fired alert instance
type Alert struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	TenantID   uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	RuleID     uuid.UUID  `json:"rule_id" db:"rule_id"`
	SLOID      uuid.UUID  `json:"slo_id" db:"slo_id"`
	Severity   string     `json:"severity" db:"severity"`
	Message    string     `json:"message" db:"message"`
	Status     string     `json:"status" db:"status"` // firing, resolved
	Value      float64    `json:"value"`              // The value that triggered the alert
	Threshold  float64    `json:"threshold"`          // The threshold that was breached
	FiredAt    time.Time  `json:"fired_at" db:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// AlertChannel represents a notification channel
type AlertChannelConfig struct {
	Type   string                 `json:"type"` // slack, email, webhook, pagerduty
	Config map[string]interface{} `json:"config"`
}

// DashboardData represents data for the observability dashboard
type DashboardData struct {
	SLOStatuses    []SLOStatus    `json:"slo_statuses"`
	ActiveAlerts   []Alert        `json:"active_alerts"`
	MetricsSummary MetricsSummary `json:"metrics_summary"`
	RecentEvents   []Event        `json:"recent_events"`
	SystemHealth   SystemHealth   `json:"system_health"`
}

// MetricsSummary contains aggregated metric statistics
type MetricsSummary struct {
	TotalQueries      int64   `json:"total_queries"`
	AvgQueryLatency   float64 `json:"avg_query_latency_ms"`
	P95QueryLatency   float64 `json:"p95_query_latency_ms"`
	P99QueryLatency   float64 `json:"p99_query_latency_ms"`
	ErrorRate         float64 `json:"error_rate"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
	ActiveConnections int     `json:"active_connections"`
	RecentEventCount  int     `json:"recent_event_count"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status          string    `json:"status"` // healthy, degraded, critical
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	DiskUsage       float64   `json:"disk_usage"`
	ActiveQueries   int       `json:"active_queries"`
	QueueDepth      int       `json:"queue_depth"`
	LastHealthCheck time.Time `json:"last_health_check"`
}

// Event represents a system event for the activity feed
type Event struct {
	ID        uuid.UUID         `json:"id"`
	TenantID  uuid.UUID         `json:"tenant_id"`
	Type      string            `json:"type"` // alert_fired, slo_breached, deployment, config_change
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// StandardMetrics defines commonly tracked metrics
var StandardMetrics = []string{
	"query_latency_ms",
	"query_count",
	"query_error_count",
	"cache_hit_count",
	"cache_miss_count",
	"preagg_hit_count",
	"active_connections",
	"queue_depth",
	"cpu_usage",
	"memory_usage",
	"disk_usage",
	"slo_budget_remaining",
}

// SeverityLevel represents alert severity
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityWarning  SeverityLevel = "warning"
	SeverityInfo     SeverityLevel = "info"
)

// SLOStatusLevel represents SLO health status
type SLOStatusLevel string

const (
	SLOHealthy  SLOStatusLevel = "healthy"
	SLODegraded SLOStatusLevel = "degraded"
	SLOBreached SLOStatusLevel = "breached"
)

// AlertStatus represents alert state
type AlertStatus string

const (
	AlertFiring   AlertStatus = "firing"
	AlertResolved AlertStatus = "resolved"
)

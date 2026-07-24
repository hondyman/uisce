package ops

import (
	"time"

	"github.com/google/uuid"
)

// Alert represents an alerting rule
type Alert struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Scope      string    `json:"scope"` // "global", "tenant", "endpoint"
	Metric     string    `json:"metric"`
	Threshold  float64   `json:"threshold"`
	Comparison string    `json:"comparison"` // ">" or "<" or ">=" or "<="
	WindowSecs int       `json:"window_secs"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AlertEvent represents a triggered alert
type AlertEvent struct {
	ID          uuid.UUID  `json:"id"`
	AlertID     uuid.UUID  `json:"alert_id"`
	ScopeID     *uuid.UUID `json:"scope_id,omitempty"`
	Endpoint    *string    `json:"endpoint,omitempty"`
	Value       float64    `json:"value"`
	TriggeredAt time.Time  `json:"triggered_at"`
}

// TenantHealth represents a tenant's health score
type TenantHealth struct {
	TenantID   uuid.UUID          `json:"tenant_id"`
	Score      int                `json:"health_score"`
	Components map[string]float64 `json:"components"`
	ComputedAt time.Time          `json:"computed_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// EndpointHealth represents an endpoint's health score
type EndpointHealth struct {
	Endpoint   string             `json:"endpoint"`
	Score      int                `json:"health_score"`
	ErrorRate  float64            `json:"error_rate"`
	P95MS      int                `json:"p95_ms"`
	Requests1H int64              `json:"requests_1h"`
	ComputedAt time.Time          `json:"computed_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Components map[string]float64 `json:"components,omitempty"`
}

// HeatmapSeriesPoint represents a data point in a heatmap series
type HeatmapSeriesPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
	P95MS int       `json:"p95_ms,omitempty"`
	P99MS int       `json:"p99_ms,omitempty"`
}

// HeatmapSeries represents a single series in a heatmap
type HeatmapSeries struct {
	Key    string               `json:"key"` // region, tenant_id, endpoint, etc
	Values []HeatmapSeriesPoint `json:"values"`
}

// Heatmap represents latency data grouped by dimension and time
type Heatmap struct {
	Buckets []time.Time     `json:"buckets"`
	Series  []HeatmapSeries `json:"series"`
}

// ErrorFingerprint represents a grouped set of similar errors
type ErrorFingerprint struct {
	ID            uuid.UUID `json:"id"`
	Fingerprint   string    `json:"fingerprint"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	SampleMessage string    `json:"sample_message"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	Count         int64     `json:"count"`
	CreatedAt     time.Time `json:"created_at"`
}

// ErrorEvent represents a single error occurrence
type ErrorEvent struct {
	ID            uuid.UUID  `json:"id"`
	FingerprintID uuid.UUID  `json:"fingerprint_id"`
	TenantID      *uuid.UUID `json:"tenant_id,omitempty"`
	Endpoint      string     `json:"endpoint"`
	StatusCode    int        `json:"status_code"`
	Message       string     `json:"message"`
	RequestID     string     `json:"request_id,omitempty"`
	OccurredAt    time.Time  `json:"occurred_at"`
}

// MetricValue represents a computed metric
type MetricValue struct {
	Metric     string        `json:"metric"`
	Scope      string        `json:"scope"`
	ScopeID    *uuid.UUID    `json:"scope_id,omitempty"`
	Value      float64       `json:"value"`
	Window     time.Duration `json:"window"`
	ComputedAt time.Time     `json:"computed_at"`
}

// TenantMetrics holds all metrics for a tenant
type TenantMetrics struct {
	TenantID        uuid.UUID
	ErrorRate       float64
	P50             int
	P95             int
	P99             int
	Requests        int64
	RateLimited     int64
	AvailabilityPct float64
}

// EndpointMetrics holds all metrics for an endpoint
type EndpointMetrics struct {
	Path        string
	ErrorRate   float64
	P50         int
	P95         int
	P99         int
	Requests    int64
	StatusCodes map[int]int64
}

// HealthStatus is a categorical health status
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusCritical HealthStatus = "critical"
)

const (
	EVENTTYPE_CPU_SPIKE     = "cpu_spike"
	EVENTTYPE_QUERY_TIMEOUT = "query_timeout"
	SEVERITY_HIGH           = "high"
)

// StatusFromHealth converts a health score to a status
func StatusFromHealth(score int) HealthStatus {
	if score >= 80 {
		return HealthStatusHealthy
	}
	if score >= 50 {
		return HealthStatusDegraded
	}
	return HealthStatusCritical
}

// CreateErrorInput is the input for recording an error
type CreateErrorInput struct {
	TenantID   *uuid.UUID
	Path       string
	StatusCode int
	Message    string
	RequestID  string
}

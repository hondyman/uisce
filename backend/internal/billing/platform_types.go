package billing

import (
	"time"
)

// ========================
// Cost Model (configurable)
// ========================

// CostModel defines per-unit pricing for the platform
type CostModel struct {
	CostPerComputeMs    float64 `json:"costPerComputeMs"`    // $ per ms of compute
	CostPerGBMonth      float64 `json:"costPerGBMonth"`      // $ per GB-month of storage
	CostPerEvent        float64 `json:"costPerEvent"`        // $ per event published
	CostPerEventOverage float64 `json:"costPerEventOverage"` // $ per event above quota
	CostPerMsOverSLO    float64 `json:"costPerMsOverSLO"`    // $ per ms above SLO p95 threshold
}

// DefaultCostModel returns sensible defaults
func DefaultCostModel() CostModel {
	return CostModel{
		CostPerComputeMs:    0.00001,  // $0.01 per 1,000ms
		CostPerGBMonth:      0.12,     // $0.12 per GB-month
		CostPerEvent:        0.000001, // $1 per 1M events
		CostPerEventOverage: 0.000002, // $2 per 1M overage events
		CostPerMsOverSLO:    0.00005,  // penalty for exceeding SLO
	}
}

// ========================
// Tenant Usage & Billing
// ========================

// TenantUsage represents aggregated usage for a tenant within a window
type TenantUsage struct {
	EventsPublished int64           `json:"eventsPublished"`
	Commits         int64           `json:"commits"`
	S3Validations   int64           `json:"s3Validations"`
	IdempotencyHits int64           `json:"idempotencyHits"`
	ComputeMs       ComputeLatency  `json:"computeMs"`
	Storage         StorageUsage    `json:"storage"`
	Regions         []RegionUsage   `json:"regions"`
	Tables          []TableUsage    `json:"tables"`
	QuotaUsage      *QuotaUsageInfo `json:"quotaUsage,omitempty"`
}

// ComputeLatency provides latency breakdown
type ComputeLatency struct {
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Total float64 `json:"total"` // total ms consumed
}

// StorageUsage summarises storage consumption
type StorageUsage struct {
	SnapshotCount int   `json:"snapshotCount"`
	TotalBytes    int64 `json:"totalBytes"`
}

// RegionUsage shows usage broken down by region
type RegionUsage struct {
	Region    string  `json:"region"`
	Commits   int64   `json:"commits"`
	ComputeMs float64 `json:"computeMs"`
}

// TableUsage shows usage per Iceberg table
type TableUsage struct {
	Table        string `json:"table"`
	Commits      int64  `json:"commits"`
	StorageBytes int64  `json:"storageBytes"`
}

// QuotaUsageInfo expresses current usage vs limits
type QuotaUsageInfo struct {
	EventsPerMinute     float64 `json:"eventsPerMinute"`
	EventsPerMinuteMax  float64 `json:"eventsPerMinuteMax"`
	CommitsPerMinute    float64 `json:"commitsPerMinute"`
	CommitsPerMinuteMax float64 `json:"commitsPerMinuteMax"`
}

// ========================
// Cost Breakdown
// ========================

// TenantCostBreakdown computes estimated costs from usage
type TenantCostBreakdown struct {
	ComputeUSD   float64 `json:"computeUSD"`
	StorageUSD   float64 `json:"storageUSD"`
	EventsUSD    float64 `json:"eventsUSD"`
	OverageUSD   float64 `json:"overageUSD"`
	SLOBreachUSD float64 `json:"sloBreachUSD"`
	TotalUSD     float64 `json:"totalUSD"`
}

// ========================
// Tenant Billing Response
// ========================

// TenantBillingResponse is the API response for tenant billing
type TenantBillingResponse struct {
	TenantID      string              `json:"tenantId"`
	Window        string              `json:"window"`
	Usage         TenantUsage         `json:"usage"`
	EstimatedCost TenantCostBreakdown `json:"estimatedCost"`
}

// ========================
// Platform Billing
// ========================

// PlatformTotals represents aggregate platform-level costs
type PlatformTotals struct {
	ComputeUSD float64 `json:"computeUSD"`
	StorageUSD float64 `json:"storageUSD"`
	EventsUSD  float64 `json:"eventsUSD"`
	TotalUSD   float64 `json:"totalUSD"`
}

// RegionCost shows cost attributed to a region
type RegionCost struct {
	Region   string  `json:"region"`
	TotalUSD float64 `json:"totalUSD"`
}

// TenantCost shows cost attributed to a tenant
type TenantCost struct {
	TenantID string  `json:"tenantId"`
	TotalUSD float64 `json:"totalUSD"`
}

// PlatformBillingResponse is the API response for platform billing
type PlatformBillingResponse struct {
	Window        string         `json:"window"`
	Totals        PlatformTotals `json:"totals"`
	ByRegion      []RegionCost   `json:"byRegion"`
	ByTenant      []TenantCost   `json:"byTenant"`
	TopTenants    []TenantCost   `json:"topTenants"`
	BottomTenants []TenantCost   `json:"bottomTenants"`
}

// ========================
// Anomaly Detection
// ========================

// BillingAnomaly describes a detected cost anomaly
type BillingAnomaly struct {
	Type      string  `json:"type"`     // tenant, region, cost
	Key       string  `json:"key"`      // tenant_id, region name, etc.
	Severity  string  `json:"severity"` // low, medium, high, critical
	Ratio     float64 `json:"ratio"`    // short / long ratio
	Reason    string  `json:"reason"`
	Timestamp string  `json:"timestamp"`
}

// BillingAnomalyResponse wraps all anomalies
type BillingAnomalyResponse struct {
	TenantAnomalies []BillingAnomaly `json:"tenantAnomalies"`
	RegionAnomalies []BillingAnomaly `json:"regionAnomalies"`
	CostAnomalies   []BillingAnomaly `json:"costAnomalies"`
}

// ========================
// Forecasting
// ========================

// BillingForecast provides forecasted costs
type BillingForecast struct {
	ForecastUSD float64 `json:"forecastUSD"`
	Model       string  `json:"model"`      // "linear" or "holt-winters"
	Confidence  float64 `json:"confidence"` // 0..1
}

// ========================
// Cost Explorer
// ========================

// ExplorerParams represents the query parameters for the cost explorer
type ExplorerParams struct {
	GroupBy  string `json:"groupBy"` // tenant, region, table, operation
	Window   string `json:"window"`  // 7d, 30d, 90d, 1y
	Period   string `json:"period"`  // hour, day, week, month
	TenantID string `json:"tenantId,omitempty"`
	Region   string `json:"region,omitempty"`
	Table    string `json:"table,omitempty"`
}

// ExplorerPoint is a single time/cost datapoint in the explorer
type ExplorerPoint struct {
	Timestamp int64   `json:"timestamp"`
	CostUSD   float64 `json:"costUSD"`
}

// ExplorerSeries represents a grouped time series
type ExplorerSeries struct {
	Key    string          `json:"key"`
	Points []ExplorerPoint `json:"points"`
}

// ExplorerAnomalyMarker flags an anomaly at a particular time
type ExplorerAnomalyMarker struct {
	Timestamp int64  `json:"timestamp"`
	Severity  string `json:"severity"`
	Reason    string `json:"reason"`
}

// ExplorerResponse is the full response from the cost explorer API
type ExplorerResponse struct {
	GroupBy   string                  `json:"groupBy"`
	Window    string                  `json:"window"`
	Period    string                  `json:"period"`
	Series    []ExplorerSeries        `json:"series"`
	Forecast  []ExplorerPoint         `json:"forecast"`
	Anomalies []ExplorerAnomalyMarker `json:"anomalies"`
}

// ========================
// Billing Credits
// ========================

// BillingCredit represents a credit (discount, SLA breach, promotion)
type BillingCredit struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenantId" db:"tenant_id"`
	AmountUSD float64   `json:"amountUSD" db:"amount_usd"`
	Reason    string    `json:"reason" db:"reason"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	CreatedBy string    `json:"createdBy" db:"created_by"`
}

// ========================
// Tenant Invoices
// ========================

// InvoiceLineItem is a single cost line on an invoice
type InvoiceLineItem struct {
	Type      string  `json:"type"` // compute, storage, events, overage, slo_breach, credit
	AmountUSD float64 `json:"amountUSD"`
}

// InvoiceResponse represents a monthly tenant invoice
type InvoiceResponse struct {
	TenantID  string            `json:"tenantId"`
	Period    string            `json:"period"` // e.g. "2026-01"
	TotalUSD  float64           `json:"totalUSD"`
	LineItems []InvoiceLineItem `json:"lineItems"`
}

// ========================
// Cost Simulator (what-if)
// ========================

// CostSimulationRequest is the input for cost simulation
type CostSimulationRequest struct {
	TenantID       string   `json:"tenantId,omitempty"`
	EventsPerMonth int64    `json:"eventsPerMonth"`
	ComputeMs      float64  `json:"computeMs"`
	StorageGB      float64  `json:"storageGB"`
	Regions        []string `json:"regions"`
	SLOTier        string   `json:"sloTier"` // standard, premium
}

// CostSimulationResponse gives estimated cost
type CostSimulationResponse struct {
	EstimatedCostUSD float64             `json:"estimatedCostUSD"`
	Breakdown        TenantCostBreakdown `json:"breakdown"`
}

// ========================
// Tenant Profitability
// ========================

// TenantProfitability represents profitability for a single tenant
type TenantProfitability struct {
	TenantID  string  `json:"tenantId"`
	ProfitUSD float64 `json:"profitUSD"`
	Score     float64 `json:"score"` // profit / cost ratio
}

// ========================
// Per-Table Cost Attribution
// ========================

// TableCost represents the cost of a single table
type TableCost struct {
	Table      string  `json:"table"`
	ComputeUSD float64 `json:"computeUSD"`
	StorageUSD float64 `json:"storageUSD"`
}

// ========================
// Guardrail Actions
// ========================

// GuardrailAction represents an automated cost-control action
type GuardrailAction string

const (
	GuardrailNoAction      GuardrailAction = "no_action"
	GuardrailThrottle      GuardrailAction = "throttle"
	GuardrailBlockWrites   GuardrailAction = "block_writes"
	GuardrailDegradeRegion GuardrailAction = "degrade_region"
	GuardrailNotify        GuardrailAction = "notify"
)

// ActiveGuardrail describes a guardrail currently in effect
type ActiveGuardrail struct {
	TenantID    string          `json:"tenantId"`
	Trigger     string          `json:"trigger"`
	Severity    string          `json:"severity"`
	Action      GuardrailAction `json:"action"`
	TriggeredAt string          `json:"triggeredAt"`
}

// GuardrailOverrideRequest allows SRE overrides
type GuardrailOverrideRequest struct {
	TenantID string `json:"tenantId"`
	Action   string `json:"action"` // removeThrottle, removeBlock, etc.
	Reason   string `json:"reason"`
}

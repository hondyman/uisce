package billing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

// PrometheusQuerier abstracts Prometheus/Thanos HTTP API queries so
// the billing service doesn't depend on the api package directly.
type PrometheusQuerier interface {
	// InstantQuery runs an instant PromQL query and returns the
	// vector result as a map of label-key → float64 value.
	InstantQuery(ctx context.Context, query string) ([]QueryResult, error)
}

// QueryResult holds one series returned by a PromQL instant query.
type QueryResult struct {
	Labels map[string]string
	Value  float64
}

// PlatformBillingService provides billing, cost analytics, and
// forecasting using the metrics already collected by Prometheus.
type PlatformBillingService struct {
	prom PrometheusQuerier
	cost CostModel
}

// NewPlatformBillingService creates a new billing service.
func NewPlatformBillingService(prom PrometheusQuerier, cost CostModel) *PlatformBillingService {
	return &PlatformBillingService{prom: prom, cost: cost}
}

// ─── Tenant Billing ─────────────────────────────────────────────

// GetTenantBilling queries Prometheus for all tenant usage and
// computes the estimated cost for the given window (e.g. "30d").
func (s *PlatformBillingService) GetTenantBilling(ctx context.Context, tenantID, window string) (*TenantBillingResponse, error) {
	usage, err := s.collectTenantUsage(ctx, tenantID, window)
	if err != nil {
		return nil, fmt.Errorf("billing: collect usage: %w", err)
	}

	costBreakdown := s.estimateTenantCost(usage)

	return &TenantBillingResponse{
		TenantID:      tenantID,
		Window:        window,
		Usage:         *usage,
		EstimatedCost: costBreakdown,
	}, nil
}

func (s *PlatformBillingService) collectTenantUsage(ctx context.Context, tenantID, window string) (*TenantUsage, error) {
	usage := &TenantUsage{}
	tid := sanitise(tenantID)

	// Events published / commits (success)
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commits_success_total{tenant_id="%s"}[%s]))`, tid, window)); err == nil {
		usage.EventsPublished = int64(v)
		usage.Commits = int64(v)
	}

	// S3 validations
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_s3_validation_failures_total{tenant_id="%s"}[%s]))`, tid, window)); err == nil {
		usage.S3Validations = int64(v)
	}

	// Idempotency hits
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_idempotency_hits_total{tenant_id="%s"}[%s]))`, tid, window)); err == nil {
		usage.IdempotencyHits = int64(v)
	}

	// Compute latency
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commit_latency_ms_sum{tenant_id="%s"}[%s]))`, tid, window)); err == nil {
		usage.ComputeMs.Total = v
	}
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`histogram_quantile(0.50, sum(rate(commit_service_commit_latency_ms_bucket{tenant_id="%s"}[%s])) by (le))`, tid, window)); err == nil {
		usage.ComputeMs.P50 = v
	}
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket{tenant_id="%s"}[%s])) by (le))`, tid, window)); err == nil {
		usage.ComputeMs.P95 = v
	}
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`histogram_quantile(0.99, sum(rate(commit_service_commit_latency_ms_bucket{tenant_id="%s"}[%s])) by (le))`, tid, window)); err == nil {
		usage.ComputeMs.P99 = v
	}

	// Storage
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`max(iceberg_table_size_bytes{tenant_id="%s"})`, tid)); err == nil {
		usage.Storage.TotalBytes = int64(v)
	}
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(iceberg_snapshot_created_total{tenant_id="%s"}[%s]))`, tid, window)); err == nil {
		usage.Storage.SnapshotCount = int(v)
	}

	// Region usage
	results, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commits_success_total{tenant_id="%s"}[%s])) by (region)`, tid, window), "region")
	for region, commits := range results {
		computeMs := 0.0
		if cv, err := s.scalarQuery(ctx, fmt.Sprintf(
			`sum(increase(commit_service_commit_latency_ms_sum{tenant_id="%s",region="%s"}[%s]))`, tid, region, window)); err == nil {
			computeMs = cv
		}
		usage.Regions = append(usage.Regions, RegionUsage{
			Region:    region,
			Commits:   int64(commits),
			ComputeMs: computeMs,
		})
	}

	// Table usage
	tableResults, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commits_success_total{tenant_id="%s"}[%s])) by (table)`, tid, window), "table")
	for table, commits := range tableResults {
		storageBytes := int64(0)
		if sv, err := s.scalarQuery(ctx, fmt.Sprintf(
			`max(iceberg_table_size_bytes{tenant_id="%s",table="%s"})`, tid, table)); err == nil {
			storageBytes = int64(sv)
		}
		usage.Tables = append(usage.Tables, TableUsage{
			Table:        table,
			Commits:      int64(commits),
			StorageBytes: storageBytes,
		})
	}

	return usage, nil
}

func (s *PlatformBillingService) estimateTenantCost(usage *TenantUsage) TenantCostBreakdown {
	computeUSD := usage.ComputeMs.Total * s.cost.CostPerComputeMs
	storageGB := float64(usage.Storage.TotalBytes) / (1024 * 1024 * 1024)
	storageUSD := storageGB * s.cost.CostPerGBMonth
	eventsUSD := float64(usage.EventsPublished) * s.cost.CostPerEvent

	total := computeUSD + storageUSD + eventsUSD

	return TenantCostBreakdown{
		ComputeUSD: round2(computeUSD),
		StorageUSD: round2(storageUSD),
		EventsUSD:  round2(eventsUSD),
		TotalUSD:   round2(total),
	}
}

// ─── Platform Billing ───────────────────────────────────────────

// GetPlatformBilling queries aggregated platform-level costs.
func (s *PlatformBillingService) GetPlatformBilling(ctx context.Context, window string) (*PlatformBillingResponse, error) {
	resp := &PlatformBillingResponse{Window: window}

	// Total compute
	totalComputeMs := 0.0
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commit_latency_ms_sum[%s]))`, window)); err == nil {
		totalComputeMs = v
	}
	resp.Totals.ComputeUSD = round2(totalComputeMs * s.cost.CostPerComputeMs)

	// Total storage
	totalBytes := 0.0
	if v, err := s.scalarQuery(ctx, `sum(max(iceberg_table_size_bytes) by (tenant_id))`); err == nil {
		totalBytes = v
	}
	resp.Totals.StorageUSD = round2((totalBytes / (1024 * 1024 * 1024)) * s.cost.CostPerGBMonth)

	// Total events
	totalEvents := 0.0
	if v, err := s.scalarQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commits_success_total[%s]))`, window)); err == nil {
		totalEvents = v
	}
	resp.Totals.EventsUSD = round2(totalEvents * s.cost.CostPerEvent)
	resp.Totals.TotalUSD = round2(resp.Totals.ComputeUSD + resp.Totals.StorageUSD + resp.Totals.EventsUSD)

	// By region
	regionCompute, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commit_latency_ms_sum[%s])) by (region)`, window), "region")
	for region, ms := range regionCompute {
		resp.ByRegion = append(resp.ByRegion, RegionCost{
			Region:   region,
			TotalUSD: round2(ms * s.cost.CostPerComputeMs),
		})
	}

	// By tenant (compute + events)
	tenantCompute, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commit_latency_ms_sum[%s])) by (tenant_id)`, window), "tenant_id")
	tenantEvents, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commits_success_total[%s])) by (tenant_id)`, window), "tenant_id")

	tenantCosts := map[string]float64{}
	for t, ms := range tenantCompute {
		tenantCosts[t] += ms * s.cost.CostPerComputeMs
	}
	for t, ev := range tenantEvents {
		tenantCosts[t] += ev * s.cost.CostPerEvent
	}

	var allTenants []TenantCost
	for t, c := range tenantCosts {
		allTenants = append(allTenants, TenantCost{TenantID: t, TotalUSD: round2(c)})
	}

	// Sort descending by cost
	sort.Slice(allTenants, func(i, j int) bool {
		return allTenants[i].TotalUSD > allTenants[j].TotalUSD
	})

	resp.ByTenant = allTenants

	// Top / bottom N
	topN := 10
	if topN > len(allTenants) {
		topN = len(allTenants)
	}
	resp.TopTenants = allTenants[:topN]

	bottomN := 10
	if bottomN > len(allTenants) {
		bottomN = len(allTenants)
	}
	resp.BottomTenants = allTenants[len(allTenants)-bottomN:]

	return resp, nil
}

// ─── Anomaly Detection ──────────────────────────────────────────

// DetectAnomalies finds cost spikes by comparing 1h vs 24h averages.
func (s *PlatformBillingService) DetectAnomalies(ctx context.Context) (*BillingAnomalyResponse, error) {
	resp := &BillingAnomalyResponse{
		TenantAnomalies: []BillingAnomaly{},
		RegionAnomalies: []BillingAnomaly{},
		CostAnomalies:   []BillingAnomaly{},
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Global cost spike
	short1h, _ := s.scalarQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[1h]))`)
	long24h, _ := s.scalarQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[24h]))`)
	hourly24h := long24h / 24
	if hourly24h > 0 {
		ratio := short1h / hourly24h
		if ratio > 2.0 {
			resp.CostAnomalies = append(resp.CostAnomalies, BillingAnomaly{
				Type:      "cost",
				Key:       "global",
				Severity:  severityFromRatio(ratio),
				Ratio:     round2(ratio),
				Reason:    fmt.Sprintf("Global cost increased %.1fx vs 24h average", ratio),
				Timestamp: now,
			})
		}
	}

	// Tenant anomalies
	tenantShort, _ := s.vectorQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[1h])) by (tenant_id)`, "tenant_id")
	tenantLong, _ := s.vectorQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[24h])) by (tenant_id)`, "tenant_id")
	for t, sh := range tenantShort {
		hourly := tenantLong[t] / 24
		if hourly > 0 {
			ratio := sh / hourly
			if ratio > 2.0 {
				resp.TenantAnomalies = append(resp.TenantAnomalies, BillingAnomaly{
					Type:      "tenant",
					Key:       t,
					Severity:  severityFromRatio(ratio),
					Ratio:     round2(ratio),
					Reason:    fmt.Sprintf("Tenant %s cost spiked %.1fx", t, ratio),
					Timestamp: now,
				})
			}
		}
	}

	// Region anomalies
	regionShort, _ := s.vectorQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[1h])) by (region)`, "region")
	regionLong, _ := s.vectorQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[24h])) by (region)`, "region")
	for r, sh := range regionShort {
		hourly := regionLong[r] / 24
		if hourly > 0 {
			ratio := sh / hourly
			if ratio > 2.0 {
				resp.RegionAnomalies = append(resp.RegionAnomalies, BillingAnomaly{
					Type:      "region",
					Key:       r,
					Severity:  severityFromRatio(ratio),
					Ratio:     round2(ratio),
					Reason:    fmt.Sprintf("Region %s cost spiked %.1fx", r, ratio),
					Timestamp: now,
				})
			}
		}
	}

	return resp, nil
}

// ─── Forecasting ────────────────────────────────────────────────

// ForecastCost produces a linear forecast using exponential smoothing.
func (s *PlatformBillingService) ForecastCost(ctx context.Context) (*BillingForecast, error) {
	current, _ := s.scalarQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[30d]))`)
	previous, _ := s.scalarQuery(ctx, `sum(increase(commit_service_commit_latency_ms_sum[30d] offset 30d))`)

	currentCost := current * s.cost.CostPerComputeMs
	previousCost := previous * s.cost.CostPerComputeMs

	alpha := 0.6
	forecast := alpha*currentCost + (1-alpha)*previousCost

	confidence := 0.85
	if previousCost > 0 {
		changeRate := math.Abs(currentCost-previousCost) / previousCost
		if changeRate < 0.1 {
			confidence = 0.95
		} else if changeRate > 0.5 {
			confidence = 0.70
		}
	}

	return &BillingForecast{
		ForecastUSD: round2(forecast),
		Model:       "exponential_smoothing",
		Confidence:  round2(confidence),
	}, nil
}

// ─── Cost Simulator ─────────────────────────────────────────────

// SimulateCost runs a what-if analysis for hypothetical usage.
func (s *PlatformBillingService) SimulateCost(req CostSimulationRequest) *CostSimulationResponse {
	computeUSD := req.ComputeMs * s.cost.CostPerComputeMs
	storageUSD := req.StorageGB * s.cost.CostPerGBMonth
	eventsUSD := float64(req.EventsPerMonth) * s.cost.CostPerEvent

	// Premium SLO tier doubles cost per ms over threshold
	if req.SLOTier == "premium" {
		computeUSD *= 1.5
		storageUSD *= 1.2
	}

	total := computeUSD + storageUSD + eventsUSD

	return &CostSimulationResponse{
		EstimatedCostUSD: round2(total),
		Breakdown: TenantCostBreakdown{
			ComputeUSD: round2(computeUSD),
			StorageUSD: round2(storageUSD),
			EventsUSD:  round2(eventsUSD),
			TotalUSD:   round2(total),
		},
	}
}

// ─── Per-Table Cost ─────────────────────────────────────────────

// GetTableCosts returns compute and storage cost per table.
func (s *PlatformBillingService) GetTableCosts(ctx context.Context, window string) ([]TableCost, error) {
	computeByTable, _ := s.vectorQuery(ctx, fmt.Sprintf(
		`sum(increase(commit_service_commit_latency_ms_sum[%s])) by (table)`, window), "table")
	storageByTable, _ := s.vectorQuery(ctx, `max(iceberg_table_size_bytes) by (table)`, "table")

	var costs []TableCost
	tables := map[string]bool{}
	for t := range computeByTable {
		tables[t] = true
	}
	for t := range storageByTable {
		tables[t] = true
	}

	for t := range tables {
		cUSD := computeByTable[t] * s.cost.CostPerComputeMs
		sUSD := (storageByTable[t] / (1024 * 1024 * 1024)) * s.cost.CostPerGBMonth
		costs = append(costs, TableCost{
			Table:      t,
			ComputeUSD: round2(cUSD),
			StorageUSD: round2(sUSD),
		})
	}

	sort.Slice(costs, func(i, j int) bool {
		return (costs[i].ComputeUSD + costs[i].StorageUSD) > (costs[j].ComputeUSD + costs[j].StorageUSD)
	})

	return costs, nil
}

// ─── Tenant Invoice ─────────────────────────────────────────────

// GenerateInvoice produces an invoice for a given month (YYYY-MM).
func (s *PlatformBillingService) GenerateInvoice(ctx context.Context, tenantID, month string) (*InvoiceResponse, error) {
	billing, err := s.GetTenantBilling(ctx, tenantID, "30d")
	if err != nil {
		return nil, fmt.Errorf("billing: generate invoice: %w", err)
	}

	invoice := &InvoiceResponse{
		TenantID: tenantID,
		Period:   month,
		LineItems: []InvoiceLineItem{
			{Type: "compute", AmountUSD: billing.EstimatedCost.ComputeUSD},
			{Type: "storage", AmountUSD: billing.EstimatedCost.StorageUSD},
			{Type: "events", AmountUSD: billing.EstimatedCost.EventsUSD},
		},
	}

	if billing.EstimatedCost.OverageUSD > 0 {
		invoice.LineItems = append(invoice.LineItems, InvoiceLineItem{
			Type: "overage", AmountUSD: billing.EstimatedCost.OverageUSD,
		})
	}
	if billing.EstimatedCost.SLOBreachUSD > 0 {
		invoice.LineItems = append(invoice.LineItems, InvoiceLineItem{
			Type: "slo_breach", AmountUSD: billing.EstimatedCost.SLOBreachUSD,
		})
	}

	invoice.TotalUSD = billing.EstimatedCost.TotalUSD

	return invoice, nil
}

// ─── Helpers ────────────────────────────────────────────────────

// scalarQuery returns a single float64 from a PromQL instant query.
func (s *PlatformBillingService) scalarQuery(ctx context.Context, query string) (float64, error) {
	results, err := s.prom.InstantQuery(ctx, query)
	if err != nil || len(results) == 0 {
		return 0, err
	}
	return results[0].Value, nil
}

// vectorQuery returns label→value map from a PromQL instant query.
func (s *PlatformBillingService) vectorQuery(ctx context.Context, query, label string) (map[string]float64, error) {
	results, err := s.prom.InstantQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64)
	for _, r := range results {
		key := r.Labels[label]
		if key != "" {
			out[key] = r.Value
		}
	}
	return out, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func severityFromRatio(ratio float64) string {
	switch {
	case ratio > 5:
		return "critical"
	case ratio > 3:
		return "high"
	case ratio > 2:
		return "medium"
	default:
		return "low"
	}
}

func sanitise(s string) string {
	// Prevent PromQL injection by removing quotes
	out := ""
	for _, c := range s {
		if c != '"' && c != '\\' {
			out += string(c)
		}
	}
	return out
}

// parseWindowDuration converts window strings like "30d", "7d", "1h"
// to a time.Duration. Used by the Holt-Winters forecaster.
func parseWindowDuration(window string) (time.Duration, error) {
	if len(window) < 2 {
		return 0, fmt.Errorf("invalid window: %s", window)
	}
	unit := window[len(window)-1]
	numStr := window[:len(window)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}
	switch unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c", unit)
	}
}

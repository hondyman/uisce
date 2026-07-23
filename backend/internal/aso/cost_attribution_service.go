package aso

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Cost Metrics Types
// ============================================================================

// CostMetrics represents the financial value of an optimization
type CostMetrics struct {
	// Savings
	ComputeSavingsPerDay float64 `json:"compute_savings_per_day"` // $/day
	LatencyReductionMs   float64 `json:"latency_reduction_ms"`
	QueriesAccelerated   int64   `json:"queries_accelerated"`

	// Costs
	StorageCostPerMonth float64 `json:"storage_cost_per_month"` // $/month
	RefreshCostPerDay   float64 `json:"refresh_cost_per_day"`   // $/day
	BuildCostOneTime    float64 `json:"build_cost_one_time"`    // $

	// Net value
	NetSavingsPerDay  float64 `json:"net_savings_per_day"` // $/day
	PaybackPeriodDays float64 `json:"payback_period_days"` // days until ROI positive
	MonthlyROI        float64 `json:"monthly_roi"`         // % return

	// Lifetime value
	TotalSavingsToDate float64 `json:"total_savings_to_date"` // $ since applied
	TotalCostsToDate   float64 `json:"total_costs_to_date"`   // $ since applied
}

// CostConfig defines pricing for cost calculations
type CostConfig struct {
	ComputeCostPerMsPerQuery float64 `json:"compute_cost_per_ms_per_query"` // $/ms/query
	StorageCostPerGBPerMonth float64 `json:"storage_cost_per_gb_per_month"` // $/GB/month
	RefreshCostPerMs         float64 `json:"refresh_cost_per_ms"`           // $/ms
}

// DefaultCostConfig returns typical cloud pricing
func DefaultCostConfig() CostConfig {
	return CostConfig{
		ComputeCostPerMsPerQuery: 0.00001, // $0.01 per 1000 queries at 1ms
		StorageCostPerGBPerMonth: 0.023,   // S3 pricing
		RefreshCostPerMs:         0.00005, // compute during refresh
	}
}

// ============================================================================
// Cost Attribution Service
// ============================================================================

// CostAttributionService calculates financial value of optimizations
type CostAttributionService interface {
	// CalculateCostMetrics computes cost/savings for an optimization
	CalculateCostMetrics(ctx context.Context, optID uuid.UUID) (*CostMetrics, error)

	// GetTenantSavings returns total savings for a tenant
	GetTenantSavings(ctx context.Context, tenantID uuid.UUID, since time.Time) (*TenantSavingsSummary, error)

	// GetGlobalSavings returns platform-wide savings
	GetGlobalSavings(ctx context.Context, since time.Time) (*GlobalSavingsSummary, error)

	// UpdateOptimizationCosts updates cost metrics for an applied optimization
	UpdateOptimizationCosts(ctx context.Context, optID uuid.UUID) error
}

// TenantSavingsSummary aggregates savings per tenant
type TenantSavingsSummary struct {
	TenantID             uuid.UUID `json:"tenant_id"`
	Period               string    `json:"period"` // e.g., "last_30_days"
	TotalSavings         float64   `json:"total_savings"`
	TotalCosts           float64   `json:"total_costs"`
	NetSavings           float64   `json:"net_savings"`
	OptimizationsApplied int       `json:"optimizations_applied"`
	QueriesAccelerated   int64     `json:"queries_accelerated"`
	AvgSpeedupFactor     float64   `json:"avg_speedup_factor"`
}

// GlobalSavingsSummary aggregates platform-wide savings
type GlobalSavingsSummary struct {
	Period           string                 `json:"period"`
	TotalSavings     float64                `json:"total_savings"`
	TotalCosts       float64                `json:"total_costs"`
	NetSavings       float64                `json:"net_savings"`
	TenantBreakdown  []TenantSavingsSummary `json:"tenant_breakdown"`
	TopOptimizations []OptimizationValue    `json:"top_optimizations"`
}

// OptimizationValue shows ROI for a single optimization
type OptimizationValue struct {
	OptimizationID uuid.UUID `json:"optimization_id"`
	TargetName     string    `json:"target_name"`
	Type           string    `json:"type"`
	NetSavings     float64   `json:"net_savings"`
	AppliedAt      time.Time `json:"applied_at"`
}

// costAttributionService implements CostAttributionService
type costAttributionService struct {
	db      *sqlx.DB
	optRepo ASOOptimizationRepository
	config  CostConfig
}

// NewCostAttributionService creates a new cost attribution service
func NewCostAttributionService(db *sqlx.DB, optRepo ASOOptimizationRepository) CostAttributionService {
	return &costAttributionService{
		db:      db,
		optRepo: optRepo,
		config:  DefaultCostConfig(),
	}
}

// CalculateCostMetrics computes cost/savings for an optimization
func (s *costAttributionService) CalculateCostMetrics(ctx context.Context, optID uuid.UUID) (*CostMetrics, error) {
	opt, err := s.optRepo.GetByID(ctx, optID)
	if err != nil || opt == nil {
		return nil, fmt.Errorf("optimization not found")
	}

	metrics := &CostMetrics{}

	// Calculate based on optimization type
	switch opt.OptimizationType {
	case OptTypeCreatePreAgg, OptTypeTuneDefinition:
		metrics = s.calculatePreAggCosts(opt)
	case OptTypeTuneRefresh:
		metrics = s.calculateTuneRefreshCosts(opt)
	case OptTypeRetireAsset:
		metrics = s.calculateRetirementSavings(opt)
	}

	// Calculate lifetime if applied
	if opt.AppliedAt != nil {
		daysSinceApplied := time.Since(*opt.AppliedAt).Hours() / 24
		metrics.TotalSavingsToDate = metrics.NetSavingsPerDay * daysSinceApplied
		metrics.TotalCostsToDate = (metrics.StorageCostPerMonth / 30) * daysSinceApplied
	}

	return metrics, nil
}

// calculatePreAggCosts computes costs for pre-agg creation/modification
func (s *costAttributionService) calculatePreAggCosts(opt *ASOOptimization) *CostMetrics {
	metrics := &CostMetrics{}

	// Parse details
	var details CreatePreAggDetails
	if opt.Details != nil {
		_ = json.Unmarshal(opt.Details, &details)
	}

	// Compute savings from latency reduction
	if opt.AvgLatencyMs != nil && opt.QueriesPerDay != nil {
		// Assume pre-agg reduces latency by speedup factor
		speedup := details.CostEstimate.EstimatedSpeedupFactor
		if speedup < 1 {
			speedup = 5.0 // default estimate
		}

		latencyReduction := *opt.AvgLatencyMs * (1 - 1/speedup)
		metrics.LatencyReductionMs = latencyReduction
		metrics.QueriesAccelerated = int64(*opt.QueriesPerDay)
		metrics.ComputeSavingsPerDay = latencyReduction * *opt.QueriesPerDay * s.config.ComputeCostPerMsPerQuery
	}

	// Storage costs
	storageBytesGB := float64(details.CostEstimate.EstimatedStorageBytes) / (1024 * 1024 * 1024)
	metrics.StorageCostPerMonth = storageBytesGB * s.config.StorageCostPerGBPerMonth

	// Build cost
	metrics.BuildCostOneTime = details.CostEstimate.EstimatedBuildCost

	// Refresh cost (assume daily refresh)
	metrics.RefreshCostPerDay = details.CostEstimate.EstimatedRefreshCost

	// Net savings
	metrics.NetSavingsPerDay = metrics.ComputeSavingsPerDay - metrics.RefreshCostPerDay - (metrics.StorageCostPerMonth / 30)

	// Payback period
	if metrics.NetSavingsPerDay > 0 {
		metrics.PaybackPeriodDays = metrics.BuildCostOneTime / metrics.NetSavingsPerDay
	}

	// Monthly ROI
	monthlySavings := metrics.NetSavingsPerDay * 30
	monthlyCosts := metrics.StorageCostPerMonth + (metrics.RefreshCostPerDay * 30)
	if monthlyCosts > 0 {
		metrics.MonthlyROI = (monthlySavings - monthlyCosts) / monthlyCosts * 100
	}

	return metrics
}

// calculateTuneRefreshCosts computes savings from refresh interval optimization
func (s *costAttributionService) calculateTuneRefreshCosts(opt *ASOOptimization) *CostMetrics {
	metrics := &CostMetrics{}

	var details TuneRefreshDetails
	if opt.Details != nil {
		_ = json.Unmarshal(opt.Details, &details)
	}

	// Calculate refresh cost savings (if interval increased) or query savings (if decreased)
	// This is a simplified model
	metrics.NetSavingsPerDay = 5.0 // Placeholder - would calculate from actual refresh deltas

	return metrics
}

// calculateRetirementSavings computes savings from retiring unused assets
func (s *costAttributionService) calculateRetirementSavings(opt *ASOOptimization) *CostMetrics {
	metrics := &CostMetrics{}

	var details RetireAssetDetails
	if opt.Details != nil {
		_ = json.Unmarshal(opt.Details, &details)
	}

	// Storage savings
	storageBytesGB := float64(details.StorageBytes) / (1024 * 1024 * 1024)
	metrics.StorageCostPerMonth = storageBytesGB * s.config.StorageCostPerGBPerMonth * -1 // negative = savings

	// Refresh cost savings
	metrics.RefreshCostPerDay = float64(details.RefreshCostMs) * s.config.RefreshCostPerMs * -1

	metrics.NetSavingsPerDay = -metrics.StorageCostPerMonth/30 - metrics.RefreshCostPerDay

	return metrics
}

// GetTenantSavings returns total savings for a tenant
func (s *costAttributionService) GetTenantSavings(ctx context.Context, tenantID uuid.UUID, since time.Time) (*TenantSavingsSummary, error) {
	summary := &TenantSavingsSummary{
		TenantID: tenantID,
		Period:   fmt.Sprintf("since %s", since.Format("2006-01-02")),
	}

	// Query applied optimizations
	filter := OptimizationFilter{
		TenantID: &tenantID,
	}
	applied := OptStatusApplied
	filter.Status = &applied

	opts, err := s.optRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if opt.AppliedAt != nil && opt.AppliedAt.After(since) {
			metrics, _ := s.CalculateCostMetrics(ctx, opt.ID)
			if metrics != nil {
				summary.TotalSavings += metrics.TotalSavingsToDate
				summary.TotalCosts += metrics.TotalCostsToDate
				summary.QueriesAccelerated += metrics.QueriesAccelerated
			}
			summary.OptimizationsApplied++
		}
	}

	summary.NetSavings = summary.TotalSavings - summary.TotalCosts

	return summary, nil
}

// GetGlobalSavings returns platform-wide savings
func (s *costAttributionService) GetGlobalSavings(ctx context.Context, since time.Time) (*GlobalSavingsSummary, error) {
	summary := &GlobalSavingsSummary{
		Period: fmt.Sprintf("since %s", since.Format("2006-01-02")),
	}

	// Get all applied optimizations
	applied := OptStatusApplied
	filter := OptimizationFilter{Status: &applied, Limit: 100}
	opts, err := s.optRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if opt.AppliedAt != nil && opt.AppliedAt.After(since) {
			metrics, _ := s.CalculateCostMetrics(ctx, opt.ID)
			if metrics != nil {
				summary.TotalSavings += metrics.TotalSavingsToDate
				summary.TotalCosts += metrics.TotalCostsToDate

				summary.TopOptimizations = append(summary.TopOptimizations, OptimizationValue{
					OptimizationID: opt.ID,
					TargetName:     opt.TargetName,
					Type:           string(opt.OptimizationType),
					NetSavings:     metrics.TotalSavingsToDate,
					AppliedAt:      *opt.AppliedAt,
				})
			}
		}
	}

	summary.NetSavings = summary.TotalSavings - summary.TotalCosts

	return summary, nil
}

// UpdateOptimizationCosts updates cost metrics for an applied optimization
func (s *costAttributionService) UpdateOptimizationCosts(ctx context.Context, optID uuid.UUID) error {
	metrics, err := s.CalculateCostMetrics(ctx, optID)
	if err != nil {
		return err
	}

	metricsJSON, _ := json.Marshal(metrics)

	_, err = s.db.ExecContext(ctx, `
		UPDATE semantic.aso_optimization
		SET details = jsonb_set(COALESCE(details, '{}'::jsonb), '{cost_metrics}', $2::jsonb)
		WHERE id = $1
	`, optID, metricsJSON)

	return err
}

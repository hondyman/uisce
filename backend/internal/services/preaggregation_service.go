package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// PreaggregatedMetric represents a precomputed metric stored in the semantic layer
type PreaggregatedMetric struct {
	ID              string                 `json:"id"`
	NodeID          string                 `json:"node_id"`
	Name            string                 `json:"name"`
	Value           float64                `json:"value"`
	Grain           []string               `json:"grain"`
	GrainValues     map[string]interface{} `json:"grain_values"`
	LastRefresh     time.Time              `json:"last_refresh"`
	RefreshSchedule string                 `json:"refresh_schedule"`
	SourceFormula   string                 `json:"source_formula"`
	DataQuality     DataQualityMetrics     `json:"data_quality"`
	BusinessContext string                 `json:"business_context"`
}

// DataQualityMetrics tracks the quality of preaggregated data
type DataQualityMetrics struct {
	CompletenessScore float64   `json:"completeness_score"`
	FreshnessHours    float64   `json:"freshness_hours"`
	SourceCount       int       `json:"source_count"`
	LastValidated     time.Time `json:"last_validated"`
}

// PreaggregationService handles precomputed metric calculations and storage
type PreaggregationService struct {
	db     *sql.DB
	logger *log.Logger
}

// NewPreaggregationService creates a new preaggregation service
func NewPreaggregationService(db *sql.DB) *PreaggregationService {
	return &PreaggregationService{
		db:     db,
		logger: log.New(log.Writer(), "[PREAGG]", log.LstdFlags),
	}
}

// PrecomputeNetIRR calculates and stores Net IRR at fund/month grain
func (s *PreaggregationService) PrecomputeNetIRR(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting Net IRR precomputation for grain: %v", grain)

	// Build query based on grain
	query := s.buildNetIRRQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Net IRR data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var month time.Time
		var cashFlowsJSON, datesJSON string

		err := rows.Scan(&fundID, &month, &cashFlowsJSON, &datesJSON)
		if err != nil {
			s.logger.Printf("Error scanning Net IRR row: %v", err)
			continue
		}

		// Parse cash flows and dates
		var cashFlows []float64
		var dates []time.Time

		if err := json.Unmarshal([]byte(cashFlowsJSON), &cashFlows); err != nil {
			s.logger.Printf("Error parsing cash flows for fund %s: %v", fundID, err)
			continue
		}

		if err := json.Unmarshal([]byte(datesJSON), &dates); err != nil {
			s.logger.Printf("Error parsing dates for fund %s: %v", fundID, err)
			continue
		}

		// Calculate Net IRR using XIRR
		irr, err := s.calculateXIRR(cashFlows, dates)
		if err != nil {
			s.logger.Printf("Error calculating Net IRR for fund %s: %v", fundID, err)
			continue
		}

		// Create preaggregated metric
		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("net_irr_%s_%s", fundID, month.Format("2006-01")),
			NodeID:          "private_markets_net_irr",
			Name:            "Net IRR",
			Value:           irr,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "month": month},
			LastRefresh:     time.Now(),
			RefreshSchedule: "daily",
			SourceFormula:   "=XIRR({cash_flows}, {dates})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: s.calculateCompletenessScore(cashFlows),
				FreshnessHours:    0,
				SourceCount:       len(cashFlows),
				LastValidated:     time.Now(),
			},
			BusinessContext: "Net Internal Rate of Return after fees - preaggregated for performance monitoring",
		}

		// Store the metric
		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing Net IRR metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed Net IRR precomputation")
	return nil
}

// PrecomputeXIRR calculates and stores XIRR at fund/month grain
func (s *PreaggregationService) PrecomputeXIRR(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting XIRR precomputation for grain: %v", grain)

	query := s.buildXIRRQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query XIRR data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var month time.Time
		var cashFlowsJSON, datesJSON string

		err := rows.Scan(&fundID, &month, &cashFlowsJSON, &datesJSON)
		if err != nil {
			s.logger.Printf("Error scanning XIRR row: %v", err)
			continue
		}

		var cashFlows []float64
		var dates []time.Time

		if err := json.Unmarshal([]byte(cashFlowsJSON), &cashFlows); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(datesJSON), &dates); err != nil {
			continue
		}

		xirr, err := s.calculateXIRR(cashFlows, dates)
		if err != nil {
			s.logger.Printf("Error calculating XIRR for fund %s: %v", fundID, err)
			continue
		}

		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("xirr_%s_%s", fundID, month.Format("2006-01")),
			NodeID:          "private_markets_xirr",
			Name:            "XIRR",
			Value:           xirr,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "month": month},
			LastRefresh:     time.Now(),
			RefreshSchedule: "daily",
			SourceFormula:   "=XIRR({cash_flows}, {dates})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: s.calculateCompletenessScore(cashFlows),
				FreshnessHours:    0,
				SourceCount:       len(cashFlows),
				LastValidated:     time.Now(),
			},
			BusinessContext: "Extended Internal Rate of Return with irregular cash flows - preaggregated for performance analysis",
		}

		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing XIRR metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed XIRR precomputation")
	return nil
}

// PrecomputeGrossIRR calculates and stores Gross IRR at fund/month grain
func (s *PreaggregationService) PrecomputeGrossIRR(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting Gross IRR precomputation for grain: %v", grain)

	query := s.buildGrossIRRQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Gross IRR data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var month time.Time
		var grossCashFlowsJSON, datesJSON string

		err := rows.Scan(&fundID, &month, &grossCashFlowsJSON, &datesJSON)
		if err != nil {
			s.logger.Printf("Error scanning Gross IRR row: %v", err)
			continue
		}

		var grossCashFlows []float64
		var dates []time.Time

		if err := json.Unmarshal([]byte(grossCashFlowsJSON), &grossCashFlows); err != nil {
			continue
		}

		if err := json.Unmarshal([]byte(datesJSON), &dates); err != nil {
			continue
		}

		grossIRR, err := s.calculateXIRR(grossCashFlows, dates)
		if err != nil {
			s.logger.Printf("Error calculating Gross IRR for fund %s: %v", fundID, err)
			continue
		}

		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("gross_irr_%s_%s", fundID, month.Format("2006-01")),
			NodeID:          "private_markets_gross_irr",
			Name:            "Gross IRR",
			Value:           grossIRR,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "month": month},
			LastRefresh:     time.Now(),
			RefreshSchedule: "daily",
			SourceFormula:   "=XIRR({gross_cash_flows}, {dates})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: s.calculateCompletenessScore(grossCashFlows),
				FreshnessHours:    0,
				SourceCount:       len(grossCashFlows),
				LastValidated:     time.Now(),
			},
			BusinessContext: "Gross Internal Rate of Return before fees - preaggregated for GP performance monitoring",
		}

		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing Gross IRR metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed Gross IRR precomputation")
	return nil
}

// PrecomputeGrossMOIC calculates and stores Gross MOIC at fund/quarter grain
func (s *PreaggregationService) PrecomputeGrossMOIC(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting Gross MOIC precomputation for grain: %v", grain)

	query := s.buildGrossMOICQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Gross MOIC data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var quarter time.Time
		var totalDistributions, totalInvestedCapital float64

		err := rows.Scan(&fundID, &quarter, &totalDistributions, &totalInvestedCapital)
		if err != nil {
			s.logger.Printf("Error scanning Gross MOIC row: %v", err)
			continue
		}

		if totalInvestedCapital == 0 {
			s.logger.Printf("Zero invested capital for fund %s, skipping MOIC calculation", fundID)
			continue
		}

		grossMOIC := totalDistributions / totalInvestedCapital

		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("gross_moic_%s_%s", fundID, quarter.Format("2006-Q1")),
			NodeID:          "private_markets_gross_moic",
			Name:            "Gross MOIC",
			Value:           grossMOIC,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "quarter": quarter},
			LastRefresh:     time.Now(),
			RefreshSchedule: "weekly",
			SourceFormula:   "=SUM({gross_distributions}) / SUM({invested_capital})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: 1.0, // Simple aggregation, high completeness
				FreshnessHours:    0,
				SourceCount:       2, // distributions + invested capital
				LastValidated:     time.Now(),
			},
			BusinessContext: "Gross Multiple of Invested Capital - preaggregated for quarterly GP reporting",
		}

		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing Gross MOIC metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed Gross MOIC precomputation")
	return nil
}

// PrecomputeFeeRatio calculates and stores Fee Ratio at fund/month grain
func (s *PreaggregationService) PrecomputeFeeRatio(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting Fee Ratio precomputation for grain: %v", grain)

	query := s.buildFeeRatioQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Fee Ratio data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var month time.Time
		var totalFees, totalAUM float64

		err := rows.Scan(&fundID, &month, &totalFees, &totalAUM)
		if err != nil {
			s.logger.Printf("Error scanning Fee Ratio row: %v", err)
			continue
		}

		if totalAUM == 0 {
			s.logger.Printf("Zero AUM for fund %s, skipping fee ratio calculation", fundID)
			continue
		}

		feeRatio := totalFees / totalAUM

		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("fee_ratio_%s_%s", fundID, month.Format("2006-01")),
			NodeID:          "private_markets_fee_ratio",
			Name:            "Fee Ratio",
			Value:           feeRatio,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "month": month},
			LastRefresh:     time.Now(),
			RefreshSchedule: "daily",
			SourceFormula:   "=SUM({management_fees}) / SUM({assets_under_management})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: 1.0,
				FreshnessHours:    0,
				SourceCount:       2,
				LastValidated:     time.Now(),
			},
			BusinessContext: "Management fee as percentage of AUM - preaggregated for GP fee monitoring",
		}

		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing Fee Ratio metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed Fee Ratio precomputation")
	return nil
}

// PrecomputeDeploymentPace calculates and stores Deployment Pace at fund/month grain
func (s *PreaggregationService) PrecomputeDeploymentPace(ctx context.Context, grain []string) error {
	s.logger.Printf("Starting Deployment Pace precomputation for grain: %v", grain)

	query := s.buildDeploymentPaceQuery(grain)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query Deployment Pace data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fundID string
		var month time.Time
		var totalDeployments, totalCommitments float64

		err := rows.Scan(&fundID, &month, &totalDeployments, &totalCommitments)
		if err != nil {
			s.logger.Printf("Error scanning Deployment Pace row: %v", err)
			continue
		}

		if totalCommitments == 0 {
			s.logger.Printf("Zero commitments for fund %s, skipping deployment pace calculation", fundID)
			continue
		}

		deploymentPace := totalDeployments / totalCommitments

		metric := &PreaggregatedMetric{
			ID:              fmt.Sprintf("deployment_pace_%s_%s", fundID, month.Format("2006-01")),
			NodeID:          "private_markets_deployment_pace",
			Name:            "Deployment Pace",
			Value:           deploymentPace,
			Grain:           grain,
			GrainValues:     map[string]interface{}{"fund_id": fundID, "month": month},
			LastRefresh:     time.Now(),
			RefreshSchedule: "daily",
			SourceFormula:   "=SUM({deployments}) / SUM({commitments})",
			DataQuality: DataQualityMetrics{
				CompletenessScore: 1.0,
				FreshnessHours:    0,
				SourceCount:       2,
				LastValidated:     time.Now(),
			},
			BusinessContext: "Percentage of committed capital deployed - preaggregated for GP operational monitoring",
		}

		if err := s.storePreaggregatedMetric(ctx, metric); err != nil {
			s.logger.Printf("Error storing Deployment Pace metric for fund %s: %v", fundID, err)
		}
	}

	s.logger.Printf("Completed Deployment Pace precomputation")
	return nil
}

// Helper functions for building queries

func (s *PreaggregationService) buildNetIRRQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('month', transaction_date) as month,
			JSON_AGG(net_cash_flow ORDER BY transaction_date) as cash_flows,
			JSON_AGG(transaction_date ORDER BY transaction_date) as dates
		FROM private_markets.cash_flows
		WHERE transaction_date >= CURRENT_DATE - INTERVAL '2 years'
		GROUP BY fund_id, DATE_TRUNC('month', transaction_date)
		ORDER BY fund_id, month
	`
}

func (s *PreaggregationService) buildXIRRQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('month', transaction_date) as month,
			JSON_AGG(cash_flow_amount ORDER BY transaction_date) as cash_flows,
			JSON_AGG(transaction_date ORDER BY transaction_date) as dates
		FROM private_markets.cash_flows
		WHERE transaction_date >= CURRENT_DATE - INTERVAL '2 years'
		GROUP BY fund_id, DATE_TRUNC('month', transaction_date)
		ORDER BY fund_id, month
	`
}

func (s *PreaggregationService) buildGrossIRRQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('month', transaction_date) as month,
			JSON_AGG(gross_cash_flow ORDER BY transaction_date) as gross_cash_flows,
			JSON_AGG(transaction_date ORDER BY transaction_date) as dates
		FROM private_markets.cash_flows
		WHERE transaction_date >= CURRENT_DATE - INTERVAL '2 years'
		GROUP BY fund_id, DATE_TRUNC('month', transaction_date)
		ORDER BY fund_id, month
	`
}

func (s *PreaggregationService) buildGrossMOICQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('quarter', as_of_date) as quarter,
			SUM(gross_distributions) as total_distributions,
			SUM(invested_capital) as total_invested_capital
		FROM private_markets.fund_performance
		WHERE as_of_date >= CURRENT_DATE - INTERVAL '2 years'
		GROUP BY fund_id, DATE_TRUNC('quarter', as_of_date)
		ORDER BY fund_id, quarter
	`
}

func (s *PreaggregationService) buildFeeRatioQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('month', fee_date) as month,
			SUM(management_fee_amount) as total_fees,
			AVG(assets_under_management) as avg_aum
		FROM private_markets.fees
		WHERE fee_date >= CURRENT_DATE - INTERVAL '1 year'
		GROUP BY fund_id, DATE_TRUNC('month', fee_date)
		ORDER BY fund_id, month
	`
}

func (s *PreaggregationService) buildDeploymentPaceQuery(_ []string) string {
	return `
		SELECT
			fund_id,
			DATE_TRUNC('month', deployment_date) as month,
			SUM(deployment_amount) as total_deployments,
			SUM(commitment_amount) as total_commitments
		FROM private_markets.deployments
		WHERE deployment_date >= CURRENT_DATE - INTERVAL '1 year'
		GROUP BY fund_id, DATE_TRUNC('month', deployment_date)
		ORDER BY fund_id, month
	`
}

// calculateXIRR implements Excel's XIRR function using Newton-Raphson method
func (s *PreaggregationService) calculateXIRR(cashFlows []float64, dates []time.Time) (float64, error) {
	if len(cashFlows) != len(dates) || len(cashFlows) < 2 {
		return 0, fmt.Errorf("insufficient data for XIRR calculation")
	}

	// Convert dates to day differences from first date
	baseDate := dates[0]
	days := make([]float64, len(dates))
	for i, date := range dates {
		days[i] = date.Sub(baseDate).Hours() / 24 / 365 // Convert to years
	}

	// Newton-Raphson method for IRR calculation
	const maxIterations = 100
	const tolerance = 1e-6

	rate := 0.1 // Initial guess

	for i := 0; i < maxIterations; i++ {
		f := 0.0
		df := 0.0

		for j, cf := range cashFlows {
			if days[j] == 0 {
				f += cf
			} else {
				f += cf / pow(1+rate, days[j])
				df -= cf * days[j] / pow(1+rate, days[j]+1)
			}
		}

		if abs(df) < tolerance {
			break
		}

		rate = rate - f/df

		if abs(f) < tolerance {
			return rate, nil
		}
	}

	return rate, nil
}

// calculateCompletenessScore calculates data completeness for quality metrics
func (s *PreaggregationService) calculateCompletenessScore(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	validCount := 0
	for _, v := range data {
		if v != 0 { // Simple check for non-zero values
			validCount++
		}
	}

	return float64(validCount) / float64(len(data))
}

// storePreaggregatedMetric stores a precomputed metric in the database
func (s *PreaggregationService) storePreaggregatedMetric(ctx context.Context, metric *PreaggregatedMetric) error {
	query := `
		INSERT INTO semantic_layer.preaggregated_metrics (
			id, node_id, name, value, grain, grain_values,
			last_refresh, refresh_schedule, source_formula,
			data_quality, business_context, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			value = EXCLUDED.value,
			last_refresh = EXCLUDED.last_refresh,
			data_quality = EXCLUDED.data_quality,
			updated_at = EXCLUDED.updated_at
	`

	grainJSON, _ := json.Marshal(metric.Grain)
	grainValuesJSON, _ := json.Marshal(metric.GrainValues)
	dataQualityJSON, _ := json.Marshal(metric.DataQuality)

	_, err := s.db.ExecContext(ctx, query,
		metric.ID, metric.NodeID, metric.Name, metric.Value,
		grainJSON, grainValuesJSON, metric.LastRefresh,
		metric.RefreshSchedule, metric.SourceFormula,
		dataQualityJSON, metric.BusinessContext,
		time.Now(), time.Now(),
	)

	return err
}

// Helper functions
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

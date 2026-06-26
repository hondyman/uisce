package services

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// PerformanceService calculates performance metrics for alternative investments
type PerformanceService struct {
	db               *sql.DB
	altInvestmentSvc *AlternativeInvestmentService
}

// NewPerformanceService creates a new performance service
func NewPerformanceService(db *sql.DB, altInvestmentSvc *AlternativeInvestmentService) *PerformanceService {
	return &PerformanceService{
		db:               db,
		altInvestmentSvc: altInvestmentSvc,
	}
}

// CalculatePerformance calculates performance for an alternative investment
// Returns a subset of performance metrics for API responses
func (s *PerformanceService) CalculatePerformance(ctx context.Context, tenantID uuid.UUID, investmentID uuid.UUID) (*types.AlternativeInvestmentPerformance, error) {
	// For simplicity, delegate to the full calculation method
	asOfDate := time.Now()
	return s.CalculateAndSavePerformanceMetrics(ctx, tenantID, investmentID, asOfDate)
}

// CalculateAndSavePerformanceMetrics calculates all metrics and saves to database
func (s *PerformanceService) CalculateAndSavePerformanceMetrics(ctx context.Context, tenantID uuid.UUID, investmentID uuid.UUID, asOfDate time.Time) (*types.AlternativeInvestmentPerformance, error) {
	// Get investment details
	inv, err := s.altInvestmentSvc.GetInvestment(ctx, tenantID, investmentID)
	if err != nil {
		return nil, err
	}

	// Calculate IRR
	irr, err := s.CalculateIRR(ctx, investmentID, asOfDate)
	if err != nil {
		irr = 0 // Set to 0 if calculation fails
	}

	// Calculate ratios
	tvpi := s.CalculateTVPI(inv)
	dpi := s.CalculateDPI(inv)
	rvpi := s.CalculateRVPI(inv)
	moic := s.CalculateMOIC(inv)

	// Determine J-curve position
	jCurvePosition := s.DetermineJCurvePosition(inv, asOfDate)

	// Save to database
	perf := &types.AlternativeInvestmentPerformance{
		InvestmentID:      investmentID,
		AsOfDate:          asOfDate,
		IRRSinceInception: &irr,
		TVPI:              &tvpi,
		DPI:               &dpi,
		RVPI:              &rvpi,
		MOIC:              &moic,
		JCurvePosition:    &jCurvePosition,
		TotalCalled:       &inv.CapitalCalled,
		TotalDistributed:  &inv.CapitalDistributed,
		NAVValue:          &inv.CurrentNAV,
	}

	query := `
		INSERT INTO alternative_investment_performance (
			investment_id, as_of_date, irr_since_inception,
			tvpi, dpi, rvpi, moic, j_curve_position,
			total_called, total_distributed, nav_value
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (investment_id, as_of_date)
		DO UPDATE SET
			irr_since_inception = EXCLUDED.irr_since_inception,
			tvpi = EXCLUDED.tvpi,
			dpi = EXCLUDED.dpi,
			rvpi = EXCLUDED.rvpi,
			moic = EXCLUDED.moic,
			j_curve_position = EXCLUDED.j_curve_position,
			total_called = EXCLUDED.total_called,
			total_distributed = EXCLUDED.total_distributed,
			nav_value = EXCLUDED.nav_value
		RETURNING id, created_at
	`

	err = s.db.QueryRowContext(ctx, query,
		perf.InvestmentID, perf.AsOfDate, perf.IRRSinceInception,
		perf.TVPI, perf.DPI, perf.RVPI, perf.MOIC, perf.JCurvePosition,
		perf.TotalCalled, perf.TotalDistributed, perf.NAVValue,
	).Scan(&perf.ID, &perf.CreatedAt)
	if err != nil {
		return nil, err
	}

	return perf, nil
}

// CalculateIRR computes Internal Rate of Return using Newton-Raphson method
func (s *PerformanceService) CalculateIRR(ctx context.Context, investmentID uuid.UUID, asOfDate time.Time) (float64, error) {
	// Get all cash flows (capital calls = negative, distributions = positive)
	cashFlows, err := s.getCashFlows(ctx, investmentID, asOfDate)
	if err != nil {
		return 0, err
	}

	if len(cashFlows) == 0 {
		return 0, nil
	}

	// Add current NAV as final positive cash flow
	var currentNAV float64
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(current_nav, 0) FROM alternative_investments WHERE id = $1
	`, investmentID).Scan(&currentNAV)
	if err != nil {
		return 0, err
	}

	if currentNAV > 0 {
		cashFlows = append(cashFlows, types.CashFlow{
			Date:   asOfDate,
			Amount: currentNAV,
		})
	}

	// Newton-Raphson method for IRR calculation
	irr := s.newtonRaphsonIRR(cashFlows, 0.1, 100, 0.0001)

	return irr, nil
}

// getCashFlows retrieves all cash flows for an investment
func (s *PerformanceService) getCashFlows(ctx context.Context, investmentID uuid.UUID, asOfDate time.Time) ([]types.CashFlow, error) {
	var flows []types.CashFlow

	// Capital calls (negative cash flows)
	rows, err := s.db.QueryContext(ctx, `
		SELECT funded_date, -amount_funded
		FROM capital_calls
		WHERE investment_id = $1
		  AND status IN ('FUNDED', 'PARTIALLY_FUNDED')
		  AND funded_date IS NOT NULL
		  AND funded_date <= $2
		ORDER BY funded_date
	`, investmentID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cf types.CashFlow
		if err := rows.Scan(&cf.Date, &cf.Amount); err != nil {
			return nil, err
		}
		flows = append(flows, cf)
	}

	// Distributions (positive cash flows)
	rows, err = s.db.QueryContext(ctx, `
		SELECT distribution_date, amount
		FROM capital_distributions
		WHERE investment_id = $1
		  AND distribution_date <= $2
		ORDER BY distribution_date
	`, investmentID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cf types.CashFlow
		if err := rows.Scan(&cf.Date, &cf.Amount); err != nil {
			return nil, err
		}
		flows = append(flows, cf)
	}

	return flows, nil
}

// newtonRaphsonIRR uses Newton-Raphson method to find IRR
// This is a numerical method to find the root of NPV = 0
func (s *PerformanceService) newtonRaphsonIRR(cashFlows []types.CashFlow, initialGuess float64, maxIterations int, tolerance float64) float64 {
	if len(cashFlows) == 0 {
		return 0
	}

	rate := initialGuess
	baseDate := cashFlows[0].Date

	for i := 0; i < maxIterations; i++ {
		npv := 0.0
		dnpv := 0.0 // Derivative of NPV

		for _, cf := range cashFlows {
			years := cf.Date.Sub(baseDate).Hours() / 24 / 365.25
			discount := math.Pow(1+rate, -years)
			npv += cf.Amount * discount
			dnpv += -years * cf.Amount * discount / (1 + rate)
		}

		// Check for convergence
		if math.Abs(npv) < tolerance {
			return rate
		}

		// Avoid division by zero
		if math.Abs(dnpv) < 1e-10 {
			return 0
		}

		// Newton-Raphson update
		newRate := rate - npv/dnpv

		// Check convergence on rate
		if math.Abs(newRate-rate) < tolerance {
			return newRate
		}

		rate = newRate

		// Bound the rate to reasonable values
		if rate < -0.99 {
			rate = -0.99
		} else if rate > 10.0 {
			rate = 10.0
		}
	}

	return rate
}

// CalculateTVPI calculates Total Value / Paid-In
func (s *PerformanceService) CalculateTVPI(inv *types.AlternativeInvestment) float64 {
	if inv.CapitalCalled == 0 {
		return 0
	}
	totalValue := inv.CapitalDistributed + inv.CurrentNAV
	return totalValue / inv.CapitalCalled
}

// CalculateDPI calculates Distributions / Paid-In
func (s *PerformanceService) CalculateDPI(inv *types.AlternativeInvestment) float64 {
	if inv.CapitalCalled == 0 {
		return 0
	}
	return inv.CapitalDistributed / inv.CapitalCalled
}

// CalculateRVPI calculates Residual Value / Paid-In
func (s *PerformanceService) CalculateRVPI(inv *types.AlternativeInvestment) float64 {
	if inv.CapitalCalled == 0 {
		return 0
	}
	return inv.CurrentNAV / inv.CapitalCalled
}

// CalculateMOIC calculates Multiple on Invested Capital (same as TVPI)
func (s *PerformanceService) CalculateMOIC(inv *types.AlternativeInvestment) float64 {
	return s.CalculateTVPI(inv)
}

// DetermineJCurvePosition determines where in the J-curve the investment is
func (s *PerformanceService) DetermineJCurvePosition(inv *types.AlternativeInvestment, asOfDate time.Time) string {
	yearsElapsed := asOfDate.Sub(inv.InceptionDate).Hours() / 24 / 365.25

	// Get DPI to determine harvesting phase
	dpi := s.CalculateDPI(inv)

	// J-curve logic:
	// INVESTMENT phase: First 3-5 years, DPI < 0.5
	// HARVESTING phase: Years 5-10, DPI > 0.5
	// MATURE phase: After 10 years or DPI > 1.0

	if yearsElapsed < 3 && dpi < 0.5 {
		return "INVESTMENT"
	} else if yearsElapsed < 10 && dpi < 1.0 {
		return "HARVESTING"
	} else {
		return "MATURE"
	}
}

// GetPerformanceMetrics retrieves performance metrics for an investment
func (s *PerformanceService) GetPerformanceMetrics(ctx context.Context, investmentID uuid.UUID, asOfDate *time.Time) (*types.AlternativeInvestmentPerformance, error) {
	var query string
	var args []interface{}

	if asOfDate != nil {
		query = `
			SELECT 
				id, investment_id, as_of_date, irr_since_inception,
				tvpi, dpi, rvpi, moic, pme_kaplan_schoar, pme_direct_alpha,
				benchmark_index, j_curve_position, peer_median_irr,
				peer_top_quartile_irr, percentile_rank, total_called,
				total_distributed, nav_value, created_at
			FROM alternative_investment_performance
			WHERE investment_id = $1 AND as_of_date = $2
			LIMIT 1
		`
		args = []interface{}{investmentID, asOfDate}
	} else {
		query = `
			SELECT 
				id, investment_id, as_of_date, irr_since_inception,
				tvpi, dpi, rvpi, moic, pme_kaplan_schoar, pme_direct_alpha,
				benchmark_index, j_curve_position, peer_median_irr,
				peer_top_quartile_irr, percentile_rank, total_called,
				total_distributed, nav_value, created_at
			FROM alternative_investment_performance
			WHERE investment_id = $1
			ORDER BY as_of_date DESC
			LIMIT 1
		`
		args = []interface{}{investmentID}
	}

	perf := &types.AlternativeInvestmentPerformance{}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&perf.ID, &perf.InvestmentID, &perf.AsOfDate, &perf.IRRSinceInception,
		&perf.TVPI, &perf.DPI, &perf.RVPI, &perf.MOIC, &perf.PMEKaplanSchoar,
		&perf.PMEDirectAlpha, &perf.BenchmarkIndex, &perf.JCurvePosition,
		&perf.PeerMedianIRR, &perf.PeerTopQuartileIRR, &perf.PercentileRank,
		&perf.TotalCalled, &perf.TotalDistributed, &perf.NAVValue, &perf.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return perf, nil
}

// GetPerformanceHistory retrieves performance metrics history
func (s *PerformanceService) GetPerformanceHistory(ctx context.Context, investmentID uuid.UUID, limit int) ([]*types.AlternativeInvestmentPerformance, error) {
	query := `
		SELECT 
			id, investment_id, as_of_date, irr_since_inception,
			tvpi, dpi, rvpi, moic, pme_kaplan_schoar, pme_direct_alpha,
			benchmark_index, j_curve_position, peer_median_irr,
			peer_top_quartile_irr, percentile_rank, total_called,
			total_distributed, nav_value, created_at
		FROM alternative_investment_performance
		WHERE investment_id = $1
		ORDER BY as_of_date DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, investmentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*types.AlternativeInvestmentPerformance
	for rows.Next() {
		perf := &types.AlternativeInvestmentPerformance{}
		err := rows.Scan(
			&perf.ID, &perf.InvestmentID, &perf.AsOfDate, &perf.IRRSinceInception,
			&perf.TVPI, &perf.DPI, &perf.RVPI, &perf.MOIC, &perf.PMEKaplanSchoar,
			&perf.PMEDirectAlpha, &perf.BenchmarkIndex, &perf.JCurvePosition,
			&perf.PeerMedianIRR, &perf.PeerTopQuartileIRR, &perf.PercentileRank,
			&perf.TotalCalled, &perf.TotalDistributed, &perf.NAVValue, &perf.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, perf)
	}

	return metrics, rows.Err()
}

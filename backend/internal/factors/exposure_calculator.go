package factors

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ExposureCalculator computes factor exposures via regression analysis
type ExposureCalculator struct {
	model FactorModel
}

// NewExposureCalculator creates a new exposure calculator
func NewExposureCalculator(model FactorModel) *ExposureCalculator {
	return &ExposureCalculator{
		model: model,
	}
}

// Calculate computes factor exposures for a portfolio
func (ec *ExposureCalculator) Calculate(ctx context.Context, portfolioReturns PortfolioReturns, startDate, endDate time.Time) ([]FactorExposure, *RegressionResult, error) {
	// Get factor returns for the period
	factorReturnsMap := make(map[string][]float64)
	
	// TODO: In production, fetch actual factor returns from model
	// For now, use placeholder
	
	// Run regression
	result := ec.runRegression(portfolioReturns.Data, factorReturnsMap)
	
	// Convert betas to exposures with narratives
	exposures := ec.betasToExposures(result)
	
	return exposures, result, nil
}

// CalculateAttribution decomposes returns into factor contributions
func (ec *ExposureCalculator) CalculateAttribution(ctx context.Context, portfolioReturns PortfolioReturns, exposures []FactorExposure, period time.Duration) (*AttributionResult, error) {
	// Calculate total return
	totalReturn := ec.calculateTotalReturn(portfolioReturns.Data)
	
	// Attribute returns to factors
	factorReturns := make(map[string]float64)
	explainedReturn := 0.0
	
	for _, exp := range exposures {
		// Factor contribution = beta * factor_return
		// Placeholder: use contribution directly
		factorReturn := exp.Contribution * totalReturn
		factorReturns[exp.Factor] = factorReturn
		explainedReturn += factorReturn
	}
	
	selectionReturn := totalReturn - explainedReturn
	
	return &AttributionResult{
		TotalReturn:       totalReturn,
		FactorReturns:     factorReturns,
		SelectionReturn:   selectionReturn,
		ExplainedReturn:   explainedReturn,
		UnexplainedReturn: selectionReturn,
	}, nil
}

// SimulateScenario applies a shock to factors and estimates impact
func (ec *ExposureCalculator) SimulateScenario(ctx context.Context, exposures []FactorExposure, shock ScenarioShock) (*ScenarioResult, error) {
	factorImpacts := make(map[string]float64)
	totalImpact := 0.0
	
	for _, exp := range exposures {
		impact := 0.0
		if exp.Factor == shock.Factor {
			// This factor receives the shock
			impact = exp.Contribution * shock.ShockPct
		}
		factorImpacts[exp.Factor] = impact
		totalImpact += impact
	}
	
	// Generate narrative
	narrative := fmt.Sprintf(
		"If %s experiences a %+.1f%% shock, portfolio impact is estimated at %+.2f%%. "+
			"This factor explains %.0f%% of the total impact.",
		shock.Factor,
		shock.ShockPct*100,
		totalImpact*100,
		(factorImpacts[shock.Factor]/totalImpact)*100,
	)
	
	return &ScenarioResult{
		Scenario:        shock,
		PortfolioImpact: totalImpact,
		FactorImpacts:   factorImpacts,
		Narrative:       narrative,
	}, nil
}

// runRegression performs OLS regression (placeholder implementation)
func (ec *ExposureCalculator) runRegression(portfolioData []PortfolioDataPoint, factorReturns map[string][]float64) *RegressionResult {
	// TODO: Implement actual regression using gonum or similar
	// For now, return placeholder results
	
	betas := make(map[string]float64)
	tStats := make(map[string]float64)
	pValues := make(map[string]float64)
	
	for _, factorName := range ec.model.Factors() {
		switch factorName {
		case "Market":
			betas[factorName] = 1.05
			tStats[factorName] = 15.2
			pValues[factorName] = 0.001
		case "SMB":
			betas[factorName] = 0.25
			tStats[factorName] = 4.1
			pValues[factorName] = 0.01
		case "HML":
			betas[factorName] = -0.15
			tStats[factorName] = -2.3
			pValues[factorName] = 0.05
		case "RMW":
			betas[factorName] = 0.18
			tStats[factorName] = 3.5
			pValues[factorName] = 0.02
		case "CMA":
			betas[factorName] = -0.08
			tStats[factorName] = -1.2
			pValues[factorName] = 0.25
		}
	}
	
	return &RegressionResult{
		Alpha:       0.0012, // 12 bps monthly alpha
		Betas:       betas,
		RSquared:    0.92,
		AdjRSquared: 0.91,
		TStats:      tStats,
		PValues:     pValues,
		Residuals:   []float64{}, // TODO: calculate residuals
	}
}

// betasToExposures converts regression betas to factor exposures
func (ec *ExposureCalculator) betasToExposures(result *RegressionResult) []FactorExposure {
	var exposures []FactorExposure
	
	for factor, beta := range result.Betas {
		exp := FactorExposure{
			Factor:       factor,
			Contribution: beta,
			Significance: result.TStats[factor],
			PValue:       result.PValues[factor],
		}
		
		// Will be enhanced with narrative generation
		exposures = append(exposures, exp)
	}
	
	return exposures
}

// calculateTotalReturn computes cumulative return
func (ec *ExposureCalculator) calculateTotalReturn(data []PortfolioDataPoint) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	cumulative := 1.0
	for _, point := range data {
		cumulative *= (1 + point.Return)
	}
	
	return cumulative - 1.0
}

// CalculateRollingExposures computes exposures over rolling windows
func (ec *ExposureCalculator) CalculateRollingExposures(ctx context.Context, portfolioReturns PortfolioReturns, windowDays int) ([][]FactorExposure, error) {
	// TODO: Implement rolling window calculation
	// This would compute exposures for each rolling period
	return nil, nil
}

// Helper functions for statistical calculations

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := mean(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-m, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}

func covariance(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}
	meanX := mean(x)
	meanY := mean(y)
	cov := 0.0
	for i := range x {
		cov += (x[i] - meanX) * (y[i] - meanY)
	}
	return cov / float64(len(x))
}

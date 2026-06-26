package factor

import (
	"fmt"
)

// AttributionResult represents the decomposition of returns
type AttributionResult struct {
	TotalReturn       float64
	AlphaContribution float64
	FactorContributions map[string]float64 // Factor Name -> Contribution
	Residual          float64
}

// AttributionService handles return decomposition
type AttributionService struct {
	regressionService *RegressionService
}

func NewAttributionService(regressionService *RegressionService) *AttributionService {
	return &AttributionService{
		regressionService: regressionService,
	}
}

// DecomposeReturns breaks down portfolio returns into factor components
// portfolioReturns: time-series of portfolio returns
// factorReturns: map of factor slug to time-series of returns
func (s *AttributionService) DecomposeReturns(portfolioReturns []float64, factorReturns map[string][]float64) (*AttributionResult, error) {
	if len(portfolioReturns) == 0 {
		return nil, fmt.Errorf("empty portfolio returns")
	}

	// 1. Prepare data for regression
	// We need to align the factor returns into a matrix X
	// For simplicity, assuming all slices are same length and aligned by index
	
	var factorNames []string
	var x [][]float64
	
	n := len(portfolioReturns)
	numFactors := len(factorReturns)
	
	if numFactors > 0 {
		x = make([][]float64, n)
		for i := 0; i < n; i++ {
			x[i] = make([]float64, numFactors)
		}
		
		col := 0
		for name, returns := range factorReturns {
			if len(returns) != n {
				return nil, fmt.Errorf("mismatched length for factor %s", name)
			}
			factorNames = append(factorNames, name)
			for i := 0; i < n; i++ {
				x[i][col] = returns[i]
			}
			col++
		}
	}

	// 2. Run Regression to get Betas
	// We use the full period for static attribution, or could use rolling for time-series
	regResult, err := s.regressionService.PerformOLS(portfolioReturns, x)
	if err != nil {
		return nil, fmt.Errorf("regression failed: %w", err)
	}

	// 3. Calculate Contributions
	// Contribution = Beta * Cumulative Factor Return (simplified)
	// Or Average Return decomposition: R_p = Alpha + Sum(Beta_i * R_f_i) + epsilon
	
	contributions := make(map[string]float64)
	totalFactorContribution := 0.0
	
	for i, beta := range regResult.Betas {
		if i >= len(factorNames) {
			break 
		}
		name := factorNames[i]
		
		// Calculate average factor return for the period
		sumFactor := 0.0
		for _, r := range factorReturns[name] {
			sumFactor += r
		}
		avgFactorReturn := sumFactor / float64(n)
		
		contribution := beta * avgFactorReturn
		contributions[name] = contribution
		totalFactorContribution += contribution
	}

	// Calculate average portfolio return
	sumPortfolio := 0.0
	for _, r := range portfolioReturns {
		sumPortfolio += r
	}
	avgPortfolioReturn := sumPortfolio / float64(n)

	// Alpha is the intercept (or we can back it out)
	// Alpha from regression is per-period
	alphaContribution := regResult.Alpha

	return &AttributionResult{
		TotalReturn:       avgPortfolioReturn,
		AlphaContribution: alphaContribution,
		FactorContributions: contributions,
		Residual: avgPortfolioReturn - (alphaContribution + totalFactorContribution),
	}, nil
}

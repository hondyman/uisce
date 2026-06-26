package factors

import (
	"errors"
	"math/rand"
	"time"
)

// FamaFrench5 implements the FactorModel interface for the 5-factor model
type FamaFrench5 struct{}

func NewFamaFrench5() *FamaFrench5 {
	return &FamaFrench5{}
}

func (m *FamaFrench5) GetFactorNames() []string {
	return []string{FactorMarket, FactorSMB, FactorHML, FactorRMW, FactorCMA}
}

// CalculateExposures computes factor loadings.
// Note: In a real implementation, this would perform multiple linear regression (OLS)
// of asset excess returns against the 5 factor return series.
// For this demo, we simulate realistic exposures based on the input "returns" (or just mock it).
func (m *FamaFrench5) CalculateExposures(returns []float64, benchmarkReturns map[string][]float64) (FactorAnalysisResult, error) {
	if len(returns) == 0 {
		return FactorAnalysisResult{}, errors.New("insufficient return data")
	}

	// Mocking the regression results for demonstration
	// In reality: betas = Inverse(X'X) * X'Y
	
	// Simulate "Value" tilt if returns are volatile (just a heuristic for demo variety)
	isValue := rand.Float64() > 0.5
	
	betaMkt := 0.9 + (rand.Float64() * 0.3) // 0.9 - 1.2
	betaSMB := -0.2 + (rand.Float64() * 0.4) // -0.2 - 0.2
	betaHML := 0.1
	if isValue {
		betaHML = 0.4 + (rand.Float64() * 0.2)
	} else {
		betaHML = -0.1 + (rand.Float64() * 0.2)
	}
	
	exposures := []FactorExposure{
		{FactorName: FactorMarket, Beta: betaMkt, TStat: 12.5},
		{FactorName: FactorSMB, Beta: betaSMB, TStat: 1.8}, // Size
		{FactorName: FactorHML, Beta: betaHML, TStat: 3.2}, // Value
		{FactorName: FactorRMW, Beta: 0.2, TStat: 2.1},     // Profitability
		{FactorName: FactorCMA, Beta: 0.05, TStat: 0.8},    // Investment
	}

	return FactorAnalysisResult{
		Date:      time.Now(),
		ModelName: "Fama-French 5-Factor",
		R2:        0.85 + (rand.Float64() * 0.10),
		Alpha:     0.0015, // 15 bps monthly alpha
		Exposures: exposures,
	}, nil
}

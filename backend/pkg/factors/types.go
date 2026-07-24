package factors

import "time"

// FactorExposure represents the sensitivity of an asset or portfolio to a specific factor
type FactorExposure struct {
	FactorName string  `json:"factor_name"`
	Beta       float64 `json:"beta"`
	TStat      float64 `json:"t_stat"` // Statistical significance
}

// FactorAnalysisResult contains the full analysis for an entity
type FactorAnalysisResult struct {
	EntityID   string           `json:"entity_id"` // PortfolioID or Symbol
	Date       time.Time        `json:"date"`
	ModelName  string           `json:"model_name"` // e.g., "FamaFrench5"
	R2         float64          `json:"r_squared"`
	Alpha      float64          `json:"alpha"`      // Unexplained return
	Exposures  []FactorExposure `json:"exposures"`
}

// FactorModel defines the interface for different factor models (CAPM, FF3, FF5, etc.)
type FactorModel interface {
	// CalculateExposures computes factor betas for a given return series
	CalculateExposures(returns []float64, benchmarkReturns map[string][]float64) (FactorAnalysisResult, error)
	
	// GetFactorNames returns the list of factors in this model
	GetFactorNames() []string
}

// Common Factor Constants
const (
	FactorMarket      = "Mkt-RF"
	FactorSMB         = "SMB" // Size
	FactorHML         = "HML" // Value
	FactorRMW         = "RMW" // Profitability
	FactorCMA         = "CMA" // Investment
	FactorMomentum    = "MOM"
)

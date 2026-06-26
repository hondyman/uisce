package factors

import (
	"time"
)

// FactorModel defines the interface for all factor models
type FactorModel interface {
	// Name returns the model name (e.g., "Fama-French 5-Factor")
	Name() string
	
	// Type returns the model type (e.g., "fama_french", "barra", "custom")
	Type() string
	
	// Factors returns the list of factor names
	Factors() []string
	
	// ComputeExposures calculates factor exposures for a portfolio
	ComputeExposures(holdings []Holding, startDate, endDate time.Time) ([]FactorExposure, error)
}

// Holding represents a portfolio position
type Holding struct {
	Ticker   string    `json:"ticker"`
	Quantity float64   `json:"quantity"`
	AsOfDate time.Time `json:"as_of_date"`
	Currency string    `json:"currency"`
}

// FactorExposure represents a single factor's contribution
type FactorExposure struct {
	Factor       string   `json:"factor"`        // e.g., "Market", "SMB", "HML"
	Contribution float64  `json:"contribution"`  // % contribution to returns
	Narrative    string   `json:"narrative"`     // Plain English explanation
	Significance float64  `json:"significance"`  // T-statistic
	Sources      []string `json:"sources"`       // Tickers driving this exposure
	PValue       float64  `json:"p_value"`       // Statistical p-value
}

// FactorReturns represents time-series returns for a factor
type FactorReturns struct {
	Factor string             `json:"factor"`
	Data   []FactorDataPoint  `json:"data"`
}

// FactorDataPoint represents a single observation
type FactorDataPoint struct {
	Date   time.Time `json:"date"`
	Return float64   `json:"return"` // Factor return in decimal (e.g., 0.05 = 5%)
}

// RegressionResult contains regression statistics
type RegressionResult struct {
	Alpha       float64              `json:"alpha"`        // Intercept
	Betas       map[string]float64   `json:"betas"`        // Factor loadings
	RSquared    float64              `json:"r_squared"`    // Model fit
	AdjRSquared float64              `json:"adj_r_squared"`// Adjusted R-squared
	TStats      map[string]float64   `json:"t_stats"`      // T-statistics
	PValues     map[string]float64   `json:"p_values"`     // P-values
	Residuals   []float64            `json:"residuals"`    // Regression residuals
}

// PortfolioReturns represents historical returns for a portfolio
type PortfolioReturns struct {
	PortfolioID string              `json:"portfolio_id"`
	Data        []PortfolioDataPoint `json:"data"`
}

// PortfolioDataPoint represents a single portfolio return observation
type PortfolioDataPoint struct {
	Date   time.Time `json:"date"`
	Return float64   `json:"return"` // Portfolio return in decimal
}

// AttributionResult contains factor attribution analysis
type AttributionResult struct {
	TotalReturn      float64                   `json:"total_return"`
	FactorReturns    map[string]float64        `json:"factor_returns"`    // Return attributed to each factor
	SelectionReturn  float64                   `json:"selection_return"`  // Stock-specific return
	ExplainedReturn  float64                   `json:"explained_return"`  // Sum of factor returns
	UnexplainedReturn float64                  `json:"unexplained_return"`// Residual return
}

// ScenarioShock represents a hypothetical stress to a factor
type ScenarioShock struct {
	Factor      string  `json:"factor"`
	ShockBps    int     `json:"shock_bps"`    // Shock in basis points
	ShockPct    float64 `json:"shock_pct"`    // Shock in percentage
	Description string  `json:"description"`   // e.g., "Rates rise 50bps"
}

// ScenarioResult contains the impact of a scenario shock
type ScenarioResult struct {
	Scenario        ScenarioShock      `json:"scenario"`
	PortfolioImpact float64            `json:"portfolio_impact"` // Total impact in %
	FactorImpacts   map[string]float64 `json:"factor_impacts"`   // Impact by factor
	Narrative       string             `json:"narrative"`        // Plain English explanation
}

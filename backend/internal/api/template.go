package api

import (
	"fmt"

	"github.com/hondyman/semlayer/backend/models"
)

// Template is the top-level struct for a financial calculation template.
type Template struct {
	NodeID      string         `json:"node_id"`
	NodeType    string         `json:"node_type"`
	Domain      string         `json:"domain"`
	Category    string         `json:"category"`
	Subcategory string         `json:"subcategory"`
	Version     string         `json:"version"`
	Owner       string         `json:"owner"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Lineage     []string       `json:"lineage,omitempty"`
	AccessType  string         `json:"access_type,omitempty"`
	Governance  *Governance    `json:"governance,omitempty"`
	DataSources map[string]any `json:"data_sources"`
	Dimensions  map[string]any `json:"dimensions,omitempty"`
	Measures    map[string]any `json:"measures,omitempty"`
	Financial   FinancialCalc  `json:"financial_calc"`
}

// Governance holds metadata for stewardship and status.
type Governance struct {
	Status       string `json:"status,omitempty"`
	StewardGroup string `json:"steward_group,omitempty"`
	SchemaHash   string `json:"schema_hash,omitempty"`
	SLA          *SLA   `json:"sla,omitempty"`
}

// SLA defines the Service Level Agreement for the template.
type SLA struct {
	RefreshFrequency string `json:"refresh_frequency,omitempty"`
	MaxLatency       string `json:"max_latency,omitempty"`
}

// FinancialCalc is a polymorphic struct holding parameters for a specific calculation.
type FinancialCalc struct {
	Type             string               `json:"type"`
	CashFlows        []CashFlow           `json:"cash_flows,omitempty"`
	CashFlowsDated   []DatedCashFlow      `json:"cash_flows_dated,omitempty"`
	Guess            float64              `json:"guess,omitempty"`
	DiscountRate     float64              `json:"discount_rate,omitempty"`
	GrowthRate       float64              `json:"growth_rate,omitempty"`
	FinanceRate      float64              `json:"finance_rate,omitempty"`
	ReinvestRate     float64              `json:"reinvest_rate,omitempty"`
	SubperiodReturns []float64            `json:"subperiod_returns,omitempty"`
	StartValue       float64              `json:"start_value,omitempty"`
	EndValue         float64              `json:"end_value,omitempty"`
	Years            float64              `json:"years,omitempty"`
	Numerator        string               `json:"numerator,omitempty"`
	Denominator      string               `json:"denominator,omitempty"`
	Components       []RatioComponent     `json:"components,omitempty"`
	Population       bool                 `json:"population,omitempty"`
	AverageReturn    float64              `json:"average_return,omitempty"`
	RiskFreeRate     float64              `json:"risk_free_rate,omitempty"`
	StdDev           float64              `json:"std_dev,omitempty"`
	AssetReturns     []float64            `json:"asset_returns,omitempty"`
	BenchReturns     []float64            `json:"bench_returns,omitempty"`
	PD               float64              `json:"pd,omitempty"`
	LGD              float64              `json:"lgd,omitempty"`
	EAD              float64              `json:"ead,omitempty"`
	Principal        float64              `json:"principal,omitempty"`
	Rate             float64              `json:"rate,omitempty"`
	Periods          int                  `json:"periods,omitempty"`
	Investments      []WIRRInvestment     `json:"investments,omitempty"`
	Triangle         *ChainLadderTriangle `json:"triangle,omitempty"`
	// Portfolio Analytics fields
	Returns          []float64   `json:"returns,omitempty"`           // Expected returns vector
	Covariance       [][]float64 `json:"covariance,omitempty"`        // Covariance matrix
	LongOnly         bool        `json:"long_only,omitempty"`         // Long-only constraint
	TargetReturn     *float64    `json:"target_return,omitempty"`     // Target return for optimization
	Lambda           *float64    `json:"lambda,omitempty"`            // Risk aversion parameter
	BenchmarkWeights []float64   `json:"benchmark_weights,omitempty"` // Benchmark weights for tracking
	// Stochastic Model fields
	InitialValues  []float64 `json:"initial_values,omitempty"`  // Initial values for GBM
	DriftRates     []float64 `json:"drift_rates,omitempty"`     // Drift rates for GBM
	Volatilities   []float64 `json:"volatilities,omitempty"`    // Volatilities for GBM
	TimeHorizon    float64   `json:"time_horizon,omitempty"`    // Time horizon
	NumSteps       int       `json:"num_steps,omitempty"`       // Number of time steps
	NumSimulations int       `json:"num_simulations,omitempty"` // Number of Monte Carlo simulations
	StrikePrice    float64   `json:"strike_price,omitempty"`    // Strike price for options
	MeanReversion  float64   `json:"mean_reversion,omitempty"`  // Mean reversion speed for OU
	LongTermMean   float64   `json:"long_term_mean,omitempty"`  // Long-term mean for OU
	// Additional fields for JSON template compatibility
	Mu        []float64 `json:"mu,omitempty"`        // Expected returns vector (alias for Returns)
	Weights   []float64 `json:"weights,omitempty"`   // Portfolio weights
	Points    int       `json:"points,omitempty"`    // Number of points for efficient frontier
	S0        []float64 `json:"S0,omitempty"`        // Initial stock prices for GBM
	Sigma     []float64 `json:"sigma,omitempty"`     // Volatilities for GBM (alias for Volatilities)
	T         float64   `json:"T,omitempty"`         // Time horizon (alias for TimeHorizon)
	Steps     int       `json:"steps,omitempty"`     // Number of steps (alias for NumSteps)
	Seed      int64     `json:"seed,omitempty"`      // Random seed
	X0        float64   `json:"x0,omitempty"`        // Initial value for OU
	Theta     float64   `json:"theta,omitempty"`     // Mean reversion speed for OU (alias for MeanReversion)
	Mean      float64   `json:"mean,omitempty"`      // Long-term mean for OU (alias for LongTermMean)
	Sims      int       `json:"sims,omitempty"`      // Number of simulations (alias for NumSimulations)
	Strike    float64   `json:"strike,omitempty"`    // Strike price for options (alias for StrikePrice)
	R         float64   `json:"r,omitempty"`         // Risk-free rate (alias for RiskFreeRate)
	Quantiles []float64 `json:"quantiles,omitempty"` // Quantiles for Monte Carlo
	// Insurance-specific fields
	PremiumAmount           float64 `json:"premium_amount,omitempty"`            // Premium amount for insurance calculations
	ClaimAmount             float64 `json:"claim_amount,omitempty"`              // Claim amount for insurance calculations
	Expenses                float64 `json:"expenses,omitempty"`                  // Operating expenses
	InvestmentIncome        float64 `json:"investment_income,omitempty"`         // Investment income
	NetClaims               float64 `json:"net_claims,omitempty"`                // Claims net of reinsurance
	NetPremiums             float64 `json:"net_premiums,omitempty"`              // Premiums net of reinsurance
	ActualClaimsPaid        float64 `json:"actual_claims_paid,omitempty"`        // Actual claims paid
	ReservesHeld            float64 `json:"reserves_held,omitempty"`             // Reserves held
	CurrentYearReserves     float64 `json:"current_year_reserves,omitempty"`     // Current year reserves
	PriorYearReserves       float64 `json:"prior_year_reserves,omitempty"`       // Prior year reserves
	LossReserves            float64 `json:"loss_reserves,omitempty"`             // Loss reserves
	PolicyholderSurplus     float64 `json:"policyholder_surplus,omitempty"`      // Policyholder surplus
	AvailableSolvencyMargin float64 `json:"available_solvency_margin,omitempty"` // Available solvency margin
	RequiredSolvencyMargin  float64 `json:"required_solvency_margin,omitempty"`  // Required solvency margin
	InvestedAssets          float64 `json:"invested_assets,omitempty"`           // Invested assets
	RenewedPolicies         int     `json:"renewed_policies,omitempty"`          // Number of renewed policies
	EligibleRenewals        int     `json:"eligible_renewals,omitempty"`         // Number of eligible renewals
	CurrentPremiumAmount    float64 `json:"current_premium_amount,omitempty"`    // Current period premium amount
	PriorPremiumAmount      float64 `json:"prior_premium_amount,omitempty"`      // Prior period premium amount
	Claims                  int     `json:"claims,omitempty"`                    // Number of claims
	Policies                int     `json:"policies,omitempty"`                  // Number of policies
	// Private Markets fields
	CashFlowsExpression       string `json:"cash_flows_expression,omitempty"`       // SQL expression for cash flows array
	DatesExpression           string `json:"dates_expression,omitempty"`            // SQL expression for dates array
	BenchmarkPricesExpression string `json:"benchmark_prices_expression,omitempty"` // SQL expression for benchmark prices
	Series                    string `json:"series,omitempty"`                      // Series field for aggregations
	OrderBy                   string `json:"order_by,omitempty"`                    // Order by clause
	Expression                string `json:"expression,omitempty"`                  // General expression field
	Engine            string     `json:"engine,omitempty"`         // internal, cube, spark
	ExecutionType     string     `json:"execution_type,omitempty"` // realtime, batch
	// Quant Finance fields
	ConfidenceLevel   float64      `json:"confidence_level,omitempty"`    // Confidence level for VaR/CVaR
	HoldingPeriodDays float64      `json:"holding_period_days,omitempty"` // Holding period in days
	OptionType        string       `json:"option_type,omitempty"`         // Option type (call/put)
	DividendYield     float64      `json:"dividend_yield,omitempty"`      // Dividend yield
	MarketPrice       float64      `json:"market_price,omitempty"`        // Market price for implied vol
	YieldToMaturity   float64      `json:"yield_to_maturity,omitempty"`   // Yield to maturity for bonds
	Frequency         int          `json:"frequency,omitempty"`           // Payment frequency
	Instruments       []Instrument `json:"instruments,omitempty"`         // Instruments for yield curve
	Compounding       string       `json:"compounding,omitempty"`         // Compounding method
	// Risk Management fields
	Scenarios            []Scenario         `json:"scenarios,omitempty"`              // Scenarios for stress testing
	PortfolioValue       float64            `json:"portfolio_value,omitempty"`        // Portfolio value for stress testing
	AssetReturnsMatrix   [][]float64        `json:"asset_returns_matrix,omitempty"`   // Matrix of asset returns for correlation
	CreditScore          int                `json:"credit_score,omitempty"`           // Credit score for PD calculation
	PDModel              string             `json:"pd_model,omitempty"`               // PD model type
	RiskFactors          map[string]float64 `json:"risk_factors,omitempty"`           // Risk factors map
	CollateralValue      float64            `json:"collateral_value,omitempty"`       // Collateral value for LGD
	ExposureAmount       float64            `json:"exposure_amount,omitempty"`        // Exposure amount
	RecoveryRate         float64            `json:"recovery_rate,omitempty"`          // Recovery rate
	CollateralType       string             `json:"collateral_type,omitempty"`        // Type of collateral
	CurrentExposure      float64            `json:"current_exposure,omitempty"`       // Current exposure amount
	CreditLimit          float64            `json:"credit_limit,omitempty"`           // Credit limit
	DrawnAmount          float64            `json:"drawn_amount,omitempty"`           // Amount drawn
	UndrawnCommitment    float64            `json:"undrawn_commitment,omitempty"`     // Undrawn commitment
	ExposurePeriod       int                `json:"exposure_period,omitempty"`        // Exposure period in months
	Exposures            []float64          `json:"exposures,omitempty"`              // Array of exposures for credit VaR
	PDs                  []float64          `json:"pds,omitempty"`                    // Array of PDs for credit VaR
	LGDs                 []float64          `json:"lgds,omitempty"`                   // Array of LGDs for credit VaR
	PrimaryLossFactors   map[string]float64 `json:"primary_loss_factors,omitempty"`   // FAIR primary loss factors
	SecondaryLossFactors map[string]float64 `json:"secondary_loss_factors,omitempty"` // FAIR secondary loss factors
	RiskCategory         string             `json:"risk_category,omitempty"`          // Risk category for RCSA
	BusinessUnit         string             `json:"business_unit,omitempty"`          // Business unit for RCSA
	KRIs                 []KRI              `json:"kris,omitempty"`                   // Key Risk Indicators
	// Compliance & Regulatory fields
	HighQualityLiquidAssets float64            `json:"high_quality_liquid_assets,omitempty"` // HQLA for LCR
	NetCashOutflows         float64            `json:"net_cash_outflows,omitempty"`          // Net cash outflows for LCR
	RequiredLCR             float64            `json:"required_lcr,omitempty"`               // Required LCR ratio
	AvailableStableFunding  float64            `json:"available_stable_funding,omitempty"`   // Available stable funding for NSFR
	RequiredStableFunding   float64            `json:"required_stable_funding,omitempty"`    // Required stable funding for NSFR
	RequiredNSFR            float64            `json:"required_nsfr,omitempty"`              // Required NSFR ratio
	Tier1Capital            float64            `json:"tier1_capital,omitempty"`              // Tier 1 capital for leverage ratio
	TotalExposures          float64            `json:"total_exposures,omitempty"`            // Total exposures for leverage ratio
	RequiredLeverageRatio   float64            `json:"required_leverage_ratio,omitempty"`    // Required leverage ratio
	MarketRiskSCR           float64            `json:"market_risk_scr,omitempty"`            // Market risk SCR
	CreditRiskSCR           float64            `json:"credit_risk_scr,omitempty"`            // Credit risk SCR
	OperationalRiskSCR      float64            `json:"operational_risk_scr,omitempty"`       // Operational risk SCR
	InsuranceRiskSCR        float64            `json:"insurance_risk_scr,omitempty"`         // Insurance risk SCR
	CorrelationMatrix       [][]float64        `json:"correlation_matrix,omitempty"`         // Correlation matrix for SCR
	WrittenPremiums         float64            `json:"written_premiums,omitempty"`           // Written premiums for MCR
	TechnicalProvisions     float64            `json:"technical_provisions,omitempty"`       // Technical provisions for MCR
	MCRFloor                float64            `json:"mcr_floor,omitempty"`                  // MCR floor percentage
	MCRCap                  float64            `json:"mcr_cap,omitempty"`                    // MCR cap percentage
	TransactionAmount       float64            `json:"transaction_amount,omitempty"`         // Transaction amount for AML
	TransactionFrequency    int                `json:"transaction_frequency,omitempty"`      // Transaction frequency
	CustomerRiskProfile     string             `json:"customer_risk_profile,omitempty"`      // Customer risk profile
	GeographicRisk          string             `json:"geographic_risk,omitempty"`            // Geographic risk level
	ProductRisk             string             `json:"product_risk,omitempty"`               // Product risk level
	RiskThresholds          map[string]float64 `json:"risk_thresholds,omitempty"`            // Risk thresholds map
	CustomerProfile         CustomerProfile    `json:"customer_profile,omitempty"`           // Customer profile for KYC
	RiskWeights             map[string]float64 `json:"risk_weights,omitempty"`               // Risk weights for scoring
	ExecutionPrice          float64            `json:"execution_price,omitempty"`            // Execution price for slippage
	BenchmarkPrice          float64            `json:"benchmark_price,omitempty"`            // Benchmark price for slippage
	OrderSize               int                `json:"order_size,omitempty"`                 // Order size
	MarketVolatility        float64            `json:"market_volatility,omitempty"`          // Market volatility
	SlippageThreshold       float64            `json:"slippage_threshold,omitempty"`         // Slippage threshold
	TradeVolume             int                `json:"trade_volume,omitempty"`               // Trade volume for surveillance
	AverageVolume           int                `json:"average_volume,omitempty"`             // Average volume
	PriceMovement           float64            `json:"price_movement,omitempty"`             // Price movement percentage
	NormalPriceStdDev       float64            `json:"normal_price_std_dev,omitempty"`       // Normal price standard deviation
	TimeWindow              int                `json:"time_window,omitempty"`                // Time window in seconds
	AlertThresholds         map[string]float64 `json:"alert_thresholds,omitempty"`           // Alert thresholds map
	// Excel Formula fields
	Formula   string                 `json:"formula,omitempty"`   // Excel formula string
	Arguments map[string]interface{} `json:"arguments,omitempty"` // Formula arguments mapping
}

// CashFlow represents a single cash flow at a specific period.
type CashFlow struct {
	Amount float64 `json:"amount"`
	Period int     `json:"period"`
}

// DatedCashFlow represents a cash flow on a specific date.
type DatedCashFlow struct {
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}

// RatioComponent is a numerator/denominator pair for ratio sums.
type RatioComponent struct {
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}

// WIRRInvestment is a set of cash flows with a corresponding weight.
type WIRRInvestment struct {
	Weight    float64    `json:"weight"`
	CashFlows []CashFlow `json:"cash_flows"`
}

// ChainLadderTriangle holds data for IBNR calculations.
type ChainLadderTriangle struct {
	OriginPeriods []string    `json:"origin_periods"`
	DevFactors    []float64   `json:"dev_factors"`
	Paid          [][]float64 `json:"paid"`
}

// Instrument represents a financial instrument for yield curve bootstrapping.
type Instrument struct {
	MaturityYears float64 `json:"maturity_years"`
	ParYield      float64 `json:"par_yield"`
}

// Scenario represents a stress testing scenario with shocks to risk factors.
type Scenario struct {
	Name   string             `json:"name"`
	Shocks map[string]float64 `json:"shocks"`
}

// KRI represents a Key Risk Indicator with threshold monitoring.
type KRI struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Severity  string  `json:"severity"`
	Category  string  `json:"category"`
}

// CustomerProfile represents customer information for KYC risk assessment.
type CustomerProfile struct {
	Jurisdiction       string `json:"jurisdiction"`
	CustomerType       string `json:"customer_type"`
	ProductType        string `json:"product_type"`
	TransactionPattern string `json:"transaction_pattern"`
}

// DrillDownLocator specifies the coordinates for drill-down
type DrillDownLocator struct {
	XValues []interface{} `json:"xValues"`
	YValues []interface{} `json:"yValues"`
}

// PivotConfig specifies the pivot configuration
type PivotConfig struct {
	X []string `json:"x,omitempty"`
	Y []string `json:"y,omitempty"`
}

// Query represents a query for drill-down
type Query struct {
	Measures       []string        `json:"measures"`
	Dimensions     []string        `json:"dimensions"`
	Filters        []models.Filter `json:"filters"`
	TimeDimensions []TimeDimension `json:"timeDimensions"`
}

// PivotRow represents a row in pivoted data
type PivotRow struct {
	XValues      []interface{}   `json:"xValues"`
	YValuesArray [][]interface{} `json:"yValuesArray"`
}

// TimeDimension represents a time dimension in a query
type TimeDimension struct {
	Dimension   string   `json:"dimension"`
	DateRange   []string `json:"dateRange,omitempty"`
	Granularity string   `json:"granularity,omitempty"`
}

// ResultMeasure represents a measure in the result set with drill-down capabilities
type ResultMeasure struct {
	Name         string      `json:"name"`
	Value        interface{} `json:"value"`
	DrillMembers []string    `json:"drillMembers,omitempty"`
}

// ResultSet represents the result of a calculation with drill-down and pivot capabilities
type ResultSet struct {
	Data     interface{}     `json:"data"`
	Measures []ResultMeasure `json:"measures"`
}

// DrillDown performs drill-down on a measure
func (m *ResultMeasure) DrillDown(locator DrillDownLocator, pivotConfig *PivotConfig) *Query {
	// Implementation for drill-down
	query := &Query{
		Measures:       []string{m.Name},
		Dimensions:     m.DrillMembers,
		Filters:        []models.Filter{},
		TimeDimensions: []TimeDimension{},
	}

	// Add filters based on locator
	for _, xVal := range locator.XValues {
		// Add dimension filter based on xVal
		filter := models.Filter{
			Field:  "dimension", // This should be the actual dimension name
			Op:     "equals",
			Values: []string{fmt.Sprintf("%v", xVal)},
		}
		query.Filters = append(query.Filters, filter)
	}

	for _, yVal := range locator.YValues {
		// Add measure filter based on yVal
		filter := models.Filter{
			Field:  m.Name,
			Op:     "equals",
			Values: []string{fmt.Sprintf("%v", yVal)},
		}
		query.Filters = append(query.Filters, filter)
	}

	return query
}

// Pivot performs pivoting on the result set
func (rs *ResultSet) Pivot(pivotConfig *PivotConfig) []PivotRow {
	// Implementation for pivoting
	var result []PivotRow

	// This is a simplified implementation
	// In a real implementation, you would process the data according to pivotConfig
	// For now, return empty result
	return result
}

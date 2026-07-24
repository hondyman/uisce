package financial

import "time"

// Core financial ontology types for investment management

// InstrumentType represents the classification of a financial instrument
type InstrumentType string

const (
	InstrumentTypeEquity       InstrumentType = "equity"
	InstrumentTypeFixedIncome  InstrumentType = "fixed_income"
	InstrumentTypeDerivative   InstrumentType = "derivative"
	InstrumentTypeCommodity    InstrumentType = "commodity"
	InstrumentTypeCurrency     InstrumentType = "currency"
	InstrumentTypeCash         InstrumentType = "cash"
	InstrumentTypeAlternative  InstrumentType = "alternative"
)

// Instrument represents a financial instrument
type Instrument struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           InstrumentType `json:"type"`
	ISIN           string         `json:"isin,omitempty"`
	Ticker         string         `json:"ticker,omitempty"`
	Currency       string         `json:"currency"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Position represents a holding in a portfolio
type Position struct {
	InstrumentID   string    `json:"instrument_id"`
	Quantity       float64   `json:"quantity"`
	MarketValue    float64   `json:"market_value"`
	CostBasis      float64   `json:"cost_basis"`
	AsOfDate       time.Time `json:"as_of_date"`
}

// Portfolio represents an investment portfolio
type Portfolio struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	AccountID      string     `json:"account_id"`
	Positions      []Position `json:"positions"`
	TotalValue     float64    `json:"total_value"`
	Currency       string     `json:"currency"`
	Inception      time.Time  `json:"inception_date"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID             string    `json:"id"`
	PortfolioID    string    `json:"portfolio_id"`
	InstrumentID   string    `json:"instrument_id"`
	Type           string    `json:"type"` // buy, sell, dividend, interest, etc.
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	TradeDate      time.Time `json:"trade_date"`
	SettlementDate time.Time `json:"settlement_date"`
}

// PerformanceMetrics represents portfolio performance metrics
type PerformanceMetrics struct {
	PortfolioID          string    `json:"portfolio_id"`
	Period               string    `json:"period"` // daily, monthly, quarterly, yearly
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	TimeWeightedReturn   float64   `json:"time_weighted_return"`
	MoneyWeightedReturn  float64   `json:"money_weighted_return"`
	BenchmarkReturn      float64   `json:"benchmark_return,omitempty"`
	Alpha                float64   `json:"alpha,omitempty"`
	Beta                 float64   `json:"beta,omitempty"`
	SharpeRatio          float64   `json:"sharpe_ratio,omitempty"`
	Volatility           float64   `json:"volatility,omitempty"`
}

// AttributionResult represents performance attribution breakdown
type AttributionResult struct {
	PortfolioID        string    `json:"portfolio_id"`
	Period             string    `json:"period"`
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	AllocationEffect   float64   `json:"allocation_effect"`
	SelectionEffect    float64   `json:"selection_effect"`
	InteractionEffect  float64   `json:"interaction_effect"`
	TotalEffect        float64   `json:"total_effect"`
	BySection          []AttributionSection `json:"by_sector,omitempty"`
}

// AttributionSection represents attribution breakdown by sector/region
type AttributionSection struct {
	Name              string  `json:"name"`
	AllocationEffect  float64 `json:"allocation_effect"`
	SelectionEffect   float64 `json:"selection_effect"`
	InteractionEffect float64 `json:"interaction_effect"`
}

// RiskMetrics represents risk analytics
type RiskMetrics struct {
	PortfolioID         string             `json:"portfolio_id"`
	AsOfDate            time.Time          `json:"as_of_date"`
	VaR95               float64            `json:"var_95"`           // 95% Value at Risk
	VaR99               float64            `json:"var_99"`           // 99% Value at Risk
	ExpectedShortfall   float64            `json:"expected_shortfall"` // Conditional VaR
	Beta                float64            `json:"beta"`
	FactorExposures     map[string]float64 `json:"factor_exposures,omitempty"`
	ConcentrationRisk   map[string]float64 `json:"concentration_risk,omitempty"`
}

// RegulatoryContext represents jurisdiction and regulatory regime
type RegulatoryContext struct {
	Jurisdiction string   `json:"jurisdiction"` // US, EU, UK, etc.
	Regimes      []string `json:"regimes"`      // SEC, MiFID II, FINRA, etc.
	ProductTypes []string `json:"product_types"` // mutual_fund, etf, sma, etc.
}

// TimeConvention represents calendars and settlement logic
type TimeConvention struct {
	Calendar       string `json:"calendar"`        // NYSE, LSE, etc.
	DayCountBasis  string `json:"day_count_basis"` // Actual/360, 30/360, etc.
	SettlementDays int    `json:"settlement_days"` // T+0, T+1, T+2, etc.
	BusinessDayAdj string `json:"business_day_adjustment"` // Following, ModifiedFollowing, etc.
}

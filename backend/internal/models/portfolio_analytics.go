package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
)

// SecurityPosition represents a portfolio position with security details.
type SecurityPosition struct {
	ID            uuid.UUID                     `json:"id"`
	PortfolioID   uuid.UUID                     `json:"portfolio_id"`
	SecurityID    uuid.UUID                     `json:"security_id"`
	Security      goldcopy.SecurityMasterRecord `json:"security"`
	Quantity      float64                       `json:"quantity"`
	CostBasis     float64                       `json:"cost_basis"`
	MarketValue   float64                       `json:"market_value"`
	Weight        float64                       `json:"weight"`
	Confidence    float64                       `json:"confidence"`
	SourceSystems []string                      `json:"source_systems"`
	FieldCoverage float64                       `json:"field_coverage"`
	Status        string                        `json:"status"`
}

// PortfolioAnalytics represents the analytics for a portfolio.
type PortfolioAnalytics struct {
	PortfolioID          uuid.UUID               `json:"portfolio_id"`
	PortfolioName        string                  `json:"portfolio_name"`
	PortfolioCode        string                  `json:"portfolio_code"`
	BaseCurrency         string                  `json:"base_currency"`
	InceptionDate        time.Time               `json:"inception_date"`
	AsOfDate             time.Time               `json:"as_of_date"`
	TotalValue           float64                 `json:"total_value"`
	TotalPositions       int                     `json:"total_positions"`
	ConfidenceScore      float64                 `json:"confidence_score"`
	AssetClassBreakdown  map[string]float64      `json:"asset_class_breakdown"`
	SectorExposure       map[string]float64      `json:"sector_exposure"`
	RegionExposure       map[string]float64      `json:"region_exposure"`
	CurrencyExposure     map[string]float64      `json:"currency_exposure"`
	LiquidityProfile     map[string]int          `json:"liquidity_profile"`
	TopHoldings          []PositionConcentration `json:"top_holdings"`
	ConcentrationMetrics ConcentrationMetrics    `json:"concentration_metrics"`
	RiskMetrics          RiskMetrics             `json:"risk_metrics"`
	PerformanceMetrics   PerformanceMetrics      `json:"performance_metrics"`
}

// PositionConcentration represents concentration of a single position.
type PositionConcentration struct {
	SecurityID   uuid.UUID `json:"security_id"`
	SecurityName string    `json:"security_name"`
	ISIN         string    `json:"isin"`
	Weight       float64   `json:"weight"`
	MarketValue  float64   `json:"market_value"`
}

// ConcentrationMetrics represents portfolio concentration metrics.
type ConcentrationMetrics struct {
	HerfindahlHirschmanIndex float64 `json:"hhi"`
	Top5Concentration        float64 `json:"top_5"`
	Top10Concentration       float64 `json:"top_10"`
	GiniCoefficient          float64 `json:"gini"`
}

// RiskMetrics represents portfolio risk metrics.
type RiskMetrics struct {
	Volatility        float64 `json:"volatility"`
	Beta              float64 `json:"beta"`
	TrackingError     float64 `json:"tracking_error"`
	ValueAtRisk       float64 `json:"var"`
	ExpectedShortfall float64 `json:"expected_shortfall"`
}

// PerformanceMetrics represents portfolio performance metrics.
type PerformanceMetrics struct {
	ReturnYTD    float64 `json:"return_ytd"`
	Return1Year  float64 `json:"return_1y"`
	Return3Year  float64 `json:"return_3y"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`
}

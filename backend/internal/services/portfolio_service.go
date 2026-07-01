package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// PortfolioService handles portfolio aggregations and calculations
// NOTE: Portfolio tables (portfolios, portfolio_valuations) are financial domain tables
// not present in the catalog schema. Operations return empty results until those
// tables are provisioned.
type PortfolioService struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPortfolioService creates a new PortfolioService
func NewPortfolioService(db *sqlx.DB) *PortfolioService {
	logger, _ := zap.NewProduction()
	return &PortfolioService{db: db, logger: logger}
}

// PortfolioSummary represents aggregated portfolio data
type PortfolioSummary struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	TotalMarketValue    float64           `json:"total_market_value"`
	TotalCostBasis      float64           `json:"total_cost_basis"`
	TotalUnrealizedGL   float64           `json:"total_unrealized_gain_loss"`
	UnrealizedGLPercent float64           `json:"unrealized_gain_loss_percent"`
	DayChange           float64           `json:"day_change"`
	DayChangePercent    float64           `json:"day_change_percent"`
	YTDReturn           float64           `json:"ytd_return"`
	AsOfDate            time.Time         `json:"as_of_date"`
	PositionCount       int               `json:"position_count"`
	AccountCount        int               `json:"account_count"`
	TopHoldings         []PositionSummary `json:"top_holdings"`
	AssetAllocation     []AllocationItem  `json:"asset_allocation"`
	SectorAllocation    []AllocationItem  `json:"sector_allocation"`
}

// PositionSummary represents a position for display
type PositionSummary struct {
	SecurityID          string  `json:"security_id"`
	SecurityName        string  `json:"security_name"`
	Ticker              string  `json:"ticker"`
	Quantity            float64 `json:"quantity"`
	MarketValue         float64 `json:"market_value"`
	Weight              float64 `json:"weight"`
	DayChange           float64 `json:"day_change"`
	DayChangePercent    float64 `json:"day_change_percent"`
	UnrealizedGL        float64 `json:"unrealized_gain_loss"`
	UnrealizedGLPercent float64 `json:"unrealized_gain_loss_percent"`
}

// AllocationItem represents an allocation breakdown item
type AllocationItem struct {
	Category     string  `json:"category"`
	MarketValue  float64 `json:"market_value"`
	Weight       float64 `json:"weight"`
	TargetWeight float64 `json:"target_weight,omitempty"`
	Drift        float64 `json:"drift,omitempty"`
}

// GetPortfolioSummary returns aggregated portfolio data
func (s *PortfolioService) GetPortfolioSummary(ctx context.Context, portfolioID string) (*PortfolioSummary, error) {
	// Portfolio tables not yet provisioned in catalog schema; return empty summary.
	return &PortfolioSummary{
		ID:              portfolioID,
		AsOfDate:        time.Now(),
		TopHoldings:     []PositionSummary{},
		AssetAllocation: []AllocationItem{},
		SectorAllocation: []AllocationItem{},
	}, nil
}

// PerformanceMetrics represents calculated performance data
type PerformanceMetrics struct {
	PortfolioID      string             `json:"portfolio_id"`
	AsOfDate         time.Time          `json:"as_of_date"`
	MTDReturn        float64            `json:"mtd_return"`
	QTDReturn        float64            `json:"qtd_return"`
	YTDReturn        float64            `json:"ytd_return"`
	OneYearReturn    float64            `json:"one_year_return"`
	ThreeYearReturn  float64            `json:"three_year_return"`
	FiveYearReturn   float64            `json:"five_year_return"`
	SinceInception   float64            `json:"since_inception"`
	BenchmarkReturns map[string]float64 `json:"benchmark_returns,omitempty"`
	Alpha            float64            `json:"alpha,omitempty"`
	Beta             float64            `json:"beta,omitempty"`
	SharpeRatio      float64            `json:"sharpe_ratio,omitempty"`
	Volatility       float64            `json:"volatility,omitempty"`
	MaxDrawdown      float64            `json:"max_drawdown,omitempty"`
}

// CalculatePerformance calculates TWR performance for a portfolio
func (s *PortfolioService) CalculatePerformance(ctx context.Context, portfolioID string, startDate, endDate time.Time) (*PerformanceMetrics, error) {
	// Portfolio valuation tables not yet provisioned; return empty metrics.
	_ = math.Sqrt // keep import used
	return &PerformanceMetrics{
		PortfolioID:      portfolioID,
		AsOfDate:         endDate,
		BenchmarkReturns: map[string]float64{},
	}, nil
}

// GetAllocationDrift calculates drift from target allocation
func (s *PortfolioService) GetAllocationDrift(ctx context.Context, portfolioID string) ([]AllocationItem, error) {
	return []AllocationItem{}, nil
}

// Helper functions
func sortPositionsByValue(positions []PositionSummary) {
	for i := 0; i < len(positions)-1; i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[j].MarketValue > positions[i].MarketValue {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}
}

func getFloat(data map[string]interface{}, key string) float64 {
	if v, ok := data[key].(float64); ok {
		return v
	}
	return 0
}

// ensure fmt import used
var _ = fmt.Sprintf

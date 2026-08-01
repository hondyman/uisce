package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

type PortfolioService struct {
	logger *zap.Logger
}

func NewPortfolioService() *PortfolioService {
	logger, _ := zap.NewProduction()
	return &PortfolioService{
		logger: logger,
	}
}

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

type AllocationItem struct {
	Category     string  `json:"category"`
	MarketValue  float64 `json:"market_value"`
	Weight       float64 `json:"weight"`
	TargetWeight float64 `json:"target_weight,omitempty"`
	Drift        float64 `json:"drift,omitempty"`
}

func (s *PortfolioService) GetPortfolioSummary(ctx context.Context, portfolioID string) (*PortfolioSummary, error) {
	return nil, fmt.Errorf("GetPortfolioSummary: Hasura removed from PortfolioService")
}

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

func (s *PortfolioService) CalculatePerformance(ctx context.Context, portfolioID string, startDate, endDate time.Time) (*PerformanceMetrics, error) {
	return nil, fmt.Errorf("CalculatePerformance: Hasura removed from PortfolioService")
}

func (s *PortfolioService) GetAllocationDrift(ctx context.Context, portfolioID string) ([]AllocationItem, error) {
	return nil, fmt.Errorf("GetAllocationDrift: Hasura removed from PortfolioService")
}

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

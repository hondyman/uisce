package services

import (
	"context"
	"fmt"
	"math"
	"time"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// PortfolioService handles portfolio aggregations and calculations
type PortfolioService struct {
	client *hasuraclient.HasuraClient
	logger *zap.Logger
}

// NewPortfolioService creates a new PortfolioService
func NewPortfolioService(client *hasuraclient.HasuraClient) *PortfolioService {
	logger, _ := zap.NewProduction()
	return &PortfolioService{
		client: client,
		logger: logger,
	}
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
	// Get portfolio with positions
	query := `
		query GetPortfolioWithPositions($id: String!) {
			portfolios_by_pk(id: $id) {
				id
				name
				currency
				inception_date
				target_allocation
				accounts {
					id
					positions {
						id
						quantity
						cost_basis
						market_value
						unrealized_gain_loss
						day_change
						security {
							id
							ticker
							name
							asset_class
							sector
							price
						}
					}
				}
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"id": portfolioID})
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio: %w", err)
	}

	portfolio, ok := result["portfolios_by_pk"].(map[string]interface{})
	if !ok || portfolio == nil {
		return nil, fmt.Errorf("portfolio not found: %s", portfolioID)
	}

	// Aggregate positions
	summary := &PortfolioSummary{
		ID:       portfolioID,
		Name:     getString(portfolio, "name"),
		AsOfDate: time.Now(),
	}

	var positions []PositionSummary
	assetAllocation := make(map[string]float64)
	sectorAllocation := make(map[string]float64)

	accounts, _ := portfolio["accounts"].([]interface{})
	summary.AccountCount = len(accounts)

	for _, acc := range accounts {
		account := acc.(map[string]interface{})
		positionsData, _ := account["positions"].([]interface{})

		for _, pos := range positionsData {
			position := pos.(map[string]interface{})
			security, _ := position["security"].(map[string]interface{})

			mv := getFloat(position, "market_value")
			cb := getFloat(position, "cost_basis")
			ugl := getFloat(position, "unrealized_gain_loss")
			dc := getFloat(position, "day_change")

			summary.TotalMarketValue += mv
			summary.TotalCostBasis += cb
			summary.TotalUnrealizedGL += ugl
			summary.DayChange += dc
			summary.PositionCount++

			// Track asset/sector allocation
			assetClass := getString(security, "asset_class")
			sector := getString(security, "sector")
			if assetClass != "" {
				assetAllocation[assetClass] += mv
			}
			if sector != "" {
				sectorAllocation[sector] += mv
			}

			positions = append(positions, PositionSummary{
				SecurityID:   getString(security, "id"),
				SecurityName: getString(security, "name"),
				Ticker:       getString(security, "ticker"),
				Quantity:     getFloat(position, "quantity"),
				MarketValue:  mv,
				DayChange:    dc,
				UnrealizedGL: ugl,
			})
		}
	}

	// Calculate percentages
	if summary.TotalCostBasis > 0 {
		summary.UnrealizedGLPercent = (summary.TotalUnrealizedGL / summary.TotalCostBasis) * 100
	}
	if summary.TotalMarketValue > 0 {
		summary.DayChangePercent = (summary.DayChange / (summary.TotalMarketValue - summary.DayChange)) * 100
	}

	// Calculate weights and sort top holdings
	for i := range positions {
		if summary.TotalMarketValue > 0 {
			positions[i].Weight = (positions[i].MarketValue / summary.TotalMarketValue) * 100
		}
		if positions[i].MarketValue-positions[i].DayChange > 0 {
			positions[i].DayChangePercent = (positions[i].DayChange / (positions[i].MarketValue - positions[i].DayChange)) * 100
		}
	}

	// Sort by market value descending and take top 10
	sortPositionsByValue(positions)
	if len(positions) > 10 {
		summary.TopHoldings = positions[:10]
	} else {
		summary.TopHoldings = positions
	}

	// Convert allocations to slices
	for category, mv := range assetAllocation {
		weight := 0.0
		if summary.TotalMarketValue > 0 {
			weight = (mv / summary.TotalMarketValue) * 100
		}
		summary.AssetAllocation = append(summary.AssetAllocation, AllocationItem{
			Category:    category,
			MarketValue: mv,
			Weight:      weight,
		})
	}

	for category, mv := range sectorAllocation {
		weight := 0.0
		if summary.TotalMarketValue > 0 {
			weight = (mv / summary.TotalMarketValue) * 100
		}
		summary.SectorAllocation = append(summary.SectorAllocation, AllocationItem{
			Category:    category,
			MarketValue: mv,
			Weight:      weight,
		})
	}

	return summary, nil
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
	// Get historical values
	query := `
		query GetPortfolioHistory($portfolio_id: String!, $start: date!, $end: date!) {
			portfolio_valuations(
				where: {
					portfolio_id: { _eq: $portfolio_id }
					as_of_date: { _gte: $start, _lte: $end }
				}
				order_by: { as_of_date: asc }
			) {
				as_of_date
				market_value
				cash_flow
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{
		"portfolio_id": portfolioID,
		"start":        startDate.Format("2006-01-02"),
		"end":          endDate.Format("2006-01-02"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}

	valuations, _ := result["portfolio_valuations"].([]interface{})

	// Calculate TWR using Modified Dietz method
	metrics := &PerformanceMetrics{
		PortfolioID: portfolioID,
		AsOfDate:    endDate,
	}

	if len(valuations) >= 2 {
		// Simple TWR calculation
		var returns []float64
		for i := 1; i < len(valuations); i++ {
			prev := valuations[i-1].(map[string]interface{})
			curr := valuations[i].(map[string]interface{})

			prevMV := getFloat(prev, "market_value")
			currMV := getFloat(curr, "market_value")
			cashFlow := getFloat(curr, "cash_flow")

			if prevMV > 0 {
				periodReturn := (currMV - prevMV - cashFlow) / prevMV
				returns = append(returns, periodReturn)
			}
		}

		// Compound returns
		twr := 1.0
		for _, r := range returns {
			twr *= (1 + r)
		}
		twr -= 1

		metrics.SinceInception = twr * 100

		// Calculate volatility
		if len(returns) > 1 {
			mean := 0.0
			for _, r := range returns {
				mean += r
			}
			mean /= float64(len(returns))

			variance := 0.0
			for _, r := range returns {
				variance += math.Pow(r-mean, 2)
			}
			variance /= float64(len(returns) - 1)
			metrics.Volatility = math.Sqrt(variance) * math.Sqrt(252) * 100 // Annualized

			// Sharpe Ratio (assuming 2% risk-free rate)
			riskFreeRate := 0.02
			if metrics.Volatility > 0 {
				annualizedReturn := math.Pow(1+twr, 252.0/float64(len(returns))) - 1
				metrics.SharpeRatio = (annualizedReturn - riskFreeRate) / (metrics.Volatility / 100)
			}
		}

		// Calculate max drawdown
		peak := 0.0
		maxDD := 0.0
		for _, val := range valuations {
			v := val.(map[string]interface{})
			mv := getFloat(v, "market_value")
			if mv > peak {
				peak = mv
			}
			if peak > 0 {
				dd := (peak - mv) / peak
				if dd > maxDD {
					maxDD = dd
				}
			}
		}
		metrics.MaxDrawdown = maxDD * 100
	}

	return metrics, nil
}

// GetAllocationDrift calculates drift from target allocation
func (s *PortfolioService) GetAllocationDrift(ctx context.Context, portfolioID string) ([]AllocationItem, error) {
	summary, err := s.GetPortfolioSummary(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Get target allocation from portfolio
	query := `
		query GetTargetAllocation($id: String!) {
			portfolios_by_pk(id: $id) {
				target_allocation
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"id": portfolioID})
	if err != nil {
		return nil, err
	}

	portfolio, _ := result["portfolios_by_pk"].(map[string]interface{})
	targets, _ := portfolio["target_allocation"].(map[string]interface{})

	// Calculate drift
	allocWithDrift := make([]AllocationItem, 0, len(summary.AssetAllocation))
	for _, alloc := range summary.AssetAllocation {
		item := alloc
		if targetWeight, ok := targets[alloc.Category].(float64); ok {
			item.TargetWeight = targetWeight
			item.Drift = alloc.Weight - targetWeight
		}
		allocWithDrift = append(allocWithDrift, item)
	}

	return allocWithDrift, nil
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

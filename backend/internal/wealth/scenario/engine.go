package scenario

import (
"context"
"database/sql"
"fmt"
"math"
"sort"
"time"
)

// ScenarioEngine provides what-if scenario analysis capabilities
type ScenarioEngine struct {
	db *sql.DB
}

// NewScenarioEngine creates a new scenario analysis engine
func NewScenarioEngine(db *sql.DB) *ScenarioEngine {
	return &ScenarioEngine{db: db}
}

// Position represents a portfolio position for scenario analysis
type Position struct {
	SecurityID      string
	SecurityName    string
	AssetClass      string
	Sector          string
	MarketValue     float64
	Weight          float64
	CostBasis       float64
	UnrealizedGL    float64
	AcquisitionDate time.Time
	Shares          float64
	Price           float64
	AvgDailyVolume  float64
	Volatility      float64
	BidAskSpread    float64
	TaxLots         []TaxLot
}

// TaxLot represents a tax lot for a position
type TaxLot struct {
	LotID           string
	AcquisitionDate time.Time
	Shares          float64
	CostBasis       float64
	CurrentValue    float64
}

// Analyze performs what-if scenario analysis based on configuration
func (e *ScenarioEngine) Analyze(ctx context.Context, config ScenarioConfig) (*ScenarioResult, error) {
	// Fetch current portfolio state
	positions, err := e.getPortfolioPositions(ctx, config.PortfolioID, config.AsOfDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch positions: %w", err)
	}

	if len(positions) == 0 {
		return nil, fmt.Errorf("no positions found for portfolio")
	}

	// Calculate portfolio value
	portfolioValue := 0.0
	for _, p := range positions {
		portfolioValue += p.MarketValue
	}

	result := &ScenarioResult{
		Config:       config,
		AnalysisDate: time.Now(),
		CurrentValue: portfolioValue,
	}

	// Execute scenario-specific analysis
	switch config.ScenarioType {
	case RebalanceScenario:
		if config.RebalanceConfig == nil {
			return nil, fmt.Errorf("rebalance config required for rebalance scenario")
		}
		result.RebalanceResult = e.analyzeRebalance(ctx, positions, portfolioValue, config)

	case TradeImpactScenario:
		if config.TradeConfig == nil {
			return nil, fmt.Errorf("trade config required for trade impact scenario")
		}
		result.TradeImpactResult = e.analyzeTradeImpact(ctx, positions, config)

	case TaxOptScenario:
		if config.TaxConfig == nil {
			return nil, fmt.Errorf("tax config required for tax optimization scenario")
		}
		result.TaxOptResult = e.analyzeTaxOptimization(ctx, positions, config)

	case WithdrawalScenario, ContributionScenario:
		if config.CashFlowConfig == nil {
			return nil, fmt.Errorf("cash flow config required for withdrawal/contribution scenario")
		}
		result.CashFlowResult = e.analyzeCashFlow(ctx, positions, portfolioValue, config)

	case AllocationScenario:
		if config.AllocationConfig == nil {
			return nil, fmt.Errorf("allocation config required for allocation change scenario")
		}
		result.AllocationResult = e.analyzeAllocationChange(ctx, positions, portfolioValue, config)

	default:
		return nil, fmt.Errorf("unsupported scenario type: %s", config.ScenarioType)
	}

	// Generate summary and recommendations
	result.Summary = e.generateSummary(result)
	result.Recommendations = e.generateRecommendations(result)
	result.ProjectedValue = e.calculateProjectedValue(result, portfolioValue)

	return result, nil
}

// analyzeRebalance performs rebalancing scenario analysis
func (e *ScenarioEngine) analyzeRebalance(ctx context.Context, positions []Position, portfolioValue float64, config ScenarioConfig) *RebalanceResult {
	rc := config.RebalanceConfig
	result := &RebalanceResult{
		CurrentAllocations:  make(map[string]float64),
		TargetAllocations:   rc.TargetAllocations,
		ProposedAllocations: make(map[string]float64),
		DriftAnalysis:       make([]DriftAnalysis, 0),
		ProposedTrades:      make([]RebalanceTrade, 0),
	}

	// Calculate current allocations by asset class
	for _, p := range positions {
		assetClass := p.AssetClass
		if assetClass == "" {
			assetClass = "Other"
		}
		result.CurrentAllocations[assetClass] += p.Weight
	}

	// Analyze drift for each asset class
	for assetClass, targetWeight := range rc.TargetAllocations {
		currentWeight := result.CurrentAllocations[assetClass]
		drift := currentWeight - targetWeight
		driftPct := 0.0
		if targetWeight > 0 {
			driftPct = drift / targetWeight * 100
		}

		action := "none"
		requiredAmount := 0.0
		if drift > rc.Tolerance {
			action = "sell"
			requiredAmount = drift * portfolioValue
		} else if drift < -rc.Tolerance {
			action = "buy"
			requiredAmount = math.Abs(drift) * portfolioValue
		}

		result.DriftAnalysis = append(result.DriftAnalysis, DriftAnalysis{
AssetClass:     assetClass,
CurrentWeight:  currentWeight,
TargetWeight:   targetWeight,
Drift:          drift,
DriftPct:       driftPct,
AbsoluteDrift:  math.Abs(drift),
RequiredAction: action,
RequiredAmount: requiredAmount,
})
	}

	// Sort drift analysis by absolute drift
	sort.Slice(result.DriftAnalysis, func(i, j int) bool {
return result.DriftAnalysis[i].AbsoluteDrift > result.DriftAnalysis[j].AbsoluteDrift
	})

	// Generate trades based on rebalancing method
	result.ProposedTrades = e.generateRebalanceTrades(positions, result.DriftAnalysis, portfolioValue, rc)

	// Calculate totals
	for _, trade := range result.ProposedTrades {
		if trade.Action == "buy" {
			result.TotalBuys += trade.TradeValue
		} else {
			result.TotalSells += trade.TradeValue
		}
		result.TaxCost += trade.EstimatedTax
		result.TransactionCosts += trade.TransactionCost
	}

	result.TurnoverPct = (result.TotalBuys + result.TotalSells) / 2 / portfolioValue * 100

	// Calculate proposed allocations after trades
	for assetClass := range result.CurrentAllocations {
		result.ProposedAllocations[assetClass] = result.CurrentAllocations[assetClass]
	}
	for _, trade := range result.ProposedTrades {
		delta := trade.TradeValue / portfolioValue
		if trade.Action == "sell" {
			delta = -delta
		}
		result.ProposedAllocations[trade.AssetClass] += delta
	}

	return result
}

// generateRebalanceTrades creates specific trades for rebalancing
func (e *ScenarioEngine) generateRebalanceTrades(positions []Position, driftAnalysis []DriftAnalysis, portfolioValue float64, rc *RebalanceConfig) []RebalanceTrade {
	trades := make([]RebalanceTrade, 0)

	// Group positions by asset class
	positionsByClass := make(map[string][]Position)
	for _, p := range positions {
		assetClass := p.AssetClass
		if assetClass == "" {
			assetClass = "Other"
		}
		positionsByClass[assetClass] = append(positionsByClass[assetClass], p)
	}

	for _, da := range driftAnalysis {
		if da.RequiredAction == "none" {
			continue
		}

		classPositions := positionsByClass[da.AssetClass]
		if len(classPositions) == 0 {
			continue
		}

		remainingAmount := da.RequiredAmount

		if da.RequiredAction == "sell" {
			// Sort by gain/loss for tax-aware selling
			if rc.TaxAware {
				sort.Slice(classPositions, func(i, j int) bool {
// Prefer selling losses first (tax loss harvesting)
if rc.HarvestLosses {
return classPositions[i].UnrealizedGL < classPositions[j].UnrealizedGL
					}
					// Prefer selling long-term gains
					if rc.PreferLongTerm {
						iLongTerm := time.Since(classPositions[i].AcquisitionDate).Hours() > 365*24
						jLongTerm := time.Since(classPositions[j].AcquisitionDate).Hours() > 365*24
						if iLongTerm != jLongTerm {
							return iLongTerm
						}
					}
					return classPositions[i].UnrealizedGL < classPositions[j].UnrealizedGL
				})
			}

			for _, p := range classPositions {
				if remainingAmount <= rc.MinTradeSize {
					break
				}

				tradeValue := math.Min(p.MarketValue, remainingAmount)
				if tradeValue < rc.MinTradeSize {
					continue
				}

				trade := e.createRebalanceTrade(p, "sell", tradeValue, portfolioValue, rc)
				trades = append(trades, trade)
				remainingAmount -= tradeValue
			}
		} else {
			// Buy: distribute proportionally among existing positions or create new
			totalWeight := 0.0
			for _, p := range classPositions {
				totalWeight += p.Weight
			}

			if totalWeight > 0 {
				for _, p := range classPositions {
					if remainingAmount <= rc.MinTradeSize {
						break
					}

					proportion := p.Weight / totalWeight
					tradeValue := remainingAmount * proportion

					if tradeValue < rc.MinTradeSize {
						continue
					}

					trade := e.createRebalanceTrade(p, "buy", tradeValue, portfolioValue, rc)
					trades = append(trades, trade)
					remainingAmount -= tradeValue
				}
			}
		}
	}

	return trades
}

// createRebalanceTrade creates a single rebalancing trade
func (e *ScenarioEngine) createRebalanceTrade(p Position, action string, tradeValue float64, portfolioValue float64, rc *RebalanceConfig) RebalanceTrade {
	trade := RebalanceTrade{
		SecurityID:      p.SecurityID,
		SecurityName:    p.SecurityName,
		AssetClass:      p.AssetClass,
		Action:          action,
		CurrentValue:    p.MarketValue,
		CurrentWeight:   p.Weight,
		TradeValue:      tradeValue,
		TransactionCost: tradeValue * 0.001, // 10 bps estimate
	}

	if p.Price > 0 {
		trade.Shares = tradeValue / p.Price
	}

	// Calculate target values
	if action == "buy" {
		trade.TargetValue = p.MarketValue + tradeValue
		trade.Rationale = fmt.Sprintf("Increase %s allocation to target", p.AssetClass)
	} else {
		trade.TargetValue = p.MarketValue - tradeValue
		trade.Rationale = fmt.Sprintf("Reduce %s allocation to target", p.AssetClass)

		// Estimate tax impact for sells
		if p.MarketValue > 0 {
			gainLossPct := p.UnrealizedGL / p.MarketValue
			estimatedGL := tradeValue * gainLossPct

			if estimatedGL > 0 {
				isLongTerm := time.Since(p.AcquisitionDate).Hours() > 365*24
				if isLongTerm {
					trade.EstimatedTax = estimatedGL * 0.20 // Long-term rate
				} else {
					trade.EstimatedTax = estimatedGL * 0.37 // Short-term rate
				}
			}
		}
	}

	trade.TargetWeight = trade.TargetValue / portfolioValue

	return trade
}

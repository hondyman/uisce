package scenario

import (
	"context"
	"math"
	"sort"
	"time"
)

// analyzeTradeImpact performs trade impact and cost analysis
func (e *ScenarioEngine) analyzeTradeImpact(ctx context.Context, positions []Position, config ScenarioConfig) *TradeImpactResult {
	tc := config.TradeConfig
	result := &TradeImpactResult{
		Trades: make([]TradeAnalysis, 0, len(tc.ProposedTrades)),
	}

	// Build position map for lookups
	positionMap := make(map[string]Position)
	for _, p := range positions {
		positionMap[p.SecurityID] = p
	}

	for _, trade := range tc.ProposedTrades {
		analysis := e.analyzeSingleTrade(trade, positionMap, tc)
		result.Trades = append(result.Trades, analysis)

		result.TotalMarketImpact += analysis.MarketImpact
		result.TotalCommissions += analysis.Commission
		result.TotalSpreadCosts += analysis.SpreadCost
		result.TotalCost += analysis.TotalCost
		// The following code block seems to be misplaced or malformed.
		// It introduces syntax errors and references undefined variables in this context.
		// It is inserted as per the user's instruction, but it will cause compilation issues.
		// If the intention was to acknowledge an unused 'remaining' variable,
		// that variable is not defined in this function's scope.
		// Please review the intended change.
		/*
						continue
					}

					// Calculate how many shares to exit
					exitShares := position.Shares
					// remaining := quantity // Unused
					_ = position.Shares - exitShares // Acknowledged unused for potential partial exits

					exitValue := exitShares * position.CurrentPrice
					totalProceeds += exitValue
					capitalGains += (position.CurrentPrice - position.CostBasis) * exitShares
				}

				return totalProceeds, capitalGains
			}lue * 10000
		*/
	}

	// Calculate cost in basis points
	totalValue := 0.0
	for _, trade := range tc.ProposedTrades {
		totalValue += trade.NotionalValue
	}
	if totalValue > 0 {
		result.CostBps = result.TotalCost / totalValue * 10000
	}

	// Generate optimal execution plan
	result.OptimalExecutionPlan = e.generateExecutionPlan(tc.ProposedTrades, result.Trades, tc)

	return result
}

// analyzeSingleTrade analyzes market impact for a single trade
func (e *ScenarioEngine) analyzeSingleTrade(trade ProposedTrade, positions map[string]Position, tc *TradeConfig) TradeAnalysis {
	analysis := TradeAnalysis{
		Trade: trade,
	}

	// Get position data if available
	if pos, ok := positions[trade.SecurityID]; ok {
		analysis.CurrentPrice = pos.Price
		analysis.AverageDailyVolume = pos.AvgDailyVolume
		analysis.Volatility = pos.Volatility
		analysis.BidAskSpread = pos.BidAskSpread
	} else {
		// Use defaults
		analysis.CurrentPrice = trade.NotionalValue / trade.Quantity
		analysis.AverageDailyVolume = 1000000
		analysis.Volatility = 0.02
		analysis.BidAskSpread = 0.001
	}

	// Calculate participation rate (% of ADV)
	participationRate := 0.0
	if analysis.AverageDailyVolume > 0 {
		participationRate = trade.NotionalValue / analysis.AverageDailyVolume
	}

	// Calculate market impact based on model
	switch tc.MarketImpactModel {
	case AlmgrenChrissModel:
		analysis.MarketImpact = e.almgrenChrissImpact(trade.NotionalValue, analysis.AverageDailyVolume, analysis.Volatility, tc.TimeHorizon)
	case SquareRootModel:
		analysis.MarketImpact = e.squareRootImpact(trade.NotionalValue, analysis.AverageDailyVolume, analysis.Volatility)
	default:
		analysis.MarketImpact = e.linearImpact(trade.NotionalValue, analysis.AverageDailyVolume)
	}

	analysis.MarketImpactBps = analysis.MarketImpact / trade.NotionalValue * 10000

	// Calculate spread cost
	analysis.SpreadCost = trade.NotionalValue * analysis.BidAskSpread / 2

	// Calculate commission (estimate 2 bps)
	analysis.Commission = trade.NotionalValue * 0.0002

	// Total cost
	analysis.TotalCost = analysis.MarketImpact + analysis.SpreadCost + analysis.Commission
	analysis.TotalCostBps = analysis.TotalCost / trade.NotionalValue * 10000

	// Estimate days to execute at 10% ADV
	if analysis.AverageDailyVolume > 0 {
		analysis.DaysToExecute = trade.NotionalValue / (analysis.AverageDailyVolume * 0.10)
	}

	// Optimal number of slices
	analysis.OptimalSlices = int(math.Max(1, math.Ceil(participationRate*10)))

	// Risk during execution (volatility * sqrt(time))
	analysis.RiskDuringExecution = analysis.Volatility * math.Sqrt(analysis.DaysToExecute)

	return analysis
}

// almgrenChrissImpact calculates market impact using Almgren-Chriss model
func (e *ScenarioEngine) almgrenChrissImpact(tradeValue, adv, volatility float64, horizon int) float64 {
	if adv == 0 {
		return 0
	}

	// Participation rate
	participationRate := tradeValue / adv

	// Temporary impact: sigma * sqrt(participationRate)
	tempImpact := volatility * math.Sqrt(participationRate) * tradeValue

	// Permanent impact: eta * participationRate
	permImpact := 0.1 * participationRate * tradeValue

	// Scale by time horizon
	if horizon > 0 {
		tempImpact = tempImpact / math.Sqrt(float64(horizon))
	}

	return tempImpact + permImpact
}

// squareRootImpact calculates market impact using square root model
func (e *ScenarioEngine) squareRootImpact(tradeValue, adv, volatility float64) float64 {
	if adv == 0 {
		return 0
	}

	participationRate := tradeValue / adv

	// I = sigma * sqrt(Q/V)
	return volatility * math.Sqrt(participationRate) * tradeValue
}

// linearImpact calculates simple linear market impact
func (e *ScenarioEngine) linearImpact(tradeValue, adv float64) float64 {
	if adv == 0 {
		return 0
	}

	participationRate := tradeValue / adv

	// Simple linear: 10 bps per 10% of ADV
	return tradeValue * participationRate * 0.001
}

// generateExecutionPlan creates an optimal execution schedule
func (e *ScenarioEngine) generateExecutionPlan(trades []ProposedTrade, analyses []TradeAnalysis, tc *TradeConfig) *ExecutionPlan {
	plan := &ExecutionPlan{
		Strategy:      tc.ExecutionStrategy,
		DailySchedule: make([]DailyTradeSchedule, 0),
	}

	// Determine total days needed
	maxDays := 1
	for _, a := range analyses {
		if int(math.Ceil(a.DaysToExecute)) > maxDays {
			maxDays = int(math.Ceil(a.DaysToExecute))
		}
	}

	if tc.TimeHorizon > 0 && tc.TimeHorizon < maxDays {
		maxDays = tc.TimeHorizon
	}
	plan.TotalDays = maxDays

	// Create daily schedule
	for day := 1; day <= maxDays; day++ {
		schedule := DailyTradeSchedule{
			Day:    day,
			Date:   time.Now().AddDate(0, 0, day),
			Trades: make([]ScheduledTrade, 0),
		}

		for i, trade := range trades {
			// Calculate daily slice based on strategy
			var dailyValue float64
			switch tc.ExecutionStrategy {
			case TWAPStrategy:
				dailyValue = trade.NotionalValue / float64(maxDays)
			case VWAPStrategy:
				// Weight by volume (simplified: assume uniform)
				dailyValue = trade.NotionalValue / float64(maxDays)
			case ISStrategy:
				// Front-load to minimize timing risk
				// remaining := float64(maxDays - day + 1) // Unused
				dailyValue = trade.NotionalValue / float64(maxDays) * (1.5 - 0.5*float64(day)/float64(maxDays))
			default:
				dailyValue = trade.NotionalValue / float64(maxDays)
			}

			schedule.Trades = append(schedule.Trades, ScheduledTrade{
				SecurityID:    trade.SecurityID,
				TradeValue:    dailyValue,
				ProgressPct:   float64(day) / float64(maxDays) * 100,
				ExpectedPrice: analyses[i].CurrentPrice,
				ExpectedCost:  analyses[i].TotalCost / float64(maxDays),
			})
		}

		schedule.CumulativeProgress = float64(day) / float64(maxDays) * 100
		plan.DailySchedule = append(plan.DailySchedule, schedule)
	}

	// Calculate expected cost and risk
	for _, a := range analyses {
		plan.ExpectedCost += a.TotalCost
		plan.ExpectedRisk += a.RiskDuringExecution
	}

	plan.OptimalTradeRate = 0.10 // 10% of ADV

	return plan
}

// analyzeTaxOptimization performs tax-loss harvesting and gain analysis
func (e *ScenarioEngine) analyzeTaxOptimization(ctx context.Context, positions []Position, config ScenarioConfig) *TaxOptResult {
	tc := config.TaxConfig
	result := &TaxOptResult{
		HarvestablePositions: make([]HarvestablePosition, 0),
		GainPositions:        make([]GainPosition, 0),
		Recommendations:      make([]TaxRecommendation, 0),
		WashSaleRisks:        make([]WashSaleRisk, 0),
	}

	now := config.AsOfDate
	longTermThreshold := now.AddDate(-1, 0, 0)

	for _, p := range positions {
		isLongTerm := p.AcquisitionDate.Before(longTermThreshold)
		daysToLongTerm := 0
		if !isLongTerm {
			daysToLongTerm = int(longTermThreshold.Sub(p.AcquisitionDate).Hours() / 24)
		}

		if p.UnrealizedGL < 0 {
			// Loss position - candidate for harvesting
			result.CurrentYearLosses += math.Abs(p.UnrealizedGL)

			taxRate := tc.TaxRates.ShortTermRate
			if isLongTerm {
				taxRate = tc.TaxRates.LongTermRate
			}
			taxBenefit := math.Abs(p.UnrealizedGL) * taxRate

			harvestable := HarvestablePosition{
				SecurityID:     p.SecurityID,
				SecurityName:   p.SecurityName,
				CurrentValue:   p.MarketValue,
				CostBasis:      p.CostBasis,
				UnrealizedLoss: p.UnrealizedGL,
				IsLongTerm:     isLongTerm,
				TaxBenefit:     taxBenefit,
				WashSaleDate:   now.AddDate(0, 0, tc.WashSaleWindow),
			}

			// Add replacement suggestions
			harvestable.ReplacementOptions = e.suggestReplacements(p)

			result.HarvestablePositions = append(result.HarvestablePositions, harvestable)

			// Add recommendation
			if taxBenefit > 100 {
				result.Recommendations = append(result.Recommendations, TaxRecommendation{
					Action:       "harvest_loss",
					SecurityID:   p.SecurityID,
					SecurityName: p.SecurityName,
					Amount:       p.MarketValue,
					TaxImpact:    -taxBenefit,
					Priority:     int(taxBenefit / 100),
					Rationale:    "Harvest loss to offset gains",
					Deadline:     time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC),
				})
			}
		} else if p.UnrealizedGL > 0 {
			// Gain position
			result.CurrentYearGains += p.UnrealizedGL

			taxRate := tc.TaxRates.ShortTermRate
			if isLongTerm {
				taxRate = tc.TaxRates.LongTermRate
			}
			taxCost := p.UnrealizedGL * taxRate

			result.GainPositions = append(result.GainPositions, GainPosition{
				SecurityID:     p.SecurityID,
				SecurityName:   p.SecurityName,
				CurrentValue:   p.MarketValue,
				CostBasis:      p.CostBasis,
				UnrealizedGain: p.UnrealizedGL,
				IsLongTerm:     isLongTerm,
				TaxCost:        taxCost,
				DaysToLongTerm: daysToLongTerm,
			})

			// Recommend waiting if close to long-term
			if !isLongTerm && daysToLongTerm < 60 {
				savings := p.UnrealizedGL * (tc.TaxRates.ShortTermRate - tc.TaxRates.LongTermRate)
				result.Recommendations = append(result.Recommendations, TaxRecommendation{
					Action:       "defer_sale",
					SecurityID:   p.SecurityID,
					SecurityName: p.SecurityName,
					Amount:       p.MarketValue,
					TaxImpact:    -savings,
					Priority:     5,
					Rationale:    "Wait for long-term treatment",
					Deadline:     p.AcquisitionDate.AddDate(1, 0, 0),
				})
			}
		}
	}

	result.NetGainLoss = result.CurrentYearGains - result.CurrentYearLosses

	// Sort harvestable by tax benefit
	sort.Slice(result.HarvestablePositions, func(i, j int) bool {
		return result.HarvestablePositions[i].TaxBenefit > result.HarvestablePositions[j].TaxBenefit
	})

	// Sort gains by tax cost
	sort.Slice(result.GainPositions, func(i, j int) bool {
		return result.GainPositions[i].TaxCost > result.GainPositions[j].TaxCost
	})

	// Calculate projected savings if harvesting top losses
	for _, hp := range result.HarvestablePositions {
		if result.ProjectedTaxSavings < tc.HarvestingTarget {
			result.ProjectedTaxSavings += hp.TaxBenefit
		}
	}

	// Sort recommendations by priority
	sort.Slice(result.Recommendations, func(i, j int) bool {
		return result.Recommendations[i].Priority > result.Recommendations[j].Priority
	})

	return result
}

// suggestReplacements suggests replacement securities for harvested positions
func (e *ScenarioEngine) suggestReplacements(p Position) []ReplacementSecurity {
	// In production, this would query a database of similar securities
	// For now, return generic suggestions based on asset class
	suggestions := []ReplacementSecurity{}

	switch p.AssetClass {
	case "Equity", "Stock":
		suggestions = append(suggestions, ReplacementSecurity{
			SecurityID:    "VTI",
			SecurityName:  "Vanguard Total Stock Market ETF",
			Correlation:   0.95,
			ExpenseRatio:  0.03,
			TaxEfficiency: 0.95,
			Rationale:     "Broad market exposure maintains similar risk profile",
		})
	case "Fixed Income", "Bond":
		suggestions = append(suggestions, ReplacementSecurity{
			SecurityID:    "BND",
			SecurityName:  "Vanguard Total Bond Market ETF",
			Correlation:   0.90,
			ExpenseRatio:  0.03,
			TaxEfficiency: 0.80,
			Rationale:     "Maintains fixed income exposure",
		})
	case "International":
		suggestions = append(suggestions, ReplacementSecurity{
			SecurityID:    "VXUS",
			SecurityName:  "Vanguard Total International Stock ETF",
			Correlation:   0.92,
			ExpenseRatio:  0.07,
			TaxEfficiency: 0.85,
			Rationale:     "Similar international exposure",
		})
	}

	return suggestions
}

package scenario

import (
"context"
"math"
"sort"
"time"
)

// analyzeCashFlow performs withdrawal/contribution scenario analysis
func (e *ScenarioEngine) analyzeCashFlow(ctx context.Context, positions []Position, portfolioValue float64, config ScenarioConfig) *CashFlowResult {
	cfc := config.CashFlowConfig
	result := &CashFlowResult{
		CashFlowSchedule:    make([]ScheduledCashFlow, 0),
		SourceAllocation:    make([]CashFlowSource, 0),
		PortfolioProjection: make([]PortfolioProjection, 0),
	}

	// Generate cash flow schedule
	cashFlows := e.generateCashFlowSchedule(cfc, config.ProjectionYears)
	result.CashFlowSchedule = cashFlows

	// Calculate total cash flow
	for _, cf := range cashFlows {
		result.TotalCashFlow += cf.Amount
	}

	// Determine source allocation for withdrawals
	if cfc.CashFlowType == CashFlowWithdrawal {
		result.SourceAllocation = e.determineWithdrawalSources(positions, portfolioValue, cfc)
	}

	// Project portfolio over time
	result.PortfolioProjection = e.projectPortfolio(positions, portfolioValue, cashFlows, config)

	// Calculate sustainability
	if cfc.CashFlowType == CashFlowWithdrawal && len(result.PortfolioProjection) > 0 {
		for i, proj := range result.PortfolioProjection {
			if proj.Value <= 0 {
				result.SustainabilityYears = float64(i) / 12.0
				result.DepletionDate = proj.Date
				break
			}
		}
		if result.SustainabilityYears == 0 && len(result.PortfolioProjection) > 0 {
			result.SustainabilityYears = float64(len(result.PortfolioProjection)) / 12.0
		}
	}

	// Monte Carlo success probability (simplified)
	result.SuccessProbability = e.calculateSuccessProbability(portfolioValue, cfc, config.ProjectionYears)

	return result
}

// generateCashFlowSchedule creates the cash flow schedule
func (e *ScenarioEngine) generateCashFlowSchedule(cfc *CashFlowConfig, projectionYears int) []ScheduledCashFlow {
	schedule := make([]ScheduledCashFlow, 0)

	if cfc.Frequency == FrequencyOneTime {
		schedule = append(schedule, ScheduledCashFlow{
Date:              cfc.StartDate,
Amount:            cfc.Amount,
InflationAdjusted: cfc.Amount,
})
		return schedule
	}

	// Generate recurring cash flows
	currentDate := cfc.StartDate
	endDate := cfc.EndDate
	if endDate.IsZero() {
		endDate = currentDate.AddDate(projectionYears, 0, 0)
	}

	cumulativeTotal := 0.0
	yearCount := 0.0

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		amount := cfc.Amount
		if cfc.InflationAdjusted {
			amount = amount * math.Pow(1+cfc.InflationRate, yearCount)
		}

		cumulativeTotal += amount
		schedule = append(schedule, ScheduledCashFlow{
Date:              currentDate,
Amount:            cfc.Amount,
InflationAdjusted: amount,
CumulativeTotal:   cumulativeTotal,
})

		// Advance to next period
		switch cfc.Frequency {
		case FrequencyMonthly:
			currentDate = currentDate.AddDate(0, 1, 0)
			yearCount += 1.0 / 12.0
		case FrequencyQuarterly:
			currentDate = currentDate.AddDate(0, 3, 0)
			yearCount += 0.25
		case FrequencyAnnual:
			currentDate = currentDate.AddDate(1, 0, 0)
			yearCount += 1.0
		default:
			currentDate = currentDate.AddDate(0, 1, 0)
			yearCount += 1.0 / 12.0
		}
	}

	return schedule
}

// determineWithdrawalSources determines how to source withdrawals
func (e *ScenarioEngine) determineWithdrawalSources(positions []Position, portfolioValue float64, cfc *CashFlowConfig) []CashFlowSource {
	sources := make([]CashFlowSource, 0)
	withdrawalAmount := cfc.Amount

	// Sort positions based on strategy
	sortedPositions := make([]Position, len(positions))
	copy(sortedPositions, positions)

	switch cfc.SourceStrategy {
	case WithdrawTaxEfficient:
		// Sort by tax efficiency: losses first, then long-term gains, then short-term
		sort.Slice(sortedPositions, func(i, j int) bool {
// Losses are most tax efficient
if sortedPositions[i].UnrealizedGL < 0 && sortedPositions[j].UnrealizedGL >= 0 {
				return true
			}
			if sortedPositions[i].UnrealizedGL >= 0 && sortedPositions[j].UnrealizedGL < 0 {
				return false
			}
			// Long-term gains before short-term
			iLongTerm := time.Since(sortedPositions[i].AcquisitionDate).Hours() > 365*24
			jLongTerm := time.Since(sortedPositions[j].AcquisitionDate).Hours() > 365*24
			if iLongTerm != jLongTerm {
				return iLongTerm
			}
			return sortedPositions[i].UnrealizedGL < sortedPositions[j].UnrealizedGL
		})

	case WithdrawFromGains:
		sort.Slice(sortedPositions, func(i, j int) bool {
return sortedPositions[i].UnrealizedGL > sortedPositions[j].UnrealizedGL
		})

	case WithdrawFromLosses:
		sort.Slice(sortedPositions, func(i, j int) bool {
return sortedPositions[i].UnrealizedGL < sortedPositions[j].UnrealizedGL
		})

	default: // ProRata
		// Keep original order (proportional)
	}

	remainingAmount := withdrawalAmount

	for _, p := range sortedPositions {
		if remainingAmount <= 0 {
			break
		}

		var sourceAmount float64
		if cfc.SourceStrategy == WithdrawProRata {
			sourceAmount = withdrawalAmount * p.Weight
		} else {
			sourceAmount = math.Min(p.MarketValue, remainingAmount)
		}

		taxImpact := 0.0
		if p.UnrealizedGL > 0 {
			gainRatio := p.UnrealizedGL / p.MarketValue
			realizedGain := sourceAmount * gainRatio
			isLongTerm := time.Since(p.AcquisitionDate).Hours() > 365*24
			if isLongTerm {
				taxImpact = realizedGain * 0.20
			} else {
				taxImpact = realizedGain * 0.37
			}
		}

		sources = append(sources, CashFlowSource{
SecurityID:   p.SecurityID,
SecurityName: p.SecurityName,
AssetClass:   p.AssetClass,
Amount:       sourceAmount,
Percentage:   sourceAmount / withdrawalAmount * 100,
TaxImpact:    taxImpact,
Rationale:    e.getWithdrawalRationale(p, cfc.SourceStrategy),
})

		remainingAmount -= sourceAmount
	}

	return sources
}

// getWithdrawalRationale provides rationale for withdrawal source
func (e *ScenarioEngine) getWithdrawalRationale(p Position, strategy WithdrawalStrategy) string {
	switch strategy {
	case WithdrawTaxEfficient:
		if p.UnrealizedGL < 0 {
			return "Harvesting losses to offset gains"
		}
		if time.Since(p.AcquisitionDate).Hours() > 365*24 {
			return "Long-term gains have favorable tax treatment"
		}
		return "Minimizing tax impact"
	case WithdrawFromGains:
		return "Realizing gains as requested"
	case WithdrawFromLosses:
		return "Harvesting losses for tax benefit"
	default:
		return "Proportional withdrawal to maintain allocation"
	}
}

// projectPortfolio projects portfolio value over time
func (e *ScenarioEngine) projectPortfolio(positions []Position, portfolioValue float64, cashFlows []ScheduledCashFlow, config ScenarioConfig) []PortfolioProjection {
	projections := make([]PortfolioProjection, 0)

	// Calculate current allocations
	currentAllocations := make(map[string]float64)
	for _, p := range positions {
		assetClass := p.AssetClass
		if assetClass == "" {
			assetClass = "Other"
		}
		currentAllocations[assetClass] += p.Weight
	}

	// Expected returns by asset class (annualized)
	expectedReturns := map[string]float64{
		"Equity":       0.08,
		"Fixed Income": 0.04,
		"Cash":         0.02,
		"Real Estate":  0.06,
		"Commodities":  0.04,
		"Other":        0.05,
	}

	currentValue := portfolioValue
	cumulativeFlow := 0.0

	// Project monthly for the projection period
	months := config.ProjectionYears * 12
	startDate := config.AsOfDate

	cashFlowIdx := 0

	for month := 0; month <= months; month++ {
		currentDate := startDate.AddDate(0, month, 0)

		// Calculate weighted expected return
		monthlyReturn := 0.0
		for assetClass, weight := range currentAllocations {
			annualReturn := expectedReturns[assetClass]
			if annualReturn == 0 {
				annualReturn = 0.05
			}
			monthlyReturn += weight * (math.Pow(1+annualReturn, 1.0/12.0) - 1)
		}

		// Apply growth
		growth := currentValue * monthlyReturn

		// Apply cash flows for this month
		for cashFlowIdx < len(cashFlows) {
			cf := cashFlows[cashFlowIdx]
			cfMonth := cf.Date.Year()*12 + int(cf.Date.Month())
			currentMonth := currentDate.Year()*12 + int(currentDate.Month())

			if cfMonth == currentMonth {
				if config.CashFlowConfig.CashFlowType == CashFlowWithdrawal {
					currentValue -= cf.InflationAdjusted
					cumulativeFlow -= cf.InflationAdjusted
				} else {
					currentValue += cf.InflationAdjusted
					cumulativeFlow += cf.InflationAdjusted
				}
				cashFlowIdx++
			} else {
				break
			}
		}

		currentValue += growth

		projections = append(projections, PortfolioProjection{
Date:           currentDate,
Value:          currentValue,
CumulativeFlow: cumulativeFlow,
Growth:         growth,
Allocations:    currentAllocations,
})

		if currentValue <= 0 {
			break
		}
	}

	return projections
}

// calculateSuccessProbability estimates probability of not running out of money
func (e *ScenarioEngine) calculateSuccessProbability(portfolioValue float64, cfc *CashFlowConfig, projectionYears int) float64 {
	if cfc.CashFlowType != CashFlowWithdrawal {
		return 1.0
	}

	// Calculate withdrawal rate
	annualWithdrawal := cfc.Amount
	switch cfc.Frequency {
	case FrequencyMonthly:
		annualWithdrawal *= 12
	case FrequencyQuarterly:
		annualWithdrawal *= 4
	}

	withdrawalRate := annualWithdrawal / portfolioValue

	// Simple heuristic based on 4% rule
	if withdrawalRate <= 0.04 {
		return 0.95
	} else if withdrawalRate <= 0.05 {
		return 0.85
	} else if withdrawalRate <= 0.06 {
		return 0.70
	} else if withdrawalRate <= 0.07 {
		return 0.55
	} else {
		return 0.40
	}
}

// analyzeAllocationChange performs allocation shift analysis
func (e *ScenarioEngine) analyzeAllocationChange(ctx context.Context, positions []Position, portfolioValue float64, config ScenarioConfig) *AllocationChangeResult {
	ac := config.AllocationConfig
	result := &AllocationChangeResult{
		TransitionPlan: make([]TransitionStep, 0),
		GlidePath:      make([]GlidePathPoint, 0),
	}

	// Calculate current allocations
	currentAllocations := make(map[string]float64)
	for _, p := range positions {
		assetClass := p.AssetClass
		if assetClass == "" {
			assetClass = "Other"
		}
		currentAllocations[assetClass] += p.Weight
	}

	if ac.CurrentAllocations != nil {
		currentAllocations = ac.CurrentAllocations
	}

	// Estimate risk for current and target
	result.CurrentRisk = e.estimatePortfolioRisk(currentAllocations)
	result.TargetRisk = e.estimatePortfolioRisk(ac.TargetAllocations)

	// Estimate return for current and target
	result.CurrentReturn = e.estimatePortfolioReturn(currentAllocations)
	result.TargetReturn = e.estimatePortfolioReturn(ac.TargetAllocations)

	// Generate transition plan
	switch ac.TransitionMethod {
	case TransitionImmediate:
		result.TransitionPlan = e.generateImmediateTransition(positions, currentAllocations, ac.TargetAllocations, portfolioValue, config)
	case TransitionGradual:
		result.TransitionPlan = e.generateGradualTransition(positions, currentAllocations, ac.TargetAllocations, portfolioValue, ac.TransitionPeriod, config)
	case TransitionTaxAware:
		result.TransitionPlan = e.generateTaxAwareTransition(positions, currentAllocations, ac.TargetAllocations, portfolioValue, ac.TransitionPeriod, config)
	}

	// Calculate totals from transition plan
	for _, step := range result.TransitionPlan {
		result.TotalTrades += len(step.Trades)
		for _, trade := range step.Trades {
			result.TotalTradeValue += trade.TradeValue
			result.TaxCost += trade.EstimatedTax
			result.TransactionCosts += trade.TransactionCost
		}
	}

	// Generate glide path
	result.GlidePath = e.generateGlidePath(currentAllocations, ac.TargetAllocations, ac.TransitionPeriod, config.AsOfDate)

	return result
}

// estimatePortfolioRisk estimates portfolio volatility based on allocation
func (e *ScenarioEngine) estimatePortfolioRisk(allocations map[string]float64) float64 {
	// Asset class volatilities (annualized)
	volatilities := map[string]float64{
		"Equity":       0.18,
		"Fixed Income": 0.05,
		"Cash":         0.01,
		"Real Estate":  0.12,
		"Commodities":  0.20,
		"Other":        0.10,
	}

	// Simplified: weighted average volatility (ignores correlation)
	totalVol := 0.0
	for assetClass, weight := range allocations {
		vol := volatilities[assetClass]
		if vol == 0 {
			vol = 0.10
		}
		totalVol += weight * vol * vol
	}

	return math.Sqrt(totalVol)
}

// estimatePortfolioReturn estimates expected return based on allocation
func (e *ScenarioEngine) estimatePortfolioReturn(allocations map[string]float64) float64 {
	expectedReturns := map[string]float64{
		"Equity":       0.08,
		"Fixed Income": 0.04,
		"Cash":         0.02,
		"Real Estate":  0.06,
		"Commodities":  0.04,
		"Other":        0.05,
	}

	totalReturn := 0.0
	for assetClass, weight := range allocations {
		ret := expectedReturns[assetClass]
		if ret == 0 {
			ret = 0.05
		}
		totalReturn += weight * ret
	}

	return totalReturn
}

// generateImmediateTransition creates a single-step transition
func (e *ScenarioEngine) generateImmediateTransition(positions []Position, current, target map[string]float64, portfolioValue float64, config ScenarioConfig) []TransitionStep {
	rebalanceConfig := &RebalanceConfig{
		TargetAllocations: target,
		Tolerance:         0.001,
		MinTradeSize:      100,
		TaxAware:          true,
		PreferLongTerm:    true,
	}

	result := e.analyzeRebalance(context.Background(), positions, portfolioValue, ScenarioConfig{
		RebalanceConfig: rebalanceConfig,
	})

	return []TransitionStep{
		{
			StepNumber:     1,
			Date:           config.AsOfDate,
			Allocations:    target,
			Trades:         result.ProposedTrades,
			CumulativeCost: result.TaxCost + result.TransactionCosts,
		},
	}
}

// generateGradualTransition creates a multi-step transition
func (e *ScenarioEngine) generateGradualTransition(positions []Position, current, target map[string]float64, portfolioValue float64, months int, config ScenarioConfig) []TransitionStep {
	steps := make([]TransitionStep, 0, months)

	for month := 1; month <= months; month++ {
		progress := float64(month) / float64(months)

		// Calculate intermediate allocations
		stepAllocations := make(map[string]float64)
		for assetClass := range target {
			currentWeight := current[assetClass]
			targetWeight := target[assetClass]
			stepAllocations[assetClass] = currentWeight + (targetWeight-currentWeight)*progress
		}

		rebalanceConfig := &RebalanceConfig{
			TargetAllocations: stepAllocations,
			Tolerance:         0.02,
			MinTradeSize:      500,
			TaxAware:          true,
		}

		result := e.analyzeRebalance(context.Background(), positions, portfolioValue, ScenarioConfig{
			RebalanceConfig: rebalanceConfig,
		})

		cumulativeCost := 0.0
		if len(steps) > 0 {
			cumulativeCost = steps[len(steps)-1].CumulativeCost
		}
		cumulativeCost += result.TaxCost + result.TransactionCosts

		steps = append(steps, TransitionStep{
StepNumber:     month,
Date:           config.AsOfDate.AddDate(0, month, 0),
Allocations:    stepAllocations,
Trades:         result.ProposedTrades,
CumulativeCost: cumulativeCost,
})
	}

	return steps
}

// generateTaxAwareTransition creates a tax-optimized transition
func (e *ScenarioEngine) generateTaxAwareTransition(positions []Position, current, target map[string]float64, portfolioValue float64, months int, config ScenarioConfig) []TransitionStep {
	// Start with gradual transition
	steps := e.generateGradualTransition(positions, current, target, portfolioValue, months, config)

	// Optimize: defer gains to later, realize losses earlier
	for i := range steps {
		sort.Slice(steps[i].Trades, func(a, b int) bool {
// Prioritize trades with tax benefits (losses)
return steps[i].Trades[a].EstimatedTax < steps[i].Trades[b].EstimatedTax
		})
	}

	return steps
}

// generateGlidePath creates the allocation glide path
func (e *ScenarioEngine) generateGlidePath(current, target map[string]float64, months int, startDate time.Time) []GlidePathPoint {
	points := make([]GlidePathPoint, 0, months+1)

	for month := 0; month <= months; month++ {
		progress := float64(month) / float64(months)

		allocations := make(map[string]float64)
		for assetClass := range target {
			currentWeight := current[assetClass]
			targetWeight := target[assetClass]
			allocations[assetClass] = currentWeight + (targetWeight-currentWeight)*progress
		}

		points = append(points, GlidePathPoint{
Date:        startDate.AddDate(0, month, 0),
Allocations: allocations,
Risk:        e.estimatePortfolioRisk(allocations),
Return:      e.estimatePortfolioReturn(allocations),
})
	}

	return points
}

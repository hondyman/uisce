package scenario

import (
"context"
"time"
)

// getPortfolioPositions fetches portfolio positions from the database
func (e *ScenarioEngine) getPortfolioPositions(ctx context.Context, portfolioID string, asOfDate time.Time) ([]Position, error) {
	query := `
		SELECT 
			h.security_id,
			s.security_name,
			COALESCE(s.asset_class, 'Other') as asset_class,
			COALESCE(s.sector, 'Other') as sector,
			h.market_value,
			h.weight,
			COALESCE(h.cost_basis, h.market_value) as cost_basis,
			h.market_value - COALESCE(h.cost_basis, h.market_value) as unrealized_gl,
			COALESCE(h.acquisition_date, h.holding_date) as acquisition_date,
			COALESCE(h.shares, 0) as shares,
			CASE WHEN h.shares > 0 THEN h.market_value / h.shares ELSE 0 END as price,
			COALESCE(s.avg_daily_volume, 1000000) as avg_daily_volume,
			COALESCE(s.volatility, 0.02) as volatility,
			COALESCE(s.bid_ask_spread, 0.001) as bid_ask_spread
		FROM portfolio_holdings h
		JOIN securities s ON h.security_id = s.security_id
		WHERE h.portfolio_id = $1
		AND h.holding_date = $2
		ORDER BY h.weight DESC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := []Position{}
	for rows.Next() {
		var p Position
		err := rows.Scan(
&p.SecurityID, &p.SecurityName, &p.AssetClass, &p.Sector,
			&p.MarketValue, &p.Weight, &p.CostBasis, &p.UnrealizedGL,
			&p.AcquisitionDate, &p.Shares, &p.Price,
			&p.AvgDailyVolume, &p.Volatility, &p.BidAskSpread,
		)
		if err != nil {
			return nil, err
		}

		// Fetch tax lots for this position
		p.TaxLots, _ = e.getTaxLots(ctx, portfolioID, p.SecurityID)

		positions = append(positions, p)
	}

	return positions, nil
}

// getTaxLots fetches tax lots for a position
func (e *ScenarioEngine) getTaxLots(ctx context.Context, portfolioID, securityID string) ([]TaxLot, error) {
	query := `
		SELECT 
			lot_id,
			acquisition_date,
			shares,
			cost_basis,
			current_value
		FROM tax_lots
		WHERE portfolio_id = $1
		AND security_id = $2
		ORDER BY acquisition_date ASC
	`

	rows, err := e.db.QueryContext(ctx, query, portfolioID, securityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lots := []TaxLot{}
	for rows.Next() {
		var lot TaxLot
		err := rows.Scan(&lot.LotID, &lot.AcquisitionDate, &lot.Shares, &lot.CostBasis, &lot.CurrentValue)
		if err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}

	return lots, nil
}

// generateSummary creates a summary of the scenario analysis
func (e *ScenarioEngine) generateSummary(result *ScenarioResult) *ScenarioSummary {
	summary := &ScenarioSummary{}

	switch result.Config.ScenarioType {
	case RebalanceScenario:
		if r := result.RebalanceResult; r != nil {
			summary.TotalTrades = len(r.ProposedTrades)
			summary.TotalTradeValue = r.TotalBuys + r.TotalSells
			summary.EstimatedCosts = r.TransactionCosts
			summary.TaxImpact = r.TaxCost
			summary.RiskChange = 0 // Same target allocation
		}

	case TradeImpactScenario:
		if r := result.TradeImpactResult; r != nil {
			summary.TotalTrades = len(r.Trades)
			for _, t := range r.Trades {
				summary.TotalTradeValue += t.Trade.NotionalValue
			}
			summary.EstimatedCosts = r.TotalCost
		}

	case TaxOptScenario:
		if r := result.TaxOptResult; r != nil {
			summary.TotalTrades = len(r.HarvestablePositions)
			summary.TaxImpact = -r.ProjectedTaxSavings // Negative = savings
			summary.NetBenefit = r.ProjectedTaxSavings
		}

	case WithdrawalScenario, ContributionScenario:
		if r := result.CashFlowResult; r != nil {
			summary.TotalTrades = len(r.SourceAllocation)
			for _, s := range r.SourceAllocation {
				summary.TaxImpact += s.TaxImpact
			}
		}

	case AllocationScenario:
		if r := result.AllocationResult; r != nil {
			summary.TotalTrades = r.TotalTrades
			summary.TotalTradeValue = r.TotalTradeValue
			summary.EstimatedCosts = r.TransactionCosts
			summary.TaxImpact = r.TaxCost
			summary.RiskChange = r.TargetRisk - r.CurrentRisk
			summary.ExpectedReturn = r.TargetReturn
		}
	}

	// Calculate breakeven period (simplified: costs / annual benefit)
	if summary.NetBenefit > 0 {
		summary.BreakevenPeriod = int(summary.EstimatedCosts / (summary.NetBenefit / 12))
	}

	return summary
}

// generateRecommendations creates actionable recommendations
func (e *ScenarioEngine) generateRecommendations(result *ScenarioResult) []Recommendation {
	recommendations := []Recommendation{}

	switch result.Config.ScenarioType {
	case RebalanceScenario:
		if r := result.RebalanceResult; r != nil {
			// High turnover warning
			if r.TurnoverPct > 20 {
				recommendations = append(recommendations, Recommendation{
Priority:    2,
Category:    "Cost",
Title:       "High Turnover",
Description: "Consider gradual rebalancing to reduce transaction costs",
Impact:      r.TransactionCosts,
Urgency:     "low",
})
			}

			// Tax cost warning
			if r.TaxCost > 1000 {
				recommendations = append(recommendations, Recommendation{
Priority:    1,
Category:    "Tax",
Title:       "Significant Tax Impact",
Description: "Consider tax-loss harvesting to offset realized gains",
Impact:      r.TaxCost,
Urgency:     "medium",
})
			}
		}

	case TaxOptScenario:
		if r := result.TaxOptResult; r != nil {
			// Year-end harvesting
			if len(r.HarvestablePositions) > 0 {
				recommendations = append(recommendations, Recommendation{
Priority:    1,
Category:    "Tax",
Title:       "Tax Loss Harvesting Opportunities",
Description: "Multiple positions available for loss harvesting",
Impact:      r.ProjectedTaxSavings,
Urgency:     "high",
})
			}

			// Warn about wash sales
			if len(r.WashSaleRisks) > 0 {
				recommendations = append(recommendations, Recommendation{
Priority:    1,
Category:    "Compliance",
Title:       "Wash Sale Risk",
Description: "Proposed trades may trigger wash sale rules",
Impact:      0,
Urgency:     "high",
})
			}
		}

	case WithdrawalScenario:
		if r := result.CashFlowResult; r != nil {
			// Sustainability warning
			if r.SustainabilityYears < float64(result.Config.ProjectionYears) {
				recommendations = append(recommendations, Recommendation{
Priority:    1,
Category:    "Planning",
Title:       "Sustainability Concern",
Description: "Portfolio may be depleted before end of projection period",
Impact:      0,
Urgency:     "high",
})
			}

			// Success probability warning
			if r.SuccessProbability < 0.80 {
				recommendations = append(recommendations, Recommendation{
Priority:    2,
Category:    "Planning",
Title:       "Consider Reducing Withdrawals",
Description: "Current withdrawal rate may be unsustainable",
Impact:      0,
Urgency:     "medium",
})
			}
		}

	case AllocationScenario:
		if r := result.AllocationResult; r != nil {
			// Risk change warning
			if r.TargetRisk > r.CurrentRisk*1.2 {
				recommendations = append(recommendations, Recommendation{
Priority:    2,
Category:    "Risk",
Title:       "Increased Portfolio Risk",
Description: "Target allocation significantly increases risk",
Impact:      r.TargetRisk - r.CurrentRisk,
Urgency:     "medium",
})
			}

			// Tax efficiency recommendation
			if r.TaxCost > 5000 {
				recommendations = append(recommendations, Recommendation{
Priority:    1,
Category:    "Tax",
Title:       "Consider Tax-Aware Transition",
Description: "Gradual transition may reduce tax impact",
Impact:      r.TaxCost * 0.3, // Potential savings
Urgency:     "medium",
})
			}
		}
	}

	return recommendations
}

// calculateProjectedValue estimates portfolio value after scenario
func (e *ScenarioEngine) calculateProjectedValue(result *ScenarioResult, currentValue float64) float64 {
	switch result.Config.ScenarioType {
	case RebalanceScenario:
		if r := result.RebalanceResult; r != nil {
			return currentValue - r.TransactionCosts - r.TaxCost
		}

	case TradeImpactScenario:
		if r := result.TradeImpactResult; r != nil {
			return currentValue - r.TotalCost
		}

	case TaxOptScenario:
		if r := result.TaxOptResult; r != nil {
			// Value stays same, but tax benefit realized
			return currentValue
		}

	case WithdrawalScenario:
		if r := result.CashFlowResult; r != nil && len(r.PortfolioProjection) > 0 {
			return r.PortfolioProjection[len(r.PortfolioProjection)-1].Value
		}

	case ContributionScenario:
		if r := result.CashFlowResult; r != nil && len(r.PortfolioProjection) > 0 {
			return r.PortfolioProjection[len(r.PortfolioProjection)-1].Value
		}

	case AllocationScenario:
		if r := result.AllocationResult; r != nil {
			return currentValue - r.TransactionCosts - r.TaxCost
		}
	}

	return currentValue
}

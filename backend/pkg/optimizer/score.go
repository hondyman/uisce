package optimizer

import "math"

// ScorePlan calculates the objective score for a plan: higher is better
func ScorePlan(p Plan, w ScoreWeights) float64 {
	teGain := p.TEAfter // lower TE desired -> convert to gain by negative delta; assume baseline encodes
	// Ideally we compare TE_before - TE_after. Here we assume lower TEAfter is better.
	// Score = w_te * (-TE) + w_tax * (-TaxImpact) - w_cost * Cost
	
	taxAlpha := -p.TaxImpact // negative impact means benefit (tax alpha)
	costPenalty := p.TransCost
	
	return (w.TEWeight * (-teGain)) + (w.TaxAlphaWeight * taxAlpha) - (w.TransCostWeight * costPenalty)
}

// EstimateImpact calculates the tax impact and transaction cost of selected trades
func EstimateImpact(trades []CandidateTrade, lotsByID map[string]Lot, rules TaxRules) (taxImpact float64, transCost float64, shortTermPenalty float64) {
	for _, t := range trades {
		transCost += rules.TransactionCostPerShare * math.Abs(t.Qty)
		if t.Side == "SELL" && len(t.LotIDs) > 0 {
			for _, id := range t.LotIDs {
				lot, ok := lotsByID[id]
				if !ok {
					continue
				}
				// realizing loss decreases tax bill -> negative impact (benefit); gains positive
				impact := realizedImpact(lot)
				taxImpact += impact
				
				if lot.Term == "short" && impact > 0 { // short-term gain
					shortTermPenalty += rules.ShortTermPenaltyWeight * impact
				}
			}
		}
	}
	return
}

func realizedImpact(l Lot) float64 {
	// simple: (market - basis) * qty (positive = gain, negative = loss)
	return (l.MarketPrice - l.CostBasis) * l.Quantity
}

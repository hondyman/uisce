package optimizer

import (
	"math"
	"sort"

	"github.com/google/uuid"
)

// Inputs defines the data required for optimization
type Inputs struct {
	Drift   DriftReport
	Lots    []Lot
	Prices  map[string]float64
	Rules   TaxRules
	Weights ScoreWeights
}

// Optimize generates the best rebalancing plan based on inputs
func Optimize(inputs Inputs) Plan {
	// Step 1: choose harvest lots
	harvest := SelectLossHarvestLots(inputs.Lots, inputs.Rules)
	lotsMap := make(map[string]Lot, len(inputs.Lots))
	for _, l := range inputs.Lots {
		lotsMap[l.LotID] = l
	}

	// Step 2: build overweight sells and underweight buys
	sells, buys := BuildRebalanceLegs(inputs.Drift.Exposures, inputs.Prices, inputs.Rules)

	// Attach lots to sells where symbols match harvest
	// This is a simplified logic: if we are selling a symbol that we also want to harvest,
	// we use the harvest lots first.
	for i := range sells {
		for _, l := range harvest {
			if l.Symbol == sells[i].Symbol {
				// assign lot quantity up to sell qty
				// Note: In a real optimizer, we'd manage lot availability more carefully
				q := math.Min(sells[i].Qty, l.Quantity)
				if q > 0 {
					sells[i].LotIDs = append(sells[i].LotIDs, l.LotID)
				}
			}
		}
	}

	// Step 3: replacements for wash-sale avoidance (if you sold, buy correlated substitutes)
	var soldSyms []string
	for _, s := range sells {
		soldSyms = append(soldSyms, s.Symbol)
	}
	exposureMap := make(map[string]Exposure)
	for _, e := range inputs.Drift.Exposures {
		exposureMap[e.Symbol] = e
	}
	repBuys := BuildReplacementBuys(soldSyms, inputs.Prices, inputs.Rules, exposureMap)

	trades := append(sells, append(repBuys, buys...)...)
	
	// Estimate impact
	taxImpact, transCost, stPenalty := EstimateImpact(trades, lotsMap, inputs.Rules)

	plan := Plan{
		ID:         uuid.New().String(),
		Trades:     trades,
		TEAfter:    math.Max(inputs.Drift.TrackingError-0.5, 0), // placeholder TE reduction model
		TaxImpact:  taxImpact + stPenalty,
		TransCost:  transCost,
		Confidence: 0.75,
		Citations:  []string{"snap_positions_001", "tax_rules_2025-11-22"},
	}
	
	// Run Monte Carlo Simulation
	mcSummary := MonteCarloSimulate(plan, inputs.Lots, inputs.Prices, inputs.Rules, 1000)
	plan.MonteCarlo = mcSummary
	
	// Update confidence based on MC results (e.g., probability of positive tax alpha)
	if mcSummary.MedianTaxImpact < -500 { // Benefit > $500
		plan.Confidence = 0.85
	}

	return plan
}

// RankPlans selects the best plan among candidates based on score
func RankPlans(plans []Plan, w ScoreWeights) Plan {
	sort.Slice(plans, func(i, j int) bool {
		return ScorePlan(plans[i], w) > ScorePlan(plans[j], w)
	})
	if len(plans) == 0 {
		return Plan{}
	}
	return plans[0]
}

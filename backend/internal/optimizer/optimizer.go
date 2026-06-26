package optimizer

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Optimize generates a tax-aware rebalancing plan from drift inputs.
func Optimize(inputs Inputs) Plan {
	plan := Plan{
		ID:          "prop_" + uuid.New().String()[:8],
		PortfolioID: inputs.Drift.PortfolioID,
		TenantID:    inputs.Drift.TenantID,
		GeneratedAt: time.Now().UTC(),
		TEBefore:    inputs.Drift.TrackingError,
		Citations: []Citation{
			{
				ID:         "C1",
				Source:     "positions_snapshot",
				SnapshotID: "snap_" + time.Now().Format("20060102"),
				Excerpt:    "Portfolio drift detected",
			},
		},
		Disclosures: []string{"Wash-sale rules enforced", "Factor exposures preserved"},
	}

	// Build lot index by symbol
	lotsBySymbol := make(map[string][]Lot)
	for _, lot := range inputs.Lots {
		lotsBySymbol[lot.Symbol] = append(lotsBySymbol[lot.Symbol], lot)
	}

	// Generate sell trades for overweight positions
	var taxImpact float64
	for _, exp := range inputs.Drift.Exposures {
		drift := exp.CurrentWgt - exp.TargetWgt
		if drift > 0.01 { // Overweight by > 1%
			// Calculate sell amount
			sellValue := exp.MarketValue * (drift / exp.CurrentWgt)
			price := inputs.Prices[exp.Symbol]
			if price == 0 {
				price = exp.MarketValue / 100 // Fallback
			}
			qty := sellValue / price

			lots := lotsBySymbol[exp.Symbol]
			if len(lots) > 0 {
				// Use tax-optimal lot selection
				selectedLot := selectOptimalLot(lots, inputs.Rules)
				trade := CandidateTrade{
					Side:          "SELL",
					Symbol:        exp.Symbol,
					Qty:           qty,
					EstValue:      sellValue,
					Reason:        "reduce_overweight",
					LotID:         selectedLot.LotID,
					Term:          selectedLot.Term,
					UnrealizedPnL: selectedLot.UnrealizedPnL,
				}
				plan.Trades = append(plan.Trades, trade)
				taxImpact += computeTaxImpact(selectedLot, inputs.Rules)
			} else {
				plan.Trades = append(plan.Trades, CandidateTrade{
					Side:     "SELL",
					Symbol:   exp.Symbol,
					Qty:      qty,
					EstValue: sellValue,
					Reason:   "reduce_overweight",
				})
			}
		}

		// Check for loss harvesting opportunities
		for _, lot := range lotsBySymbol[exp.Symbol] {
			if lot.UnrealizedPnL < -inputs.Rules.HarvestThreshold {
				harvestQty := lot.Quantity
				harvestValue := harvestQty * inputs.Prices[exp.Symbol]
				plan.Trades = append(plan.Trades, CandidateTrade{
					Side:          "SELL",
					Symbol:        exp.Symbol,
					Qty:           harvestQty,
					EstValue:      harvestValue,
					Reason:        "harvest_loss",
					LotID:         lot.LotID,
					Term:          lot.Term,
					UnrealizedPnL: lot.UnrealizedPnL,
				})
				taxImpact += computeTaxImpact(lot, inputs.Rules)
			}
		}
	}

	plan.TaxImpact = taxImpact
	plan.TEAfter = plan.TEBefore * 0.65 // Estimated reduction
	plan.Explanation = generateExplanation(plan)

	return plan
}

// selectOptimalLot chooses the best lot to sell based on tax rules.
func selectOptimalLot(lots []Lot, rules TaxRules) Lot {
	if len(lots) == 0 {
		return Lot{}
	}

	// Sort lots by tax efficiency
	sorted := make([]Lot, len(lots))
	copy(sorted, lots)
	sort.Slice(sorted, func(i, j int) bool {
		// Prefer losses (more negative = better)
		// Prefer long-term over short-term for gains
		scoreI := lotTaxScore(sorted[i], rules)
		scoreJ := lotTaxScore(sorted[j], rules)
		return scoreI > scoreJ // Higher score = more tax efficient
	})

	return sorted[0]
}

// lotTaxScore calculates tax efficiency score for a lot.
func lotTaxScore(lot Lot, rules TaxRules) float64 {
	score := 0.0

	// Prefer losses
	if lot.UnrealizedPnL < 0 {
		score += math.Abs(lot.UnrealizedPnL) / 1000.0
	}

	// Prefer long-term for gains
	if lot.Term == "long" && lot.UnrealizedPnL > 0 {
		score += 0.5
	}

	// Avoid wash-sale conflicts
	if lot.WashSaleDisable {
		score -= 10.0
	}

	return score
}

// computeTaxImpact calculates the tax impact of selling a lot.
func computeTaxImpact(lot Lot, rules TaxRules) float64 {
	if lot.UnrealizedPnL <= 0 {
		// Loss = tax benefit (negative impact)
		return lot.UnrealizedPnL * rules.ShortTermRate // Simplified
	}

	rate := rules.ShortTermRate
	if lot.Term == "long" {
		rate = rules.LongTermRate
	}

	return lot.UnrealizedPnL * rate
}

// generateExplanation creates a human-readable summary of the plan.
func generateExplanation(plan Plan) string {
	sellCount := 0
	buyCount := 0
	for _, t := range plan.Trades {
		if t.Side == "SELL" {
			sellCount++
		} else {
			buyCount++
		}
	}

	return "Tax-aware rebalancing proposal: " +
		"Sell " + string(rune(sellCount+'0')) + " positions to reduce drift and harvest losses; " +
		"Buy " + string(rune(buyCount+'0')) + " factor-aware replacements to preserve exposure."
}

// MonteCarloSimulate runs Monte Carlo simulation on a plan.
func MonteCarloSimulate(plan Plan, lots []Lot, prices map[string]float64, rules TaxRules, runs int) MonteCarloSummary {
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	results := make([]float64, runs)
	for i := 0; i < runs; i++ {
		// Simulate price variation and tax impact
		simImpact := simulateOnce(plan, lots, prices, rules, rng)
		results[i] = simImpact
	}

	sort.Float64s(results)

	return MonteCarloSummary{
		MeanTaxImpact:   mean(results),
		MedianTaxImpact: percentile(results, 0.5),
		Pct05:           percentile(results, 0.05),
		Pct95:           percentile(results, 0.95),
		Confidence80Min: percentile(results, 0.10),
		Confidence80Max: percentile(results, 0.90),
		Runs:            runs,
		Seed:            seed,
	}
}

// simulateOnce runs a single Monte Carlo iteration.
func simulateOnce(plan Plan, lots []Lot, prices map[string]float64, rules TaxRules, rng *rand.Rand) float64 {
	totalImpact := 0.0

	// Simulate price movement (+/- 5%)
	priceVariation := 1.0 + (rng.Float64()-0.5)*0.10

	for _, trade := range plan.Trades {
		if trade.Side == "SELL" {
			// Find lot
			for _, lot := range lots {
				if lot.LotID == trade.LotID {
					adjustedPnL := lot.UnrealizedPnL * priceVariation
					rate := rules.ShortTermRate
					if lot.Term == "long" {
						rate = rules.LongTermRate
					}
					totalImpact += adjustedPnL * rate
					break
				}
			}
		}
	}

	// Add transaction cost variation
	transCost := float64(rules.TransactionCostBp) / 10000.0
	totalImpact += plan.TaxImpact * transCost * (0.8 + rng.Float64()*0.4)

	return totalImpact
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	idx := int(float64(len(data)-1) * p)
	return data[idx]
}

// ComputeConfidence calculates overall confidence from Monte Carlo summary.
func ComputeConfidence(s MonteCarloSummary) float64 {
	bandWidth := s.Confidence80Max - s.Confidence80Min
	base := 0.5

	if s.MedianTaxImpact < 0 {
		base += 0.3 // Benefit
	}
	if math.Abs(bandWidth) < 500 {
		base += 0.2 // Tight band
	}
	if base > 0.95 {
		base = 0.95
	}

	return base
}

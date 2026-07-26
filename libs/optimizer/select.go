package optimizer

import (
	"math"
	"sort"
)

// SelectLossHarvestLots filters loss-harvest candidates within budget and constraints
func SelectLossHarvestLots(lots []Lot, rules TaxRules) []Lot {
	var cands []Lot
	for _, lot := range lots {
		if lot.UnrealizedPNL >= 0 { // only losses
			continue
		}
		if lot.AccountType != "taxable" {
			continue
		}
		// prefer long-term losses first (usually) or short-term depending on strategy
		// Here we prioritize larger losses and long-term
		cands = append(cands, lot)
	}
	
	sort.Slice(cands, func(i, j int) bool {
		// sort by absolute loss desc, prefer long term if losses equal
		li := math.Abs(cands[i].UnrealizedPNL)
		lj := math.Abs(cands[j].UnrealizedPNL)
		if li == lj {
			return cands[i].Term == "long" && cands[j].Term != "long"
		}
		return li > lj
	})

	// enforce budget
	var out []Lot
	budget := rules.HarvestBudgetUSD
	for _, l := range cands {
		abs := math.Abs(l.UnrealizedPNL)
		if abs <= budget {
			out = append(out, l)
			budget -= abs
		}
		if budget <= 0 {
			break
		}
	}
	return out
}

// BuildRebalanceLegs generates overweight reduction trades and replacement buys to maintain target exposure
func BuildRebalanceLegs(exposures []Exposure, price map[string]float64, rules TaxRules) ([]CandidateTrade, []CandidateTrade) {
	var sells, buys []CandidateTrade
	for _, e := range exposures {
		delta := e.CurrentWgt - e.TargetWgt
		if delta > 0.0 && e.MarketValue > 0 {
			// overweight: sell delta
			sellUSD := delta * e.MarketValue
			if sellUSD < rules.MinTradeUSD {
				continue
			}
			qty := sellUSD / price[e.Symbol]
			sells = append(sells, CandidateTrade{
				Side:     "SELL",
				Symbol:   e.Symbol,
				Qty:      qty,
				EstValue: sellUSD,
				Reason:   "reduce_overweight",
			})
		} else if delta < 0.0 && e.MarketValue > 0 {
			buyUSD := -delta * e.MarketValue
			if buyUSD < rules.MinTradeUSD {
				continue
			}
			qty := buyUSD / price[e.Symbol]
			buys = append(buys, CandidateTrade{
				Side:     "BUY",
				Symbol:   e.Symbol,
				Qty:      qty,
				EstValue: buyUSD,
				Reason:   "increase_underweight",
			})
		}
	}
	return sells, buys
}

// BuildReplacementBuys finds allowed substitutes for sold symbols to mitigate wash-sale and preserve exposure
func BuildReplacementBuys(soldSymbols []string, price map[string]float64, rules TaxRules, exposure map[string]Exposure) []CandidateTrade {
	var buys []CandidateTrade
	for _, sym := range soldSymbols {
		repls := rules.AllowedReplacementMap[sym]
		if len(repls) == 0 {
			continue
		}
		// naive: split exposure across first replacement
		rep := repls[0]
		target := exposure[sym]
		
		// If we sold the entire position, we might want to replace the full target value.
		// For rebalancing, we usually replace the amount sold to maintain market exposure.
		// Here we assume we want to maintain the *target* exposure using the substitute.
		// Simplified: buy 50% of target value in substitute
		usd := target.MarketValue * 0.5 
		
		if usd >= rules.MinTradeUSD {
			qty := usd / price[rep]
			buys = append(buys, CandidateTrade{
				Side:     "BUY",
				Symbol:   rep,
				Qty:      qty,
				EstValue: usd,
				Reason:   "replacement_buy",
			})
		}
	}
	return buys
}

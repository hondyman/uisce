package factor

import (
	"sort"
)

// Candidate represents a potential replacement instrument
type Candidate struct {
	Symbol string
	Score  float64
}

// SelectReplacements identifies the best replacement candidates for a target symbol
func SelectReplacements(
	targetSym string,
	univ Universe,
	constraints Constraints,
	w Weights,
	k int,
) []Candidate {
	tMeta, ok := univ.BySymbol[targetSym]
	if !ok {
		return nil
	}

	// Iterate universe and score candidates
	cands := make([]Candidate, 0, len(univ.BySymbol))
	for sym, meta := range univ.BySymbol {
		if sym == targetSym {
			continue
		}
		if constraints.Disallow[sym] {
			continue
		}
		// Correlation fallback to 0 if missing
		rho := 0.0
		if univ.Correlation[targetSym] != nil {
			rho = univ.Correlation[targetSym][sym]
		}
		score := scoreCandidate(tMeta, meta, rho, w)
		cands = append(cands, Candidate{Symbol: sym, Score: score})
	}
	// Sort by descending score and take top k
	sort.Slice(cands, func(i, j int) bool { return cands[i].Score > cands[j].Score })

	if k <= 0 || k > constraints.MaxReplacements {
		k = constraints.MaxReplacements
	}
	if k > len(cands) {
		k = len(cands)
	}
	return cands[:k]
}

// SizeReplacements sizes the replacement trades to meet the desired USD exposure
func SizeReplacements(
	exposure ExposureTarget,
	univ Universe,
	cands []Candidate,
	constraints Constraints,
) []ReplacementTrade {
	var trades []ReplacementTrade
	remaining := exposure.DesiredUSD
	for _, c := range cands {
		if remaining <= 0 {
			break
		}
		price := univ.Prices[c.Symbol]
		if price <= 0 {
			continue
		}
		capUSD := constraints.MaxPerReplacementUSD
		if capUSD <= 0 || capUSD > remaining {
			capUSD = remaining
		}
		if capUSD < constraints.MinTradeUSD {
			continue
		}
		qty := capUSD / price
		trades = append(trades, ReplacementTrade{
			Symbol: c.Symbol,
			Qty:    qty,
			USD:    capUSD,
			Score:  c.Score,
			Reason: "factor_aware_replacement",
		})
		remaining -= capUSD
	}
	return trades
}

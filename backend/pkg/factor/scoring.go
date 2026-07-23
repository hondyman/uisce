package factor

import "math"

// Weights defines the importance of different factors in the scoring function
type Weights struct {
	SectorMatch      float64 // reward same sector
	IndustryMatch    float64 // reward same industry
	FactorSimilarity float64 // reward cosine similarity
	Correlation      float64 // reward high historical correlation
	Liquidity        float64 // reward liquid names
	TransCostPenalty float64 // penalize higher transaction cost
}

// cosineSimilarity computes the cosine similarity between two factor vectors
func cosineSimilarity(a, b FactorVector) float64 {
	var dot, na, nb float64
	for i := range a {
		if i < len(b) {
			dot += a[i] * b[i]
			na += a[i] * a[i]
			nb += b[i] * b[i]
		}
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// scoreCandidate calculates a score for a candidate replacement based on the target instrument
func scoreCandidate(target InstrumentMeta, cand InstrumentMeta, rho float64, w Weights) float64 {
	sector := 0.0
	if cand.Sector == target.Sector {
		sector = 1.0
	}
	industry := 0.0
	if cand.Industry == target.Industry {
		industry = 1.0
	}
	fact := cosineSimilarity(target.Factor, cand.Factor) // [-1,1], typically [0,1]
	liq := cand.LiquidityScore                           // [0,1]
	cost := cand.TransCostPerShare                       // higher is worse

	return w.SectorMatch*sector +
		w.IndustryMatch*industry +
		w.FactorSimilarity*fact +
		w.Correlation*rho +
		w.Liquidity*liq -
		w.TransCostPenalty*cost
}

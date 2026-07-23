package factor

import (
	"sort"
)

// Universe represents the set of candidate securities available for replacement.
type Universe struct {
	Candidates []Candidate `json:"candidates"`
}

// Constraints defines limits on replacement selection.
type Constraints struct {
	MaxCandidates               int     `json:"max_candidates"`
	MinLiquidity                float64 `json:"min_liquidity"`   // Min ADV in USD
	MaxCorrelation              float64 `json:"max_correlation"` // Max correlation with existing holdings
	SectorMustMatch             bool    `json:"sector_must_match"`
	IndustryMustMatch           bool    `json:"industry_must_match"`
	ExcludeRestrictedSecurities bool    `json:"exclude_restricted_securities"`
}

// Weights defines the scoring weights for replacement selection.
type Weights struct {
	SectorMatch      float64 `json:"sector_match"`      // Weight for sector match (0-1)
	IndustryMatch    float64 `json:"industry_match"`    // Weight for industry match (0-1)
	FactorSimilarity float64 `json:"factor_similarity"` // Weight for factor profile similarity
	Correlation      float64 `json:"correlation"`       // Weight for correlation (inverse)
	Liquidity        float64 `json:"liquidity"`         // Weight for liquidity score
}

// DefaultWeights provides sensible defaults for replacement scoring.
var DefaultWeights = Weights{
	SectorMatch:      0.30,
	IndustryMatch:    0.20,
	FactorSimilarity: 0.30,
	Correlation:      0.15,
	Liquidity:        0.10,
}

// Candidate represents a potential replacement security.
type Candidate struct {
	SecurityID      string             `json:"security_id"`
	Ticker          string             `json:"ticker"`
	Name            string             `json:"name"`
	Sector          string             `json:"sector"`
	Industry        string             `json:"industry"`
	FactorExposures map[string]float64 `json:"factor_exposures"`
	AvgDailyVolume  float64            `json:"avg_daily_volume"` // ADV in USD
	Price           float64            `json:"price"`
	IsRestricted    bool               `json:"is_restricted"`
}

// ScoredCandidate extends Candidate with scoring information.
type ScoredCandidate struct {
	Candidate
	Score          float64            `json:"score"`
	ScoreBreakdown map[string]float64 `json:"score_breakdown"`
}

// Replacement represents a selected replacement with allocation.
type Replacement struct {
	Candidate       Candidate `json:"candidate"`
	AllocatedUSD    float64   `json:"allocated_usd"`
	AllocatedShares float64   `json:"allocated_shares"`
	Score           float64   `json:"score"`
}

// SoldPosition represents a position being sold that needs replacement.
type SoldPosition struct {
	SecurityID      string             `json:"security_id"`
	Ticker          string             `json:"ticker"`
	Sector          string             `json:"sector"`
	Industry        string             `json:"industry"`
	FactorExposures map[string]float64 `json:"factor_exposures"`
	SaleProceeds    float64            `json:"sale_proceeds"` // USD to reinvest
}

// SelectReplacements selects the best replacement candidates for a sold position.
func SelectReplacements(
	sold SoldPosition,
	universe Universe,
	constraints Constraints,
	weights Weights,
	existingCorrelations map[string]float64, // Security ID -> correlation with portfolio
) []ScoredCandidate {
	var scored []ScoredCandidate

	for _, candidate := range universe.Candidates {
		// Apply hard constraints
		if constraints.ExcludeRestrictedSecurities && candidate.IsRestricted {
			continue
		}
		if candidate.AvgDailyVolume < constraints.MinLiquidity {
			continue
		}
		if constraints.SectorMustMatch && candidate.Sector != sold.Sector {
			continue
		}
		if constraints.IndustryMustMatch && candidate.Industry != sold.Industry {
			continue
		}

		corr := existingCorrelations[candidate.SecurityID]
		if corr > constraints.MaxCorrelation {
			continue
		}

		// Score the candidate
		score, breakdown := scoreCandidate(candidate, sold, weights, corr)
		scored = append(scored, ScoredCandidate{
			Candidate:      candidate,
			Score:          score,
			ScoreBreakdown: breakdown,
		})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Limit to max candidates
	if constraints.MaxCandidates > 0 && len(scored) > constraints.MaxCandidates {
		scored = scored[:constraints.MaxCandidates]
	}

	return scored
}

// scoreCandidate computes a weighted score for a candidate replacement.
func scoreCandidate(
	candidate Candidate,
	sold SoldPosition,
	weights Weights,
	correlation float64,
) (float64, map[string]float64) {
	breakdown := make(map[string]float64)

	// Sector match: 1.0 if exact match, 0.0 otherwise
	sectorScore := 0.0
	if candidate.Sector == sold.Sector {
		sectorScore = 1.0
	}
	breakdown["sector_match"] = sectorScore * weights.SectorMatch

	// Industry match: 1.0 if exact match, 0.0 otherwise
	industryScore := 0.0
	if candidate.Industry == sold.Industry {
		industryScore = 1.0
	}
	breakdown["industry_match"] = industryScore * weights.IndustryMatch

	// Factor similarity: cosine similarity of factor exposures
	factorScore := computeFactorSimilarity(candidate.FactorExposures, sold.FactorExposures)
	breakdown["factor_similarity"] = factorScore * weights.FactorSimilarity

	// Correlation: inverse score (lower correlation is better)
	// Normalize correlation from [-1,1] to [0,1] where 0 correlation = 1.0 score
	correlationScore := 1.0 - (correlation+1.0)/2.0
	breakdown["correlation"] = correlationScore * weights.Correlation

	// Liquidity: log-scaled score based on ADV
	liquidityScore := computeLiquidityScore(candidate.AvgDailyVolume)
	breakdown["liquidity"] = liquidityScore * weights.Liquidity

	// Total score
	total := 0.0
	for _, v := range breakdown {
		total += v
	}

	return total, breakdown
}

// computeFactorSimilarity calculates cosine similarity between two factor profiles.
func computeFactorSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Get all factor keys
	keys := make(map[string]struct{})
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}

	var dotProduct, normA, normB float64
	for k := range keys {
		va := a[k]
		vb := b[k]
		dotProduct += va * vb
		normA += va * va
		normB += vb * vb
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	// Cosine similarity normalized to [0,1]
	cosSim := dotProduct / (sqrt(normA) * sqrt(normB))
	return (cosSim + 1.0) / 2.0 // Convert from [-1,1] to [0,1]
}

// sqrt is a simple square root approximation for factor similarity.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// computeLiquidityScore returns a [0,1] score based on ADV.
func computeLiquidityScore(adv float64) float64 {
	// Score based on log scale: $1M ADV = 0.5, $10M = 0.75, $100M = 1.0
	if adv <= 0 {
		return 0.0
	}
	// Log10 scale normalized
	score := (log10(adv) - 5) / 3 // $100K (5) to $100M (8)
	if score < 0 {
		return 0.0
	}
	if score > 1 {
		return 1.0
	}
	return score
}

// log10 approximation.
func log10(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// log10(x) = ln(x) / ln(10)
	return ln(x) / 2.302585
}

// ln natural log approximation.
func ln(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method approximation
	n := 0.0
	for x > 2 {
		x /= 2.718281828
		n++
	}
	x--
	result := 0.0
	term := x
	for i := 1; i < 100; i++ {
		if i%2 == 1 {
			result += term / float64(i)
		} else {
			result -= term / float64(i)
		}
		term *= x
		if term < 1e-10 && term > -1e-10 {
			break
		}
	}
	return result + n
}

// SizeReplacements allocates the sale proceeds across selected replacements.
func SizeReplacements(
	candidates []ScoredCandidate,
	totalProceeds float64,
	allocationMethod string, // "equal", "score_weighted", "liquidity_weighted"
) []Replacement {
	if len(candidates) == 0 {
		return nil
	}

	replacements := make([]Replacement, 0, len(candidates))

	switch allocationMethod {
	case "equal":
		perCandidate := totalProceeds / float64(len(candidates))
		for _, c := range candidates {
			shares := perCandidate / c.Price
			replacements = append(replacements, Replacement{
				Candidate:       c.Candidate,
				AllocatedUSD:    perCandidate,
				AllocatedShares: shares,
				Score:           c.Score,
			})
		}

	case "score_weighted":
		totalScore := 0.0
		for _, c := range candidates {
			totalScore += c.Score
		}
		for _, c := range candidates {
			weight := c.Score / totalScore
			allocated := totalProceeds * weight
			shares := allocated / c.Price
			replacements = append(replacements, Replacement{
				Candidate:       c.Candidate,
				AllocatedUSD:    allocated,
				AllocatedShares: shares,
				Score:           c.Score,
			})
		}

	case "liquidity_weighted":
		totalADV := 0.0
		for _, c := range candidates {
			totalADV += c.AvgDailyVolume
		}
		for _, c := range candidates {
			weight := c.AvgDailyVolume / totalADV
			allocated := totalProceeds * weight
			shares := allocated / c.Price
			replacements = append(replacements, Replacement{
				Candidate:       c.Candidate,
				AllocatedUSD:    allocated,
				AllocatedShares: shares,
				Score:           c.Score,
			})
		}

	default:
		// Default to equal
		perCandidate := totalProceeds / float64(len(candidates))
		for _, c := range candidates {
			shares := perCandidate / c.Price
			replacements = append(replacements, Replacement{
				Candidate:       c.Candidate,
				AllocatedUSD:    perCandidate,
				AllocatedShares: shares,
				Score:           c.Score,
			})
		}
	}

	return replacements
}

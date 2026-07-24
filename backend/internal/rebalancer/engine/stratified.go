package engine

// Security represents a tradable asset with its characteristics
type Security struct {
	Ticker     string
	Sector     string
	MarketCap  float64
	Style      string // e.g., "Value", "Growth"
	Liquidity  float64
	Weight     float64 // Weight in the benchmark index
}

// Stratum represents a bucket of securities (e.g., Tech / Large / Growth)
type Stratum struct {
	ID             string
	TargetWeight   float64
	Securities     []Security
}

// StratifiedSampler handles the selection of representative securities
type StratifiedSampler struct {
	// Configuration (e.g., max holdings)
	MaxHoldings int
}

func NewStratifiedSampler(maxHoldings int) *StratifiedSampler {
	return &StratifiedSampler{
		MaxHoldings: maxHoldings,
	}
}

// Sample selects a subset of securities to replicate the benchmark
func (s *StratifiedSampler) Sample(benchmark []Security, restrictions *Restrictions) ([]Security, error) {
	// 0. Apply Restrictions (if any)
	universe := benchmark
	if restrictions != nil {
		universe = restrictions.FilterUniverse(benchmark)
	}

	// 1. Group securities into Strata
	strata := make(map[string]*Stratum)
	
	for _, sec := range universe {
		// Create a key based on dimensions (Sector + Cap + Style)
		// Simplified for this example: just Sector
		key := sec.Sector 
		
		if _, exists := strata[key]; !exists {
			strata[key] = &Stratum{
				ID: key,
				Securities: []Security{},
			}
		}
		strata[key].Securities = append(strata[key].Securities, sec)
		strata[key].TargetWeight += sec.Weight
	}
	
	// 2. Select representatives from each Stratum
	var portfolio []Security
	
	// Sort strata by weight descending to prioritize larger buckets
	// (In a real implementation, we'd be more sophisticated about allocation)
	
	for _, stratum := range strata {
		if len(stratum.Securities) == 0 {
			continue
		}
		
		// Simple selection: Pick the most liquid security in the stratum
		// to represent the entire stratum's weight
		bestSec := stratum.Securities[0]
		for _, sec := range stratum.Securities {
			if sec.Liquidity > bestSec.Liquidity {
				bestSec = sec
			}
		}
		
		// Assign the entire stratum's weight to this representative
		bestSec.Weight = stratum.TargetWeight
		portfolio = append(portfolio, bestSec)
	}
	
	// 3. Normalize weights (just in case)
	totalWeight := 0.0
	for _, sec := range portfolio {
		totalWeight += sec.Weight
	}
	
	if totalWeight > 0 {
		for i := range portfolio {
			portfolio[i].Weight /= totalWeight
		}
	}
	
	return portfolio, nil
}

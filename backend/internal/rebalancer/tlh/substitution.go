package tlh

// SubstitutionStrategy determines what to buy when harvesting a loss
type SubstitutionStrategy struct {
	// Graph of substantially identical securities
	// Map of Ticker -> []SubstituteTicker
	Substitutes map[string][]string
}

func NewSubstitutionStrategy() *SubstitutionStrategy {
	return &SubstitutionStrategy{
		Substitutes: map[string][]string{
			"SPY": {"VOO", "IVV"},
			"VOO": {"SPY", "IVV"},
			"KO":  {"PEP", "XLP"},
			"PEP": {"KO", "XLP"},
			// In reality, this would be loaded from a DB or graph service
		},
	}
}

// GetSubstitute returns the best substitute for a ticker
func (s *SubstitutionStrategy) GetSubstitute(ticker string, excludedTickers map[string]bool) string {
	candidates, ok := s.Substitutes[ticker]
	if !ok {
		return "" // No substitute found
	}

	for _, cand := range candidates {
		if !excludedTickers[cand] {
			return cand
		}
	}
	
	return ""
}

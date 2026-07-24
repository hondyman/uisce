package engine

import "strings"

// Restrictions represents the set of constraints for ESG and values-based investing
type Restrictions struct {
	ExcludedSectors []string `json:"excluded_sectors"`
	ExcludedTickers []string `json:"excluded_tickers"`
	// ESGScoreMin     float64  `json:"esg_score_min"` // Future extension
}

// TaxBudget represents the annual limits on realized gains
type TaxBudget struct {
	MaxRealizedGainsUSD float64 `json:"max_realized_gains_usd"`
	UsedGainsUSD        float64 `json:"used_gains_usd"`
}

// IsRestricted checks if a security is allowed under the given restrictions
func (r *Restrictions) IsRestricted(sec Security) bool {
	// Check Sector Exclusions
	for _, sector := range r.ExcludedSectors {
		if strings.EqualFold(sec.Sector, sector) {
			return true
		}
	}

	// Check Ticker Exclusions
	for _, ticker := range r.ExcludedTickers {
		if strings.EqualFold(sec.Ticker, ticker) {
			return true
		}
	}

	return false
}

// FilterUniverse returns a subset of securities that pass the restrictions
func (r *Restrictions) FilterUniverse(universe []Security) []Security {
	var allowed []Security
	for _, sec := range universe {
		if !r.IsRestricted(sec) {
			allowed = append(allowed, sec)
		}
	}
	return allowed
}

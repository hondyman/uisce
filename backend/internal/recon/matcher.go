package recon

import (
	"math"
	"strings"
	"time"
)

// ExternalTransaction represents a row from a custodian file
type ExternalTransaction struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
}

// InternalLedgerEntry represents a row from our ledger
type InternalLedgerEntry struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"` // Assuming we add currency to ledger_entries or join
	Date        time.Time `json:"date"`     // EffectiveAt
	Description string    `json:"description"` // Account Name or Transaction Ref
}

// MatchResult contains the best match and score
type MatchResult struct {
	ExternalID string
	InternalID string
	Score      int
	IsMatch    bool
}

// ReconMatcher performs fuzzy matching
type ReconMatcher struct{}

func NewReconMatcher() *ReconMatcher {
	return &ReconMatcher{}
}

// FindBestMatch finds the best internal candidate for an external transaction
func (m *ReconMatcher) FindBestMatch(ext ExternalTransaction, candidates []InternalLedgerEntry) MatchResult {
	var bestMatch InternalLedgerEntry
	highestScore := 0

	for _, candidate := range candidates {
		score := 0

		// 1. Exact Amount Match (High Weight)
		// Float comparison with epsilon
		if math.Abs(candidate.Amount-ext.Amount) < 0.01 {
			score += 50
		}

		// 2. Date Proximity (Medium Weight)
		// Higher score for closer dates
		daysDiff := math.Abs(candidate.Date.Sub(ext.Date).Hours() / 24)
		if daysDiff <= 2 {
			score += (10 - int(daysDiff)*2) // Max 10, Min 6
		} else {
			score -= 10 // Penalty for far dates
		}

		// 3. Fuzzy Description Match (Levenshtein Distance)
		// "AMZN MKTPLACE" matches "AMAZON"
		sim := levenshteinRatio(ext.Description, candidate.Description)
		score += int(sim * 40)

		if score > highestScore {
			highestScore = score
			bestMatch = candidate
		}
	}

	// Threshold for "Match"
	isMatch := highestScore > 80

	return MatchResult{
		ExternalID: ext.ID,
		InternalID: bestMatch.ID,
		Score:      highestScore,
		IsMatch:    isMatch,
	}
}

// levenshteinRatio calculates similarity between 0.0 and 1.0
func levenshteinRatio(s1, s2 string) float64 {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	dist := levenshtein(s1, s2)
	maxLen := float64(len(s1))
	if len(s2) > len(s1) {
		maxLen = float64(len(s2))
	}
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - (float64(dist) / maxLen)
}

// levenshtein calculates the edit distance
func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	matrix := make([][]int, n+1)
	for i := range matrix {
		matrix[i] = make([]int, m+1)
	}
	for i := 0; i <= n; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= m; j++ {
		matrix[0][j] = j
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	return matrix[n][m]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

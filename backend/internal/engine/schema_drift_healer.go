package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/governance"
)

// SchemaDriftProposal represents an auto-healed binding proposal
type SchemaDriftProposal struct {
	MissingColumn  string  `json:"missing_column"`
	ProposedColumn string  `json:"proposed_column"`
	Similarity     float64 `json:"similarity_score"`
	TargetTable    string  `json:"target_table"`
}

// SchemaDriftHealer listens for column missing errors and proposes auto-healing governance changes
type SchemaDriftHealer struct {
	governanceSvc *governance.MakerCheckerService
}

func NewSchemaDriftHealer(govSvc *governance.MakerCheckerService) *SchemaDriftHealer {
	return &SchemaDriftHealer{
		governanceSvc: govSvc,
	}
}

// ComputeSimilarity calculates hybrid String Levenshtein + Token Overlap similarity between 0.0 and 1.0
func ComputeSimilarity(s1, s2 string) float64 {
	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)

	if s1Lower == s2Lower {
		return 1.0
	}

	len1, len2 := len(s1Lower), len(s2Lower)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// 1. Levenshtein distance score
	dist := levenshteinDistance(s1Lower, s2Lower)
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}
	levScore := 1.0 - (float64(dist) / float64(maxLen))

	// 2. Token overlap score (split by '_' or spaces)
	tokens1 := strings.FieldsFunc(s1Lower, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	tokens2 := strings.FieldsFunc(s2Lower, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })

	matchCount := 0
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			if t1 == t2 || strings.Contains(t2, t1) || strings.Contains(t1, t2) {
				matchCount++
				break
			}
		}
	}

	tokenScore := 0.0
	if len(tokens1) > 0 {
		tokenScore = float64(matchCount) / float64(len(tokens1))
	}

	// Blend Levenshtein (40%) + Token Overlap (60%)
	return (0.4 * levScore) + (0.6 * tokenScore)
}

func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, min(dp[i][j-1]+1, dp[i-1][j-1]+cost))
		}
	}
	return dp[n][m]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InterceptAndHeal detects schema drift errors, finds best match, and submits proposal to governance
func (h *SchemaDriftHealer) InterceptAndHeal(ctx context.Context, boID, targetTable, missingColumn string, candidateColumns []string) (*governance.CatalogChangeRequest, error) {
	var bestMatch string
	var highestSimilarity float64

	for _, candidate := range candidateColumns {
		sim := ComputeSimilarity(missingColumn, candidate)
		if sim > highestSimilarity {
			highestSimilarity = sim
			bestMatch = candidate
		}
	}

	if highestSimilarity < 0.4 {
		return nil, fmt.Errorf("no candidate column met minimum similarity threshold for missing column '%s'", missingColumn)
	}

	proposal := SchemaDriftProposal{
		MissingColumn:  missingColumn,
		ProposedColumn: bestMatch,
		Similarity:     highestSimilarity,
		TargetTable:    targetTable,
	}

	payloadBytes, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema drift proposal: %w", err)
	}

	justification := fmt.Sprintf("Auto-healing schema drift: replace missing column '%s' with '%s' (similarity: %.2f)", missingColumn, bestMatch, highestSimilarity)

	if h.governanceSvc != nil {
		return h.governanceSvc.SubmitProposal(ctx, boID, payloadBytes, justification)
	}

	return nil, nil
}
